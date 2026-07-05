package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"project/pkg/cache"
	rediscache "project/pkg/cache/redis"
	"project/pkg/config"
	"project/pkg/database"
	"project/pkg/logger"
	"project/pkg/metrics"
	"project/pkg/middleware"
	otelinit "project/pkg/otel"
	"project/pkg/trace"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

type Dependencies struct {
	Config *config.Config
	DB     *gorm.DB
	Cache  cache.Cache
}

type HTTPRegistrar func(r *gin.Engine, deps Dependencies)
type GRPCRegistrar func(s *grpc.Server, deps Dependencies)

type App struct {
	name          string
	configPath    string
	deps          Dependencies
	httpRegistrar HTTPRegistrar
	grpcRegistrar GRPCRegistrar
	grpcServer    *grpc.Server
	httpServer    *http.Server
	redisCache    *rediscache.RedisCache
	onShutdown    []func()
}

func New(name string) *App {
	return &App{
		name:       name,
		configPath: "config/config.yaml",
	}
}

func (a *App) WithConfigPath(path string) *App {
	a.configPath = path
	return a
}

func (a *App) RegisterHTTP(fn HTTPRegistrar) {
	a.httpRegistrar = fn
}

func (a *App) RegisterGRPC(fn GRPCRegistrar) {
	a.grpcRegistrar = fn
}

func (a *App) OnShutdown(fn func()) {
	a.onShutdown = append(a.onShutdown, fn)
}

func (a *App) Deps() Dependencies {
	return a.deps
}

func (a *App) Run() {
	cfg, err := config.Load(a.configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger.Setup(cfg.App.Env)
	slog.SetDefault(logger.NewWithService(cfg.App.Env, a.name))
	slog.Info("starting service", "env", cfg.App.Env)

	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	if cfg.Database.AutoMigrate {
		if err := database.MigrateUp(cfg.Database, cfg.Database.MigrationsPath); err != nil {
			slog.Error("auto-migrate failed", "error", err)
			os.Exit(1)
		}
		slog.Info("database schema up to date", "path", cfg.Database.MigrationsPath)
	}

	var c cache.Cache
	rc, err := rediscache.New(cfg.Redis)
	if err != nil {
		slog.Warn("redis unavailable, running without cache", "error", err)
	} else {
		c = rc
		a.redisCache = rc
	}

	// Distributed tracing (optional): spans export to the OTLP collector from
	// the monitoring profile; empty endpoint = disabled.
	if shutdown, err := otelinit.Init(context.Background(), a.name, cfg.OTel.Endpoint); err != nil {
		slog.Warn("otel init failed — tracing disabled", "error", err)
	} else {
		a.onShutdown = append(a.onShutdown, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = shutdown(ctx)
		})
	}

	a.deps = Dependencies{Config: cfg, DB: db, Cache: c}

	if a.grpcRegistrar != nil {
		a.startGRPC(cfg.App.GRPCPort)
	}

	if a.httpRegistrar != nil {
		a.startHTTP(cfg.App.Port)
	}

	a.waitForShutdown()
}

func (a *App) startHTTP(port int) {
	router := gin.New()
	router.Use(middleware.PanicRecovery())
	// otelgin opens the server span; RequestLogging then installs the legacy
	// x-trace-id; the bridge stamps that id onto the span so a log line can be
	// cross-referenced with its Jaeger trace.
	router.Use(otelgin.Middleware(a.name))
	router.Use(middleware.RequestLogging())
	router.Use(spanTraceIDBridge())
	router.Use(metrics.HTTPMiddleware(a.name))
	router.GET("/metrics", metrics.Handler())
	// /health is liveness only: the process is up. Orchestrators must use
	// /readyz before routing traffic — it verifies the dependencies.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": a.name})
	})
	router.GET("/readyz", a.readyz)

	a.httpRegistrar(router, a.deps)

	a.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		slog.Info("HTTP server starting", "port", port)
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server error", "error", err)
		}
	}()
}

// readyz reports whether the service can actually serve traffic: DB reachable
// and (when configured) Redis reachable. Kafka is intentionally not probed —
// consumers retry on their own and a broker blip should not pull an otherwise
// healthy API out of rotation.
func (a *App) readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := gin.H{}
	ready := true

	if sqlDB, err := a.deps.DB.DB(); err != nil || sqlDB.PingContext(ctx) != nil {
		checks["db"] = "down"
		ready = false
	} else {
		checks["db"] = "ok"
	}

	switch {
	case a.redisCache == nil:
		checks["redis"] = "disabled"
	case a.redisCache.Ping(ctx) != nil:
		// Redis is a cache the services already degrade without — report it
		// but do not fail readiness over it.
		checks["redis"] = "down"
	default:
		checks["redis"] = "ok"
	}

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"service": a.name, "ready": ready, "checks": checks})
}

// spanTraceIDBridge attaches the legacy x-trace-id to the active server span.
func spanTraceIDBridge() gin.HandlerFunc {
	return func(c *gin.Context) {
		if id := trace.FromContext(c.Request.Context()); id != "" {
			if span := oteltrace.SpanFromContext(c.Request.Context()); span.SpanContext().IsValid() {
				span.SetAttributes(attribute.String("app.trace_id", id))
			}
		}
		c.Next()
	}
}

func (a *App) startGRPC(port int) {
	a.grpcServer = grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(middleware.UnaryLogging()),
	)
	a.grpcRegistrar(a.grpcServer, a.deps)

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			slog.Error("failed to listen grpc", "error", err)
			return
		}
		slog.Info("gRPC server starting", "port", port)
		if err := a.grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server error", "error", err)
		}
	}()
}

func (a *App) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...", "service", a.name)

	// Stop accepting new HTTP requests and drain in-flight ones first, so a
	// rolling deploy never cuts a request mid-response.
	if a.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		if err := a.httpServer.Shutdown(ctx); err != nil {
			slog.Warn("http shutdown incomplete", "error", err)
		}
		cancel()
	}

	if a.grpcServer != nil {
		a.grpcServer.GracefulStop()
	}

	for _, fn := range a.onShutdown {
		fn()
	}

	if a.redisCache != nil {
		a.redisCache.Close()
	}

	sqlDB, _ := a.deps.DB.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	slog.Info("server stopped", "service", a.name)
}
