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
// uploader may be nil when MinIO is unavailable — registration is then disabled.
func buildShipperHandlers(deps app.Dependencies, rbacUC usecase.RBACUsecase, uploader usecase.FileUploader) *shipperHandlers {
	shipperRepo := persistence.NewShipperProfileGormRepository(deps.DB)
	roleRepo := persistence.NewRoleGormRepository(deps.DB)
	outboxRepo := persistence.NewOutboxGormRepository(deps.DB)
	auditLogger := audit.NewGormLogger(deps.DB)

	adminShipperUC := usecase.NewAdminShipperUsecase(deps.DB, shipperRepo, roleRepo, outboxRepo, rbacUC, auditLogger)

	out := &shipperHandlers{
		adminShipperHandler: v1.NewAdminShipperHandler(adminShipperUC),
	}

	if uploader != nil {
		registerUC := usecase.NewShipperRegisterUsecase(deps.DB, shipperRepo, outboxRepo, uploader, auditLogger)
		out.shipperHandler = v1.NewShipperHandler(registerUC)
	} else {
		slog.Warn("minio uploader unavailable — shipper registration disabled")
	}

	slog.Info("shipper handlers wired")
	return out
}

// newMinioUploader builds the MinIO client, returning nil on failure (logged).
func newMinioUploader(deps app.Dependencies) usecase.FileUploader {
	client, err := storage.NewClient(deps.Config.MinIO)
	if err != nil {
		slog.Error("minio client init failed — uploads disabled", "err", err)
		return nil
	}
	return client
}
