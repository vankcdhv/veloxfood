package persistence

import (
	"context"

	"project/services/reporting/internal/entity"
	"project/services/reporting/internal/repository"

	"gorm.io/gorm"
)

type processedEventGormRepository struct {
	db *gorm.DB
}

// NewProcessedEventGormRepository returns a ProcessedEventRepository backed by GORM.
func NewProcessedEventGormRepository(db *gorm.DB) repository.ProcessedEventRepository {
	return &processedEventGormRepository{db: db}
}

// MarkProcessed inserts event_id. Returns (true, nil) on first insert,
// (false, nil) when already present — caller should skip processing.
func (r *processedEventGormRepository) MarkProcessed(ctx context.Context, tx *gorm.DB, eventID string) (bool, error) {
	result := tx.WithContext(ctx).
		Exec("INSERT INTO processed_events (event_id) VALUES (?) ON CONFLICT (event_id) DO NOTHING", eventID)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

var _ repository.ProcessedEventRepository = (*processedEventGormRepository)(nil)

// ensure entity is referenced (suppresses unused import linter warning).
var _ = entity.ProcessedEvent{}
