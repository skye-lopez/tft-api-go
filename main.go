package tftgo

import (
	"encoding/json"
	"errors"
	"io/ioutil"
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
		time.Sleep(1 * time.Second) // NOTE: This is hard coded to 1, but may want to make it variable
		return t.Request(url, isAltRegion, target, (retryCount - 1))
	}
	defer res.Body.Close()

	if res.StatusCode == 403 {
		return errors.New("Status Code 403: Foribidden. This likely means the key given is either invalid or does not have access to the selected endpoint.")
	}

	body, err := ioutil.ReadAll(res.Body)
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
