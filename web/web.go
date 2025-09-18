package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sosso/discord"

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
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+authToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var e EventRequest
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if e.Event == "match_status_finished" && e.Payload.Game == "cs2" && e.Payload.Entity.Name == "20 Divisioona S11" {
		fmt.Printf("[Webhook] Event: %s | ID: %s | Game: %s | Entity: %s\n",
			e.Event, e.Payload.ID, e.Payload.Game, e.Payload.Entity.Name)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))

		discord.SendMessageInfo(BotSession, e.Payload.ID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
