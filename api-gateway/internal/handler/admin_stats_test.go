package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseAdminTrendQueryDefaultsAndCaps(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		target          string
		wantGranularity string
		wantLimit       int32
	}{
		{
			name:            "invalid values use day default",
			target:          "/?granularity=minute&limit=-1",
			wantGranularity: "day",
			wantLimit:       adminTrendDefaultLimit,
		},
		{
			name:            "hour limit capped",
			target:          "/?granularity=hour&limit=999",
			wantGranularity: "hour",
			wantLimit:       adminTrendMaxHourLimit,
		},
		{
			name:            "day limit capped",
			target:          "/?granularity=day&limit=999",
			wantGranularity: "day",
			wantLimit:       adminTrendMaxDayLimit,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, tt.target, nil)

			granularity, limit := parseAdminTrendQuery(c)
			if granularity != tt.wantGranularity || limit != tt.wantLimit {
				t.Fatalf("expected %s/%d, got %s/%d", tt.wantGranularity, tt.wantLimit, granularity, limit)
			}
		})
	}
}
