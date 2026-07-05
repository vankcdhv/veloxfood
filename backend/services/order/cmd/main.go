package main

import (
	"log/slog"

	"project/pkg/app"
	"project/pkg/audit"
	"project/pkg/middleware"
	orderv1 "project/proto/order/v1"
	orderevent "project/services/order/internal/handler/event"
	grpchandler "project/services/order/internal/handler/grpc"
	handlerhttp "project/services/order/internal/handler/http"
	v1 "project/services/order/internal/handler/http/v1"
	"project/services/order/internal/infrastructure/grpcclient"
	"project/services/order/internal/infrastructure/persistence"
	"project/services/order/internal/usecase"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {
	a := app.New("order-service").WithConfigPath("config/order.yaml")

	a.RegisterHTTP(func(r *gin.Engine, deps app.Dependencies) {
		r.Use(middleware.RequestMetadata())

		// ── Repositories ──────────────────────────────────────────────────────
		orderRepo := persistence.NewOrderGormRepository(deps.DB)
		cartRepo := persistence.NewCartGormRepository(deps.DB)
		outboxRepo := persistence.NewOutboxGormRepository(deps.DB)
		processedRepo := persistence.NewProcessedEventGormRepository(deps.DB)

		// ── Cross-service gRPC clients ─────────────────────────────────────────
		storeClient := buildStoreClient(deps)
		promoClient := buildPromotionClient(deps)
		paymentClient := buildPaymentClient(deps)
		userClient := buildUserClient(deps)
		locationClient := buildLocationClient(deps)

		// ── Usecases ──────────────────────────────────────────────────────────
		cartUC := usecase.NewCartUsecase(cartRepo)
		compRepo := persistence.NewCompensationGormRepository(deps.DB)
		// Convert failed-dial nil pointers into true nil interfaces so the
		// usecase's nil guards work (a typed-nil in an interface is non-nil).
		var storeGW usecase.StoreGateway
		if storeClient != nil {
			storeGW = storeClient
		}
		var promoGW usecase.PromotionGateway
		if promoClient != nil {
			promoGW = promoClient
		}
		var paymentGW usecase.PaymentGateway
		if paymentClient != nil {
			paymentGW = paymentClient
		}
		placeOrderUC := usecase.NewPlaceOrderUsecase(deps.DB, orderRepo, cartRepo, outboxRepo, compRepo, storeGW, promoGW, paymentGW)
		lifecycleUC := usecase.NewOrderLifecycleUsecase(deps.DB, orderRepo, cartRepo, outboxRepo, storeClient, promoClient, paymentClient, audit.NewGormLogger(deps.DB))

		// ── HTTP router ───────────────────────────────────────────────────────
		cfg := handlerhttp.RouterConfig{}
		if authMW, permChecker, err := buildAuth(deps); err != nil {
			slog.Error("order: auth setup failed — protected routes disabled", "err", err)
		} else {
			cfg.AuthMiddleware = authMW
			cfg.CartHandler = v1.NewCustomerCartHandler(cartUC)
			cfg.CustomerHandler = v1.NewCustomerOrderHandler(placeOrderUC, lifecycleUC)
			// Pass a true nil interface when the store client failed to dial so the
			// owner handler's nil guard catches it (avoids a typed-nil interface).
			var ownerStore v1.StoreOwnershipResolver
			if storeClient != nil {
				ownerStore = storeClient
			}
			cfg.OwnerHandler = v1.NewOwnerOrderHandler(lifecycleUC, userClient, locationClient, ownerStore, permChecker)
		}
		handlerhttp.RegisterRoutes(r, cfg)

		// ── Event consumers ───────────────────────────────────────────────────
		paymentHandler := orderevent.NewPaymentEventHandler(deps.DB, orderRepo, processedRepo)
		deliveryHandler := orderevent.NewDeliveryEventHandler(deps.DB, orderRepo, outboxRepo, processedRepo)
		storeHandler := orderevent.NewStoreEventHandler()

		startPaymentEventConsumer(a, deps, paymentHandler)
		startDeliveryEventConsumer(a, deps, deliveryHandler)
		startStoreEventConsumer(a, deps, storeHandler)

		// ── Background workers ────────────────────────────────────────────────
		startOutboxDispatcher(a, deps)
		startCompensationWorker(a, compRepo, paymentClient, promoClient)
	})

	a.RegisterGRPC(func(s *grpc.Server, deps app.Dependencies) {
		orderRepo := persistence.NewOrderGormRepository(deps.DB)
		cartRepo := persistence.NewCartGormRepository(deps.DB)
		outboxRepo := persistence.NewOutboxGormRepository(deps.DB)
		storeClient := buildStoreClient(deps)
		promoClient := buildPromotionClient(deps)
		paymentClient := buildPaymentClient(deps)

		lifecycleUC := usecase.NewOrderLifecycleUsecase(
			deps.DB, orderRepo, cartRepo, outboxRepo,
			storeClient, promoClient, paymentClient,
			audit.NewGormLogger(deps.DB),
		)
		orderv1.RegisterOrderServiceServer(s, grpchandler.NewOrderServiceServer(lifecycleUC))
	})

	a.Run()
}

// buildStoreClient dials the store service. Returns nil on failure — place-order
// will return ErrStoreNotFound so no panic occurs.
func buildStoreClient(deps app.Dependencies) *grpcclient.StoreClient {
	client, err := grpcclient.NewStoreClient(deps.Config.StoreService.GRPCAddr)
	if err != nil {
		slog.Error("order: store grpc client failed — order placement disabled", "err", err)
		return nil
	}
	return client
}

// buildPromotionClient dials the promotion service. Returns nil on failure —
// orders without vouchers continue normally; voucher codes will be rejected.
func buildPromotionClient(deps app.Dependencies) *grpcclient.PromotionClient {
	client, err := grpcclient.NewPromotionClient(deps.Config.PromotionService.GRPCAddr)
	if err != nil {
		slog.Error("order: promotion grpc client failed — vouchers disabled", "err", err)
		return nil
	}
	return client
}

// buildPaymentClient dials the payment service. Returns nil on failure —
// COD orders still work; WALLET/MOMO orders will fail at capture.
func buildPaymentClient(deps app.Dependencies) *grpcclient.PaymentClient {
	client, err := grpcclient.NewPaymentClient(deps.Config.PaymentService.GRPCAddr)
	if err != nil {
		slog.Error("order: payment grpc client failed — online payment disabled", "err", err)
		return nil
	}
	return client
}

// buildUserClient dials the user service. Returns nil on failure — the owner
// order view degrades to empty customer name/phone.
func buildUserClient(deps app.Dependencies) *grpcclient.UserClient {
	client, err := grpcclient.NewUserClient(deps.Config.UserService.GRPCAddr)
	if err != nil {
		slog.Error("order: user grpc client failed — owner order contact disabled", "err", err)
		return nil
	}
	return client
}

// buildLocationClient dials the location service. Returns nil on failure — the
// owner order view degrades to the raw location id.
func buildLocationClient(deps app.Dependencies) *grpcclient.LocationClient {
	client, err := grpcclient.NewLocationClient(deps.Config.LocationService.GRPCAddr)
	if err != nil {
		slog.Error("order: location grpc client failed — owner order path disabled", "err", err)
		return nil
	}
	return client
}
