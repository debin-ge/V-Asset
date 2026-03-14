package models

type BillingAccountResponse struct {
	UserID              string `json:"user_id"`
	CurrencyCode        string `json:"currency_code"`
	AvailableBalanceFen int64  `json:"available_balance_fen"`
	ReservedBalanceFen  int64  `json:"reserved_balance_fen"`
	TotalRechargedFen   int64  `json:"total_recharged_fen"`
	TotalSpentFen       int64  `json:"total_spent_fen"`
	TotalTrafficBytes   int64  `json:"total_traffic_bytes"`
	Status              int32  `json:"status"`
	Version             int32  `json:"version"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

type BillingStatementRequest struct {
	Page     int   `form:"page,default=1"`
	PageSize int   `form:"page_size,default=20"`
	Type     int32 `form:"type"`
	Status   int32 `form:"status"`
}

type BillingStatementItem struct {
	StatementID  string `json:"statement_id"`
	Type         int32  `json:"type"`
	HistoryID    int64  `json:"history_id"`
	TrafficBytes int64  `json:"traffic_bytes"`
	AmountFen    int64  `json:"amount_fen"`
	Status       int32  `json:"status"`
	Remark       string `json:"remark"`
	CreatedAt    string `json:"created_at"`
}

type BillingEstimateRequest struct {
	URL            string          `json:"url" binding:"required"`
	Platform       string          `json:"platform"`
	Mode           string          `json:"mode"`
	SelectedFormat *SelectedFormat `json:"selected_format,omitempty"`
}

type BillingEstimateResponse struct {
	EstimatedTrafficBytes int64  `json:"estimated_traffic_bytes"`
	EstimatedCostFen      int64  `json:"estimated_cost_fen"`
	PricingVersion        int32  `json:"pricing_version"`
	IsEstimated           bool   `json:"is_estimated"`
	EstimateReason        string `json:"estimate_reason,omitempty"`
}

type AdminBillingListRequest struct {
	Query    string `form:"query"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Status   int32  `form:"status"`
}

type AdminBillingAccount struct {
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

type AdminAdjustBillingBalanceRequest struct {
	OperationID string `json:"operation_id"`
	AmountFen   int64  `json:"amount_fen" binding:"required"`
	Remark      string `json:"remark" binding:"required"`
}

type AdminBillingShortfallRequest struct {
	UserID   string `form:"user_id"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}

type AdminReconcileBillingShortfallRequest struct {
	Remark string `json:"remark"`
}

type AdminBillingShortfallOrder struct {
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

type AdminBillingLedgerRequest struct {
	UserID    string `form:"user_id"`
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=20"`
	EntryType int32  `form:"entry_type"`
}

type AdminBillingLedgerEntry struct {
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

type AdminBillingUsageRequest struct {
	UserID    string `form:"user_id"`
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=20"`
	Direction int32  `form:"direction"`
}

type AdminBillingUsageRecord struct {
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

type AdminBillingPricing struct {
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

type AdminUpdateBillingPricingRequest struct {
	IngressPriceFenPerGiB string `json:"ingress_price_fen_per_gib" binding:"required"`
	EgressPriceFenPerGiB  string `json:"egress_price_fen_per_gib" binding:"required"`
	DefaultEstimateBytes  int64  `json:"default_estimate_bytes" binding:"required"`
	Remark                string `json:"remark"`
}
