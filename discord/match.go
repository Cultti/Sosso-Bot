package discord

import (
	"fmt"
	"regexp"
	"sosso/db"
	"sosso/faceit"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func SendMessageInfo(s *discordgo.Session, matchId, league string) {
	const maxTries = 10
	const retryDelay = time.Minute

	var match *faceit.MatchData
	var err error

	for attempt := 1; attempt <= maxTries; attempt++ {
		match, err = faceit.FetchMatchInfo(matchId)
		if err != nil {
			fmt.Printf("Error fetching match info (attempt %d/%d): %v\n", attempt, maxTries, err)
			return
		}
		if len(match.Rounds) == 2 {
			break
		}
		if attempt < maxTries {
			time.Sleep(retryDelay)
		} else {
			fmt.Printf("Error: match %s still missing rounds after %d attempts\n", matchId, maxTries)
			return
		}
	}

	if match == nil || len(match.Rounds) != 2 {
		fmt.Printf("Match %s data incomplete, skipping send\n", matchId)
		return
	}

	embed := buildMatchEmbed(match, league)
	subs, err := db.GetSubscriptionsByLeague(league)
	if err != nil || subs == nil || len(*subs) == 0 {
		return
	}
	for _, sub := range *subs {
		if _, err := s.ChannelMessageSendEmbed(sub.ChannelID, embed); err != nil {
			fmt.Printf("Error sending embed to channel %s: %v\n", sub.ChannelID, err)
		}
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

	// Stats containers
	wins := map[string]int{team1: 0, team2: 0}
	roundsFor := map[string]int{team1: 0, team2: 0}
	roundsAgainst := map[string]int{team1: 0, team2: 0}

	// Player sets
	seen1 := make(map[string]bool)
	seen2 := make(map[string]bool)
	var players1, players2 []string

	// Build map lines and gather players
	var mapLines []string
	for _, r := range m.Rounds {
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

		// Map line
		mapLines = append(mapLines, fmt.Sprintf("`%s`  %d:%d", r.RoundStats.Map, s1, s2))

		// Players (avoid duplicates)
		for _, p := range t1.Players {
			if !seen1[p.Nickname] {
				seen1[p.Nickname] = true
				players1 = append(players1, fmt.Sprintf("- %s", p.Nickname))
			}
		}
		for _, p := range t2.Players {
			if !seen2[p.Nickname] {
				seen2[p.Nickname] = true
				players2 = append(players2, fmt.Sprintf("- %s", p.Nickname))
			}
		}
	}

	// Decide emojis
	var emoji1, emoji2 string
	switch {
	case wins[team1] > wins[team2]:
		emoji1, emoji2 = "🏆", "💔"
	case wins[team1] < wins[team2]:
		emoji1, emoji2 = "💔", "🏆"
	default:
		emoji1, emoji2 = "🤝", "🤝"
	}

	// Round diff
	diff1 := roundsFor[team1] - roundsAgainst[team1]
	diff2 := roundsFor[team2] - roundsAgainst[team2]

	title := fmt.Sprintf("%s %s vs %s %s", emoji1, team1, team2, emoji2)

	var description string
	url, err := LeagueToURL(league)
	if err != nil {
		description = league
	} else {
		description = fmt.Sprintf("%s\n[Unofficial stats](%s)", league, url)
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x2ecc71,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Powered by ArmaFinland.fi",
			IconURL: "https://armafinland.fi/logot/images/armafin-logo-200px.png",
		},
		URL: m.FaceitURL(),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Maps & Scores",
				Value: strings.Join(mapLines, "\n"),
			},
			{
				Name: team1,
				Value: fmt.Sprintf(
					"Rounds: **%d**\nDiff: **%+d**\n\n%s",
					roundsFor[team1],
					diff1,
					strings.Join(players1, "\n"),
				),
				Inline: true,
			},
			{
				Name: team2,
				Value: fmt.Sprintf(
					"Rounds: **%d**\nDiff: **%+d**\n\n%s",
					roundsFor[team2],
					diff2,
					strings.Join(players2, "\n"),
				),
				Inline: true,
			},
		},
	}
}

// LeagueToURL converts league name like
//
//	"20 Divisioona S11" or "Mestaruussarja S11 Playoffs"
//
// into its stats page URL.
func LeagueToURL(name string) (string, error) {
	base := "https://tuntematonjr.github.io/Pappaliiga-statsit"

	// Normalize string
	s := strings.TrimSpace(strings.ToLower(name))

	// Regex to capture: [division] [divisioona] S[season] (Playoffs)?
	re := regexp.MustCompile(`(?i)^(mestaruussarja|(\d+)\s+divisioona)\s+s(\d+)(?:\s+playoffs)?$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return "", fmt.Errorf("unrecognized league name: %q", name)
	}

	var division string
	if strings.HasPrefix(matches[1], "mestaruussarja") {
		division = "0"
	} else {
		division = matches[2]
	}
	season := matches[3]

	url := fmt.Sprintf("%s/div%s-s%s.html", base, division, season)
	return url, nil
}
