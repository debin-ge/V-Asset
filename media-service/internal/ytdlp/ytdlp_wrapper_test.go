package ytdlp

import (
	"encoding/json"
	"testing"
)

func TestVideoInfoUnmarshalAcceptsFractionalDuration(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"id": "BV1Qb4y117rv_p1",
		"title": "Bilibili video",
		"duration": 212.23,
		"formats": []
	}`)

	var info VideoInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	if info.Duration != 212 {
		t.Fatalf("Duration = %d, want 212", info.Duration)
	}
}
