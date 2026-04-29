package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseProxyUsagePaginationCapsPageAndPageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=999999&page_size=999", nil)

	page, err := parsePositiveInt32Query(c, "page", proxyUsageDefaultPage, proxyUsageMaxPage)
	if err != nil {
		t.Fatalf("parse page returned error: %v", err)
	}
	if page != proxyUsageMaxPage {
		t.Fatalf("expected page capped to %d, got %d", proxyUsageMaxPage, page)
	}

	pageSize, err := parsePositiveInt32Query(c, "page_size", proxyUsageDefaultPageSize, proxyUsageMaxPageSize)
	if err != nil {
		t.Fatalf("parse page_size returned error: %v", err)
	}
	if pageSize != proxyUsageMaxPageSize {
		t.Fatalf("expected page_size capped to %d, got %d", proxyUsageMaxPageSize, pageSize)
	}
}

func TestParseProxyUsagePaginationDefaultsInvalidLowValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=0&page_size=-1", nil)

	page, err := parsePositiveInt32Query(c, "page", proxyUsageDefaultPage, proxyUsageMaxPage)
	if err != nil {
		t.Fatalf("parse page returned error: %v", err)
	}
	if page != proxyUsageDefaultPage {
		t.Fatalf("expected default page %d, got %d", proxyUsageDefaultPage, page)
	}

	pageSize, err := parsePositiveInt32Query(c, "page_size", proxyUsageDefaultPageSize, proxyUsageMaxPageSize)
	if err != nil {
		t.Fatalf("parse page_size returned error: %v", err)
	}
	if pageSize != proxyUsageDefaultPageSize {
		t.Fatalf("expected default page_size %d, got %d", proxyUsageDefaultPageSize, pageSize)
	}
}

func TestParseProxyListPaginationCapsPageAndPageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=999999&page_size=999", nil)

	page, err := parsePositiveInt32Query(c, "page", proxyListDefaultPage, proxyListMaxPage)
	if err != nil {
		t.Fatalf("parse page returned error: %v", err)
	}
	if page != proxyListMaxPage {
		t.Fatalf("expected page capped to %d, got %d", proxyListMaxPage, page)
	}

	pageSize, err := parsePositiveInt32Query(c, "page_size", proxyListDefaultPageSize, proxyListMaxPageSize)
	if err != nil {
		t.Fatalf("parse page_size returned error: %v", err)
	}
	if pageSize != proxyListMaxPageSize {
		t.Fatalf("expected page_size capped to %d, got %d", proxyListMaxPageSize, pageSize)
	}
}

func TestParseProxyListSortQuery(t *testing.T) {
	t.Parallel()

	sortBy, sortOrder, err := parseProxyListSortQuery("risk_score", "")
	if err != nil {
		t.Fatalf("parse sort returned error: %v", err)
	}
	if sortBy != "risk_score" || sortOrder != "desc" {
		t.Fatalf("expected risk_score desc, got %q %q", sortBy, sortOrder)
	}

	sortBy, sortOrder, err = parseProxyListSortQuery("updated_at", "asc")
	if err != nil {
		t.Fatalf("parse sort returned error: %v", err)
	}
	if sortBy != "updated_at" || sortOrder != "asc" {
		t.Fatalf("expected updated_at asc, got %q %q", sortBy, sortOrder)
	}
}

func TestParseProxyListSortQueryRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	if _, _, err := parseProxyListSortQuery("host", "desc"); err == nil {
		t.Fatal("expected invalid sort_by error")
	}
	if _, _, err := parseProxyListSortQuery("risk_score", "sideways"); err == nil {
		t.Fatal("expected invalid sort_order error")
	}
}

func TestPublicProxyUsageSourceType(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"manual_pool": "manual",
		"dynamic_api": "dynamic",
		"manual":      "manual",
		"dynamic":     "dynamic",
		"":            "",
	}

	for input, expected := range cases {
		if got := publicProxyUsageSourceType(input); got != expected {
			t.Fatalf("publicProxyUsageSourceType(%q) = %q, want %q", input, got, expected)
		}
	}
}
