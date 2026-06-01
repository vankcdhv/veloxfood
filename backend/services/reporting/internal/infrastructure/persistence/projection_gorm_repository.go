package persistence

import (
	"context"
	"time"

	"project/services/reporting/internal/entity"
	"project/services/reporting/internal/repository"

	"gorm.io/gorm"
)

type projectionGormRepository struct {
	db *gorm.DB
}

// NewProjectionGormRepository returns a ProjectionRepository backed by GORM.
func NewProjectionGormRepository(db *gorm.DB) repository.ProjectionRepository {
	return &projectionGormRepository{db: db}
}

func (r *projectionGormRepository) UpsertOrderFact(ctx context.Context, tx *gorm.DB, f *entity.OrderFact) error {
	return tx.WithContext(ctx).
		Exec(`INSERT INTO order_facts
			(order_id, code, store_id, customer_id, date, fulfillment,
			 items_total, ship_fee, discount, grand_total,
			 payment_method, status, settled, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,NOW(),NOW())
			ON CONFLICT (order_id) DO UPDATE SET
			  status     = EXCLUDED.status,
			  settled    = EXCLUDED.settled,
			  updated_at = NOW()`,
			f.OrderID, f.Code, f.StoreID, f.CustomerID, f.Date, f.Fulfillment,
			f.ItemsTotal, f.ShipFee, f.Discount, f.GrandTotal,
			f.PaymentMethod, f.Status, f.Settled,
		).Error
}

func (r *projectionGormRepository) UpdateOrderFactStatus(ctx context.Context, tx *gorm.DB, orderID, status string) error {
	return tx.WithContext(ctx).
		Exec(`UPDATE order_facts SET status=?, updated_at=NOW() WHERE order_id=?`, status, orderID).
		Error
}

func (r *projectionGormRepository) SetOrderFactSettled(ctx context.Context, tx *gorm.DB, orderID string) error {
	return tx.WithContext(ctx).
		Exec(`UPDATE order_facts SET settled=true, updated_at=NOW() WHERE order_id=?`, orderID).
		Error
}

func (r *projectionGormRepository) IncrementRevenueDaily(
	ctx context.Context, tx *gorm.DB,
	storeID string, date time.Time,
	foodDelta, shipDelta int64,
	ordersDelta, cancelledDelta int,
) error {
	return tx.WithContext(ctx).
		Exec(`INSERT INTO revenue_daily (store_id, date, total_food, total_ship, orders_count, cancelled_count)
			  VALUES (?,?,?,?,?,?)
			  ON CONFLICT (store_id, date) DO UPDATE SET
			    total_food      = revenue_daily.total_food      + EXCLUDED.total_food,
			    total_ship      = revenue_daily.total_ship      + EXCLUDED.total_ship,
			    orders_count    = revenue_daily.orders_count    + EXCLUDED.orders_count,
			    cancelled_count = revenue_daily.cancelled_count + EXCLUDED.cancelled_count`,
			storeID, date, foodDelta, shipDelta, ordersDelta, cancelledDelta,
		).Error
}

func (r *projectionGormRepository) IncrementUserGrowth(
	ctx context.Context, tx *gorm.DB,
	date time.Time, customersDelta, vendorsDelta int,
) error {
	return tx.WithContext(ctx).
		Exec(`INSERT INTO user_growth_daily (date, new_customers, new_vendors)
			  VALUES (?,?,?)
			  ON CONFLICT (date) DO UPDATE SET
			    new_customers = user_growth_daily.new_customers + EXCLUDED.new_customers,
			    new_vendors   = user_growth_daily.new_vendors   + EXCLUDED.new_vendors`,
			date, customersDelta, vendorsDelta,
		).Error
}

func (r *projectionGormRepository) UpsertStorePerformance(
	ctx context.Context, tx *gorm.DB,
	storeID string, ratingDelta int64,
) error {
	return tx.WithContext(ctx).
		Exec(`INSERT INTO store_performance (store_id, rating_sum, rating_count, avg_rating)
			  VALUES (?, ?, 1, ?)
			  ON CONFLICT (store_id) DO UPDATE SET
			    rating_sum   = store_performance.rating_sum   + EXCLUDED.rating_sum,
			    rating_count = store_performance.rating_count + 1,
			    avg_rating   = ROUND(
			      (store_performance.rating_sum + EXCLUDED.rating_sum)::NUMERIC /
			      (store_performance.rating_count + 1), 2
			    )`,
			storeID, ratingDelta, float64(ratingDelta),
		).Error
}

var _ repository.ProjectionRepository = (*projectionGormRepository)(nil)
