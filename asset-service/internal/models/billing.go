package models

import (
	"time"

	"vasset/asset-service/internal/money"
)

const (
	BillingAccountStatusActive = 1
	BillingAccountStatusFrozen = 2
	BillingAccountStatusClosed = 3
)

const (
	BillingSceneDownload   = 1
	BillingSceneRedownload = 2
	BillingSceneAdmin      = 3
)

const (
	BillingOrderStatusHeld              = 1
	BillingOrderStatusPartialCaptured   = 2
	BillingOrderStatusCaptured          = 3
	BillingOrderStatusReleased          = 4
	BillingOrderStatusAwaitingShortfall = 5
)

const (
	BillingHoldTypeDownloadTotal = 1
	BillingHoldTypeFileTransfer  = 2
)

const (
	BillingFundingSourceNewReserve      = 1
	BillingFundingSourceExistingReserve = 2
)

const (
	BillingHoldStatusHeld            = 1
	BillingHoldStatusPartialCaptured = 2
	BillingHoldStatusCaptured        = 3
	BillingHoldStatusReleased        = 4
	BillingHoldStatusExpired         = 5
)

const (
	TrafficDirectionIngress = 1
	TrafficDirectionEgress  = 2
)

const (
	TrafficUsageStatusRecorded  = 1
	TrafficUsageStatusConfirmed = 2
	TrafficUsageStatusCancelled = 3
)

const (
	LedgerEntryTypeManualTopup      = 1
	LedgerEntryTypeManualAdjustment = 2
	LedgerEntryTypeHold             = 3
	LedgerEntryTypeCapture          = 4
	LedgerEntryTypeRelease          = 5
)

const (
	StatementTypeRecharge   = 1
	StatementTypeDownload   = 2
	StatementTypeAdjustment = 3
)

type BillingAccount struct {
	ID                  int64         `db:"id"`
	UserID              string        `db:"user_id"`
	CurrencyCode        string        `db:"currency_code"`
	AvailableBalanceFen money.Decimal `db:"available_balance_fen"`
	ReservedBalanceFen  money.Decimal `db:"reserved_balance_fen"`
	TotalRechargedFen   money.Decimal `db:"total_recharged_fen"`
	TotalSpentFen       money.Decimal `db:"total_spent_fen"`
	TotalTrafficBytes   int64         `db:"total_traffic_bytes"`
	Status              int32         `db:"status"`
	Version             int32         `db:"version"`
	CreatedAt           time.Time     `db:"created_at"`
	UpdatedAt           time.Time     `db:"updated_at"`
}

type BillingPricing struct {
	ID                    int64         `db:"id"`
	Version               int32         `db:"version"`
	IngressPriceFenPerGiB money.Decimal `db:"ingress_price_fen_per_gib"`
	EgressPriceFenPerGiB  money.Decimal `db:"egress_price_fen_per_gib"`
	Enabled               bool          `db:"enabled"`
	Remark                string        `db:"remark"`
	UpdatedByUserID       string        `db:"updated_by_user_id"`
	EffectiveAt           time.Time     `db:"effective_at"`
	CreatedAt             time.Time     `db:"created_at"`
}

type BillingChargeOrder struct {
	ID                    int64         `db:"id"`
	OrderNo               string        `db:"order_no"`
	UserID                string        `db:"user_id"`
	HistoryID             int64         `db:"history_id"`
	TaskID                string        `db:"task_id"`
	Scene                 int32         `db:"scene"`
	Status                int32         `db:"status"`
	PricingVersion        int32         `db:"pricing_version"`
	EstimatedIngressBytes int64         `db:"estimated_ingress_bytes"`
	EstimatedEgressBytes  int64         `db:"estimated_egress_bytes"`
	EstimatedTrafficBytes int64         `db:"estimated_traffic_bytes"`
	ActualIngressBytes    int64         `db:"actual_ingress_bytes"`
	ActualEgressBytes     int64         `db:"actual_egress_bytes"`
	ActualTrafficBytes    int64         `db:"actual_traffic_bytes"`
	HeldAmountFen         money.Decimal `db:"held_amount_fen"`
	CapturedAmountFen     money.Decimal `db:"captured_amount_fen"`
	ReleasedAmountFen     money.Decimal `db:"released_amount_fen"`
	ShortfallFen          money.Decimal `db:"shortfall_fen"`
	Remark                string        `db:"remark"`
	CreatedAt             time.Time     `db:"created_at"`
	UpdatedAt             time.Time     `db:"updated_at"`
	ClosedAt              *time.Time    `db:"closed_at"`
}

type BillingHold struct {
	ID                int64         `db:"id"`
	HoldNo            string        `db:"hold_no"`
	OrderNo           string        `db:"order_no"`
	UserID            string        `db:"user_id"`
	HistoryID         int64         `db:"history_id"`
	TaskID            string        `db:"task_id"`
	TransferID        string        `db:"transfer_id"`
	HoldType          int32         `db:"hold_type"`
	FundingSource     int32         `db:"funding_source"`
	Status            int32         `db:"status"`
	AmountFen         money.Decimal `db:"amount_fen"`
	CapturedAmountFen money.Decimal `db:"captured_amount_fen"`
	ReleasedAmountFen money.Decimal `db:"released_amount_fen"`
	ExpiresAt         *time.Time    `db:"expires_at"`
	CreatedAt         time.Time     `db:"created_at"`
	UpdatedAt         time.Time     `db:"updated_at"`
}

type TrafficUsageRecord struct {
	ID                 int64         `db:"id"`
	UsageNo            string        `db:"usage_no"`
	OrderNo            string        `db:"order_no"`
	UserID             string        `db:"user_id"`
	HistoryID          int64         `db:"history_id"`
	TaskID             string        `db:"task_id"`
	TransferID         string        `db:"transfer_id"`
	Direction          int32         `db:"direction"`
	TrafficBytes       int64         `db:"traffic_bytes"`
	UnitPriceFenPerGiB money.Decimal `db:"unit_price_fen_per_gib"`
	AmountFen          money.Decimal `db:"amount_fen"`
	PricingVersion     int32         `db:"pricing_version"`
	SourceService      string        `db:"source_service"`
	Status             int32         `db:"status"`
	CreatedAt          time.Time     `db:"created_at"`
	ConfirmedAt        *time.Time    `db:"confirmed_at"`
}

type BillingLedgerEntry struct {
	ID                       int64         `db:"id"`
	EntryNo                  string        `db:"entry_no"`
	AccountID                int64         `db:"account_id"`
	UserID                   string        `db:"user_id"`
	OrderNo                  string        `db:"order_no"`
	HoldNo                   string        `db:"hold_no"`
	HistoryID                int64         `db:"history_id"`
	TaskID                   string        `db:"task_id"`
	TransferID               string        `db:"transfer_id"`
	OperationID              string        `db:"operation_id"`
	EntryType                int32         `db:"entry_type"`
	Scene                    int32         `db:"scene"`
	ActionAmountFen          money.Decimal `db:"action_amount_fen"`
	AvailableDeltaFen        money.Decimal `db:"available_delta_fen"`
	ReservedDeltaFen         money.Decimal `db:"reserved_delta_fen"`
	BalanceAfterAvailableFen money.Decimal `db:"balance_after_available_fen"`
	BalanceAfterReservedFen  money.Decimal `db:"balance_after_reserved_fen"`
	OperatorUserID           string        `db:"operator_user_id"`
	Remark                   string        `db:"remark"`
	CreatedAt                time.Time     `db:"created_at"`
}

type BillingStatementItem struct {
	StatementID  string        `db:"statement_id"`
	Type         int32         `db:"type"`
	HistoryID    int64         `db:"history_id"`
	TrafficBytes int64         `db:"traffic_bytes"`
	AmountFen    money.Decimal `db:"amount_fen"`
	Status       int32         `db:"status"`
	Remark       string        `db:"remark"`
	CreatedAt    time.Time     `db:"created_at"`
}

type BillingStatementResult struct {
	Total    int64
	Page     int
	PageSize int
	Items    []BillingStatementItem
}

type BillingShortfallOrder struct {
	OrderNo            string
	UserID             string
	HistoryID          int64
	TaskID             string
	Scene              int32
	Status             int32
	PricingVersion     int32
	ActualIngressBytes int64
	ActualEgressBytes  int64
	ActualTrafficBytes int64
	HeldAmountFen      money.Decimal
	CapturedAmountFen  money.Decimal
	ReleasedAmountFen  money.Decimal
	ShortfallFen       money.Decimal
	Remark             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type BillingShortfallFilter struct {
	UserID   string
	Page     int
	PageSize int
}

type BillingShortfallResult struct {
	Total    int64
	Page     int
	PageSize int
	Items    []BillingShortfallOrder
}

type BillingLedgerFilter struct {
	UserID    string
	Page      int
	PageSize  int
	EntryType int32
}

type BillingLedgerResult struct {
	Total    int64
	Page     int
	PageSize int
	Items    []BillingLedgerEntry
}

type BillingAccountFilter struct {
	UserIDs  []string
	Page     int
	PageSize int
	Status   int32
}

type BillingAccountResult struct {
	Total    int64
	Page     int
	PageSize int
	Items    []BillingAccount
}

type TrafficUsageFilter struct {
	UserID    string
	Page      int
	PageSize  int
	Direction int32
}

type TrafficUsageResult struct {
	Total    int64
	Page     int
	PageSize int
	Items    []TrafficUsageRecord
}

type BillingEstimate struct {
	EstimatedIngressBytes int64
	EstimatedEgressBytes  int64
	EstimatedTrafficBytes int64
	EstimatedCostFen      money.Decimal
	PricingVersion        int32
	IsEstimated           bool
	EstimateReason        string
}
