package tftgo

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

type TFTGO struct {
	key        string
	Region     string
	AltRegion  string
	ShowLogs   bool
	RateLimit  bool
	RetryCount int
}

func TftGo(RiotApiKey string, region string, showLogs bool, rateLimit bool, retryCount int) (TFTGO, error) {
	t := TFTGO{
		key:        RiotApiKey,
		Region:     region,
		ShowLogs:   showLogs,
		RateLimit:  rateLimit,
		RetryCount: retryCount,
	}

	// Verify region and get AltRegion
	regionMap := make(map[string]string)
	regionMap["na1"] = "americas"

	// NOTE: The altRegion is really just what cluster is being called.
	// and should likely just be americas always. This is the setup for testing
	// but should likely be an entirely different argument instead.
	val, ok := regionMap[region]
	if !ok {
		return t, errors.New("TftGo - Invalid region provided")
	}
	t.AltRegion = val

	// Validate API key
	var result interface{}
	err := t.Request("tft/status/v1/platform-data", false, &result, t.RetryCount)
	if err != nil {
		return t, errors.New("TftGo - Invalid API key")
	}

	return t, nil
}

// TODO: Rate limit (eventually ~ the api already gates this)
func (t *TFTGO) Request(url string, isAltRegion bool, target interface{}, retryCount int) error {
	// proper region mapping
	u := ""
	if isAltRegion {
		u = "https://" + t.AltRegion + ".api.riotgames.com/" + url
	} else {
		u = "https://" + t.Region + ".api.riotgames.com/" + url
	}

	if t.ShowLogs {
		log.Printf("TFTGO Request - %v\n", u)
	}

	client := http.Client{}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}

	req.Header = http.Header{
		"X-Riot-Token": {t.key},
	}

	// Only retry on the DO
	res, err := client.Do(req)
	if err != nil && retryCount == 0 {
		return err
	} else if err != nil && retryCount > 0 {
		// NOTE: This is hard coded to 1, but may want to make it variable
		time.Sleep(1 * time.Second)
		return t.Request(url, isAltRegion, target, (retryCount - 1))
	}
	defer res.Body.Close()

	if res.StatusCode == 403 {
		return errors.New("Status Code 403: Foribidden. This likely means the key given is either invalid or does not have access to the selected endpoint.")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &target)
	if err != nil {
		return err
	}

	return nil
}

type TftLeagueEntry struct {
	SummonerId   string `json:"summonerId"`
	LeaguePoints int    `json:"leaguePoints"`
	Rank         string `json:"rank"`
	Wins         int    `json:"wins"`
	Losses       int    `json:"losses"`
	Veteran      bool   `json:"veteran"`
	Inactive     bool   `json:"inactive"`
	FreshBlood   bool   `json:"freshBlood"`
	HotStreak    bool   `json:"hotStreak"`
}

type TftLeagueResponse struct {
	Tier     string           `json:"tier"`
	LeagueId string           `json:"leagueId"`
	Queue    string           `json:"queue"`
	Name     string           `json:"name"`
	Entries  []TftLeagueEntry `json:"entries"`
}

func (t *TFTGO) TftLeagueV1Challenger() (TftLeagueResponse, error) {
	url := "tft/league/v1/challenger?queue=RANKED_TFT"
	challenger := TftLeagueResponse{}
	err := t.Request(url, false, &challenger, t.RetryCount)
	if err != nil {
		return challenger, err
	}

	return challenger, nil
}

func (t *TFTGO) TftLeagueV1Grandmaster() (TftLeagueResponse, error) {
	url := "tft/league/v1/grandmaster?queue=RANKED_TFT"
	grandmaster := TftLeagueResponse{}
	err := t.Request(url, false, &grandmaster, t.RetryCount)
	if err != nil {
		return grandmaster, err
	}

	return grandmaster, nil
}

func (t *TFTGO) TftLeagueV1Master() (TftLeagueResponse, error) {
	url := "tft/league/v1/master?queue=RANKED_TFT"
	master := TftLeagueResponse{}
	err := t.Request(url, false, &master, t.RetryCount)
	if err != nil {
		return master, err
	}

	return master, nil
}

type TftSummonerResponse struct {
	Id            string `json:"id"`
	AccountId     string `json:"accountId"`
	Puuid         string `json:"puuid"`
	ProfileIconId int    `json:"profileIconId"`
	RevisionDate  int    `json:"revisionDate"`
	SummonerLevel int    `json:"summonerLevel"`
}

func (t *TFTGO) TftSummonerV1SummonerId(summonerId string) (TftSummonerResponse, error) {
	url := "tft/summoner/v1/summoners/" + summonerId
	summoner := TftSummonerResponse{}
	err := t.Request(url, false, &summoner, t.RetryCount)
	if err != nil {
		return summoner, err
	}

	return summoner, err
}

func (t *TFTGO) TftMatchV1MatchesByPuuid(puuid string) ([]string, error) {
	url := "tft/match/v1/matches/by-puuid/" + puuid + "/ids?start=0&count=200"
	ids := make([]string, 0)
	err := t.Request(url, true, &ids, t.RetryCount)
	if err != nil {
		return ids, err
	}

	return ids, nil
}

// NOTE: This is not parsing all the data, and I may be able to cut some more out
type TftMatchMetaData struct {
	DataVersion  string   `json:"data_version"`
	MatchId      string   `json:"match_id"`
	Participants []string `json:"participants"`
}

type TftMatchParticipantTrait struct {
	Name        string `json:"name"`
	NumUnits    int    `json:"num_units"`
	Style       int    `json:"style"`
	TierCurrent int    `json:"tier_current"`
	TierTotal   int    `json:"tier_total"`
}

type TftMatchParticipantUnit struct {
	CharacterId string   `json:"character_id"`
	ItemNames   []string `json:"itemNames"`
	Tier        int      `json:"tier"`
}

type TftMatchParticipant struct {
	Traits               []TftMatchParticipantTrait `json:"traits"`
	Units                []TftMatchParticipantUnit  `json:"units"`
	Augments             []string                   `json:"augments"`
	GoldLeft             int                        `json:"gold_left"`
	LastRound            int                        `json:"last_round"`
	Level                int                        `json:"level"`
	Placement            int                        `json:"placement"`
	TotalDamageToPlayers int                        `json:"total_damage_to_players"`
}

type TftMatchDataInfo struct {
	Participants     []TftMatchParticipant `json:"participants"`
	TftGameType      string                `json:"tft_game_type"`
	TftSetCoreName   string                `json:"tft_set_core_name"`
	EndOfGameResult  string                `json:"endOfGameResult"`
	QueueId          int                   `json:"queueId"`
	QueueIdAlternate int                   `json:"queue_id"`
	TftSetNumber     int                   `json:"tft_set_number"`
	GameCreation     int                   `json:"gameCreation"`
}

type TftMatchResponse struct {
	Metadata TftMatchMetaData `json:"metadata"`
	Info     TftMatchDataInfo `json:"info"`
}

func (t *TFTGO) TftMatchV1MatchesById(matchId string) (TftMatchResponse, error) {
	url := "tft/match/v1/matches/" + matchId
	matchData := TftMatchResponse{}
	err := t.Request(url, true, &matchData, t.RetryCount)
	if err != nil {
		return matchData, err
	}

	return matchData, err
}
