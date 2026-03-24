package models

type BillingAccountOverviewResponse struct {
	UserID               string `json:"user_id"`
	CurrencyCode         string `json:"currency_code"`
	AvailableBalanceYuan string `json:"available_balance_yuan"`
	ReservedBalanceYuan  string `json:"reserved_balance_yuan"`
	TotalRechargedYuan   string `json:"total_recharged_yuan"`
	TotalSpentYuan       string `json:"total_spent_yuan"`
	TotalTrafficBytes    int64  `json:"total_traffic_bytes"`
	Status               int32  `json:"status"`
	Version              int32  `json:"version"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

type BillingStatementListResponse struct {
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	Items    []BillingStatementItem `json:"items"`
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
	AmountYuan   string `json:"amount_yuan"`
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
	EstimatedCostYuan     string `json:"estimated_cost_yuan"`
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
	UserID               string `json:"user_id"`
	Email                string `json:"email,omitempty"`
	Nickname             string `json:"nickname,omitempty"`
	AvailableBalanceYuan string `json:"available_balance_yuan"`
	ReservedBalanceYuan  string `json:"reserved_balance_yuan"`
	TotalRechargedYuan   string `json:"total_recharged_yuan"`
	TotalSpentYuan       string `json:"total_spent_yuan"`
	TotalTrafficBytes    int64  `json:"total_traffic_bytes"`
	Status               int32  `json:"status"`
	Version              int32  `json:"version"`
	UpdatedAt            string `json:"updated_at"`
}

type AdminAdjustBillingBalanceRequest struct {
	OperationID string `json:"operation_id"`
	AmountYuan  string `json:"amount_yuan" binding:"required"`
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
	HeldAmountYuan     string `json:"held_amount_yuan"`
	CapturedAmountYuan string `json:"captured_amount_yuan"`
	ReleasedAmountYuan string `json:"released_amount_yuan"`
	ShortfallYuan      string `json:"shortfall_yuan"`
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
	EntryNo                   string `json:"entry_no"`
	UserID                    string `json:"user_id"`
	Email                     string `json:"email,omitempty"`
	Nickname                  string `json:"nickname,omitempty"`
	OrderNo                   string `json:"order_no"`
	HoldNo                    string `json:"hold_no"`
	HistoryID                 int64  `json:"history_id"`
	TaskID                    string `json:"task_id"`
	TransferID                string `json:"transfer_id"`
	OperationID               string `json:"operation_id"`
	EntryType                 int32  `json:"entry_type"`
	Scene                     int32  `json:"scene"`
	ActionAmountYuan          string `json:"action_amount_yuan"`
	AvailableDeltaYuan        string `json:"available_delta_yuan"`
	ReservedDeltaYuan         string `json:"reserved_delta_yuan"`
	BalanceAfterAvailableYuan string `json:"balance_after_available_yuan"`
	BalanceAfterReservedYuan  string `json:"balance_after_reserved_yuan"`
	OperatorUserID            string `json:"operator_user_id"`
	Remark                    string `json:"remark"`
	CreatedAt                 string `json:"created_at"`
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
	UnitPriceYuanPerGB string `json:"unit_price_yuan_per_gb"`
	AmountYuan         string `json:"amount_yuan"`
	PricingVersion     int32  `json:"pricing_version"`
	SourceService      string `json:"source_service"`
	Status             int32  `json:"status"`
	CreatedAt          string `json:"created_at"`
	ConfirmedAt        string `json:"confirmed_at,omitempty"`
}

type AdminBillingPricing struct {
	Version               int32  `json:"version"`
	IngressPriceYuanPerGB string `json:"ingress_price_yuan_per_gb"`
	EgressPriceYuanPerGB  string `json:"egress_price_yuan_per_gb"`
	Enabled               bool   `json:"enabled"`
	Remark                string `json:"remark"`
	UpdatedByUserID       string `json:"updated_by_user_id"`
	EffectiveAt           string `json:"effective_at"`
	CreatedAt             string `json:"created_at"`
}

type AdminUpdateBillingPricingRequest struct {
	IngressPriceYuanPerGB string `json:"ingress_price_yuan_per_gb" binding:"required"`
	EgressPriceYuanPerGB  string `json:"egress_price_yuan_per_gb" binding:"required"`
	Remark                string `json:"remark"`
}

type AdminWelcomeCreditSettings struct {
	Enabled      bool   `json:"enabled"`
	AmountYuan   string `json:"amount_yuan"`
	CurrencyCode string `json:"currency_code"`
	UpdatedAt    string `json:"updated_at"`
	UpdatedBy    string `json:"updated_by"`
}

type AdminUpdateWelcomeCreditSettingsRequest struct {
	Enabled      bool   `json:"enabled"`
	AmountYuan   string `json:"amount_yuan" binding:"required"`
	CurrencyCode string `json:"currency_code" binding:"required"`
}
