package logger

import (
	"log/slog"
	"os"
)

func Setup(env string) {
	slog.SetDefault(slog.New(newHandler(env)))
}

func NewWithService(env, serviceName string) *slog.Logger {
	return slog.New(newHandler(env)).With(slog.String("service", serviceName))
}

func newHandler(env string) slog.Handler {
	var inner slog.Handler
	switch env {
	case "production":
		inner = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
		})
	default:
		inner = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	return newTraceHandler(inner)
}
