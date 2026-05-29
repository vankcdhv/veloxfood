package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"project/pkg/cache"
	rediscache "project/pkg/cache/redis"
	"project/pkg/config"
	"project/pkg/database"
	"project/pkg/logger"
	"project/pkg/middleware"

	"github.com/gin-gonic/gin"
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

	var c cache.Cache
	rc, err := rediscache.New(cfg.Redis)
	if err != nil {
		slog.Warn("redis unavailable, running without cache", "error", err)
	} else {
		c = rc
		a.redisCache = rc
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
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogging())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": a.name})
	})

	a.httpRegistrar(router, a.deps)

	go func() {
		addr := fmt.Sprintf(":%d", port)
		slog.Info("HTTP server starting", "port", port)
		if err := router.Run(addr); err != nil {
			slog.Error("http server error", "error", err)
		}
	}()
}

func (a *App) startGRPC(port int) {
	a.grpcServer = grpc.NewServer(
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
	_ = context.Background()
}
