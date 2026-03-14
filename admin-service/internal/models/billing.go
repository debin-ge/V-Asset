package models

type BillingAccount struct {
	UserID              string `json:"user_id"`
	Email               string `json:"email,omitempty"`
	Nickname            string `json:"nickname,omitempty"`
	AvailableBalanceFen int64  `json:"available_balance_fen"`
	ReservedBalanceFen  int64  `json:"reserved_balance_fen"`
	TotalRechargedFen   int64  `json:"total_recharged_fen"`
	TotalSpentFen       int64  `json:"total_spent_fen"`
	TotalTrafficBytes   int64  `json:"total_traffic_bytes"`
	Status              int32  `json:"status"`
	Version             int32  `json:"version"`
	UpdatedAt           string `json:"updated_at"`
}

type BillingAccountListResponse struct {
	Total    int64            `json:"total"`
	Page     int32            `json:"page"`
	PageSize int32            `json:"page_size"`
	Items    []BillingAccount `json:"items"`
}

type BillingLedgerEntry struct {
	EntryNo                  string `json:"entry_no"`
	UserID                   string `json:"user_id"`
	Email                    string `json:"email,omitempty"`
	Nickname                 string `json:"nickname,omitempty"`
	OrderNo                  string `json:"order_no"`
	HoldNo                   string `json:"hold_no"`
	HistoryID                int64  `json:"history_id"`
	TaskID                   string `json:"task_id"`
	TransferID               string `json:"transfer_id"`
	OperationID              string `json:"operation_id"`
	EntryType                int32  `json:"entry_type"`
	Scene                    int32  `json:"scene"`
	ActionAmountFen          int64  `json:"action_amount_fen"`
	AvailableDeltaFen        int64  `json:"available_delta_fen"`
	ReservedDeltaFen         int64  `json:"reserved_delta_fen"`
	BalanceAfterAvailableFen int64  `json:"balance_after_available_fen"`
	BalanceAfterReservedFen  int64  `json:"balance_after_reserved_fen"`
	OperatorUserID           string `json:"operator_user_id"`
	Remark                   string `json:"remark"`
	CreatedAt                string `json:"created_at"`
}

type BillingLedgerListResponse struct {
	Total    int64                `json:"total"`
	Page     int32                `json:"page"`
	PageSize int32                `json:"page_size"`
	Items    []BillingLedgerEntry `json:"items"`
}

type BillingShortfallOrder struct {
	OrderNo            string `json:"order_no"`
	UserID             string `json:"user_id"`
	Email              string `json:"email,omitempty"`
	Nickname           string `json:"nickname,omitempty"`
	HistoryID          int64  `json:"history_id"`
	TaskID             string `json:"task_id"`
	Scene              int32  `json:"scene"`
	Status             int32  `json:"status"`
	PricingVersion     int32  `json:"pricing_version"`
	ActualIngressBytes int64  `json:"actual_ingress_bytes"`
	ActualEgressBytes  int64  `json:"actual_egress_bytes"`
	ActualTrafficBytes int64  `json:"actual_traffic_bytes"`
	HeldAmountFen      int64  `json:"held_amount_fen"`
	CapturedAmountFen  int64  `json:"captured_amount_fen"`
	ReleasedAmountFen  int64  `json:"released_amount_fen"`
	ShortfallFen       int64  `json:"shortfall_fen"`
	Remark             string `json:"remark"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type BillingShortfallListResponse struct {
	Total    int64                   `json:"total"`
	Page     int32                   `json:"page"`
	PageSize int32                   `json:"page_size"`
	Items    []BillingShortfallOrder `json:"items"`
}

type BillingUsageRecord struct {
	UsageNo            string `json:"usage_no"`
	OrderNo            string `json:"order_no"`
	UserID             string `json:"user_id"`
	Email              string `json:"email,omitempty"`
	Nickname           string `json:"nickname,omitempty"`
	HistoryID          int64  `json:"history_id"`
	TaskID             string `json:"task_id"`
	TransferID         string `json:"transfer_id"`
	Direction          int32  `json:"direction"`
	TrafficBytes       int64  `json:"traffic_bytes"`
	UnitPriceFenPerGiB string `json:"unit_price_fen_per_gib"`
	AmountFen          int64  `json:"amount_fen"`
	PricingVersion     int32  `json:"pricing_version"`
	SourceService      string `json:"source_service"`
	Status             int32  `json:"status"`
	CreatedAt          string `json:"created_at"`
	ConfirmedAt        string `json:"confirmed_at,omitempty"`
}

type BillingUsageListResponse struct {
	Total    int64                `json:"total"`
	Page     int32                `json:"page"`
	PageSize int32                `json:"page_size"`
	Items    []BillingUsageRecord `json:"items"`
}

type BillingPricing struct {
	Version               int32  `json:"version"`
	IngressPriceFenPerGiB string `json:"ingress_price_fen_per_gib"`
	EgressPriceFenPerGiB  string `json:"egress_price_fen_per_gib"`
	DefaultEstimateBytes  int64  `json:"default_estimate_bytes"`
	Enabled               bool   `json:"enabled"`
	Remark                string `json:"remark"`
	UpdatedByUserID       string `json:"updated_by_user_id"`
	EffectiveAt           string `json:"effective_at"`
	CreatedAt             string `json:"created_at"`
}
