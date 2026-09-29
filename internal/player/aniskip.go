package player

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SkipTimestamps struct {
	OpStart float64
	OpEnd   float64
	EdStart float64
	EdEnd   float64
}

var ActiveSkipTimestamps = SkipTimestamps{
	OpStart: -1,
	OpEnd:   -1,
	EdStart: -1,
	EdEnd:   -1,
}

type aniSkipResponse struct {
	Found   bool `json:"found"`
	Results []struct {
		SkipType string `json:"skipType"`
		Interval struct {
			StartTime float64 `json:"startTime"`
			EndTime   float64 `json:"endTime"`
		} `json:"interval"`
	} `json:"results"`
}

// ResetSkipTimestamps clears active intro/outro markers.
func ResetSkipTimestamps() {
	ActiveSkipTimestamps = SkipTimestamps{
		OpStart: -1,
		OpEnd:   -1,
		EdStart: -1,
		EdEnd:   -1,
	}
}

// FetchAniSkip queries the open-source AniSkip API for opening and ending timestamps.
func FetchAniSkip(malID int, epNum string) SkipTimestamps {
	res := SkipTimestamps{OpStart: -1, OpEnd: -1, EdStart: -1, EdEnd: -1}
	if malID <= 0 || epNum == "" {
		return res
	}

	url := fmt.Sprintf("https://api.aniskip.com/v2/skip-times/%d/%s?types=op&types=ed&episodeLength=0", malID, epNum)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return res
	}
	defer resp.Body.Close()

	var data aniSkipResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || !data.Found {
		return res
	}

	for _, item := range data.Results {
		if item.SkipType == "op" {
			res.OpStart = item.Interval.StartTime
			res.OpEnd = item.Interval.EndTime
		} else if item.SkipType == "ed" {
			res.EdStart = item.Interval.StartTime
			res.EdEnd = item.Interval.EndTime
		}
	}
	ActiveSkipTimestamps = res
	return res
}
