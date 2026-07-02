package main

import (
	"log/slog"

	"project/pkg/app"
	"project/pkg/audit"
	"project/pkg/storage"
	v1 "project/services/user/internal/handler/http/v1"
	"project/services/user/internal/infrastructure/persistence"
	"project/services/user/internal/usecase"
)

// shipperHandlers groups shipper self-service + admin approval handlers.
type shipperHandlers struct {
	shipperHandler      *v1.ShipperHandler
	adminShipperHandler *v1.AdminShipperHandler
}

// buildShipperHandlers wires shipper registration + admin approval.
// kyc may be nil when MinIO is unavailable — registration is then disabled and
// the admin list falls back to raw object keys instead of presigned URLs.
func buildShipperHandlers(deps app.Dependencies, rbacUC usecase.RBACUsecase, kyc *storage.Client) *shipperHandlers {
	shipperRepo := persistence.NewShipperProfileGormRepository(deps.DB)
	roleRepo := persistence.NewRoleGormRepository(deps.DB)
	outboxRepo := persistence.NewOutboxGormRepository(deps.DB)
	auditLogger := audit.NewGormLogger(deps.DB)

	var signer usecase.ObjectURLSigner
	if kyc != nil {
		signer = kyc
	}
	adminShipperUC := usecase.NewAdminShipperUsecase(deps.DB, shipperRepo, roleRepo, outboxRepo, rbacUC, auditLogger, signer)

	out := &shipperHandlers{
		adminShipperHandler: v1.NewAdminShipperHandler(adminShipperUC),
	}

	if kyc != nil {
		registerUC := usecase.NewShipperRegisterUsecase(deps.DB, shipperRepo, outboxRepo, kyc, auditLogger)
		out.shipperHandler = v1.NewShipperHandler(registerUC)
	} else {
		slog.Warn("minio uploader unavailable — shipper registration disabled")
	}

	slog.Info("shipper handlers wired")
	return out
}

// newKYCStorage builds the private-bucket MinIO client for KYC documents,
// returning nil on failure (logged). Documents in this bucket are only
// reachable through presigned URLs.
func newKYCStorage(deps app.Dependencies) *storage.Client {
	client, err := storage.NewPrivateClient(deps.Config.MinIO)
	if err != nil {
		slog.Error("minio private client init failed — KYC uploads disabled", "err", err)
		return nil
	}
	return client
}
