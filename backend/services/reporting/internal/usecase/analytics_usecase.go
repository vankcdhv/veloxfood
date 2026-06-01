package usecase

import (
	"context"
	"time"

	"project/services/reporting/internal/entity"
	"project/services/reporting/internal/repository"
)

// AnalyticsUsecase handles the read-side analytics queries for HTTP handlers.
type AnalyticsUsecase struct {
	queryRepo repository.QueryRepository
}

func NewAnalyticsUsecase(queryRepo repository.QueryRepository) *AnalyticsUsecase {
	return &AnalyticsUsecase{queryRepo: queryRepo}
}

// GetAnalytics returns platform-wide aggregated metrics for the given period
// ("day", "week", "month"). Defaults to "day" if unrecognised.
func (u *AnalyticsUsecase) GetAnalytics(ctx context.Context, period string) (*repository.AnalyticsSummary, error) {
	from, to := periodWindow(period)
	return u.queryRepo.GetAnalytics(ctx, from, to)
}

// ListRecentOrders returns the most recent order_facts rows up to limit.
func (u *AnalyticsUsecase) ListRecentOrders(ctx context.Context, limit int) ([]*entity.OrderFact, error) {
	return u.queryRepo.ListRecentOrders(ctx, limit)
}

// GetStoreRevenue returns revenue_daily rows for a store within the period window.
func (u *AnalyticsUsecase) GetStoreRevenue(ctx context.Context, storeID, period string) ([]*entity.RevenueDaily, error) {
	from, to := periodWindow(period)
	return u.queryRepo.GetStoreRevenue(ctx, storeID, from, to)
}

// GetStoreReport returns an aggregated report for a store within the period window.
func (u *AnalyticsUsecase) GetStoreReport(ctx context.Context, storeID, period string) (*repository.StoreReport, error) {
	from, to := periodWindow(period)
	return u.queryRepo.GetStoreReport(ctx, storeID, from, to)
}

// periodWindow converts a period string to [from, to) UTC timestamps anchored
// to the start of today. "day" = last 24 h, "week" = last 7 days,
// "month" = last 30 days, default = "day".
func periodWindow(period string) (from, to time.Time) {
	now := time.Now().UTC()
	// anchor to start of today
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	to = today.AddDate(0, 0, 1) // exclusive upper bound = start of tomorrow
	switch period {
	case "week":
		from = today.AddDate(0, 0, -6)
	case "month":
		from = today.AddDate(0, 0, -29)
	default: // "day"
		from = today
	}
	return from, to
}
