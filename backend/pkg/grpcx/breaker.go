package grpcx

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Circuit breaker per target address. When a downstream keeps failing, the
// breaker opens and callers get an immediate Unavailable instead of stacking
// timed-out requests on a dead service. Thresholds: within a 30s window, ≥5
// requests with ≥60% failures opens the circuit for 15s; half-open allows 3
// probes before fully closing again.
var (
	breakersMu sync.Mutex
	breakers   = map[string]*gobreaker.CircuitBreaker{}
)

func breakerFor(addr string) *gobreaker.CircuitBreaker {
	breakersMu.Lock()
	defer breakersMu.Unlock()
	if cb, ok := breakers[addr]; ok {
		return cb
	}
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        addr,
		Interval:    30 * time.Second,
		Timeout:     15 * time.Second,
		MaxRequests: 3,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			slog.Warn("grpc circuit breaker state change", "target", name, "from", from.String(), "to", to.String())
		},
		// Only infrastructure-level failures should trip the breaker.
		// Application errors (NotFound, InvalidArgument…) mean the service is
		// alive and answering.
		IsSuccessful: func(err error) bool {
			switch status.Code(err) {
			case codes.Unavailable, codes.DeadlineExceeded:
				return false
			}
			return true
		},
	})
	breakers[addr] = cb
	return cb
}

// BreakerUnaryInterceptor wraps unary calls in the target's circuit breaker.
// While open, calls fail immediately with codes.Unavailable.
func BreakerUnaryInterceptor(addr string) grpc.UnaryClientInterceptor {
	cb := breakerFor(addr)
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		_, err := cb.Execute(func() (any, error) {
			return nil, invoker(ctx, method, req, reply, cc, opts...)
		})
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return status.Error(codes.Unavailable, "circuit breaker open for "+addr)
		}
		return err
	}
}
