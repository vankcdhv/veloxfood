package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"project/pkg/audit"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

// FileUploader abstracts object storage (implemented by pkg/storage.Client).
// Defined here so the usecase layer stays free of infrastructure imports.
type FileUploader interface {
	Put(ctx context.Context, objectKey, contentType string, r io.Reader, size int64) (string, error)
}

// UploadFile is one in-memory file from the multipart request.
type UploadFile struct {
	Reader      io.Reader
	Size        int64
	ContentType string
	Ext         string // e.g. ".jpg"
}

// ShipperRegisterInput carries the two verification photos.
type ShipperRegisterInput struct {
	IDDocument UploadFile
	Portrait   UploadFile
}

// ShipperRegisterOutput is returned after a successful application.
type ShipperRegisterOutput struct {
	Status string `json:"status"`
}

// ShipperStatusOutput is the caller's own shipper application state.
type ShipperStatusOutput struct {
	Status     string     `json:"status"`
	AppliedAt  time.Time  `json:"applied_at"`
	ApprovedAt *time.Time `json:"approved_at"`
}

// ShipperRegisterUsecase handles shipper self-registration.
type ShipperRegisterUsecase interface {
	Register(ctx context.Context, userID string, in ShipperRegisterInput) (*ShipperRegisterOutput, error)
	// GetMine returns the caller's application state, or nil if they never applied.
	GetMine(ctx context.Context, userID string) (*ShipperStatusOutput, error)
}

type shipperRegisterUsecase struct {
	db          *gorm.DB
	shipperRepo repository.ShipperProfileRepository
	outboxRepo  repository.OutboxRepository
	uploader    FileUploader
	auditLogger audit.Logger
}

func NewShipperRegisterUsecase(
	db *gorm.DB,
	shipperRepo repository.ShipperProfileRepository,
	outboxRepo repository.OutboxRepository,
	uploader FileUploader,
	auditLogger audit.Logger,
) ShipperRegisterUsecase {
	return &shipperRegisterUsecase{
		db:          db,
		shipperRepo: shipperRepo,
		outboxRepo:  outboxRepo,
		uploader:    uploader,
		auditLogger: auditLogger,
	}
}

func (uc *shipperRegisterUsecase) Register(ctx context.Context, userID string, in ShipperRegisterInput) (*ShipperRegisterOutput, error) {
	if in.IDDocument.Reader == nil || in.Portrait.Reader == nil {
		return nil, ErrShipperDocsRequired
	}

	// Reject duplicate applications (any prior profile).
	if _, err := uc.shipperRepo.GetByUserID(ctx, userID); err == nil {
		return nil, ErrShipperAlreadyApplied
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check shipper profile: %w", err)
	}

	idKey := fmt.Sprintf("shippers/%s/id-document%s", userID, in.IDDocument.Ext)
	portraitKey := fmt.Sprintf("shippers/%s/portrait%s", userID, in.Portrait.Ext)

	if _, err := uc.uploader.Put(ctx, idKey, in.IDDocument.ContentType, in.IDDocument.Reader, in.IDDocument.Size); err != nil {
		slog.ErrorContext(ctx, "shipper register: id upload failed", "user_id", userID, "err", err)
		return nil, ErrShipperUploadFailed
	}
	if _, err := uc.uploader.Put(ctx, portraitKey, in.Portrait.ContentType, in.Portrait.Reader, in.Portrait.Size); err != nil {
		slog.ErrorContext(ctx, "shipper register: portrait upload failed", "user_id", userID, "err", err)
		return nil, ErrShipperUploadFailed
	}

	profile := &entity.ShipperProfile{
		UserID:             userID,
		IDDocumentPhotoURL: idKey,
		PortraitPhotoURL:   portraitKey,
		Status:             entity.ShipperStatusPending,
	}

	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Create(profile).Error; err != nil {
			return fmt.Errorf("create shipper profile: %w", err)
		}
		payload, _ := json.Marshal(map[string]string{"user_id": userID})
		evt := &entity.OutboxEvent{
			AggregateType: "shipper",
			AggregateID:   userID,
			EventType:     "shipper.requested",
			Payload:       json.RawMessage(payload),
		}
		return uc.outboxRepo.Append(ctx, tx, evt)
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "shipper register: tx failed", "user_id", userID, "err", txErr)
		return nil, txErr
	}

	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &userID,
		Action:      "shipper.registered",
		TargetType:  "shipper",
		TargetID:    &userID,
	})
	slog.InfoContext(ctx, "shipper registered", "user_id", userID)
	return &ShipperRegisterOutput{Status: "pending_admin_approval"}, nil
}

func (uc *shipperRegisterUsecase) GetMine(ctx context.Context, userID string) (*ShipperStatusOutput, error) {
	p, err := uc.shipperRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // never applied
		}
		return nil, err
	}
	return &ShipperStatusOutput{
		Status:     string(p.Status),
		AppliedAt:  p.CreatedAt,
		ApprovedAt: p.ApprovedAt,
	}, nil
}
