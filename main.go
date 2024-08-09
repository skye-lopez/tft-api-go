// TODO: Can break this into multiple files/packages
// TODO: Backup pg data
package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	goquery "github.com/skye-lopez/go-query"
)

// TODO: Better errors.
func handleError(e error, context string) {
	if e != nil {
		fmt.Println(context)
		panic(e)
	}
}

func main() {
	err := godotenv.Load(".env")
	handleError(err, "Error loading .env file.")

	gq, err := Connect()
	handleError(err, "Error connecting to DB")
	gq.AddQueriesToMap("queries")

	var wg sync.WaitGroup

	wg.Add(1)
	go CollectRegion("na1", &gq, &wg)

	wg.Add(1)
	go CollectRegion("jp1", &gq, &wg)

	wg.Add(1)
	go CollectRegion("kr", &gq, &wg)

	wg.Wait()
}

func CollectRegion(region string, gq *goquery.GoQuery, wg *sync.WaitGroup) {
	defer wg.Done()
	riotKey := os.Getenv("RIOT_KEY")

	tft, err := TftGo(riotKey, region, true, true, 5)
	handleError(err, "TftGo error")

	// get entries ----------------------------------------------------------
	entries := make([]TftLeagueEntry, 0)

	challengerEntries, _, _ := tft.TftLeagueV1Challenger()
	entries = append(entries, challengerEntries.Entries...)

	grandmasterEntries, _, _ := tft.TftLeagueV1Grandmaster()
	entries = append(entries, grandmasterEntries.Entries...)

	masterEntries, _, _ := tft.TftLeagueV1Master()
	entries = append(entries, masterEntries.Entries...)

	// get summoners from entries --------------------------------------------
	summoners := make([]TftSummonerResponse, 0)
	for _, entry := range entries {
		summonerData, _, statusCode := tft.TftSummonerV1SummonerId(entry.SummonerId)
		if statusCode == 200 {
			summoners = append(summoners, summonerData)
		} else {
			fmt.Println(entry)
		}
	}

	// get recent match ids from summoners -----------------------------------
	uniqueMatchIds := mapset.NewSet[string]()
	for _, summoner := range summoners {
		if len(summoner.Puuid) < 2 {
			continue
		}
		matchIds, _, statusCode := tft.TftMatchV1MatchesByPuuid(summoner.Puuid)
		if statusCode == 200 {
			for _, id := range matchIds {
				uniqueMatchIds.Add(id)
			}
		} else {
			fmt.Println(region, summoner)
		}
	}
	matchIds := uniqueMatchIds.ToSlice()
	fmt.Println("Processing matches: ", len(matchIds))

	// Process each match ----------------------------------------------------
	for _, matchId := range matchIds {
		rows, err := gq.Query("queries/get_match", matchId)
		handleError(err, "queries/get_match err:")
		if rows[0].([]interface{})[0] != "none" {
			fmt.Println("Skipping match")
			continue
		}

		matchData, _, statusCode := tft.TftMatchV1MatchesById(matchId)
		if statusCode != 200 {
			continue
		}

		gq.Query("queries/upsert_match", matchId)

		set := matchData.Info.TftSetNumber
		patch := tft.parsePatchNumber(matchData.Info.GameVersion)

		for _, p := range matchData.Info.Participants {
			unitKeys := make([]string, 0)
			unitNames := make([]string, 0)

			// Units
			for _, u := range p.Units {
				unitKey := fmt.Sprintf("%s~%d~%s", u.CharacterId, set, patch)
				unitKeys = append(unitKeys, unitKey)
				unitNames = append(unitNames, u.CharacterId)
				gq.Query("queries/upsert_unit", unitKey, u.CharacterId, p.Placement)

				// For now we only want to track units that completed the match
				// with 3 items
				if len(u.ItemNames) == 3 {
					itemId := fmt.Sprintf("%s~%s~%s~%s", unitKey, u.ItemNames[0], u.ItemNames[1], u.ItemNames[2])
					gq.Query("queries/upsert_unit_item", itemId, unitKey, p.Placement)
				}
			}

			// augments
			for _, a := range p.Augments {
				augmentKey := fmt.Sprintf("%s~%d~%s", a, set, patch)
				gq.Query("queries/upsert_augment", augmentKey, a, p.Placement)
			}

			// teams
			slices.Sort(unitNames)
			slices.Sort(unitKeys)

			formattedUnits := strings.Join(unitNames[:], "~")
			teamKey := fmt.Sprintf("%s~%d~%s", formattedUnits, set, patch)
			gq.Query("queries/upsert_team", teamKey, set, patch, pq.Array(unitKeys), p.Placement)
		}
	}
}

func Connect() (goquery.GoQuery, error) {
	connString := fmt.Sprintf("user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("PG_USER"),
		os.Getenv("PG_PWD"),
		os.Getenv("PG_DBNAME"),
		os.Getenv("PG_PORT"))
	fmt.Println(connString)

	conn, err := sql.Open("postgres", connString)
	if err != nil {
		panic(err)
	}
	gq := goquery.NewGoQuery(conn)

	_, err = gq.Conn.Exec("SELECT 1 as test")
	if err != nil {
		panic(err)
	}
	return gq, err
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

	regionMap := make(map[string]string, 0)
	regionMap["na1"] = "americas"
	regionMap["kr"] = "asia"
	regionMap["jp1"] = "asia"

	t.AltRegion = regionMap[region]

	// Validate API key
	var result interface{}
	err, _ := t.Request("tft/status/v1/platform-data", false, &result, t.RetryCount)
	if err != nil {
		return t, errors.New("TftGo - Invalid API key")
	}

	return t, nil
}

// NOTE: Rate limits should apply by region, idk if that means na1,euw1,kr... or asia,americas,europe groups.
func (t *TFTGO) Request(url string, isAltRegion bool, target interface{}, retryCount int) (error, int) {
	// proper region mapping
	u := ""
	if isAltRegion {
		u = "https://" + t.AltRegion + ".api.riotgames.com/" + url
	} else {
		u = "https://" + t.Region + ".api.riotgames.com/" + url
	}

	client := http.Client{}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err, 0
	}

	req.Header = http.Header{
		"X-Riot-Token": {t.key},
	}

	// TODO: This really needs to be better. Likely by keeping a global req count and doing some math.
	// Also on non-test keys should be able to lower it greatly.
	if t.RateLimit {
		time.Sleep(500 * time.Millisecond)
	}

	// Only retry on the DO
	res, err := client.Do(req)
	if err != nil && retryCount == 0 {
		fmt.Println("request err: ", err)
		return err, 0
	} else if err != nil && retryCount > 0 {
		// NOTE: This is hard coded to 1, but may want to make it variable
		time.Sleep(1 * time.Second)
		return t.Request(url, isAltRegion, target, (retryCount - 1))
	}
	defer res.Body.Close()

	if t.ShowLogs {
		fmt.Println("\n+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+")
		log.Printf("  ::TFTGO Request::\n%v\n", u)
		fmt.Println(res.StatusCode)
		fmt.Println("+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+~+")
	}

	if res.StatusCode == 403 {
		return errors.New("Status Code 403: Foribidden. This likely means the key given is either invalid or does not have access to the selected endpoint."), 403
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err, 0
	}

	err = json.Unmarshal(body, &target)
	if err != nil {
		return err, 0
	}

	return nil, res.StatusCode
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

func (t *TFTGO) TftLeagueV1Challenger() (TftLeagueResponse, error, int) {
	url := "tft/league/v1/challenger?queue=RANKED_TFT"
	challenger := TftLeagueResponse{}
	err, statusCode := t.Request(url, false, &challenger, t.RetryCount)
	if err != nil {
		return challenger, err, statusCode
	}

	return challenger, nil, statusCode
}

func (t *TFTGO) TftLeagueV1Grandmaster() (TftLeagueResponse, error, int) {
	url := "tft/league/v1/grandmaster?queue=RANKED_TFT"
	grandmaster := TftLeagueResponse{}
	err, statusCode := t.Request(url, false, &grandmaster, t.RetryCount)
	if err != nil {
		return grandmaster, err, statusCode
	}

	return grandmaster, nil, statusCode
}

func (t *TFTGO) TftLeagueV1Master() (TftLeagueResponse, error, int) {
	url := "tft/league/v1/master?queue=RANKED_TFT"
	master := TftLeagueResponse{}
	err, statusCode := t.Request(url, false, &master, t.RetryCount)
	if err != nil {
		return master, err, statusCode
	}

	return master, nil, statusCode
}

type TftSummonerResponse struct {
	Id            string `json:"id"`
	AccountId     string `json:"accountId"`
	Puuid         string `json:"puuid"`
	ProfileIconId int    `json:"profileIconId"`
	RevisionDate  int    `json:"revisionDate"`
	SummonerLevel int    `json:"summonerLevel"`
}

func (t *TFTGO) TftSummonerV1SummonerId(summonerId string) (TftSummonerResponse, error, int) {
	url := "tft/summoner/v1/summoners/" + summonerId
	summoner := TftSummonerResponse{}
	err, statusCode := t.Request(url, false, &summoner, t.RetryCount)
	if err != nil {
		return summoner, err, statusCode
	}

	return summoner, err, statusCode
}

func (t *TFTGO) TftMatchV1MatchesByPuuid(puuid string) ([]string, error, int) {
	// NOTE: Update startTime per set
	url := "tft/match/v1/matches/by-puuid/" + puuid + "/ids?start=0&startTime=1722474000&count=1000"
	ids := make([]string, 0)
	err, statusCode := t.Request(url, true, &ids, t.RetryCount)
	if err != nil {
		return ids, err, statusCode
	}

	return ids, nil, statusCode
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

func (t *TFTGO) TftMatchV1MatchesById(matchId string) (TftMatchResponse, error, int) {
	url := "tft/match/v1/matches/" + matchId
	matchData := TftMatchResponse{}
	err, statusCode := t.Request(url, true, &matchData, t.RetryCount)
	if err != nil {
		return matchData, err, statusCode
	}

	return matchData, nil, statusCode
}

func (t *TFTGO) parsePatchNumber(gameVersion string) string {
	gameVersion = strings.TrimSpace(gameVersion)
	splitGameVersion := strings.Split(gameVersion, "<Releases/")
	gameVersion = splitGameVersion[len(splitGameVersion)-1]
	gameVersion = strings.Trim(gameVersion, ">")
	return gameVersion
}
