package v1

import (
	"context"
	"net/http"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/payment/internal/infrastructure/grpcclient"
	"project/services/payment/internal/usecase"

	"github.com/gin-gonic/gin"
)

// StoreOwnershipResolver resolves a store's vendor so the handler can authorize
// a store owner. Satisfied by *grpcclient.StoreClient.
type StoreOwnershipResolver interface {
	GetStoreOwnership(ctx context.Context, storeID string) (*grpcclient.StoreOwnership, error)
}

// AdminPayoutHandler serves payout management endpoints (admin auth required).
type AdminPayoutHandler struct {
	payoutUC    usecase.PayoutUsecase
	storeClient StoreOwnershipResolver
	checker     authmw.PermissionChecker
}

func NewAdminPayoutHandler(payoutUC usecase.PayoutUsecase, storeClient StoreOwnershipResolver, checker authmw.PermissionChecker) *AdminPayoutHandler {
	return &AdminPayoutHandler{payoutUC: payoutUC, storeClient: storeClient, checker: checker}
}

// authorizeStoreOwner verifies the caller holds store.manage on the store's
// vendor. The /me/store-revenue route is auth-only (any logged-in user), so
// without this check anyone could read another store's revenue by store_id.
// Writes the error response and returns false on failure.
func (h *AdminPayoutHandler) authorizeStoreOwner(c *gin.Context, storeID string) bool {
	ctx := c.Request.Context()
	if h.storeClient == nil || h.checker == nil {
		response.InternalError(c)
		return false
	}
	ownership, err := h.storeClient.GetStoreOwnership(ctx, storeID)
	if err != nil {
		response.InternalError(c)
		return false
	}
	if !ownership.Found {
		response.NotFound(c, "store not found")
		return false
	}
	userID := authmw.UserIDFromContext(ctx)
	ok, err := h.checker.HasVendorPermission(ctx, userID, ownership.VendorID, "store.manage")
	if err != nil {
		response.InternalError(c)
		return false
	}
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions for this store"})
		return false
	}
	return true
}

// ListPayouts returns payout batches for a store.
// GET /api/v1/admin/payouts?store_id=<uuid>&limit=20&offset=0
func (h *AdminPayoutHandler) ListPayouts(c *gin.Context) {
	ctx := c.Request.Context()
	storeID := c.Query("store_id")
	if storeID == "" {
		response.BadRequest(c, "store_id required")
		return
	}

	batches, err := h.payoutUC.ListByStore(ctx, storeID, 20, 0)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.Success(c, batches)
}

// CreatePayout creates a new payout batch.
// POST /api/v1/admin/payouts
// Body: {"store_id","period_from","period_to","order_ids":[],"total_amount"}
func (h *AdminPayoutHandler) CreatePayout(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		StoreID    string   `json:"store_id"`
		PeriodFrom string   `json:"period_from"`
		PeriodTo   string   `json:"period_to"`
		OrderIDs   []string `json:"order_ids"`
	}
	// order_ids is an OPTIONAL subset selected by the admin (period / per-order
	// filter). total_amount is always re-derived server-side; an empty order_ids
	// means "pay out everything settleable".
	if err := c.ShouldBindJSON(&req); err != nil || req.StoreID == "" {
		response.BadRequest(c, "store_id required")
		return
	}

	batch, err := h.payoutUC.CreateBatch(ctx, usecase.CreatePayoutRequest{
		StoreID:    req.StoreID,
		PeriodFrom: req.PeriodFrom,
		PeriodTo:   req.PeriodTo,
		OrderIDs:   req.OrderIDs,
	})
	if err != nil {
		if err == usecase.ErrNothingToSettle {
			response.BadRequest(c, "nothing to settle for this store")
			return
		}
		response.InternalError(c)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": 201, "message": "created", "data": batch})
}

// GetSettlements returns the settleable balance and order breakdown for a store.
// GET /api/v1/admin/settlements?store_id=<uuid>
func (h *AdminPayoutHandler) GetSettlements(c *gin.Context) {
	ctx := c.Request.Context()
	storeID := c.Query("store_id")
	if storeID == "" {
		response.BadRequest(c, "store_id required")
		return
	}

	summary, err := h.payoutUC.GetSettleableSummary(ctx, storeID)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.Success(c, summary)
}

// GetStoreRevenue returns a store owner's earnings overview: current unsettled
// (payable) balance + the breakdown of unsettled orders + past payout batches.
// GET /api/v1/me/store-revenue?store_id=<uuid>
func (h *AdminPayoutHandler) GetStoreRevenue(c *gin.Context) {
	ctx := c.Request.Context()
	storeID := c.Query("store_id")
	if storeID == "" {
		response.BadRequest(c, "store_id required")
		return
	}
	if !h.authorizeStoreOwner(c, storeID) {
		return
	}

	summary, err := h.payoutUC.GetSettleableSummary(ctx, storeID)
	if err != nil {
		response.InternalError(c)
		return
	}
	batches, err := h.payoutUC.ListByStore(ctx, storeID, 20, 0)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.Success(c, gin.H{"summary": summary, "batches": batches})
}

// ExecutePayout atomically settles a payout batch.
// POST /api/v1/admin/payouts/:id/execute
func (h *AdminPayoutHandler) ExecutePayout(c *gin.Context) {
	ctx := c.Request.Context()
	batchID := c.Param("id")
	adminID := authmw.UserIDFromContext(ctx)

	err := h.payoutUC.ExecuteBatch(ctx, batchID, adminID)
	if err != nil {
		switch err {
		case usecase.ErrPayoutNotFound:
			response.NotFound(c, "payout batch not found")
		case usecase.ErrPayoutAlreadySettled:
			response.Success(c, gin.H{"message": "already settled"})
		default:
			response.InternalError(c)
		}
		return
	}
	response.Success(c, gin.H{"message": "settled"})
}
