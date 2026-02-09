package discord

import (
	"fmt"
	"log"
	"sosso/db"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// userID_channelID -> menuID -> []selectedValues
var tempSelections = map[string]map[string][]string{}

func interactionHandle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	data := i.MessageComponentData()
	userKey := i.Member.User.ID + "_" + i.ChannelID
	menuID := data.CustomID
	guildID := i.GuildID
	channelID := i.ChannelID

	// Initialize map
	if _, exists := tempSelections[userKey]; !exists {
		tempSelections[userKey] = make(map[string][]string)
	}

	// ---- Menu selection ----
	if strings.HasPrefix(menuID, "champ_") {
		// Save selections for this menu
		tempSelections[userKey][menuID] = data.Values

		// Merge all menu selections
		merged := []string{}
		for _, vals := range tempSelections[userKey] {
			merged = append(merged, vals...)
		}

		// Rebuild menus
		sortedList := sortChampionships(championshipList)
		menus := buildSelectMenus(sortedList, merged)

		// Append Save button
		saveButton := discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "💾 Save",
					Style:    discordgo.PrimaryButton,
					CustomID: "save_subscriptions",
				},
			},
		}
		menus = append(menus, saveButton)

		// Update ephemeral message
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    "✔ Valittu! Paina Save kun valmis.",
				Components: menus,
			},
		})
		return
	}

	// ---- Save button ----
	if menuID == "save_subscriptions" {
		// Merge all menu selections
		finalSelections := make(map[string]struct{})
		for _, vals := range tempSelections[userKey] {
			for _, v := range vals {
				finalSelections[v] = struct{}{}
			}
		}

		selectedSlice := make([]string, 0, len(finalSelections))
		for v := range finalSelections {
			selectedSlice = append(selectedSlice, v)
		}

		// Clear all subscriptions for this channel
		_, _ = db.DeleteSubscriptionsByGuildChannel(guildID, channelID, "")

		// Add new selections
		for _, league := range selectedSlice {
			_ = db.CreateSubscription(&db.Subscription{
				GuildID:   guildID,
				ChannelID: channelID,
				League:    league,
			})
		}

		// Clear temporary selections
		delete(tempSelections, userKey)

		// Update ephemeral message
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    fmt.Sprintf("✔ Tilattu %d sarjaa!", len(selectedSlice)),
				Components: []discordgo.MessageComponent{},
			},
		})
	}
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	userID := i.Member.User.ID
	if i.User != nil {
		userID = i.User.ID
	}

	isAllowed := allowed[userID]

	if i.GuildID != "" && !isAllowed {
		// Fetch guild info to check owner
		guild, err := s.State.Guild(i.GuildID)
		if err != nil {
			guild, err = s.Guild(i.GuildID) // fallback to API call if not in state
			if err != nil {
				log.Println("Failed to fetch guild info:", err)
			}
		}

		if guild != nil && guild.OwnerID == userID {
			isAllowed = true
		}
	}

	if !isAllowed {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Et voi käyttää tätä komentoa.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if i.ApplicationCommandData().Name == "subscriptions" {
		handleSubscriontionsCommand(s, i)
		return
	}

	switch i.ApplicationCommandData().Name {
	case "pelipaiva":
		vihollinen := i.ApplicationCommandData().Options[0].StringValue()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📊 Luodaan viikon pelipäivä -äänestys...",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		createPelipaivaPoll(s, i.ChannelID, vihollinen)

	case "harkka":
		kuvaus := i.ApplicationCommandData().Options[0].StringValue()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📊 Luodaan harkkapäivä -äänestys...",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		createHarkkaPoll(s, i.ChannelID, kuvaus)
	}
}

func handleSubscriontionsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userKey := i.Member.User.ID + "_" + i.ChannelID

	// Always create a new tempSelections map for this user & channel
	tempSelections[userKey] = make(map[string][]string)

	// Load current subscriptions from DB
	existingSubs, err := db.GetSubscriptionsByGuildChannel(i.GuildID, i.ChannelID)
	var current []string
	if err == nil {
		current = make([]string, len(*existingSubs))
		for idx, s := range *existingSubs {
			current[idx] = s.League
		}
	}

	// Pre-fill each menu chunk
	const chunkSize = 25
	for i := 0; i < len(championshipList); i += chunkSize {
		menuID := fmt.Sprintf("champ_%d", i/chunkSize)

		// Select the items from current subscriptions that belong to this chunk
		end := i + chunkSize
		if end > len(championshipList) {
			end = len(championshipList)
		}
		chunk := championshipList[i:end]

		selectedInChunk := []string{}
		for _, name := range chunk {
			for _, sel := range current {
				if sel == name {
					selectedInChunk = append(selectedInChunk, sel)
					break
				}
			}
		}

		tempSelections[userKey][menuID] = selectedInChunk
	}

	// Merge all current selections for initial display
	merged := []string{}
	for _, vals := range tempSelections[userKey] {
		merged = append(merged, vals...)
	}

	// Build select menus
	sortedList := sortChampionships(championshipList)
	menus := buildSelectMenus(sortedList, merged)

	// Append Save button
	saveButton := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "💾 Save",
				Style:    discordgo.PrimaryButton,
				CustomID: "save_subscriptions",
			},
		},
	}
	menus = append(menus, saveButton)

	// Respond ephemerally
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "Valitse mestaruuksia:",
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: menus,
		},
	})
}

func buildSelectMenus(championshipList []string, currentSelections []string) []discordgo.MessageComponent {
	// Build a set for fast lookup
	selectedSet := make(map[string]struct{}, len(currentSelections))
	for _, s := range currentSelections {
		selectedSet[s] = struct{}{}
	}

	const chunkSize = 25
	var menus []discordgo.MessageComponent

	min := 0

	for i := 0; i < len(championshipList); i += chunkSize {
		end := i + chunkSize
		if end > len(championshipList) {
			end = len(championshipList)
		}

		opts := make([]discordgo.SelectMenuOption, 0, end-i)
		for _, name := range championshipList[i:end] {
			_, isSelected := selectedSet[name] // true if currently selected
			opts = append(opts, discordgo.SelectMenuOption{
				Label:   name,
				Value:   name,
				Default: isSelected,
			})
		}

		menus = append(menus,
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    fmt.Sprintf("champ_%d", i/chunkSize),
						Placeholder: "Valitse sarjoja…",
						MinValues:   &min,
						MaxValues:   len(opts),
						Options:     opts,
					},
				},
			})
	}

	return menus
}
