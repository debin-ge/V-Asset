package models

import "time"

const (
	ProxyUsageSuccessAll     = "all"
	ProxyUsageSuccessOnly    = "success"
	ProxyUsageSuccessFailed  = "failed"
	ProxyUsageSortOrderAsc   = "asc"
	ProxyUsageSortOrderDesc  = "desc"
	ProxyUsageDefaultPage    = 1
	ProxyUsageDefaultPerPage = 20
	ProxyUsageMaxPerPage     = 100
)

type ProxyUsageEventFilter struct {
	TaskID        string
	ProxyID       int64
	ProxyLeaseID  string
	SourceType    string
	Stage         string
	Platform      string
	Success       string
	ErrorCategory string
	StartTime     time.Time
	EndTime       time.Time
	Page          int
	PageSize      int
	SortOrder     string
}

type ProxyUsageEvent struct {
	ID                   int64
	TaskID               string
	ProxyID              int64
	ProxyLeaseID         string
	SourceType           string
	Stage                string
	Platform             string
	Success              bool
	ErrorCategory        string
	ErrorMessage         string
	CreatedAt            time.Time
	ProxyHost            string
	ProxyPort            int32
	ProxyProtocol        string
	ProxyRegion          string
	ProxyRiskScore       int32
	ProxyCooldownUntil   *time.Time
	ProxyActiveTaskCount int32
	ProxyMaxConcurrent   int32
}

type ProxyUsageEventCount struct {
	Key   string
	Count int64
}

type ProxyUsageEventSummary struct {
	SuccessCount   int64
	FailureCount   int64
	FailureRate    float64
	CategoryCounts []ProxyUsageEventCount
	StageCounts    []ProxyUsageEventCount
	PlatformCounts []ProxyUsageEventCount
}

type ProxyUsageEventResult struct {
	Events   []ProxyUsageEvent
	Total    int64
	Page     int
	PageSize int
	Summary  ProxyUsageEventSummary
}
