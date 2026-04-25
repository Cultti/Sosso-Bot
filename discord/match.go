package discord

import (
	"fmt"
	"log"
	"sosso/db"
	"sosso/faceit"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var ownerNotifyMu sync.Mutex
var ownerNotifyLastSent = map[string]time.Time{}

var mapNameTranslations = map[string]string{
	"de_dust":     "Dust II",
	"de_ancient":  "Ancient",
	"de_anubis":   "Anubis",
	"de_inferno":  "Inferno",
	"de_mirage":   "Mirage",
	"de_nuke":     "Nuke",
	"de_overpass": "Overpass",
	"de_cache":    "Cache",
}

func SendMessageInfo(s *discordgo.Session, matchId, league string) {
	const maxTries = 240
	const retryDelay = time.Minute

	var match *faceit.MatchData
	var err error

	for attempt := 1; attempt <= maxTries; attempt++ {
		retryReason := ""

		match, err = faceit.FetchMatchInfo(matchId)
		if err != nil {
			retryReason = err.Error()
		} else if match == nil {
			retryReason = "got nil match"
		} else if !isSeriesComplete(match) {
			retryReason = "series data incomplete"
		}

		if retryReason == "" {
			break
		}

		log.Printf("Error fetching match info %s (attempt %d/%d): %s", matchId, attempt, maxTries, retryReason)
		if attempt == maxTries {
			if match == nil {
				log.Printf("Error: failed to fetch match %s after %d attempts", matchId, maxTries)
				return
			}
			log.Printf("Error: match %s still missing complete series data after %d attempts", matchId, maxTries)
			return
		}
		time.Sleep(retryDelay)
	}

	if match == nil || !isSeriesComplete(match) {
		log.Printf("Match %s data incomplete, skipping send", matchId)
		return
	}

	embed := buildMatchEmbed(match, league)
	subs, err := db.GetSubscriptionsByLeague(league)
	if err != nil || subs == nil || len(*subs) == 0 {
		return
	}
	for _, sub := range *subs {
		if _, err := s.ChannelMessageSendEmbed(sub.ChannelID, embed); err != nil {
			log.Printf("Error sending embed to channel %s: %v", sub.ChannelID, err)
			notifyGuildOwnerSendFailure(s, sub.GuildID, sub.ChannelID)
		}
	}
}

func isSeriesComplete(m *faceit.MatchData) bool {
	if m == nil || len(m.Rounds) == 0 {
		return false
	}

	bestOf := parseBestOf(m)
	if bestOf <= 0 {
		bestOf = 2
	}

	// BO2 always has two maps. BO3 can end at 2-0 or go to a third map.
	if bestOf == 2 {
		return len(m.Rounds) >= 2
	}
	if bestOf >= 3 {
		requiredWins := bestOf/2 + 1
		wins := map[string]int{}

		for _, r := range m.Rounds {
			if len(r.Teams) < 2 {
				continue
			}
			for _, t := range r.Teams {
				w, _ := strconv.Atoi(t.TeamStats.TeamWin)
				if w > 0 {
					wins[t.TeamID] += w
				}
			}
		}

		for _, w := range wins {
			if w >= requiredWins {
				return len(m.Rounds) >= 2
			}
		}
		return len(m.Rounds) >= bestOf
	}

	return len(m.Rounds) >= 2
}

func parseBestOf(m *faceit.MatchData) int {
	for _, r := range m.Rounds {
		if strings.TrimSpace(r.BestOf) == "" {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimSpace(r.BestOf)); err == nil {
			return n
		}
	}
	return 0
}

func notifyGuildOwnerSendFailure(s *discordgo.Session, guildID, channelID string) {
	if s == nil || strings.TrimSpace(guildID) == "" || strings.TrimSpace(channelID) == "" {
		return
	}

	// Resolve guild (prefer cache/state, fallback to API)
	guild, err := s.State.Guild(guildID)
	if err != nil || guild == nil {
		guild, err = s.Guild(guildID)
		if err != nil || guild == nil {
			log.Printf("Failed to resolve guild %s for send-failure notification: %v", guildID, err)
			return
		}
	}

	ownerID := strings.TrimSpace(guild.OwnerID)
	if ownerID == "" {
		return
	}

	// Rate limit: max 1 message / hour per guild owner.
	now := time.Now()
	ownerNotifyMu.Lock()
	if last, ok := ownerNotifyLastSent[ownerID]; ok {
		if now.Sub(last) < time.Hour {
			ownerNotifyMu.Unlock()
			return
		}
	}
	// Set immediately so concurrent failures won't double-send.
	ownerNotifyLastSent[ownerID] = now
	ownerNotifyMu.Unlock()

	dm, err := s.UserChannelCreate(ownerID)
	if err != nil || dm == nil {
		log.Printf("Failed to create DM channel for guild owner %s (guild %s): %v", ownerID, guildID, err)
		return
	}

	msg := fmt.Sprintf(
		"⚠ En saanut lähetettyä tilausilmoitusta kanavaan <#%s> palvelimella **%s**. Tarkista minun käyttöoikeuteni kanavaan (View Channel / Send Messages / Embed Links).",
		channelID,
		guild.Name,
	)

	if _, err := s.ChannelMessageSend(dm.ID, msg); err != nil {
		log.Printf("Failed to DM guild owner %s about channel send failure (guild %s, channel %s): %v", ownerID, guildID, channelID, err)
	}
}

func buildMatchEmbed(m *faceit.MatchData, league string) *discordgo.MessageEmbed {
	if len(m.Rounds) == 0 {
		return &discordgo.MessageEmbed{
			Title:       "Match info unavailable",
			Description: "No rounds were found for this match.",
			Color:       0xff0000,
		}
	}

	competitionID := matchCompetitionID(m)

	// Use first round just for team names (IDs should be stable)
	if len(m.Rounds[0].Teams) < 2 {
		return &discordgo.MessageEmbed{
			Title:       "Match info unavailable",
			Description: "Not enough team data.",
			Color:       0xff0000,
		}
	}
	team1 := m.Rounds[0].Teams[0].TeamStats.Team
	team2 := m.Rounds[0].Teams[1].TeamStats.Team
	teamIDs := map[string]string{
		team1: m.Rounds[0].Teams[0].TeamID,
		team2: m.Rounds[0].Teams[1].TeamID,
	}

	// Stats containers
	wins := map[string]int{team1: 0, team2: 0}
	roundsFor := map[string]int{team1: 0, team2: 0}
	roundsAgainst := map[string]int{team1: 0, team2: 0}

	// Player sets
	seen1 := make(map[string]bool)
	seen2 := make(map[string]bool)
	var players1, players2 []string

	// Collect map fields and gather stats
	var mapFields []*discordgo.MessageEmbedField
	for i, r := range m.Rounds {
		if len(r.Teams) < 2 {
			continue
		}

		t1 := r.Teams[0]
		t2 := r.Teams[1]

		// Scores
		s1, _ := strconv.Atoi(t1.TeamStats.FinalScore)
		s2, _ := strconv.Atoi(t2.TeamStats.FinalScore)

		// Wins
		w1, _ := strconv.Atoi(t1.TeamStats.TeamWin)
		w2, _ := strconv.Atoi(t2.TeamStats.TeamWin)
		wins[t1.TeamStats.Team] += w1
		wins[t2.TeamStats.Team] += w2

		// Round totals
		roundsFor[t1.TeamStats.Team] += s1
		roundsFor[t2.TeamStats.Team] += s2
		roundsAgainst[t1.TeamStats.Team] += s2
		roundsAgainst[t2.TeamStats.Team] += s1

		// Create one field per map: map name as field name, score + winner as value
		var winnerTeamName string
		if r.RoundStats.Winner != "" {
			// Find the team name by matching team ID with winner ID
			switch r.RoundStats.Winner {
			case t1.TeamID:
				winnerTeamName = t1.TeamStats.Team
			case t2.TeamID:
				winnerTeamName = t2.TeamStats.Team
			default:
				winnerTeamName = "Unknown"
			}
		} else {
			winnerTeamName = "Tie"
		}

		replayURL := fmt.Sprintf("https://replay2.pappa.aukko.net/player?faceit_match_id=%s&map_id=%d", m.MatchID, i+1)
		translatedMapName := translateMapName(r.RoundStats.Map)
		mapFieldName := fmt.Sprintf("%s %s:%s", translatedMapName, t1.TeamStats.FinalScore, t2.TeamStats.FinalScore)
		mapFieldValue := fmt.Sprintf("🏆 %s\n[2D Demo](%s)", winnerTeamName, replayURL)

		mapFields = append(mapFields, &discordgo.MessageEmbedField{
			Name:   mapFieldName,
			Value:  mapFieldValue,
			Inline: true,
		})

		// Players (avoid duplicates)
		for _, p := range t1.Players {
			key := p.PlayerID
			if key == "" {
				key = p.Nickname
			}
			if seen1[key] {
				continue
			}
			seen1[key] = true

			if competitionID != "" && p.PlayerID != "" {
				players1 = append(players1, fmt.Sprintf("- %s", markdownCodeLink(p.Nickname, playerURL(p.PlayerID, competitionID))))
			} else {
				players1 = append(players1, fmt.Sprintf("- %s", p.Nickname))
			}
		}
		for _, p := range t2.Players {
			key := p.PlayerID
			if key == "" {
				key = p.Nickname
			}
			if seen2[key] {
				continue
			}
			seen2[key] = true

			if competitionID != "" && p.PlayerID != "" {
				players2 = append(players2, fmt.Sprintf("- %s", markdownCodeLink(p.Nickname, playerURL(p.PlayerID, competitionID))))
			} else {
				players2 = append(players2, fmt.Sprintf("- %s", p.Nickname))
			}
		}
	}

	// Create prominent match result display
	team1Display := team1
	team2Display := team2
	if competitionID != "" {
		if id := teamIDs[team1]; id != "" {
			team1Display = markdownCodeLink(team1, teamURL(id, competitionID))
		}
		if id := teamIDs[team2]; id != "" {
			team2Display = markdownCodeLink(team2, teamURL(id, competitionID))
		}
	}

	var resultLine string
	if wins[team1] > wins[team2] {
		resultLine = fmt.Sprintf("🏆 **%s** %d - %d **%s** 💔", team1Display, wins[team1], wins[team2], team2Display)
	} else if wins[team1] < wins[team2] {
		resultLine = fmt.Sprintf("💔 **%s** %d - %d **%s** 🏆", team1Display, wins[team1], wins[team2], team2Display)
	} else {
		resultLine = fmt.Sprintf("🤝 **%s** %d - %d **%s** 🤝", team1Display, wins[team1], wins[team2], team2Display)
	}

	// Round diff
	diff1 := roundsFor[team1] - roundsAgainst[team1]
	diff2 := roundsFor[team2] - roundsAgainst[team2]

	title := fmt.Sprintf("%s vs %s", team1, team2)

	var description string
	url, err := LeagueToURL(competitionID)
	if err != nil {
		description = fmt.Sprintf("## %s\n\n%s", resultLine, league)
	} else {
		description = fmt.Sprintf("## %s\n\n[%s](%s)", resultLine, league, url)
	}

	// Build all fields: maps first, then empty field, then teams
	allFields := mapFields

	// Add empty field to force teams to next row
	allFields = append(allFields, &discordgo.MessageEmbedField{
		Name:   "\u200b", // Zero-width space for invisible field name
		Value:  "\u200b", // Zero-width space for invisible field value
		Inline: false,    // Non-inline to force line break
	})

	allFields = append(allFields, &discordgo.MessageEmbedField{
		Name: team1,
		Value: fmt.Sprintf(
			"Rounds: **%d** (Diff: **%+d**)\n%s",
			roundsFor[team1],
			diff1,
			strings.Join(players1, "\n"),
		),
		Inline: true,
	})
	allFields = append(allFields, &discordgo.MessageEmbedField{
		Name: team2,
		Value: fmt.Sprintf(
			"Rounds: **%d** (Diff: **%+d**)\n%s",
			roundsFor[team2],
			diff2,
			strings.Join(players2, "\n"),
		),
		Inline: true,
	})

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x2ecc71,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Powered by ArmaFinland.fi",
			IconURL: "https://armafinland.fi/logot/images/armafin-logo-200px.png",
		},
		URL:    m.FaceitURL(),
		Fields: allFields,
	}
}

// LeagueToURL returns the league (division) stats page URL by competition ID.
func LeagueToURL(competitionID string) (string, error) {
	if strings.TrimSpace(competitionID) == "" {
		return "", fmt.Errorf("missing competition ID")
	}
	return fmt.Sprintf("https://pappa.aukko.net/division/%s", competitionID), nil
}

func teamURL(teamID, competitionID string) string {
	return fmt.Sprintf("https://pappa.aukko.net/team/%s/%s", competitionID, teamID)
}

func playerURL(playerID, competitionID string) string {
	return fmt.Sprintf("https://pappa.aukko.net/player/%s/%s", competitionID, playerID)
}

func markdownCodeLink(name, url string) string {
	escapedName := strings.ReplaceAll(name, "`", "\\`")
	return fmt.Sprintf("[`%s`](%s)", escapedName, url)
}

func matchCompetitionID(m *faceit.MatchData) string {
	for _, r := range m.Rounds {
		if r.CompetitionID != nil && strings.TrimSpace(*r.CompetitionID) != "" {
			return strings.TrimSpace(*r.CompetitionID)
		}
	}
	return ""
}

func translateMapName(mapKey string) string {
	normalized := strings.ToLower(strings.TrimSpace(mapKey))
	if name, ok := mapNameTranslations[normalized]; ok {
		return name
	}
	return mapKey
}
