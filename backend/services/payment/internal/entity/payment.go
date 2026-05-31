package entity

import "time"

// PaymentMethod is the payment channel chosen by the customer.
type PaymentMethod string

const (
	MethodCOD    PaymentMethod = "COD"
	MethodMoMo   PaymentMethod = "MOMO"
	MethodWallet PaymentMethod = "WALLET"
)

// PaymentProvider is the underlying processor.
type PaymentProvider string

const (
	ProviderMoMo     PaymentProvider = "MOMO"
	ProviderInternal PaymentProvider = "INTERNAL"
)

// PaymentStatus tracks the lifecycle of a payment.
type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "PENDING"
	PaymentCaptured  PaymentStatus = "CAPTURED"
	PaymentFailed    PaymentStatus = "FAILED"
	PaymentRefunded  PaymentStatus = "REFUNDED"
)

// Payment records one payment attempt for an order or a wallet top-up.
// Exactly one of OrderID / TopupID is non-null (enforced by DB constraint).
type Payment struct {
	ID             string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID        *string         `gorm:"type:uuid"`
	TopupID        *string         `gorm:"type:uuid"`
	CustomerID     string          `gorm:"type:uuid;not null"`
	Method         PaymentMethod   `gorm:"type:varchar(20);not null"`
	Provider       PaymentProvider `gorm:"type:varchar(20);not null"`
	Env            string          `gorm:"type:varchar(10);not null;default:'demo'"`
	Amount         int64           `gorm:"not null"`
	Status         PaymentStatus   `gorm:"type:varchar(20);not null;default:'PENDING'"`
	MomoTransID    *string         `gorm:"type:varchar(100);column:momo_trans_id"`
	IdempotencyKey *string         `gorm:"type:varchar(200);uniqueIndex;column:idempotency_key"`
	CreatedAt      time.Time       `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt      time.Time       `gorm:"type:timestamptz;not null;default:now()"`
}

func (Payment) TableName() string { return "payments" }
