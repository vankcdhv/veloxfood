// Package otel wires OpenTelemetry distributed tracing for every service.
// Spans export over OTLP/gRPC to a collector (Jaeger all-in-one in the
// monitoring compose profile). Tracing is opt-in per service: an empty
// endpoint disables it entirely with zero overhead.
//
// The pre-existing x-trace-id (a plain string propagated through HTTP, gRPC
// and Kafka for log correlation) stays untouched — it is attached to every
// server span as the `app.trace_id` attribute, so a log line can be
// cross-referenced to its Jaeger trace and vice versa.
package otel

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Init installs a global TracerProvider exporting to the OTLP/gRPC endpoint.
// It returns a shutdown func that flushes buffered spans. An empty endpoint
// returns a no-op shutdown and leaves the default (no-op) tracer in place.
func Init(ctx context.Context, serviceName, endpoint string) (func(context.Context) error, error) {
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("otel: create otlp exporter: %w", err)
	}

	// NewSchemaless, not NewWithAttributes(semconv.SchemaURL, …): Merge rejects two
	// resources carrying different schema URLs, and the SDK's Default() advertises a
	// newer schema than the semconv package pinned above. A schemaless resource
	// merges cleanly with any of them.
	res, err := sdkresource.Merge(
		sdkresource.Default(),
		sdkresource.NewSchemaless(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("otel: build resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(3*time.Second)),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	// W3C traceparent + baggage across HTTP/gRPC hops.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	// The collector is optional (monitoring profile) — a missing Jaeger must
	// not spam error logs, so demote export failures to debug.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		slog.Debug("otel export error", "err", err)
	}))

	slog.Info("otel tracing enabled", "endpoint", endpoint, "service", serviceName)
	return tp.Shutdown, nil
}
