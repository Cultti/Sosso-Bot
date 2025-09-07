package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	Token   string
	GuildID string
)

func main() {
	// Read environment variables
	Token = os.Getenv("DISCORD_BOT_TOKEN")
	GuildID = os.Getenv("DISCORD_GUILD_ID")

	if Token == "" || GuildID == "" {
		fmt.Println("❌ DISCORD_BOT_TOKEN and DISCORD_GUILD_ID must be set")
		return
	}

	// Create Discord session
	sess, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	// Add interaction handler
	sess.AddHandler(interactionCreate)

	// Open connection
	err = sess.Open()
	if err != nil {
		fmt.Println("Error opening Discord session:", err)
		return
	}
	defer sess.Close()

	fmt.Println("✅ Bot is running")

	// Register /pelipaiva
	_, err = sess.ApplicationCommandCreate(sess.State.User.ID, GuildID, &discordgo.ApplicationCommand{
		Name:        "pelipaiva",
		Description: "Luo viikon pelipäivä-äänestys",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "vihollinen",
				Description: "Kenen kanssa pelataan?",
				Required:    true,
			},
		},
	})
	if err != nil {
		fmt.Println("Error creating /pelipaiva command:", err)
	}

	// Register /harkka
	_, err = sess.ApplicationCommandCreate(sess.State.User.ID, GuildID, &discordgo.ApplicationCommand{
		Name:        "harkka",
		Description: "Luo harkkapäivä-äänestys",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "kuvaus",
				Description: "Harkan kuvaus",
				Required:    true,
			},
		},
	})
	if err != nil {
		fmt.Println("Error creating /harkka command:", err)
	}

	// Wait for CTRL-C or termination
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}

// Handle slash commands
func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	// Allowed user IDs
	allowed := map[string]bool{
		"336110846306549760": true,
		"106011141670461440": true,
	}

	userID := i.Member.User.ID
	if i.User != nil {
		userID = i.User.ID
	}

	if !allowed[userID] {
		// Respond ephemeral if user not allowed
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Et voi käyttää tätä komentoa.",
				Flags:   1 << 6, // ephemeral
			},
		})
		return
	}

	switch i.ApplicationCommandData().Name {
	case "pelipaiva":
		options := i.ApplicationCommandData().Options
		vihollinen := ""
		if len(options) > 0 {
			vihollinen = options[0].StringValue()
		}

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📊 Luodaan viikon pelipäivä -äänestys...",
				Flags:   1 << 6,
			},
		})

		// Pass i.ChannelID instead of fixed ChannelID
		createPelipaivaPoll(s, i.ChannelID, 0x2ecc71, vihollinen)

	case "harkka":
		options := i.ApplicationCommandData().Options
		kuvaus := ""
		if len(options) > 0 {
			kuvaus = options[0].StringValue()
		}

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📊 Luodaan harkkapäivä -äänestys...",
				Flags:   1 << 6,
			},
		})

		// Pass i.ChannelID
		createHarkkaPoll(s, i.ChannelID, 0x2ecc71, kuvaus)
	}
}

func createPelipaivaPoll(s *discordgo.Session, channelID string, color int, vihollinen string) {
	now := time.Now()
	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}
	monday := now.AddDate(0, 0, offset)

	weekdayFi := []string{"Maanantai", "Tiistai", "Keskiviikko", "Torstai", "Perjantai", "Lauantai", "Sunnuntai"}
	emojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣"} // one emoji per weekday

	var answers []discordgo.PollAnswer
	for i := 0; i < 7; i++ {
		day := monday.AddDate(0, 0, i)
		formatted := fmt.Sprintf("%s %d.%d.", weekdayFi[i], day.Day(), int(day.Month()))
		answers = append(answers, discordgo.PollAnswer{
			Media: &discordgo.PollMedia{
				Text:  formatted,
				Emoji: &discordgo.ComponentEmoji{Name: emojis[i]},
			},
		})
	}

	// Add "Ei käy" option with ❌
	answers = append(answers, discordgo.PollAnswer{
		Media: &discordgo.PollMedia{
			Text:  "Ei käy",
			Emoji: &discordgo.ComponentEmoji{Name: "❌"},
		},
	})

	_, week := now.ISOWeek()
	description := fmt.Sprintf("Viikon %d pelipäivä. Oletusaloitus 19-20. Ilmoita jos aika ei käy. Vihollinen %s.", week, vihollinen)

	poll := &discordgo.MessageSend{
		Content: "@everyone",
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{
				discordgo.AllowedMentionTypeEveryone,
			},
		},
		Poll: &discordgo.Poll{
			Question:         discordgo.PollMedia{Text: description},
			AllowMultiselect: true,
			Answers:          answers,
			Duration:         168,
		},
	}

	_, err := s.ChannelMessageSendComplex(channelID, poll)
	if err != nil {
		fmt.Println("Error sending pelipaiva poll:", err)
	}
}

func createHarkkaPoll(s *discordgo.Session, channelID string, color int, kuvaus string) {
	weekdayFi := []string{"Tiistai", "Keskiviikko", "Torstai", "Perjantai", "Lauantai"}

	var answers []discordgo.PollAnswer
	for _, day := range weekdayFi {
		for _, hour := range []int{19, 20} {
			formatted := fmt.Sprintf("%s klo %d-%d", day, hour, hour+1)
			answers = append(answers, discordgo.PollAnswer{
				Media: &discordgo.PollMedia{
					Text:  formatted,
					Emoji: &discordgo.ComponentEmoji{Name: "♿"}, // wheelchair symbol
				},
			})
		}
	}

	poll := &discordgo.MessageSend{
		Content: "@everyone",
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{
				discordgo.AllowedMentionTypeEveryone,
			},
		},
		Poll: &discordgo.Poll{
			Question:         discordgo.PollMedia{Text: kuvaus},
			AllowMultiselect: true,
			Answers:          answers,
			Duration:         168, // max ~1 week
		},
	}

	_, err := s.ChannelMessageSendComplex(channelID, poll)
	if err != nil {
		fmt.Println("Error sending harkka poll:", err)
	}
}
