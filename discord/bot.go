package discord

import (
	"fmt"
	"os"

	"sosso/faceit"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session *discordgo.Session
}

var (
	MatchChannel     string
	championshipList []string
)

func Start(championships *[]faceit.ChampionshipItem) (*discordgo.Session, error) {
	for _, item := range *championships {
		championshipList = append(championshipList, item.Name)
	}

	championshipList = sortChampionships(championshipList)

	token := os.Getenv("DISCORD_BOT_TOKEN")
	MatchChannel = os.Getenv("DISCORD_MATCH_CHANNEL")

	if token == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN must be set")
	}

	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	sess.AddHandler(interactionCreate)
	sess.AddHandler(interactionHandle)
	sess.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		// Keep global commands in sync (and delete deprecated ones).
		registerCommands(s, "")
	})
	sess.AddHandler(func(s *discordgo.Session, g *discordgo.GuildCreate) {
		fmt.Println("Bot seen guild:", g.ID)
		// Ensure no guild-specific commands exist; use global commands only.
		clearGuildCommands(s, g.ID)
	})

	if err := sess.Open(); err != nil {
		return nil, err
	}

	return sess, nil
}
