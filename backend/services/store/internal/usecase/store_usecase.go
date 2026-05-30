package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"project/pkg/outbox"
	"project/services/store/internal/entity"
	"project/services/store/internal/repository"

	"gorm.io/gorm"
)

// UserDirectory resolves user IDs to display names for enriching store responses.
// Implemented by grpcclient.UserClient; use noopUserDirectory when unavailable.
type UserDirectory interface {
	GetUserNames(ctx context.Context, ids []string) (map[string]string, error)
}

// StoreView wraps a Store with the resolved owner display name.
type StoreView struct {
	*entity.Store
	OwnerUserName string `json:"OwnerUserName"`
}

// StoreUsecase handles store lifecycle operations.
type StoreUsecase interface {
	CreateStore(ctx context.Context, vendorID, ownerUserID, name string) (*entity.Store, error)
	GetStore(ctx context.Context, id string) (*entity.Store, error)
	GetStoreByVendorID(ctx context.Context, vendorID string) (*entity.Store, error)
	ListStores(ctx context.Context, saleStatus string) ([]*entity.Store, error)
	UpdateStore(ctx context.Context, id string, name, businessType, address, phone string) (*entity.Store, error)
	UpdateSaleStatus(ctx context.Context, id string, status string) error
	UpdatePickup(ctx context.Context, id string, enabled bool) error
	DeleteStore(ctx context.Context, id string) error

	// Enriched variants include OwnerUserName resolved via the user service.
	ListStoresEnriched(ctx context.Context, saleStatus string) ([]*StoreView, error)
	GetStoreEnriched(ctx context.Context, id string) (*StoreView, error)

	// ListMyStores returns only the stores owned by ownerUserID, enriched with OwnerUserName.
	ListMyStores(ctx context.Context, ownerUserID string) ([]*StoreView, error)
}

type storeUsecase struct {
	db            *gorm.DB
	storeRepo     repository.StoreRepository
	outboxRepo    repository.OutboxRepository
	userDirectory UserDirectory
}

// NewStoreUsecase constructs StoreUsecase.
// userDirectory may be nil — a no-op fallback is used automatically.
func NewStoreUsecase(
	db *gorm.DB,
	storeRepo repository.StoreRepository,
	outboxRepo repository.OutboxRepository,
	userDirectory UserDirectory,
) StoreUsecase {
	if userDirectory == nil {
		userDirectory = &noopUserDirectory{}
	}
	return &storeUsecase{
		db:            db,
		storeRepo:     storeRepo,
		outboxRepo:    outboxRepo,
		userDirectory: userDirectory,
	}
}

var validSaleStatuses = map[string]bool{
	"OPEN": true, "CLOSED_TODAY": true, "PAUSED": true,
}

// CreateStore creates a new store for a vendor. Idempotent — returns existing store
// if one already exists for vendorID (supports event-driven retry).
func (uc *storeUsecase) CreateStore(ctx context.Context, vendorID, ownerUserID, name string) (*entity.Store, error) {
	// Idempotency: return existing store if already created for this vendor.
	existing, err := uc.storeRepo.GetByVendorID(ctx, vendorID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check existing store: %w", err)
	}

	s := &entity.Store{
		VendorID:    vendorID,
		OwnerUserID: ownerUserID,
		Name:        name,
		SaleStatus:  "OPEN",
	}

	payload, _ := json.Marshal(map[string]string{"vendor_id": vendorID, "store_id": ""}) // store ID assigned by DB
	evt := &entity.OutboxEvent{
		AggregateType: "store",
		AggregateID:   vendorID, // placeholder — updated after insert
		EventType:     "store.created",
		Payload:       json.RawMessage(payload),
	}
	traceID := outbox.TraceIDFromCtx(ctx)
	if traceID != "" {
		evt.TraceID = &traceID
	}

	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(s).Error; err != nil {
			return fmt.Errorf("create store: %w", err)
		}
		// Patch payload with the real store ID now that DB has assigned it.
		p, _ := json.Marshal(map[string]string{"vendor_id": vendorID, "store_id": s.ID})
		evt.AggregateID = s.ID
		evt.Payload = json.RawMessage(p)
		if err := uc.outboxRepo.Append(ctx, tx, evt); err != nil {
			return fmt.Errorf("append outbox event: %w", err)
		}
		return nil
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "store: create failed", "vendor_id", vendorID, "err", txErr)
		return nil, txErr
	}

	slog.InfoContext(ctx, "store created", "store_id", s.ID, "vendor_id", vendorID)
	return s, nil
}

func (uc *storeUsecase) GetStore(ctx context.Context, id string) (*entity.Store, error) {
	s, err := uc.storeRepo.GetByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStoreNotFound
	}
	return s, err
}

func (uc *storeUsecase) GetStoreByVendorID(ctx context.Context, vendorID string) (*entity.Store, error) {
	s, err := uc.storeRepo.GetByVendorID(ctx, vendorID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStoreNotFound
	}
	return s, err
}

func (uc *storeUsecase) ListStores(ctx context.Context, saleStatus string) ([]*entity.Store, error) {
	return uc.storeRepo.List(ctx, saleStatus)
}

func (uc *storeUsecase) UpdateStore(ctx context.Context, id, name, businessType, address, phone string) (*entity.Store, error) {
	s, err := uc.storeRepo.GetByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStoreNotFound
	}
	if err != nil {
		return nil, err
	}
	s.Name = name
	s.BusinessType = businessType
	s.Address = address
	s.Phone = phone
	if err := uc.storeRepo.Update(ctx, s); err != nil {
		return nil, fmt.Errorf("update store: %w", err)
	}
	return s, nil
}

// UpdateSaleStatus sets sale_status instantly and publishes store.status_changed.
func (uc *storeUsecase) UpdateSaleStatus(ctx context.Context, id, status string) error {
	if !validSaleStatuses[status] {
		return ErrInvalidSaleStatus
	}

	payload, _ := json.Marshal(map[string]string{"store_id": id, "sale_status": status})
	traceID := outbox.TraceIDFromCtx(ctx)
	evt := &entity.OutboxEvent{
		AggregateType: "store",
		AggregateID:   id,
		EventType:     "store.status_changed",
		Payload:       json.RawMessage(payload),
	}
	if traceID != "" {
		evt.TraceID = &traceID
	}

	return uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.Store{}).Where("id = ?", id).Update("sale_status", status).Error; err != nil {
			return fmt.Errorf("update sale_status: %w", err)
		}
		if err := uc.outboxRepo.Append(ctx, tx, evt); err != nil {
			return fmt.Errorf("append outbox event: %w", err)
		}
		return nil
	})
}

func (uc *storeUsecase) UpdatePickup(ctx context.Context, id string, enabled bool) error {
	if err := uc.storeRepo.UpdatePickupEnabled(ctx, id, enabled); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrStoreNotFound
		}
		return fmt.Errorf("update pickup_enabled: %w", err)
	}
	return nil
}

func (uc *storeUsecase) DeleteStore(ctx context.Context, id string) error {
	if err := uc.storeRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrStoreNotFound
		}
		return err
	}
	return nil
}

// ListStoresEnriched returns stores with OwnerUserName resolved from the user service.
// Resolver errors are tolerated: names are left empty and a warning is logged.
func (uc *storeUsecase) ListStoresEnriched(ctx context.Context, saleStatus string) ([]*StoreView, error) {
	stores, err := uc.storeRepo.List(ctx, saleStatus)
	if err != nil {
		return nil, err
	}

	// Collect unique owner IDs for a single batch call.
	ids := make([]string, 0, len(stores))
	seen := make(map[string]struct{}, len(stores))
	for _, s := range stores {
		if _, ok := seen[s.OwnerUserID]; !ok {
			ids = append(ids, s.OwnerUserID)
			seen[s.OwnerUserID] = struct{}{}
		}
	}

	names, err := uc.userDirectory.GetUserNames(ctx, ids)
	if err != nil {
		slog.WarnContext(ctx, "store: owner name resolution failed — OwnerUserName will be empty", "err", err)
		names = map[string]string{}
	}

	views := make([]*StoreView, len(stores))
	for i, s := range stores {
		views[i] = &StoreView{Store: s, OwnerUserName: names[s.OwnerUserID]}
	}
	return views, nil
}

// GetStoreEnriched returns a single store with OwnerUserName resolved from the user service.
// Resolver errors are tolerated: name is left empty and a warning is logged.
func (uc *storeUsecase) GetStoreEnriched(ctx context.Context, id string) (*StoreView, error) {
	s, err := uc.GetStore(ctx, id)
	if err != nil {
		return nil, err
	}

	names, err := uc.userDirectory.GetUserNames(ctx, []string{s.OwnerUserID})
	if err != nil {
		slog.WarnContext(ctx, "store: owner name resolution failed — OwnerUserName will be empty", "err", err)
		names = map[string]string{}
	}

	return &StoreView{Store: s, OwnerUserName: names[s.OwnerUserID]}, nil
}

// ListMyStores returns only the stores owned by ownerUserID, enriched with OwnerUserName.
// Resolver errors are tolerated: name is left empty and a warning is logged.
func (uc *storeUsecase) ListMyStores(ctx context.Context, ownerUserID string) ([]*StoreView, error) {
	stores, err := uc.storeRepo.ListByOwner(ctx, ownerUserID)
	if err != nil {
		return nil, err
	}

	names, err := uc.userDirectory.GetUserNames(ctx, []string{ownerUserID})
	if err != nil {
		slog.WarnContext(ctx, "store: owner name resolution failed — OwnerUserName will be empty", "err", err)
		names = map[string]string{}
	}

	views := make([]*StoreView, len(stores))
	for i, s := range stores {
		views[i] = &StoreView{Store: s, OwnerUserName: names[s.OwnerUserID]}
	}
	return views, nil
}

// noopUserDirectory is used when the user service is unavailable at startup.
// All calls return an empty map without error so store responses still succeed.
type noopUserDirectory struct{}

func (n *noopUserDirectory) GetUserNames(_ context.Context, _ []string) (map[string]string, error) {
	return map[string]string{}, nil
}
