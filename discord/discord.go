package discord

import (
	"fmt"
	"log"
	"os"
	"sosso/db"
	"sosso/faceit"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var allowed = map[string]bool{
	"336110846306549760": true,
	"106011141670461440": true,
}

type Bot struct {
	Session *discordgo.Session
}

var MatchChannel string
var championshipList []string

func Start(championships *[]faceit.ChampionshipItem) (*discordgo.Session, error) {
	for _, item := range *championships {
		championshipList = append(championshipList, item.Name)
	}

	token := os.Getenv("DISCORD_BOT_TOKEN")
	MatchChannel = os.Getenv("DISCORD_MATCH_CHANNEL")

	if token == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN and DISCORD_GUILD_ID must be set")
	}

	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	sess.AddHandler(interactionCreate)
	sess.AddHandler(interactionHandle)
	sess.AddHandler(func(s *discordgo.Session, g *discordgo.GuildCreate) {
		fmt.Println("Bot joined guild:", g.ID)
		registerCommands(s, g.ID)
	})

	if err := sess.Open(); err != nil {
		return nil, err
	}

	// register commands
	registerAllCommands(sess)

	return sess, nil
}

func registerAllCommands(s *discordgo.Session) {
	for _, g := range s.State.Guilds {
		fmt.Println("Registering commands for guild:", g.ID)
		registerCommands(s, g.ID)
	}
}

func registerCommands(s *discordgo.Session, guildID string) {
	cmds := []*discordgo.ApplicationCommand{
		{
			Name:        "pelipaiva",
			Description: "Luo viikon pelipäivä-äänestys",
			Options: []*discordgo.ApplicationCommandOption{{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "vihollinen",
				Description: "Kenen kanssa pelataan?",
				Required:    true,
			}},
		},
		{
			Name:        "harkka",
			Description: "Luo harkkapäivä-äänestys",
			Options: []*discordgo.ApplicationCommandOption{{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "kuvaus",
				Description: "Harkan kuvaus",
				Required:    true,
			}},
		},
	}
	for _, c := range cmds {
		if _, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, c); err != nil {
			fmt.Println("Command create error:", err)
		}
	}

	s.ApplicationCommandCreate(s.State.User.ID, guildID, &discordgo.ApplicationCommand{
		Name:        "unsubscribe",
		Description: "Unsubscribe to Pappaliiga matches",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "liiga",
			Description: "Liiga muodossa '20 Divisioona S11'",
			Required:    false,
		}},
	})

	s.ApplicationCommandCreate(s.State.User.ID, guildID, &discordgo.ApplicationCommand{
		Name:        "subscriptions",
		Description: "Manage subscriptions for this channel",
	})
}

// userID_channelID -> menuID -> []selectedValues
var tempSelections = map[string]map[string][]string{}

func interactionHandle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	data := i.MessageComponentData()
	userKey := i.Member.User.ID + "_" + i.ChannelID
	menuID := data.CustomID
	guildID := i.GuildID
	channelID := i.ChannelID

	// Initialize map
	if _, exists := tempSelections[userKey]; !exists {
		tempSelections[userKey] = make(map[string][]string)
	}

	// ---- Menu selection ----
	if strings.HasPrefix(menuID, "champ_") {
		// Save selections for this menu
		tempSelections[userKey][menuID] = data.Values

		// Merge all menu selections
		merged := []string{}
		for _, vals := range tempSelections[userKey] {
			merged = append(merged, vals...)
		}

		// Rebuild menus
		sortedList := sortChampionships(championshipList)
		menus := buildSelectMenus(sortedList, merged)

		// Append Save button
		saveButton := discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "💾 Save",
					Style:    discordgo.PrimaryButton,
					CustomID: "save_subscriptions",
				},
			},
		}
		menus = append(menus, saveButton)

		// Update ephemeral message
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    "✔ Valittu! Paina Save kun valmis.",
				Components: menus,
			},
		})
		return
	}

	// ---- Save button ----
	if menuID == "save_subscriptions" {
		// Merge all menu selections
		finalSelections := make(map[string]struct{})
		for _, vals := range tempSelections[userKey] {
			for _, v := range vals {
				finalSelections[v] = struct{}{}
			}
		}

		selectedSlice := make([]string, 0, len(finalSelections))
		for v := range finalSelections {
			selectedSlice = append(selectedSlice, v)
		}

		// Clear all subscriptions for this channel
		_, _ = db.DeleteSubscriptionsByGuildChannel(guildID, channelID, "")

		// Add new selections
		for _, league := range selectedSlice {
			_ = db.CreateSubscription(&db.Subscription{
				GuildID:   guildID,
				ChannelID: channelID,
				League:    league,
			})
		}

		// Clear temporary selections
		delete(tempSelections, userKey)

		// Update ephemeral message
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    fmt.Sprintf("✔ Tilattu %d sarjaa!", len(selectedSlice)),
				Components: []discordgo.MessageComponent{},
			},
		})
	}
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	userID := i.Member.User.ID
	if i.User != nil {
		userID = i.User.ID
	}

	isAllowed := allowed[userID]

	if i.GuildID != "" && !isAllowed {
		// Fetch guild info to check owner
		guild, err := s.State.Guild(i.GuildID)
		if err != nil {
			guild, err = s.Guild(i.GuildID) // fallback to API call if not in state
			if err != nil {
				log.Println("Failed to fetch guild info:", err)
			}
		}

		if guild != nil && guild.OwnerID == userID {
			isAllowed = true
		}
	}

	if !isAllowed {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Et voi käyttää tätä komentoa.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if i.ApplicationCommandData().Name == "subscriptions" {
		handleSubscriontionsCommand(s, i)
		return
	}

	if i.ApplicationCommandData().Name == "unsubscribe" {
		handleUnsubscribeCommand(s, i)
		return
	}

	switch i.ApplicationCommandData().Name {
	case "pelipaiva":
		vihollinen := i.ApplicationCommandData().Options[0].StringValue()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📊 Luodaan viikon pelipäivä -äänestys...",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		createPelipaivaPoll(s, i.ChannelID, vihollinen)

	case "harkka":
		kuvaus := i.ApplicationCommandData().Options[0].StringValue()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📊 Luodaan harkkapäivä -äänestys...",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		createHarkkaPoll(s, i.ChannelID, kuvaus)
	}
}

func handleSubscriontionsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userKey := i.Member.User.ID + "_" + i.ChannelID

	// Always create a new tempSelections map for this user & channel
	tempSelections[userKey] = make(map[string][]string)

	// Load current subscriptions from DB
	existingSubs, err := db.GetSubscriptionsByGuildChannel(i.GuildID, i.ChannelID)
	var current []string
	if err == nil {
		current = make([]string, len(*existingSubs))
		for idx, s := range *existingSubs {
			current[idx] = s.League
		}
	}

	// Pre-fill each menu chunk
	const chunkSize = 25
	for i := 0; i < len(championshipList); i += chunkSize {
		menuID := fmt.Sprintf("champ_%d", i/chunkSize)

		// Select the items from current subscriptions that belong to this chunk
		end := i + chunkSize
		if end > len(championshipList) {
			end = len(championshipList)
		}
		chunk := championshipList[i:end]

		selectedInChunk := []string{}
		for _, name := range chunk {
			for _, sel := range current {
				if sel == name {
					selectedInChunk = append(selectedInChunk, sel)
					break
				}
			}
		}

		tempSelections[userKey][menuID] = selectedInChunk
	}

	// Merge all current selections for initial display
	merged := []string{}
	for _, vals := range tempSelections[userKey] {
		merged = append(merged, vals...)
	}

	// Build select menus
	sortedList := sortChampionships(championshipList)
	menus := buildSelectMenus(sortedList, merged)

	// Append Save button
	saveButton := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "💾 Save",
				Style:    discordgo.PrimaryButton,
				CustomID: "save_subscriptions",
			},
		},
	}
	menus = append(menus, saveButton)

	// Respond ephemerally
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "Valitse mestaruuksia:",
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: menus,
		},
	})
}

func buildSelectMenus(championshipList []string, currentSelections []string) []discordgo.MessageComponent {
	// Build a set for fast lookup
	selectedSet := make(map[string]struct{}, len(currentSelections))
	for _, s := range currentSelections {
		selectedSet[s] = struct{}{}
	}

	const chunkSize = 25
	var menus []discordgo.MessageComponent

	min := 0

	for i := 0; i < len(championshipList); i += chunkSize {
		end := i + chunkSize
		if end > len(championshipList) {
			end = len(championshipList)
		}

		opts := make([]discordgo.SelectMenuOption, 0, end-i)
		for _, name := range championshipList[i:end] {
			_, isSelected := selectedSet[name] // true if currently selected
			opts = append(opts, discordgo.SelectMenuOption{
				Label:   name,
				Value:   name,
				Default: isSelected,
			})
		}

		menus = append(menus,
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    fmt.Sprintf("champ_%d", i/chunkSize),
						Placeholder: "Valitse sarjoja…",
						MinValues:   &min,
						MaxValues:   len(opts),
						Options:     opts,
					},
				},
			})
	}

	return menus
}

func handleUnsubscribeCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "⚒ Käsitellään pyyntöä...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	guildId := i.GuildID
	channelId := i.ChannelID

	var league string
	if len(i.ApplicationCommandData().Options) == 1 {
		league = i.ApplicationCommandData().Options[0].StringValue()
	}

	deletedLeagues, err := db.DeleteSubscriptionsByGuildChannel(guildId, channelId, league)
	if err != nil {
		content := "❌ Tilauksen peruminen epäonnistui"

		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	} else if len(deletedLeagues) == 0 {
		content := "❔ Tilauksia ei löytynyt"

		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
	} else {
		content := "✔ Poistetut tilaukset: " + strings.Join(deletedLeagues, ", ")

		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
	}
}

func createPelipaivaPoll(s *discordgo.Session, channelID, vihollinen string) {
	now := time.Now()
	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}
	currentMonday := now.AddDate(0, 0, offset)
	nextMonday := currentMonday.AddDate(0, 0, 7)

	weekdayFi := []string{"Maanantai", "Tiistai", "Keskiviikko", "Torstai", "Perjantai", "Lauantai", "Sunnuntai"}
	emojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣"}

	var answers []discordgo.PollAnswer
	for i := 0; i < 7; i++ {
		day := nextMonday.AddDate(0, 0, i)
		answers = append(answers, discordgo.PollAnswer{
			Media: &discordgo.PollMedia{
				Text:  fmt.Sprintf("%s %d.%d.", weekdayFi[i], day.Day(), int(day.Month())),
				Emoji: &discordgo.ComponentEmoji{Name: emojis[i]},
			},
		})
	}
	answers = append(answers, discordgo.PollAnswer{
		Media: &discordgo.PollMedia{Text: "Ei käy", Emoji: &discordgo.ComponentEmoji{Name: "❌"}},
	})

	_, week := nextMonday.ISOWeek()
	description := fmt.Sprintf("Viikon %d pelipäivä. Vihollinen %s.", week, vihollinen)

	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:         "@everyone",
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeEveryone}},
		Poll: &discordgo.Poll{
			Question:         discordgo.PollMedia{Text: description},
			AllowMultiselect: true,
			Answers:          answers,
			Duration:         168,
		},
	})
	if err != nil {
		fmt.Println("poll send error:", err)
	}
}

func createHarkkaPoll(s *discordgo.Session, channelID, kuvaus string) {
	days := []string{"Tiistai", "Keskiviikko", "Torstai", "Perjantai", "Lauantai"}
	var answers []discordgo.PollAnswer
	for _, day := range days {
		for _, hour := range []int{19, 20} {
			answers = append(answers, discordgo.PollAnswer{
				Media: &discordgo.PollMedia{
					Text:  fmt.Sprintf("%s klo %d-%d", day, hour, hour+1),
					Emoji: &discordgo.ComponentEmoji{Name: "♿"},
				},
			})
		}
	}
	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:         "@everyone",
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeEveryone}},
		Poll: &discordgo.Poll{
			Question:         discordgo.PollMedia{Text: kuvaus},
			AllowMultiselect: true,
			Answers:          answers,
			Duration:         168,
		},
	})
	if err != nil {
		fmt.Println("poll send error:", err)
	}
}

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
		if len(match.Rounds) == 4 {
			break
		}
		if attempt < maxTries {
			time.Sleep(retryDelay)
		} else {
			fmt.Printf("Error: match %s still missing rounds after %d attempts\n", matchId, maxTries)
			return
		}
	}

	if match == nil || len(match.Rounds) != 4 {
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

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: league,
		Color:       0x2ecc71,
		URL:         m.FaceitURL(),
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
