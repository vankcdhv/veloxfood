package entity

import "time"

// LedgerEntryType classifies the reason for a money movement.
type LedgerEntryType string

const (
	LedgerTopup   LedgerEntryType = "TOPUP"
	LedgerPayment LedgerEntryType = "PAYMENT"
	LedgerRefund  LedgerEntryType = "REFUND"
	LedgerPayout  LedgerEntryType = "PAYOUT"
)

// LedgerRefType classifies the business entity referenced by a ledger entry.
type LedgerRefType string

const (
	LedgerRefOrder       LedgerRefType = "order"
	LedgerRefTopup       LedgerRefType = "topup"
	LedgerRefPayoutBatch LedgerRefType = "payout_batch"
)

// LedgerEntry is an append-only record of one side of a double-entry transfer.
// amount is signed: positive = credit to wallet, negative = debit from wallet.
// Every money movement produces exactly two entries whose amounts sum to zero.
type LedgerEntry struct {
	ID           string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WalletID     string          `gorm:"type:uuid;not null"`
	EntryType    LedgerEntryType `gorm:"type:varchar(20);not null"`
	Amount       int64           `gorm:"not null"` // signed VND
	RefType      LedgerRefType   `gorm:"type:varchar(20);not null"`
	RefID        string          `gorm:"type:uuid;not null"`
	BalanceAfter int64           `gorm:"not null"`
	TraceID      *string         `gorm:"type:varchar(80)"`
	CreatedAt    time.Time       `gorm:"type:timestamptz;not null;default:now()"`
}

func (LedgerEntry) TableName() string { return "ledger_entries" }
