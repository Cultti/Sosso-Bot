package discord

import (
	"fmt"
	"log"
	"sort"
	"sosso/db"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const discordMessageMaxLen = 2000

func formatSubscribedDivisions(leagues []string) string {
	if len(leagues) == 0 {
		return "(ei yhtään)"
	}

	// Leave headroom for any surrounding text.
	const maxLen = discordMessageMaxLen - 200

	lines := make([]string, 0, len(leagues))
	for _, league := range leagues {
		lines = append(lines, "• "+league)
	}

	out := strings.Join(lines, "\n")
	if len(out) <= maxLen {
		return out
	}

	// Truncate to fit the message limit; keep whole lines.
	truncated := make([]string, 0, len(lines))
	currentLen := 0
	for _, line := range lines {
		added := len(line)
		if len(truncated) > 0 {
			added += 1 // newline
		}
		if currentLen+added > maxLen {
			break
		}
		truncated = append(truncated, line)
		currentLen += added
	}

	remaining := len(lines) - len(truncated)
	if remaining > 0 {
		truncated = append(truncated, fmt.Sprintf("• …(+%d lisää)", remaining))
	}

	return strings.Join(truncated, "\n")
}

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
		sort.Strings(selectedSlice)
		selectedSlice = sortChampionships(selectedSlice)

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

		divisionsList := formatSubscribedDivisions(selectedSlice)

		// Send message to channel (non-ephemeral)
		channelMsg := fmt.Sprintf("✅ Tilaukset tallennettu. Tähän kanavaan tilatut sarjat (%d):\n%s", len(selectedSlice), divisionsList)
		_, channelSendErr := s.ChannelMessageSend(channelID, channelMsg)
		if channelSendErr != nil {
			log.Println("Failed to send subscriptions summary to channel:", channelSendErr)
		}

		// Clear temporary selections
		delete(tempSelections, userKey)

		ephemeralContent := fmt.Sprintf("✔ Tilattu %d sarjaa!", len(selectedSlice))
		if channelSendErr != nil {
			ephemeralContent = "⚠ En saanut lähetettyä viestiä kanavaan. Tarkista minun käyttöoikeuteni, jos haluat kanava-ilmoitukset."
		}

		// Update ephemeral message
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    ephemeralContent,
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
		// Check if user has administrator permissions
		perms, err := s.State.UserChannelPermissions(userID, i.ChannelID)
		if err != nil {
			// Fallback to fetching permissions via API
			perms, err = s.UserChannelPermissions(userID, i.ChannelID)
			if err != nil {
				log.Println("Failed to fetch user permissions:", err)
			}
		}

		// Only allow if we successfully got permissions and user has Administrator
		// err will be nil if either State or API call succeeded
		if err == nil && perms&discordgo.PermissionAdministrator != 0 {
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
		roleID := optionRoleID(i.ApplicationCommandData().Options, "tagi", i.GuildID)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📊 Luodaan viikon pelipäivä -äänestys...",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		createPelipaivaPoll(s, i.ChannelID, vihollinen, roleID)

	case "harkka":
		kuvaus := i.ApplicationCommandData().Options[0].StringValue()
		roleID := optionRoleID(i.ApplicationCommandData().Options, "tagi", i.GuildID)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📊 Luodaan harkkapäivä -äänestys...",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		createHarkkaPoll(s, i.ChannelID, kuvaus, roleID)
	}
}

// optionRoleID returns the role ID of the named role option, or "" if the
// option is missing or refers to the guild's @everyone role.
func optionRoleID(options []*discordgo.ApplicationCommandInteractionDataOption, name, guildID string) string {
	for _, opt := range options {
		if opt.Name == name && opt.Type == discordgo.ApplicationCommandOptionRole {
			roleID, _ := opt.Value.(string)
			if roleID == guildID {
				// Selecting the @everyone role falls back to the default mention.
				return ""
			}
			return roleID
		}
	}
	return ""
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
