package usecase

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/apperror"
	"project/pkg/outbox"
	authmw "project/pkg/auth/middleware"
	"project/services/review/internal/entity"
	"project/services/review/internal/infrastructure/grpcclient"
	"project/services/review/internal/repository"

	"gorm.io/gorm"
)

// ReviewUsecase exposes all business operations for the review service.
type ReviewUsecase interface {
	CreateReview(ctx context.Context, req CreateReviewRequest) (*entity.Review, error)
	ListStoreReviews(ctx context.Context, storeID string, page, pageSize int) ([]*entity.Review, int64, error)
	ListItemReviews(ctx context.Context, itemID string, page, pageSize int) ([]*entity.Review, int64, error)
	GetItemRatingSummaries(ctx context.Context, storeID string) ([]repository.ItemRatingSummary, error)
	ReplyToReview(ctx context.Context, reviewID, content string) (*entity.ReviewReply, error)
	ReportReview(ctx context.Context, reviewID, reason string) (*entity.ReviewReport, error)
	HideReview(ctx context.Context, reviewID string) error
	RestoreReview(ctx context.Context, reviewID string) error
	ListReportedReviews(ctx context.Context, page, pageSize int) ([]*entity.ReviewReport, int64, error)
	GetStoreRatingSummary(ctx context.Context, storeID string) (avg float64, count int32, err error)
}

// CreateReviewRequest carries the validated input for a new review.
type CreateReviewRequest struct {
	OrderID    string
	CustomerID string // from JWT
	TargetType string // STORE | ITEM | SHIPPER
	TargetID   string
	Rating     int
	Comment    string
	PhotoURLs  []string
}

type reviewUsecase struct {
	db            *gorm.DB
	reviewRepo    repository.ReviewRepository
	replyRepo     repository.ReviewReplyRepository
	reportRepo    repository.ReviewReportRepository
	outboxRepo    repository.OutboxRepository
	processedRepo repository.ProcessedEventRepository
	orderClient   grpcclient.OrderClient
	storeClient   grpcclient.StoreClient
	permChecker   authmw.PermissionChecker
}

// NewReviewUsecase constructs the ReviewUsecase.
func NewReviewUsecase(
	db *gorm.DB,
	reviewRepo repository.ReviewRepository,
	replyRepo repository.ReviewReplyRepository,
	reportRepo repository.ReviewReportRepository,
	outboxRepo repository.OutboxRepository,
	processedRepo repository.ProcessedEventRepository,
	orderClient grpcclient.OrderClient,
	storeClient grpcclient.StoreClient,
	permChecker authmw.PermissionChecker,
) ReviewUsecase {
	return &reviewUsecase{
		db: db, reviewRepo: reviewRepo, replyRepo: replyRepo,
		reportRepo: reportRepo, outboxRepo: outboxRepo, processedRepo: processedRepo,
		orderClient: orderClient, storeClient: storeClient, permChecker: permChecker,
	}
}

// CreateReview validates the order via Order.GetOrder then persists the review
// and enqueues a review.created outbox event atomically.
func (uc *reviewUsecase) CreateReview(ctx context.Context, req CreateReviewRequest) (*entity.Review, error) {
	if req.Rating < 1 || req.Rating > 5 {
		return nil, ErrInvalidRating
	}
	validTargets := map[string]bool{"STORE": true, "ITEM": true, "SHIPPER": true}
	if !validTargets[req.TargetType] {
		return nil, ErrInvalidTargetType
	}

	// Validate order: must exist, be COMPLETED, and belong to this customer.
	order, err := uc.orderClient.GetOrder(ctx, req.OrderID)
	if err != nil {
		slog.ErrorContext(ctx, "review: GetOrder failed", "order_id", req.OrderID, "err", err)
		return nil, apperror.Internal("could not validate order")
	}
	if !order.Found {
		return nil, ErrOrderNotFound
	}
	if order.Status != "COMPLETED" {
		return nil, ErrOrderNotCompleted
	}
	if order.CustomerID != req.CustomerID {
		return nil, ErrWrongCustomer
	}

	// Use store_id from the order so the client cannot spoof it.
	storeID := order.StoreID

	photoURLs := entity.StringSlice(req.PhotoURLs)
	if photoURLs == nil {
		photoURLs = entity.StringSlice{}
	}

	rev := &entity.Review{
		OrderID:    req.OrderID,
		CustomerID: req.CustomerID,
		StoreID:    storeID,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Rating:     req.Rating,
		Comment:    req.Comment,
		PhotoURLs:  photoURLs,
		Status:     "VISIBLE",
	}

	// Build outbox payload for review.created event.
	payload, err := json.Marshal(map[string]any{
		"review_id":   "", // filled after INSERT in tx
		"store_id":    storeID,
		"order_id":    req.OrderID,
		"rating":      req.Rating,
		"target_type": req.TargetType,
	})
	if err != nil {
		return nil, apperror.Internal("marshal event payload")
	}

	var created *entity.Review
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := uc.reviewRepo.Create(ctx, tx, rev); err != nil {
			return err
		}
		created = rev

		// Update payload with real review_id after insert.
		payload, _ = json.Marshal(map[string]any{
			"review_id":   rev.ID,
			"store_id":    storeID,
			"order_id":    req.OrderID,
			"rating":      req.Rating,
			"target_type": req.TargetType,
		})

		evt := &entity.OutboxEvent{
			AggregateType: "review",
			AggregateID:   rev.ID,
			EventType:     "review.created",
			Payload:       payload,
			TraceID:       outbox.TraceIDFromCtx(ctx),
			Status:        "pending",
		}
		return uc.outboxRepo.Insert(ctx, tx, evt)
	})

	if txErr != nil {
		// Detect UNIQUE(order_id, target_type, target_id) violation.
		if isDuplicateError(txErr) {
			return nil, ErrDuplicateReview
		}
		slog.ErrorContext(ctx, "review: create transaction failed", "err", txErr)
		return nil, apperror.Internal("create review failed")
	}

	slog.InfoContext(ctx, "review created",
		"review_id", created.ID, "order_id", req.OrderID, "store_id", storeID)
	return created, nil
}

func (uc *reviewUsecase) ListStoreReviews(ctx context.Context, storeID string, page, pageSize int) ([]*entity.Review, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return uc.reviewRepo.ListByStore(ctx, storeID, pageSize, (page-1)*pageSize)
}

func (uc *reviewUsecase) ListItemReviews(ctx context.Context, itemID string, page, pageSize int) ([]*entity.Review, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return uc.reviewRepo.ListByTarget(ctx, "ITEM", itemID, pageSize, (page-1)*pageSize)
}

func (uc *reviewUsecase) GetItemRatingSummaries(ctx context.Context, storeID string) ([]repository.ItemRatingSummary, error) {
	return uc.reviewRepo.ItemRatingSummaries(ctx, storeID)
}

// ReplyToReview lets a store owner respond to a review. Authorisation: the caller
// must have store.manage permission scoped to the review's store vendor.
func (uc *reviewUsecase) ReplyToReview(ctx context.Context, reviewID, content string) (*entity.ReviewReply, error) {
	if content == "" {
		return nil, apperror.BadRequest("content is required")
	}
	rev, err := uc.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, ErrReviewNotFound
	}

	userID := authmw.UserIDFromContext(ctx)
	ownership, err := uc.storeClient.GetStoreOwnership(ctx, rev.StoreID)
	if err != nil || !ownership.Found {
		return nil, ErrStoreNotFound
	}

	ok, err := uc.permChecker.HasVendorPermission(ctx, userID, ownership.VendorID, "store.manage")
	if err != nil {
		return nil, apperror.Internal("permission check failed")
	}
	if !ok {
		return nil, ErrUnauthorizedReply
	}

	reply := &entity.ReviewReply{
		ReviewID:     reviewID,
		AuthorUserID: userID,
		Content:      content,
	}
	if err := uc.replyRepo.Create(ctx, reply); err != nil {
		return nil, apperror.Internal("create reply failed")
	}
	return reply, nil
}

// ReportReview records a content complaint and enqueues a review.reported event.
func (uc *reviewUsecase) ReportReview(ctx context.Context, reviewID, reason string) (*entity.ReviewReport, error) {
	if reason == "" {
		return nil, apperror.BadRequest("reason is required")
	}
	if _, err := uc.reviewRepo.GetByID(ctx, reviewID); err != nil {
		return nil, ErrReviewNotFound
	}

	userID := authmw.UserIDFromContext(ctx)
	report := &entity.ReviewReport{
		ReviewID:   reviewID,
		ReportedBy: userID,
		Reason:     reason,
		Status:     "OPEN",
	}

	payload, _ := json.Marshal(map[string]any{"review_id": reviewID})
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := uc.reportRepo.Create(ctx, report); err != nil {
			return err
		}
		evt := &entity.OutboxEvent{
			AggregateType: "review",
			AggregateID:   reviewID,
			EventType:     "review.reported",
			Payload:       payload,
			TraceID:       outbox.TraceIDFromCtx(ctx),
			Status:        "pending",
		}
		return uc.outboxRepo.Insert(ctx, tx, evt)
	})
	if txErr != nil {
		return nil, apperror.Internal("report review failed")
	}
	return report, nil
}

// HideReview sets a review status to HIDDEN (admin action).
func (uc *reviewUsecase) HideReview(ctx context.Context, reviewID string) error {
	if _, err := uc.reviewRepo.GetByID(ctx, reviewID); err != nil {
		return ErrReviewNotFound
	}
	return uc.reviewRepo.UpdateStatus(ctx, reviewID, "HIDDEN")
}

// RestoreReview sets a review status back to VISIBLE (admin action).
func (uc *reviewUsecase) RestoreReview(ctx context.Context, reviewID string) error {
	if _, err := uc.reviewRepo.GetByID(ctx, reviewID); err != nil {
		return ErrReviewNotFound
	}
	return uc.reviewRepo.UpdateStatus(ctx, reviewID, "VISIBLE")
}

func (uc *reviewUsecase) ListReportedReviews(ctx context.Context, page, pageSize int) ([]*entity.ReviewReport, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return uc.reportRepo.ListOpen(ctx, pageSize, (page-1)*pageSize)
}

func (uc *reviewUsecase) GetStoreRatingSummary(ctx context.Context, storeID string) (float64, int32, error) {
	// The store's rating reflects STORE-targeted reviews only (per-item reviews
	// roll up to each menu item, not the store average).
	return uc.reviewRepo.RatingSummaryByTarget(ctx, "STORE", storeID)
}

// isDuplicateError detects Postgres unique-violation errors (code 23505).
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "23505") || contains(msg, "unique") || contains(msg, "duplicate")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
