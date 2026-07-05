// Package grpcx standardises internal service-to-service gRPC clients:
// plaintext transport, trace-id propagation, a default per-call deadline,
// bounded retry on transient failures, and a per-target circuit breaker —
// so a hung or dying downstream can neither stall nor melt its callers.
package grpcx

import (
	"context"
	"time"

	"project/pkg/trace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// DefaultTimeout bounds every internal unary call. All internal RPCs are
// small lookups/commands; anything slower indicates a stuck dependency and
// should fail fast so sagas can compensate.
const DefaultTimeout = 5 * time.Second

// Retry policy: only codes.Unavailable is retried — it signals a transient
// transport/connection failure where the request most likely never reached
// the server. DeadlineExceeded is NOT retried (it would multiply the caller's
// wait) and application errors must never be. All saga RPCs are idempotent by
// order_id, so a duplicate delivery after an ambiguous failure is safe.
const (
	retryAttempts    = 2
	retryBaseBackoff = 100 * time.Millisecond
)

// RetryUnaryInterceptor retries Unavailable failures with linear backoff
// (100ms, 200ms), respecting context cancellation between attempts.
func RetryUnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var err error
		for attempt := 0; ; attempt++ {
			err = invoker(ctx, method, req, reply, cc, opts...)
			if err == nil || attempt >= retryAttempts || status.Code(err) != codes.Unavailable {
				return err
			}
			select {
			case <-ctx.Done():
				return err
			case <-time.After(retryBaseBackoff * time.Duration(attempt+1)):
			}
		}
	}
}

// TimeoutUnaryInterceptor applies a default deadline to outgoing unary calls
// that don't already carry one. Callers that need a tighter or looser bound
// keep full control by setting their own deadline on ctx.
func TimeoutUnaryInterceptor(d time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, d)
			defer cancel()
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// Dial opens a connection to an internal service with the standard client
// stack, outermost first: trace propagation → circuit breaker (fail fast
// while the target is down) → timeout (bounds the whole retry sequence) →
// retry on Unavailable.
func Dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			trace.UnaryClientInterceptor(),
			BreakerUnaryInterceptor(addr),
			TimeoutUnaryInterceptor(DefaultTimeout),
			RetryUnaryInterceptor(),
		),
	)
}
