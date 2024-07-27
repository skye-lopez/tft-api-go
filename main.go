package tftgo

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
)

type TFTGO struct {
	key       string
	Region    string
	AltRegion string
	ShowLogs  bool
	// TODO: Rate limit
}

func TftGo(RiotApiKey string, region string, showLogs bool) (TFTGO, error) {
	t := TFTGO{
		key:      RiotApiKey,
		Region:   region,
		ShowLogs: showLogs,
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
	err := t.Request("tft/status/v1/platform-data", false, &result)
	if err != nil {
		return t, errors.New("TftGo - Invalid API key")
	}

	return t, nil
}

func (t *TFTGO) Request(url string, isAltRegion bool, target interface{}) error {
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

	res, err := client.Do(req)
	if err != nil {
		return err
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
	err := t.Request(url, false, &challenger)
	if err != nil {
		return challenger, err
	}

	return challenger, nil
}

func (t *TFTGO) TftLeagueV1Grandmaster() (TftLeagueResponse, error) {
	url := "tft/league/v1/grandmaster?queue=RANKED_TFT"
	grandmaster := TftLeagueResponse{}
	err := t.Request(url, false, &grandmaster)
	if err != nil {
		return grandmaster, err
	}

	return grandmaster, nil
}

func (t *TFTGO) TftLeagueV1Master() (TftLeagueResponse, error) {
	url := "tft/league/v1/master?queue=RANKED_TFT"
	master := TftLeagueResponse{}
	err := t.Request(url, false, &master)
	if err != nil {
		return master, err
	}

	return master, nil
}
