package trace

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ExtractGRPCMetadata reads x-trace-id from incoming gRPC metadata.
func ExtractGRPCMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get(HeaderHTTPLower)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

// InjectGRPCMetadata returns an outgoing-context with x-trace-id set to the
// trace id from ctx. Replaces (not appends) any existing entry so retries
// don't duplicate the header.
func InjectGRPCMetadata(ctx context.Context) context.Context {
	id := FromContext(ctx)
	if id == "" {
		return ctx
	}
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	} else {
		md = md.Copy()
	}
	md.Set(HeaderHTTPLower, id)
	return metadata.NewOutgoingContext(ctx, md)
}

// UnaryClientInterceptor injects trace_id into outgoing metadata for every
// unary call. Wire with:
//
//	grpc.Dial(addr, grpc.WithUnaryInterceptor(trace.UnaryClientInterceptor()))
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return invoker(InjectGRPCMetadata(ctx), method, req, reply, cc, opts...)
	}
}
