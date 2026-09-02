package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

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
			}, {
				Type:        discordgo.ApplicationCommandOptionRole,
				Name:        "tagi",
				Description: "Mainittava rooli (oletus @everyone)",
				Required:    false,
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
			}, {
				Type:        discordgo.ApplicationCommandOptionRole,
				Name:        "tagi",
				Description: "Mainittava rooli (oletus @everyone)",
				Required:    false,
			}},
		},
		{
			Name:        "subscriptions",
			Description: "Manage subscriptions for this channel",
		},
	}

	// Bulk overwrite is the easiest way to remove deprecated commands:
	// it replaces the entire command set for the scope (guild or global).
	if _, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildID, cmds); err != nil {
		fmt.Println("Command bulk overwrite error:", err)
	} else {
		if guildID == "" {
			fmt.Printf("Registered %d global commands\n", len(cmds))
		} else {
			fmt.Printf("Registered %d commands for guild %s\n", len(cmds), guildID)
		}
	}
}
