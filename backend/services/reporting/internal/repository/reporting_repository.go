package repository

import (
	"context"
	"time"

	"project/services/reporting/internal/entity"

	"gorm.io/gorm"
)

// ProjectionRepository is the write side: upserts read-model tables from events.
type ProjectionRepository interface {
	// UpsertOrderFact inserts or updates an order_fact row.
	UpsertOrderFact(ctx context.Context, tx *gorm.DB, f *entity.OrderFact) error

	// UpdateOrderFactStatus updates only the status (and updated_at) of an order_fact.
	UpdateOrderFactStatus(ctx context.Context, tx *gorm.DB, orderID, status string) error

	// SetOrderFactSettled marks an order_fact as settled=true.
	SetOrderFactSettled(ctx context.Context, tx *gorm.DB, orderID string) error

	// IncrementRevenueDaily adds delta amounts and counts to revenue_daily for a
	// given store+date, inserting the row if it does not exist yet (upsert).
	IncrementRevenueDaily(ctx context.Context, tx *gorm.DB, storeID string, date time.Time,
		foodDelta, shipDelta int64, ordersDelta, cancelledDelta int) error

	// IncrementUserGrowth adds deltas to user_growth_daily for the given date.
	IncrementUserGrowth(ctx context.Context, tx *gorm.DB, date time.Time, customersDelta, vendorsDelta int) error

	// UpsertStorePerformance adds ratingDelta to store_performance and recomputes avg.
	UpsertStorePerformance(ctx context.Context, tx *gorm.DB, storeID string, ratingDelta int64) error
}

// QueryRepository is the read side: serves HTTP analytics endpoints.
type QueryRepository interface {
	// GetAnalytics returns aggregated platform-wide metrics for the given period
	// window [from, to).
	GetAnalytics(ctx context.Context, from, to time.Time) (*AnalyticsSummary, error)

	// ListRecentOrders returns the most recent order_facts rows up to limit.
	ListRecentOrders(ctx context.Context, limit int) ([]*entity.OrderFact, error)

	// GetStoreRevenue returns revenue_daily rows for a store within [from, to).
	GetStoreRevenue(ctx context.Context, storeID string, from, to time.Time) ([]*entity.RevenueDaily, error)

	// GetStoreReport returns an aggregated summary for a store within [from, to).
	GetStoreReport(ctx context.Context, storeID string, from, to time.Time) (*StoreReport, error)
}

// ProcessedEventRepository deduplicates incoming Kafka events.
type ProcessedEventRepository interface {
	// MarkProcessed inserts event_id. Returns (true, nil) on first insert,
	// (false, nil) on duplicate (already processed), or (false, err) on failure.
	MarkProcessed(ctx context.Context, tx *gorm.DB, eventID string) (bool, error)
}

// AnalyticsSummary holds the aggregated numbers returned by GET /admin/analytics.
// Field names are PascalCase because the HTTP handler marshals this struct directly.
type AnalyticsSummary struct {
	OrdersTotal    int64   `json:"OrdersTotal"`
	RevenueTotal   int64   `json:"RevenueTotal"`
	NewUsersTotal  int     `json:"NewUsersTotal"`
	NewVendors     int     `json:"NewVendors"`
	CancelledTotal int     `json:"CancelledTotal"`
	AvgOrderValue  float64 `json:"AvgOrderValue"`
}

// StoreReport holds the aggregated summary for a store, served by GET /stores/:id/reports.
type StoreReport struct {
	StoreID        string  `json:"StoreID"`
	OrdersTotal    int64   `json:"OrdersTotal"`
	RevenueTotal   int64   `json:"RevenueTotal"`
	CancelledCount int     `json:"CancelledCount"`
	AvgRating      float64 `json:"AvgRating"`
	RatingCount    int     `json:"RatingCount"`
}
