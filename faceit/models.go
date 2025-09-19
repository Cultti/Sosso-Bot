package faceit

type MatchData struct {
	MatchID string
	Rounds  []Round `json:"rounds"`
}

type Round struct {
	BestOf        string     `json:"best_of"`
	CompetitionID *string    `json:"competition_id"`
	GameID        string     `json:"game_id"`
	GameMode      string     `json:"game_mode"`
	MatchID       string     `json:"match_id"`
	MatchRound    string     `json:"match_round"`
	Played        string     `json:"played"`
	RoundStats    RoundStats `json:"round_stats"`
	Teams         []Team     `json:"teams"`
}

type RoundStats struct {
	Region string `json:"Region"`
	Map    string `json:"Map"`
	Score  string `json:"Score"`
	Rounds string `json:"Rounds"`
	Winner string `json:"Winner"`
}

type Team struct {
	TeamID    string    `json:"team_id"`
	Premade   bool      `json:"premade"`
	TeamStats TeamStats `json:"team_stats"`
	Players   []Player  `json:"players"`
}

type TeamStats struct {
	Team            string `json:"Team"`
	FinalScore      string `json:"Final Score"`
	TeamHeadshots   string `json:"Team Headshots"`
	OvertimeScore   string `json:"Overtime score"`
	TeamWin         string `json:"Team Win"`
	FirstHalfScore  string `json:"First Half Score"`
	SecondHalfScore string `json:"Second Half Score"`
}

type Player struct {
	PlayerID    string      `json:"player_id"`
	Nickname    string      `json:"nickname"`
	PlayerStats PlayerStats `json:"player_stats"`
}

type PlayerStats struct {
	Assists                          string `json:"Assists"`
	UtilityDamage                    string `json:"Utility Damage"`
	UtilityUsagePerRound             string `json:"Utility Usage per Round"`
	MVPs                             string `json:"MVPs"`
	MatchEntrySuccessRate            string `json:"Match Entry Success Rate"`
	UtilityDamagePerRoundInAMatch    string `json:"Utility Damage per Round in a Match"`
	EnemiesFlashed                   string `json:"Enemies Flashed"`
	SniperKills                      string `json:"Sniper Kills"`
	FlashSuccessRatePerMatch         string `json:"Flash Success Rate per Match"`
	FlashesPerRoundInAMatch          string `json:"Flashes per Round in a Match"`
	TripleKills                      string `json:"Triple Kills"`
	ClutchKills                      string `json:"Clutch Kills"`
	PistolKills                      string `json:"Pistol Kills"`
	OneVOneCount                     string `json:"1v1Count"`
	DoubleKills                      string `json:"Double Kills"`
	EnemiesFlashedPerRoundInAMatch   string `json:"Enemies Flashed per Round in a Match"`
	Kills                            string `json:"Kills"`
	OneVTwoCount                     string `json:"1v2Count"`
	OneVTwoWins                      string `json:"1v2Wins"`
	FirstKills                       string `json:"First Kills"`
	ADR                              string `json:"ADR"`
	FlashCount                       string `json:"Flash Count"`
	UtilitySuccesses                 string `json:"Utility Successes"`
	UtilityDamageSuccessRatePerMatch string `json:"Utility Damage Success Rate per Match"`
	Headshots                        string `json:"Headshots"`
	KnifeKills                       string `json:"Knife Kills"`
	UtilitySuccessRatePerMatch       string `json:"Utility Success Rate per Match"`
	QuadroKills                      string `json:"Quadro Kills"`
	Deaths                           string `json:"Deaths"`
	MatchEntryRate                   string `json:"Match Entry Rate"`
	KRRatio                          string `json:"K/R Ratio"`
	OneVOneWins                      string `json:"1v1Wins"`
	PentaKills                       string `json:"Penta Kills"`
	Damage                           string `json:"Damage"`
	FlashSuccesses                   string `json:"Flash Successes"`
	UtilityCount                     string `json:"Utility Count"`
	SniperKillRatePerRound           string `json:"Sniper Kill Rate per Round"`
	EntryCount                       string `json:"Entry Count"`
	KDRatio                          string `json:"K/D Ratio"`
	MatchOneVOneWinRate              string `json:"Match 1v1 Win Rate"`
	UtilityEnemies                   string `json:"Utility Enemies"`
	SniperKillRatePerMatch           string `json:"Sniper Kill Rate per Match"`
	ZeusKills                        string `json:"Zeus Kills"`
	EntryWins                        string `json:"Entry Wins"`
	Result                           string `json:"Result"`
	MatchOneVTwoWinRate              string `json:"Match 1v2 Win Rate"`
	HeadshotsPercent                 string `json:"Headshots %"`
}

// ChampionshipResponse is the top-level response.
type ChampionshipResponse struct {
	Start int                `json:"start"`
	End   int                `json:"end"`
	Items []ChampionshipItem `json:"items"`
}

// ChampionshipItem is one championship record.
type ChampionshipItem struct {
	ChampionshipID string `json:"championship_id"`
	Name           string `json:"name"`
	GameID         string `json:"game_id"`
	Region         string `json:"region"`
	Status         string `json:"status"`
	// 👉 Add other fields as needed (you can map the full JSON if you need everything)
}
