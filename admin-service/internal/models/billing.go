package models

type BillingAccount struct {
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

type BillingAccountListResponse struct {
	Total    int64            `json:"total"`
	Page     int32            `json:"page"`
	PageSize int32            `json:"page_size"`
	Items    []BillingAccount `json:"items"`
}

type BillingLedgerEntry struct {
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
	HeldAmountYuan     string `json:"held_amount_yuan"`
	CapturedAmountYuan string `json:"captured_amount_yuan"`
	ReleasedAmountYuan string `json:"released_amount_yuan"`
	ShortfallYuan      string `json:"shortfall_yuan"`
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
	UnitPriceYuanPerGB string `json:"unit_price_yuan_per_gb"`
	AmountYuan         string `json:"amount_yuan"`
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
	IngressPriceYuanPerGB string `json:"ingress_price_yuan_per_gb"`
	EgressPriceYuanPerGB  string `json:"egress_price_yuan_per_gb"`
	Enabled               bool   `json:"enabled"`
	Remark                string `json:"remark"`
	UpdatedByUserID       string `json:"updated_by_user_id"`
	EffectiveAt           string `json:"effective_at"`
	CreatedAt             string `json:"created_at"`
}

type WelcomeCreditSettings struct {
	Enabled      bool   `json:"enabled"`
	AmountYuan   string `json:"amount_yuan"`
	CurrencyCode string `json:"currency_code"`
	UpdatedAt    string `json:"updated_at"`
	UpdatedBy    string `json:"updated_by"`
}
