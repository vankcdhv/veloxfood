package v1

import (
	"net/http"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/payment/internal/usecase"

	"github.com/gin-gonic/gin"
)

// AdminPayoutHandler serves payout management endpoints (admin auth required).
type AdminPayoutHandler struct {
	payoutUC usecase.PayoutUsecase
}

func NewAdminPayoutHandler(payoutUC usecase.PayoutUsecase) *AdminPayoutHandler {
	return &AdminPayoutHandler{payoutUC: payoutUC}
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
		StoreID     string   `json:"store_id"`
		PeriodFrom  string   `json:"period_from"`
		PeriodTo    string   `json:"period_to"`
		OrderIDs    []string `json:"order_ids"`
		TotalAmount int64    `json:"total_amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.StoreID == "" || req.TotalAmount <= 0 {
		response.BadRequest(c, "store_id and positive total_amount required")
		return
	}

	batch, err := h.payoutUC.CreateBatch(ctx, usecase.CreatePayoutRequest{
		StoreID:     req.StoreID,
		PeriodFrom:  req.PeriodFrom,
		PeriodTo:    req.PeriodTo,
		OrderIDs:    req.OrderIDs,
		TotalAmount: req.TotalAmount,
	})
	if err != nil {
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
