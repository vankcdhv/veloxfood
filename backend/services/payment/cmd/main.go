package main

import (
	"log/slog"

	"project/pkg/app"
	"project/pkg/middleware"
	paymentv1 "project/proto/payment/v1"
	grpchandler "project/services/payment/internal/handler/grpc"
	payevent "project/services/payment/internal/handler/event"
	handlerhttp "project/services/payment/internal/handler/http"
	v1 "project/services/payment/internal/handler/http/v1"
	"project/services/payment/internal/infrastructure/momo"
	"project/services/payment/internal/infrastructure/persistence"
	"project/services/payment/internal/usecase"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {
	a := app.New("payment-service").WithConfigPath("config/payment.yaml")

	a.RegisterHTTP(func(r *gin.Engine, deps app.Dependencies) {
		r.Use(middleware.RequestMetadata())

		// ── Repositories ──────────────────────────────────────────────────────
		walletRepo := persistence.NewWalletGormRepository(deps.DB)
		ledgerRepo := persistence.NewLedgerGormRepository(deps.DB)
		paymentRepo := persistence.NewPaymentGormRepository(deps.DB)
		payoutRepo := persistence.NewPayoutGormRepository(deps.DB)
		outboxRepo := persistence.NewOutboxGormRepository(deps.DB)
		processedRepo := persistence.NewProcessedEventGormRepository(deps.DB)

		// ── MoMo client ───────────────────────────────────────────────────────
		momoClient := momo.NewClient(deps.Config.MoMo)
		ipnURL := deps.Config.MoMo.IpnURL
		redirectURL := deps.Config.MoMo.RedirectURL

		// ── Usecases ──────────────────────────────────────────────────────────
		walletUC := usecase.NewWalletUsecase(walletRepo, ledgerRepo)
		refundUC := usecase.NewRefundUsecase(deps.DB, walletRepo, ledgerRepo, paymentRepo, outboxRepo)
		topupUC := usecase.NewTopupUsecase(deps.DB, paymentRepo, momoClient, ipnURL, redirectURL)
		ipnUC := usecase.NewIPNUsecase(deps.DB, walletRepo, ledgerRepo, paymentRepo, outboxRepo, momoClient)
		payoutUC := usecase.NewPayoutUsecase(deps.DB, walletRepo, ledgerRepo, payoutRepo, outboxRepo)

		// ── HTTP router ───────────────────────────────────────────────────────
		cfg := handlerhttp.RouterConfig{
			MoMoHandler: v1.NewMoMoHandler(ipnUC),
		}
		if authMW, _, err := buildAuth(deps); err != nil {
			slog.Error("payment: auth setup failed — protected routes disabled", "err", err)
		} else {
			cfg.AuthMiddleware = authMW
			cfg.WalletHandler = v1.NewWalletHandler(walletUC, topupUC)
			cfg.AdminHandler = v1.NewAdminPayoutHandler(payoutUC)
		}

		handlerhttp.RegisterRoutes(r, cfg)

		// ── Event consumers ───────────────────────────────────────────────────
		orderHandler := payevent.NewOrderEventHandler(
			deps.DB, paymentRepo, walletRepo, ledgerRepo, outboxRepo, processedRepo, refundUC,
		)
		deliveryHandler := payevent.NewDeliveryEventHandler(
			deps.DB, paymentRepo, walletRepo, ledgerRepo, outboxRepo, processedRepo,
		)
		startOrderEventConsumer(a, deps, orderHandler)
		startDeliveryEventConsumer(a, deps, deliveryHandler)

		// ── Background workers ────────────────────────────────────────────────
		startOutboxDispatcher(a, deps)
	})

	a.RegisterGRPC(func(s *grpc.Server, deps app.Dependencies) {
		walletRepo := persistence.NewWalletGormRepository(deps.DB)
		ledgerRepo := persistence.NewLedgerGormRepository(deps.DB)
		paymentRepo := persistence.NewPaymentGormRepository(deps.DB)
		outboxRepo := persistence.NewOutboxGormRepository(deps.DB)
		momoClient := momo.NewClient(deps.Config.MoMo)

		captureUC := usecase.NewCaptureUsecase(
			deps.DB, walletRepo, ledgerRepo, paymentRepo, outboxRepo,
			momoClient, deps.Config.MoMo.IpnURL, deps.Config.MoMo.RedirectURL,
		)
		refundUC := usecase.NewRefundUsecase(deps.DB, walletRepo, ledgerRepo, paymentRepo, outboxRepo)

		paymentv1.RegisterPaymentServiceServer(s, grpchandler.NewPaymentServiceServer(captureUC, refundUC, paymentRepo))
	})

	a.Run()
}
