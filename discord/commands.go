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
