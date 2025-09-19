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
