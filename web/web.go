package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sosso/discord"
	"time"

	"github.com/bwmarrin/discordgo"
)

var authToken string
var BotSession *discordgo.Session

func init() {
	// Read the token once at startup
	authToken = os.Getenv("WEBHOOK_AUTH_TOKEN")
	if authToken == "" {
		fmt.Println("⚠️  WEBHOOK_AUTH_TOKEN is not set – webhook will reject all requests")
	}
}

type Entity struct {
	Name string `json:"name"`
}

type Payload struct {
	ID     string `json:"id"`
	Game   string `json:"game"`
	Entity Entity `json:"entity"`
}

type EventRequest struct {
	Event   string  `json:"event"`
	Payload Payload `json:"payload"`
}

func Start(addr string, ds *discordgo.Session) error {
	BotSession = ds
	http.HandleFunc("/webhook", handler)
	fmt.Println("🌐 Webhook listening on", addr)
	return http.ListenAndServe(addr, nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	status := http.StatusOK

	// Always log the request at the end
	defer func() {
		log.Printf("[Webhook] %s %s -> %d (%s)", r.Method, r.URL.Path, status, time.Since(start))
	}()

	if r.Method != http.MethodPost {
		status = http.StatusMethodNotAllowed
		http.Error(w, "method not allowed", status)
		return
	}

	if r.Header.Get("Authorization") != "Bearer "+authToken {
		status = http.StatusUnauthorized
		http.Error(w, "unauthorized", status)
		return
	}

	var e EventRequest
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		status = http.StatusBadRequest
		http.Error(w, "bad request", status)
		return
	}

	// Log resolved event info
	log.Printf("[Webhook] Received event: %s | ID: %s | Game: %s | Entity: %s",
		e.Event, e.Payload.ID, e.Payload.Game, e.Payload.Entity.Name)

	// Check event and return code immediately
	if e.Event == "match_status_finished" &&
		e.Payload.Game == "cs2" &&
		e.Payload.Entity.Name == "20 Divisioona S11" {

		status = http.StatusAccepted
		w.WriteHeader(status)
		w.Write([]byte("processing"))

		// Process Discord message in background
		go discord.SendMessageInfo(BotSession, e.Payload.ID)
		return
	}

	status = http.StatusNoContent
	w.WriteHeader(status)
}
