package faceit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Match struct {
	MatchID         string `json:"match_id"`
	Status          string `json:"status"`
	CompetitionName string `json:"competition_name"`
	Results         struct {
		Winner string         `json:"winner"`
		Score  map[string]int `json:"score"`
	} `json:"results"`
	Teams map[string]struct {
		Name   string `json:"name"`
		Roster []struct {
			Nickname string `json:"nickname"`
		} `json:"roster"`
	} `json:"teams"`
	Voting struct {
		Map struct {
			Pick []string `json:"pick"`
		} `json:"map"`
	} `json:"voting"`
	DetailedResults []struct {
		Winner   string `json:"winner"`
		Factions map[string]struct {
			Score int `json:"score"`
		} `json:"factions"`
	} `json:"detailed_results"`
}

func (m *Match) FaceitURL() string {
	return fmt.Sprintf("https://www.faceit.com/en/cs2/room/%s", m.MatchID)
}

// FetchMatchInfo calls FaceIT API and decodes the response
func FetchMatchInfo(matchID string) (*Match, error) {
	apiKey := os.Getenv("FACEIT_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("FACEIT_API_KEY is not set")
	}

	url := fmt.Sprintf("https://open.faceit.com/data/v4/matches/%s", matchID)

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

	var match Match
	if err := json.NewDecoder(resp.Body).Decode(&match); err != nil {
		return nil, err
	}
	return &match, nil
}
