package persistence

import (
	"context"

	"project/services/store/internal/entity"
	"project/services/store/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type favoriteGormRepository struct {
	db *gorm.DB
}

func NewFavoriteGormRepository(db *gorm.DB) repository.FavoriteRepository {
	return &favoriteGormRepository{db: db}
}

func (r *favoriteGormRepository) Add(ctx context.Context, f *entity.Favorite) error {
	// ON CONFLICT DO NOTHING — re-favoriting is idempotent.
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(f).Error
}

func (r *favoriteGormRepository) Remove(ctx context.Context, userID string, targetType entity.FavoriteTargetType, targetID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Delete(&entity.Favorite{}).Error
}

func (r *favoriteGormRepository) IDs(ctx context.Context, userID string, targetType entity.FavoriteTargetType) ([]string, error) {
	ids := []string{}
	err := r.db.WithContext(ctx).Model(&entity.Favorite{}).
		Where("user_id = ? AND target_type = ?", userID, targetType).
		Order("created_at DESC").
		Pluck("target_id", &ids).Error
	return ids, err
}

func (r *favoriteGormRepository) FavoriteStores(ctx context.Context, userID string) ([]*entity.Store, error) {
	var rows []*entity.Store
	err := r.db.WithContext(ctx).Raw(`
		SELECT stores.*
		FROM favorites
		JOIN stores ON stores.id = favorites.target_id
		WHERE favorites.user_id = ?
		  AND favorites.target_type = 'STORE'
		  AND stores.deleted_at IS NULL
		ORDER BY favorites.created_at DESC
	`, userID).Scan(&rows).Error
	return rows, err
}

func (r *favoriteGormRepository) FavoriteItems(ctx context.Context, userID string) ([]repository.SearchMenuItemRow, error) {
	var rows []repository.SearchMenuItemRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			menu_items.id          AS id,
			menu_items.name        AS name,
			menu_items.price       AS price,
			menu_items.image_url   AS image_url,
			menu_items.description AS description,
			stores.id              AS store_id,
			stores.name            AS store_name,
			stores.sale_status     AS sale_status
		FROM favorites
		JOIN menu_items ON menu_items.id = favorites.target_id
		JOIN stores     ON stores.id = menu_items.store_id
		WHERE favorites.user_id = ?
		  AND favorites.target_type = 'ITEM'
		  AND menu_items.deleted_at IS NULL
		  AND stores.deleted_at IS NULL
		ORDER BY favorites.created_at DESC
	`, userID).Scan(&rows).Error
	return rows, err
}
