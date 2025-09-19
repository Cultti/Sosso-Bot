package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sosso/db"
	"sosso/discord"
	"sosso/faceit"
	"sosso/web"
	"syscall"
)

func main() {
	// Start database
	db.Init("./data/data.db")

	// Fetch championships
	championships, err := faceit.FetchAllChampionships("1bfc69fa-5a21-4ed9-9ef3-37edbd7210d8", 100)
	championships = faceit.FilterChampionships(championships, "cs2", "started")

	// Start Discord bot
	ds, err := discord.Start(&championships)
	if err != nil {
		log.Fatalf("Discord error: %v", err)
	}
	defer ds.Close()

	// Start webhook server
	go func() {
		if err := web.Start(":8080", ds); err != nil {
			log.Fatalf("Webhook error: %v", err)
		}
	}()

	fmt.Println("✅ Services running (Discord + Webhook)")

	// Wait for CTRL-C
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	fmt.Println("\n👋 Shutting down...")
}
