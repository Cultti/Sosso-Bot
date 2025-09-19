package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sosso/db"
	"sosso/discord"
	"sosso/web"
	"syscall"
)

func main() {
	// Start database
	db.Init("data.db")

	// Start Discord bot
	ds, err := discord.Start()
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
