package main

import (
	"context"
	"log/slog"

	"project/pkg/app"
	"project/pkg/messaging/kafka"
	"project/pkg/outbox"
	"project/services/payment/internal/infrastructure/persistence"
)

// startOutboxDispatcher spins up the payment outbox polling loop in a background
// goroutine and registers cancellation + producer close as OnShutdown hooks.
func startOutboxDispatcher(a *app.App, deps app.Dependencies) {
	brokers := deps.Config.Kafka.Brokers
	if len(brokers) == 0 {
		slog.Warn("kafka brokers not configured — payment outbox dispatcher disabled")
		return
	}

	producer := kafka.NewProducer(brokers)
	publisher := outbox.NewKafkaPublisher(producer)

	repo := persistence.NewOutboxGormRepository(deps.DB)
	dispatcher := outbox.NewDispatcher(
		persistence.NewOutboxDispatchAdapter(repo),
		publisher,
		outbox.DispatcherConfig{
			TickEvery:   deps.Config.Outbox.TickEvery,
			BatchSize:   deps.Config.Outbox.BatchSize,
			MaxAttempts: deps.Config.Outbox.MaxAttempts,
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	go dispatcher.Start(ctx)

	a.OnShutdown(func() {
		cancel()
		if err := producer.Close(); err != nil {
			slog.Error("payment: kafka producer close error", "err", err)
		}
	})
}
