package discord

import (
	"fmt"
	"os"
	"sosso/faceit"
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
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	userID := i.Member.User.ID
	if i.User != nil {
		userID = i.User.ID
	}

	if !allowed[userID] {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Et voi käyttää tätä komentoa.",
				Flags:   1 << 6,
			},
		})
		return
	}

	switch i.ApplicationCommandData().Name {
	case "pelipaiva":
		vihollinen := i.ApplicationCommandData().Options[0].StringValue()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📊 Luodaan viikon pelipäivä -äänestys...",
				Flags:   1 << 6,
			},
		})
		createPelipaivaPoll(s, i.ChannelID, vihollinen)

	case "harkka":
		kuvaus := i.ApplicationCommandData().Options[0].StringValue()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📊 Luodaan harkkapäivä -äänestys...",
				Flags:   1 << 6,
			},
		})
		createHarkkaPoll(s, i.ChannelID, kuvaus)
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

func SendMessageInfo(s *discordgo.Session, matchId string) {
	match, err := faceit.FetchMatchInfo(matchId)
	if err != nil {
		return
	}

	embed := buildMatchEmbed(match)

	_, err = s.ChannelMessageSendEmbed(MatchChannel, embed)
	if err != nil {
		fmt.Println("Error sending Discord embed:", err)
	}
}

// buildMatchEmbed builds a beautiful embed for a finished match
func buildMatchEmbed(m *faceit.Match) *discordgo.MessageEmbed {
	winnerName := m.Teams[m.Results.Winner].Name

	// --- Lineups ---
	var lineupFields []*discordgo.MessageEmbedField
	for key, team := range m.Teams {
		var nicks []string
		for _, p := range team.Roster {
			nicks = append(nicks, p.Nickname)
		}
		score := m.Results.Score[key]
		lineupFields = append(lineupFields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s (score %d)", team.Name, score),
			Value:  strings.Join(nicks, ", "),
			Inline: false,
		})
	}

	// --- Maps / results ---
	var mapLines []string
	for i, mp := range m.Voting.Map.Pick {
		scoreInfo := ""
		if i < len(m.DetailedResults) {
			dr := m.DetailedResults[i]
			s1 := dr.Factions["faction1"].Score
			s2 := dr.Factions["faction2"].Score
			scoreInfo = fmt.Sprintf(" — %d:%d (winner: %s)",
				s1, s2, m.Teams[dr.Winner].Name)
		}
		mapLines = append(mapLines, fmt.Sprintf("• `%s`%s", mp, scoreInfo))
	}

	return &discordgo.MessageEmbed{
		Title:       "🎯 FaceIT Match Finished",
		Description: fmt.Sprintf("League: **%s**\nWinner: **%s**\nStatus: %s", m.CompetitionName, winnerName, m.Status),
		Color:       0x2ecc71, // green
		Fields: append(
			[]*discordgo.MessageEmbedField{
				{
					Name:   "Maps",
					Value:  strings.Join(mapLines, "\n"),
					Inline: false,
				},
			},
			lineupFields...,
		),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Powered by FaceIT API",
		},
		URL: m.FaceitURL(),
	}
}
