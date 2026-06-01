package main

import (
	"log/slog"

	"project/pkg/app"
	pkgfirebase "project/pkg/firebase"
	"project/pkg/mailer"
	"project/pkg/middleware"
	notifEvent "project/services/notification/internal/handler/event"
	handlerhttp "project/services/notification/internal/handler/http"
	v1 "project/services/notification/internal/handler/http/v1"
	"project/services/notification/internal/infrastructure/persistence"
	"project/services/notification/internal/usecase"

	"github.com/gin-gonic/gin"
)

func main() {
	a := app.New("notification-service").WithConfigPath("config/notification.yaml")

	a.RegisterHTTP(func(r *gin.Engine, deps app.Dependencies) {
		r.Use(middleware.RequestMetadata())

		// ── Firebase (graceful degradation when path empty or init fails) ─────
		fbClient, err := pkgfirebase.New(deps.Config.Firebase)
		if err != nil {
			// New() already returns a no-op client on failure; this branch is not
			// reached under normal conditions but is guarded for safety.
			slog.Error("notification: firebase init returned unexpected error", "err", err)
		}
		defer a.OnShutdown(fbClient.Close)

		// ── Mailer (no-op when SMTP not configured) ────────────────────────────
		mailClient := mailer.NewSMTPMailer(deps.Config.SMTP)

		// ── Repositories ──────────────────────────────────────────────────────
		notifRepo := persistence.NewNotificationGormRepository(deps.DB)
		tokenRepo := persistence.NewDeviceTokenGormRepository(deps.DB)
		processedRepo := persistence.NewProcessedEventGormRepository(deps.DB)

		// ── Usecase ───────────────────────────────────────────────────────────
		uc := usecase.NewNotificationUsecase(
			deps.DB, notifRepo, tokenRepo, processedRepo,
			fbClient, mailClient,
		)

		// ── HTTP router ───────────────────────────────────────────────────────
		cfg := handlerhttp.RouterConfig{}
		if authMW, err := buildAuth(deps); err != nil {
			slog.Error("notification: auth setup failed — protected routes disabled", "err", err)
		} else {
			cfg.AuthMiddleware = authMW
			cfg.Handler = v1.NewNotificationHandler(uc)
		}
		handlerhttp.RegisterRoutes(r, cfg)

		// ── Event consumers ───────────────────────────────────────────────────
		orderH := notifEvent.NewOrderEventHandler(uc)
		deliveryH := notifEvent.NewDeliveryEventHandler(uc)
		payH := notifEvent.NewPaymentEventHandler(uc)
		vrH := notifEvent.NewVendorReviewEventHandler(uc)

		startAllConsumers(a, deps, orderH, deliveryH, payH, vrH)
	})

	a.Run()
}
