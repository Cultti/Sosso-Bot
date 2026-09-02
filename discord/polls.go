package discord

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func pollMention(roleID string) (string, *discordgo.MessageAllowedMentions) {
	if roleID == "" {
		return "@everyone", &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeEveryone}}
	}
	return fmt.Sprintf("<@&%s>", roleID), &discordgo.MessageAllowedMentions{Roles: []string{roleID}}
}

func createPelipaivaPoll(s *discordgo.Session, channelID, vihollinen, roleID string) {
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

	mention, allowedMentions := pollMention(roleID)
	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:         mention,
		AllowedMentions: allowedMentions,
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

func createHarkkaPoll(s *discordgo.Session, channelID, kuvaus, roleID string) {
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
	mention, allowedMentions := pollMention(roleID)
	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:         mention,
		AllowedMentions: allowedMentions,
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
