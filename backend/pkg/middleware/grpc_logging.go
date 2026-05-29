package middleware

import (
	"context"
	"log/slog"
	"time"

	"project/pkg/trace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryLogging() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		id := trace.ExtractGRPCMetadata(ctx)
		if id == "" {
			id = trace.New()
		}
		ctx = trace.WithTraceID(ctx, id)

		start := time.Now()

		resp, err := handler(ctx, req)

		latencyMs := time.Since(start).Milliseconds()
		st, _ := status.FromError(err)
		code := st.Code()

		attrs := []slog.Attr{
			slog.String("grpc_method", info.FullMethod),
			slog.String("grpc_code", code.String()),
			slog.Int64("latency_ms", latencyMs),
		}

		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
		}

		level := grpcCodeToLevel(code)
		slog.LogAttrs(ctx, level, "grpc request", attrs...)

		return resp, err
	}
}

func grpcCodeToLevel(code codes.Code) slog.Level {
	switch code {
	case codes.OK:
		return slog.LevelInfo
	case codes.NotFound, codes.InvalidArgument, codes.AlreadyExists:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}
