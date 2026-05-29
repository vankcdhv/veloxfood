package outbox

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"project/pkg/trace"
)

// DispatchRepository is the narrow data-access interface the Dispatcher depends on.
// The concrete implementation lives in services/user/internal/infrastructure/persistence.
// Defining it here prevents pkg/outbox from importing any service package.
type DispatchRepository interface {
	Pickup(ctx context.Context, limit, maxAttempts int) ([]*OutboxRow, error)
	MarkPublished(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id, errMsg string, attempts, maxAttempts int) error
}

// DispatcherConfig controls the dispatcher polling loop.
type DispatcherConfig struct {
	TickEvery   time.Duration // interval between polls (default 2s)
	BatchSize   int           // max events per tick (default 50)
	MaxAttempts int           // attempts before permanent failure (default 5)
}

// Dispatcher polls outbox_events for pending rows and publishes them to Kafka.
// Start it as a goroutine after DB is ready; cancel the context to stop it.
type Dispatcher struct {
	repo      DispatchRepository
	publisher Publisher
	cfg       DispatcherConfig
}

// NewDispatcher constructs a Dispatcher with the given dependencies.
// cfg zero-values fall back to sensible defaults.
func NewDispatcher(repo DispatchRepository, publisher Publisher, cfg DispatcherConfig) *Dispatcher {
	if cfg.TickEvery == 0 {
		cfg.TickEvery = 2 * time.Second
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 50
	}
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = 5
	}
	return &Dispatcher{repo: repo, publisher: publisher, cfg: cfg}
}

// Start runs the polling loop until ctx is cancelled (e.g. on SIGTERM).
// Designed to be called as a goroutine: go dispatcher.Start(ctx).
func (d *Dispatcher) Start(ctx context.Context) {
	slog.InfoContext(ctx, "outbox dispatcher started",
		"tick_every", d.cfg.TickEvery,
		"batch_size", d.cfg.BatchSize,
		"max_attempts", d.cfg.MaxAttempts,
	)
	ticker := time.NewTicker(d.cfg.TickEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "outbox dispatcher stopping")
			return
		case <-ticker.C:
			d.tick(ctx)
		}
	}
}

// tick performs one poll-and-publish cycle.
func (d *Dispatcher) tick(ctx context.Context) {
	events, err := d.repo.Pickup(ctx, d.cfg.BatchSize, d.cfg.MaxAttempts)
	if err != nil {
		slog.ErrorContext(ctx, "outbox pickup failed", "err", err)
		return
	}
	if len(events) == 0 {
		return
	}
	slog.DebugContext(ctx, "outbox tick", "event_count", len(events))

	for _, evt := range events {
		d.publish(ctx, evt)
	}
}

// publish sends a single outbox row to Kafka and marks it published or failed.
func (d *Dispatcher) publish(ctx context.Context, row *OutboxRow) {
	traceID := row.TraceID
	pubCtx := trace.WithTraceID(ctx, traceID)

	envelope, err := NewEnvelope(row)
	if err != nil {
		slog.ErrorContext(pubCtx, "outbox envelope marshal failed",
			"event_id", row.ID, "event_type", row.EventType, "err", err)
		// Payload is corrupt — mark failed immediately (no point retrying).
		newAttempts := row.Attempts + 1
		if merr := d.repo.MarkFailed(pubCtx, row.ID, err.Error(), newAttempts, d.cfg.MaxAttempts); merr != nil {
			slog.ErrorContext(pubCtx, "outbox mark_failed error", "event_id", row.ID, "err", merr)
		}
		return
	}

	topic := TopicFor(row.AggregateType)
	if err := d.publisher.Publish(pubCtx, topic, row.AggregateID, envelope, traceID); err != nil {
		newAttempts := row.Attempts + 1
		slog.WarnContext(pubCtx, "outbox publish failed",
			"event_id", row.ID,
			"event_type", row.EventType,
			"topic", topic,
			"attempts", newAttempts,
			"err", err,
		)
		if merr := d.repo.MarkFailed(pubCtx, row.ID, err.Error(), newAttempts, d.cfg.MaxAttempts); merr != nil {
			slog.ErrorContext(pubCtx, "outbox mark_failed error", "event_id", row.ID, "err", merr)
		}
		return
	}

	if merr := d.repo.MarkPublished(pubCtx, row.ID); merr != nil {
		// Published to Kafka but DB update failed — log and continue. Kafka consumer
		// must dedup by event_id because the row will be picked up again on next tick.
		slog.ErrorContext(pubCtx, "outbox mark_published error — possible duplicate delivery",
			"event_id", row.ID, "err", merr)
		return
	}

	slog.InfoContext(pubCtx, "outbox event published",
		"event_id", row.ID,
		"event_type", row.EventType,
		"topic", topic,
	)
}

// ensure json import used (for NewEnvelope called via outbox.go)
var _ = json.Marshal
