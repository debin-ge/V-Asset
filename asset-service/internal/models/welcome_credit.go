package models

import (
	"time"

	"vasset/asset-service/internal/money"
)

const WelcomeCreditReasonCode = "welcome_credit"

type WelcomeCreditSettings struct {
	Enabled      bool
	AmountYuan   money.Decimal
	CurrencyCode string
	UpdatedAt    time.Time
	UpdatedBy    string
}

type WelcomeCreditGrant struct {
	ID            int64
	UserID        string
	OperationID   string
	LedgerEntryNo string
	ReasonCode    string
	AmountYuan    money.Decimal
	CurrencyCode  string
	CreatedAt     time.Time
}
