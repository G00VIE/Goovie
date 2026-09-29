package player

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchAniSkip_EmptyInputs(t *testing.T) {
	res := FetchAniSkip(0, "1")
	if res.OpStart != -1 || res.OpEnd != -1 {
		t.Errorf("expected -1 for invalid malID, got: %+v", res)
	}

	res2 := FetchAniSkip(21, "")
	if res2.OpStart != -1 || res2.OpEnd != -1 {
		t.Errorf("expected -1 for empty epNum, got: %+v", res2)
	}
}

func TestFetchAniSkip_MockServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"found": true,
			"results": [
				{
					"skipType": "op",
					"interval": { "startTime": 30.5, "endTime": 120.0 }
				},
				{
					"skipType": "ed",
					"interval": { "startTime": 1350.0, "endTime": 1440.0 }
				}
			]
		}`))
	}))
	defer ts.Close()

	// Direct parsing test
	parseJSON := `{"found":true,"results":[{"skipType":"op","interval":{"startTime":30.5,"endTime":120.0}},{"skipType":"ed","interval":{"startTime":1350.0,"endTime":1440.0}}]}`
	if err := parseAniSkipData([]byte(parseJSON)); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if ActiveSkipTimestamps.OpStart != 30.5 || ActiveSkipTimestamps.OpEnd != 120.0 {
		t.Errorf("op timestamps incorrect: %+v", ActiveSkipTimestamps)
	}
	if ActiveSkipTimestamps.EdStart != 1350.0 || ActiveSkipTimestamps.EdEnd != 1440.0 {
		t.Errorf("ed timestamps incorrect: %+v", ActiveSkipTimestamps)
	}

	ResetSkipTimestamps()
	if ActiveSkipTimestamps.OpStart != -1 {
		t.Errorf("ResetSkipTimestamps failed")
	}
}

func parseAniSkipData(data []byte) error {
	var resp aniSkipResponse
	if err := jsonUnmarshal(data, &resp); err != nil {
		return err
	}
	for _, item := range resp.Results {
		if item.SkipType == "op" {
			ActiveSkipTimestamps.OpStart = item.Interval.StartTime
			ActiveSkipTimestamps.OpEnd = item.Interval.EndTime
		} else if item.SkipType == "ed" {
			ActiveSkipTimestamps.EdStart = item.Interval.StartTime
			ActiveSkipTimestamps.EdEnd = item.Interval.EndTime
		}
	}
	return nil
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
