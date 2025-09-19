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
	GuildID string
}

var MatchChannel string

func Start() (*discordgo.Session, error) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	guildID := os.Getenv("DISCORD_GUILD_ID")
	MatchChannel = os.Getenv("DISCORD_MATCH_CHANNEL")

	if token == "" || guildID == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN and DISCORD_GUILD_ID must be set")
	}

	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	sess.AddHandler(interactionCreate)

	if err := sess.Open(); err != nil {
		return nil, err
	}

	// register commands
	registerCommands(sess, guildID)

	return sess, nil
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

	s.ApplicationCommandCreate(s.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "subscribe",
		Description: "Subscribe to Pappaliiga matches",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "liiga",
			Description: "Liiga muodossa '20 Divisioona S11'",
			Required:    true,
		}},
	})

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
		Description: "Lists subscriptions for this channel",
	})
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

	if i.ApplicationCommandData().Name == "subscribe" {
		handleSubscribeCommand(s, i)
		return
	}

	if i.ApplicationCommandData().Name == "unsubscribe" {
		handleUnsubscribeCommand(s, i)
		return
	}

	if i.ApplicationCommandData().Name == "subscriptions" {
		handleSubscriptions(s, i)
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

func handleSubscribeCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "⚒ Käsitellään pyyntöä...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	guildId := i.GuildID
	channelId := i.ChannelID
	league := i.ApplicationCommandData().Options[0].StringValue()

	err := db.CreateSubscription(&db.Subscription{
		GuildID:   guildId,
		ChannelID: channelId,
		League:    league,
	})
	if err != nil {
		content := "❌ Tilauksen luonti epäonnistui"

		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	content := "✔ Tilauksen onnistui"

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
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

func handleSubscriptions(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "⚒ Käsitellään pyyntöä...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	subs, err := db.GetSubscriptionsByGuildChannel(i.GuildID, i.ChannelID)
	if err != nil {
		content := "❌ Pyyntö epäonnistui"

		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
	} else if len(*subs) == 0 {
		content := "❔ Tilauksia ei löytynyt"

		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
	} else {
		leagues := make([]string, len(*subs))
		for i, s := range *subs {
			leagues[i] = s.League
		}
		content := "✔ Tämän kanavan tilaukset: " + strings.Join(leagues, ", ")

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

func SendMessageInfo(s *discordgo.Session, matchId string, league string) {
	// Fetch match info
	match, err := faceit.FetchMatchInfo(matchId)
	if err != nil {
		fmt.Println("Error fetching match info:", err)
		return
	}

	// Build the embed once
	embed := buildMatchEmbed(match, league)

	// Fetch all subscriptions for this league
	subs, err := db.GetSubscriptionsByLeague(league)
	if err != nil {
		fmt.Println("Error fetching subscriptions:", err)
		return
	}

	if subs == nil || len(*subs) == 0 {
		fmt.Println("No subscriptions found for league:", league)
		return
	}

	// Send embed to each channel in the subscriptions
	for _, sub := range *subs {
		_, err := s.ChannelMessageSendEmbed(sub.ChannelID, embed)
		if err != nil {
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
