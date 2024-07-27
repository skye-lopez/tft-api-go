package tftgo

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

type TFTGO struct {
	key string
}

// TftGo - Struct builder
func TftGo(RiotApiKey string) (TFTGO, error) {
	// Validate API key

	t := TFTGO{
		key: RiotApiKey,
	}

	var result interface{}
	err := t.Request("https://na1.api.riotgames.com/tft/status/v1/platform-data", &result)
	if err != nil {
		return t, errors.New("TftGo - Invalid API key")
	}

	return t, nil
}

// @{Request}
// Basic request function only preforms GET requests.
func (t *TFTGO) Request(url string, target *interface{}) error {
	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
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
