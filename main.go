package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"tftgo/pg"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	gq, err := pg.Connect()
	if err != nil {
		panic(err)
	}
	gq.AddQueriesToMap("queries")

	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	riotKey := os.Getenv("RIOT_KEY")

	tft, err := TftGo(riotKey, "na1", false, false, 5)
	if err != nil {
		panic(err)
	}

	// Step 1 - Collect all entries from top tiers
	entries := make([]TftLeagueEntry, 0)

	challengerEntries, _ := tft.TftLeagueV1Challenger()
	entries = append(entries, challengerEntries.Entries...)

	grandmasterEntries, _ := tft.TftLeagueV1Grandmaster()
	entries = append(entries, grandmasterEntries.Entries...)

	masterEntries, _ := tft.TftLeagueV1Master()
	entries = append(entries, masterEntries.Entries...)

	// Step 2 - Collect all summoners from entries
	summonersChan := make(chan TftSummonerResponse)
	var summonersWg sync.WaitGroup
	summonersWg.Add(len(entries))
	for _, entry := range entries {
		go func(entry TftLeagueEntry) {
			defer summonersWg.Done()

			summonerData, _ := tft.TftSummonerV1SummonerId(entry.SummonerId)
			summonersChan <- summonerData
		}(entry)
	}

	go func() {
		summonersWg.Wait()
		close(summonersChan)
	}()

	var summoners []TftSummonerResponse
	for s := range summonersChan {
		summoners = append(summoners, s)
	}

	// Step 3 - Collect all match ids and filter down to unique ones.
	matchIdsChan := make(chan string)
	var matchIdsWg sync.WaitGroup
	for _, summoner := range summoners {
		matchIdsWg.Add(1)
		go func(summoner TftSummonerResponse) {
			defer matchIdsWg.Done()

			matchIds, _ := tft.TftMatchV1MatchesByPuuid(summoner.Puuid)
			for _, id := range matchIds {
				matchIdsChan <- id
			}
		}(summoner)
	}

	go func() {
		matchIdsWg.Wait()
		close(matchIdsChan)
	}()

	uniqueMatchIds := mapset.NewSet[string]()
	for mi := range matchIdsChan {
		uniqueMatchIds.Add(mi)
	}
	// NOTE: We may be able to just use uniqueMatchIds.Iter() instead of cloning.
	matchIds := uniqueMatchIds.ToSlice()
	var matchesWg sync.WaitGroup

	// Step 4 - Ingest match data
	for _, matchId := range matchIds {
		matchesWg.Add(1)
		go func(matchId string) {
			defer matchesWg.Done()

			matchData, _ := tft.TftMatchV1MatchesById(matchId)
			// Step 1 - make sure we mark this matchId as being processed in DB
			gq.Query("queries/upsert_match", matchId)

			// Get the current set and game version for making keys
			set := matchData.Info.TftSetNumber
			rawPatch := matchData.Info.GameVersion
		}(matchId)
	}

	go func() {
		matchesWg.Wait()
	}()
}

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

	// TODO: Do something smarter here
	if t.RateLimit {
		time.Sleep(100 * time.Millisecond)
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
	GameVersion      string                `json:"game_version"`
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

func (t *TFTGO) parsePatchNumber(gameVersion string) string {
	// "Linux Version 14.15.604.8769 (Jul 26 2024/14:34:42) [PUBLIC] "
	gameVersion = strings.TrimSpace(gameVersion)
	splitGameVersion := strings.Split(gameVersion, "Linux Version ")
	splitPatchVersion := strings.Split(splitGameVersion[1], "(")
	// TODO: For B patches we may need like 14.15.604 instead of 14.15...
	patchVersions := strings.Split(splitPatchVersion[0], ".")
	patch := fmt.Sprintf("%s.%s", patchVersions[0], patchVersions[1])
	return patch
}
