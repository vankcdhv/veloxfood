package persistence

import (
	"context"
	"time"

	"project/services/reporting/internal/entity"
	"project/services/reporting/internal/repository"

	"gorm.io/gorm"
)

type queryGormRepository struct {
	db *gorm.DB
}

// NewQueryGormRepository returns a QueryRepository backed by GORM.
func NewQueryGormRepository(db *gorm.DB) repository.QueryRepository {
	return &queryGormRepository{db: db}
}

func (r *queryGormRepository) GetAnalytics(ctx context.Context, from, to time.Time) (*repository.AnalyticsSummary, error) {
	var summary repository.AnalyticsSummary

	// Order aggregates from order_facts within the window.
	var orderRow struct {
		OrdersTotal    int64
		RevenueTotal   int64
		CancelledTotal int
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
		  COUNT(*)                                                   AS orders_total,
		  COALESCE(SUM(grand_total), 0)                             AS revenue_total,
		  COUNT(*) FILTER (WHERE status = 'CANCELLED')              AS cancelled_total
		FROM order_facts
		WHERE created_at >= ? AND created_at < ?`, from, to,
	).Scan(&orderRow).Error; err != nil {
		return nil, err
	}
	summary.OrdersTotal = orderRow.OrdersTotal
	summary.RevenueTotal = orderRow.RevenueTotal
	summary.CancelledTotal = orderRow.CancelledTotal
	if orderRow.OrdersTotal > 0 {
		summary.AvgOrderValue = float64(orderRow.RevenueTotal) / float64(orderRow.OrdersTotal)
	}

	// User growth within the window.
	var growthRow struct {
		NewCustomers int
		NewVendors   int
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
		  COALESCE(SUM(new_customers), 0) AS new_customers,
		  COALESCE(SUM(new_vendors),   0) AS new_vendors
		FROM user_growth_daily
		WHERE date >= ? AND date < ?`, from.Format("2006-01-02"), to.Format("2006-01-02"),
	).Scan(&growthRow).Error; err != nil {
		return nil, err
	}
	summary.NewUsersTotal = growthRow.NewCustomers
	summary.NewVendors = growthRow.NewVendors

	return &summary, nil
}

func (r *queryGormRepository) ListRecentOrders(ctx context.Context, limit int) ([]*entity.OrderFact, error) {
	if limit <= 0 {
		limit = 20
	}
	var facts []*entity.OrderFact
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&facts).Error; err != nil {
		return nil, err
	}
	return facts, nil
}

func (r *queryGormRepository) GetStoreRevenue(ctx context.Context, storeID string, from, to time.Time) ([]*entity.RevenueDaily, error) {
	var rows []*entity.RevenueDaily
	if err := r.db.WithContext(ctx).
		Where("store_id = ? AND date >= ? AND date < ?", storeID, from.Format("2006-01-02"), to.Format("2006-01-02")).
		Order("date ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *queryGormRepository) GetStoreReport(ctx context.Context, storeID string, from, to time.Time) (*repository.StoreReport, error) {
	var row struct {
		OrdersTotal    int64
		RevenueTotal   int64
		CancelledCount int
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
		  COUNT(*)                                              AS orders_total,
		  COALESCE(SUM(grand_total), 0)                        AS revenue_total,
		  COUNT(*) FILTER (WHERE status = 'CANCELLED')         AS cancelled_count
		FROM order_facts
		WHERE store_id = ? AND created_at >= ? AND created_at < ?`, storeID, from, to,
	).Scan(&row).Error; err != nil {
		return nil, err
	}

	var perf entity.StorePerformance
	r.db.WithContext(ctx).Where("store_id = ?", storeID).First(&perf) // ignore not-found

	return &repository.StoreReport{
		StoreID:        storeID,
		OrdersTotal:    row.OrdersTotal,
		RevenueTotal:   row.RevenueTotal,
		CancelledCount: row.CancelledCount,
		AvgRating:      perf.AvgRating,
		RatingCount:    perf.RatingCount,
	}, nil
}

var _ repository.QueryRepository = (*queryGormRepository)(nil)
