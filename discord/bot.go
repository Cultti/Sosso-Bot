package discord

import (
	"fmt"
	"log"
	"os"
	"sync"

	"sosso/db"
	"sosso/faceit"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session *discordgo.Session
}

var (
	MatchChannel     string
	championshipList []string
	cleanupOnce      sync.Once
)

func cleanupSubscriptionsForMissingGuilds(s *discordgo.Session) {
	if s == nil {
		return
	}

	var guildIDs []string
	for _, g := range s.State.Guilds {
		if g != nil && g.ID != "" {
			guildIDs = append(guildIDs, g.ID)
		}
	}

	if len(guildIDs) == 0 {
		// Avoid deleting anything if we couldn't resolve guilds.
		log.Println("[subscriptions] cleanup skipped: no guilds in state")
		return
	}

	deleted, err := db.DeleteSubscriptionsNotInGuildIDs(guildIDs)
	if err != nil {
		log.Println("[subscriptions] cleanup failed:", err)
		return
	}
	if deleted > 0 {
		log.Printf("[subscriptions] cleaned up %d subscriptions from missing guilds\n", deleted)
	}
}

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

		// Clean up stale subscriptions for guilds the bot left.
		cleanupOnce.Do(func() {
			cleanupSubscriptionsForMissingGuilds(s)
		})
	})
	sess.AddHandler(func(s *discordgo.Session, g *discordgo.GuildCreate) {
		log.Printf("Bot seen guild: %s", g.ID)
	})
	sess.AddHandler(func(s *discordgo.Session, g *discordgo.GuildDelete) {
		if g == nil || g.ID == "" {
			return
		}
		deleted, err := db.DeleteSubscriptionsByGuildID(g.ID)
		if err != nil {
			log.Printf("[subscriptions] failed to delete subscriptions for guild %s: %v\n", g.ID, err)
			return
		}
		if deleted > 0 {
			log.Printf("[subscriptions] deleted %d subscriptions for left guild %s\n", deleted, g.ID)
		}
	})

	if err := sess.Open(); err != nil {
		return nil, err
	}

	return sess, nil
}
