package persistence

import (
	"context"

	"project/services/promotion/internal/entity"
	"project/services/promotion/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type processedEventGormRepository struct {
	db *gorm.DB
}

func NewProcessedEventGormRepository(db *gorm.DB) repository.ProcessedEventRepository {
	return &processedEventGormRepository{db: db}
}

// MarkProcessed inserts event_id within tx using INSERT … ON CONFLICT DO NOTHING.
// Returns inserted=true when the row is new, false (no error) when duplicate.
func (r *processedEventGormRepository) MarkProcessed(ctx context.Context, tx *gorm.DB, eventID string) (bool, error) {
	ev := entity.ProcessedEvent{EventID: eventID}
	result := tx.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&ev)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}
