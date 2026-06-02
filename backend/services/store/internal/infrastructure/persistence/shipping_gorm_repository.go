package persistence

import (
	"context"

	"project/services/store/internal/entity"
	"project/services/store/internal/repository"

	"gorm.io/gorm"
)

type shippingGormRepository struct {
	db *gorm.DB
}

// NewShippingGormRepository returns a repository.ShippingRepository backed by GORM.
func NewShippingGormRepository(db *gorm.DB) repository.ShippingRepository {
	return &shippingGormRepository{db: db}
}

// ---- ShipFeeRules ----

func (r *shippingGormRepository) CreateShipFeeRule(ctx context.Context, sr *entity.ShipFeeRule) error {
	return r.db.WithContext(ctx).Create(sr).Error
}

func (r *shippingGormRepository) GetShipFeeRule(ctx context.Context, id string) (*entity.ShipFeeRule, error) {
	var sr entity.ShipFeeRule
	if err := r.db.WithContext(ctx).First(&sr, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &sr, nil
}

func (r *shippingGormRepository) ListShipFeeRules(ctx context.Context, storeID string) ([]*entity.ShipFeeRule, error) {
	var rows []*entity.ShipFeeRule
	return rows, r.db.WithContext(ctx).
		Where("store_id = ?", storeID).
		Order("scope ASC, unit_fee ASC").
		Find(&rows).Error
}

func (r *shippingGormRepository) UpdateShipFeeRule(ctx context.Context, sr *entity.ShipFeeRule) error {
	return r.db.WithContext(ctx).Model(sr).Updates(map[string]any{
		"scope":    sr.Scope,
		"ref_id":   sr.RefID,
		"unit_fee": sr.UnitFee,
	}).Error
}

func (r *shippingGormRepository) DeleteShipFeeRule(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.ShipFeeRule{}, "id = ?", id).Error
}

// ResolveShipFee finds the most specific fee rule for the delivery location.
// It walks the ordered (scope, id) candidate pairs and returns the first match,
// so the caller controls priority by ordering most-specific first.
// Returns nil when no rule is configured (store does not serve that location).
func (r *shippingGormRepository) ResolveShipFee(ctx context.Context, storeID string, scopes []string, ids []string) (*entity.ShipFeeRule, error) {
	for i := range scopes {
		if ids[i] == "" {
			continue
		}
		var rule entity.ShipFeeRule
		err := r.db.WithContext(ctx).
			Where("store_id = ? AND scope = ? AND ref_id = ?", storeID, scopes[i], ids[i]).
			First(&rule).Error
		if err == nil {
			return &rule, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	// No rule found — store does not serve this location.
	return nil, nil
}

// ---- OperatingHours ----

func (r *shippingGormRepository) CreateOperatingHours(ctx context.Context, oh *entity.OperatingHours) error {
	return r.db.WithContext(ctx).Create(oh).Error
}

func (r *shippingGormRepository) GetOperatingHours(ctx context.Context, id string) (*entity.OperatingHours, error) {
	var oh entity.OperatingHours
	if err := r.db.WithContext(ctx).First(&oh, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &oh, nil
}

func (r *shippingGormRepository) ListOperatingHours(ctx context.Context, storeID string) ([]*entity.OperatingHours, error) {
	var rows []*entity.OperatingHours
	return rows, r.db.WithContext(ctx).
		Where("store_id = ?", storeID).
		Order("weekday ASC").
		Find(&rows).Error
}

func (r *shippingGormRepository) UpdateOperatingHours(ctx context.Context, oh *entity.OperatingHours) error {
	return r.db.WithContext(ctx).Model(oh).Updates(map[string]any{
		"weekday":    oh.Weekday,
		"open_time":  oh.OpenTime,
		"close_time": oh.CloseTime,
	}).Error
}

func (r *shippingGormRepository) DeleteOperatingHours(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.OperatingHours{}, "id = ?", id).Error
}

// ---- ShipCutoffs ----

func (r *shippingGormRepository) CreateShipCutoff(ctx context.Context, sc *entity.ShipCutoff) error {
	return r.db.WithContext(ctx).Create(sc).Error
}

func (r *shippingGormRepository) GetShipCutoff(ctx context.Context, id string) (*entity.ShipCutoff, error) {
	var sc entity.ShipCutoff
	if err := r.db.WithContext(ctx).First(&sc, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &sc, nil
}

func (r *shippingGormRepository) ListShipCutoffs(ctx context.Context, storeID string) ([]*entity.ShipCutoff, error) {
	var rows []*entity.ShipCutoff
	return rows, r.db.WithContext(ctx).
		Where("store_id = ?", storeID).
		Order("cutoff_time ASC").
		Find(&rows).Error
}

func (r *shippingGormRepository) ListShipCutoffsForStores(ctx context.Context, storeIDs []string) (map[string][]*entity.ShipCutoff, error) {
	out := make(map[string][]*entity.ShipCutoff, len(storeIDs))
	if len(storeIDs) == 0 {
		return out, nil
	}
	var rows []*entity.ShipCutoff
	if err := r.db.WithContext(ctx).
		Where("store_id IN ?", storeIDs).
		Order("cutoff_time ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, sc := range rows {
		out[sc.StoreID] = append(out[sc.StoreID], sc)
	}
	return out, nil
}

func (r *shippingGormRepository) UpdateShipCutoff(ctx context.Context, sc *entity.ShipCutoff) error {
	return r.db.WithContext(ctx).Model(sc).Updates(map[string]any{
		"cutoff_time":  sc.CutoffTime,
		"lead_minutes": sc.LeadMinutes,
	}).Error
}

func (r *shippingGormRepository) DeleteShipCutoff(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.ShipCutoff{}, "id = ?", id).Error
}

// ---- Bulk replace ----

func (r *shippingGormRepository) DeleteOperatingHoursByStore(ctx context.Context, tx *gorm.DB, storeID string) error {
	// Hard-delete so the subsequent BulkCreate becomes the sole current set.
	return tx.WithContext(ctx).Unscoped().
		Where("store_id = ?", storeID).
		Delete(&entity.OperatingHours{}).Error
}

func (r *shippingGormRepository) BulkCreateOperatingHours(ctx context.Context, tx *gorm.DB, rows []*entity.OperatingHours) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Create(&rows).Error
}

func (r *shippingGormRepository) DeleteShipCutoffsByStore(ctx context.Context, tx *gorm.DB, storeID string) error {
	// Hard-delete so the subsequent BulkCreate becomes the sole current set.
	return tx.WithContext(ctx).Unscoped().
		Where("store_id = ?", storeID).
		Delete(&entity.ShipCutoff{}).Error
}

func (r *shippingGormRepository) BulkCreateShipCutoffs(ctx context.Context, tx *gorm.DB, rows []*entity.ShipCutoff) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Create(&rows).Error
}

// ---- OperatingHoursChangeRequests ----

func (r *shippingGormRepository) CreateChangeRequest(ctx context.Context, req *entity.OperatingHoursChangeRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *shippingGormRepository) GetChangeRequest(ctx context.Context, id string) (*entity.OperatingHoursChangeRequest, error) {
	var req entity.OperatingHoursChangeRequest
	if err := r.db.WithContext(ctx).First(&req, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *shippingGormRepository) ListChangeRequests(ctx context.Context, storeID, status string) ([]*entity.OperatingHoursChangeRequest, error) {
	q := r.db.WithContext(ctx).Where("store_id = ?", storeID).Order("created_at DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []*entity.OperatingHoursChangeRequest
	return rows, q.Find(&rows).Error
}

func (r *shippingGormRepository) UpdateChangeRequestStatus(ctx context.Context, id, status, reviewedBy string) error {
	return r.db.WithContext(ctx).
		Model(&entity.OperatingHoursChangeRequest{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      status,
			"reviewed_by": reviewedBy,
		}).Error
}
