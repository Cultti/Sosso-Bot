package faceit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func (m *MatchData) FaceitURL() string {
	return fmt.Sprintf("https://www.faceit.com/en/cs2/room/%s", m.MatchID)
}

// FetchMatchInfo calls FaceIT API and decodes the response
func FetchMatchInfo(matchID string) (*MatchData, error) {
	apiKey := os.Getenv("FACEIT_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("FACEIT_API_KEY is not set")
	}

	url := fmt.Sprintf("https://open.faceit.com/data/v4/matches/%s/stats", matchID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FaceIT API returned %s", resp.Status)
	}

	var match MatchData
	if err := json.NewDecoder(resp.Body).Decode(&match); err != nil {
		return nil, err
	}

	match.MatchID = matchID

	return &match, nil
}

func FetchAllChampionships(organizerID string, limit int) ([]ChampionshipItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	apiKey := os.Getenv("FACEIT_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("FACEIT_API_KEY is not set")
	}

	client := http.DefaultClient
	var allItems []ChampionshipItem
	offset := 0

	for {
		url := fmt.Sprintf(
			"https://open.faceit.com/data/v4/organizers/%s/championships?limit=%d&offset=%d",
			organizerID, limit, offset,
		)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("FaceIT API returned %s", resp.Status)
		}

		var page ChampionshipResponse
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			return nil, err
		}

		allItems = append(allItems, page.Items...)

		// Stop when fewer than limit items are returned
		if len(page.Items) < limit {
			break
		}

		offset += limit
	}

	return allItems, nil
}

func FilterChampionships(
	items []ChampionshipItem,
	gameID string,
	statuses ...string,
) []ChampionshipItem {
	if len(items) == 0 {
		return nil
	}

	// Convert statuses to a lookup map for O(1) checks
	statusSet := make(map[string]struct{}, len(statuses))
	for _, s := range statuses {
		statusSet[s] = struct{}{}
	}

	var filtered []ChampionshipItem
	for _, item := range items {
		if item.GameID != gameID {
			continue
		}
		if len(statusSet) > 0 {
			if _, ok := statusSet[item.Status]; !ok {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}
