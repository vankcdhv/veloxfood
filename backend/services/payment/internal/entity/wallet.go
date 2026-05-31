package entity

import "time"

// WalletOwnerType classifies whose wallet this is.
type WalletOwnerType string

const (
	WalletOwnerSystem       WalletOwnerType = "SYSTEM"
	WalletOwnerCustomer     WalletOwnerType = "CUSTOMER"
	WalletOwnerStorePayable WalletOwnerType = "STORE_PAYABLE"
)

// SystemOwnerID is the zero UUID — singleton SYSTEM wallet owner.
const SystemOwnerID = "00000000-0000-0000-0000-000000000000"

// Wallet stores a balance for one owner (SYSTEM, CUSTOMER, or STORE_PAYABLE).
// balance changes only via ledger_entries inside the same DB transaction.
type Wallet struct {
	ID        string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerType WalletOwnerType `gorm:"type:varchar(20);not null"`
	OwnerID   string          `gorm:"type:uuid;not null"`
	Balance   int64           `gorm:"not null;default:0"`
	CreatedAt time.Time       `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time       `gorm:"type:timestamptz;not null;default:now()"`
}

func (Wallet) TableName() string { return "wallets" }
