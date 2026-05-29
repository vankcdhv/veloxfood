package persistence

import (
	"context"

	"project/pkg/outbox"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"
)

// outboxDispatchAdapter bridges repository.OutboxRepository to the
// outbox.DispatchRepository interface expected by pkg/outbox.Dispatcher.
// This prevents pkg/outbox from importing the entity or repository packages.
type outboxDispatchAdapter struct {
	repo repository.OutboxRepository
}

// NewOutboxDispatchAdapter wraps an OutboxRepository for use with the outbox Dispatcher.
func NewOutboxDispatchAdapter(repo repository.OutboxRepository) outbox.DispatchRepository {
	return &outboxDispatchAdapter{repo: repo}
}

func (a *outboxDispatchAdapter) Pickup(ctx context.Context, limit, maxAttempts int) ([]*outbox.OutboxRow, error) {
	events, err := a.repo.Pickup(ctx, limit, maxAttempts)
	if err != nil {
		return nil, err
	}
	rows := make([]*outbox.OutboxRow, 0, len(events))
	for _, e := range events {
		rows = append(rows, entityToRow(e))
	}
	return rows, nil
}

func (a *outboxDispatchAdapter) MarkPublished(ctx context.Context, id string) error {
	return a.repo.MarkPublished(ctx, id)
}

func (a *outboxDispatchAdapter) MarkFailed(ctx context.Context, id, errMsg string, attempts, maxAttempts int) error {
	return a.repo.MarkFailed(ctx, id, errMsg, attempts, maxAttempts)
}

func entityToRow(e *entity.OutboxEvent) *outbox.OutboxRow {
	traceID := ""
	if e.TraceID != nil {
		traceID = *e.TraceID
	}
	return &outbox.OutboxRow{
		ID:            e.ID,
		AggregateType: e.AggregateType,
		AggregateID:   e.AggregateID,
		EventType:     e.EventType,
		Payload:       e.Payload,
		TraceID:       traceID,
		Attempts:      e.Attempts,
	}
}
