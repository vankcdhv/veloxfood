// Package grpcx standardises internal service-to-service gRPC clients:
// plaintext transport, trace-id propagation, and a default per-call deadline
// so a hung downstream can never stall an upstream request indefinitely.
package grpcx

import (
	"context"
	"time"

	"project/pkg/trace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DefaultTimeout bounds every internal unary call. All internal RPCs are
// small lookups/commands; anything slower indicates a stuck dependency and
// should fail fast so sagas can compensate.
const DefaultTimeout = 5 * time.Second

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
// stack (insecure transport, trace propagation, default timeout).
func Dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			trace.UnaryClientInterceptor(),
			TimeoutUnaryInterceptor(DefaultTimeout),
		),
	)
}
