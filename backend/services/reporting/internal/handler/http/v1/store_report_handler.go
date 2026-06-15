package v1

import (
	"context"
	"net/http"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/reporting/internal/infrastructure/grpcclient"
	"project/services/reporting/internal/usecase"

	"github.com/gin-gonic/gin"
)

// StoreOwnershipResolver resolves a store's vendor so the handler can authorize
// a store owner. Satisfied by *grpcclient.StoreClient.
type StoreOwnershipResolver interface {
	GetStoreOwnership(ctx context.Context, storeID string) (*grpcclient.StoreOwnership, error)
}

// StoreReportHandler serves per-store analytics to authenticated store owners.
type StoreReportHandler struct {
	analyticsUC *usecase.AnalyticsUsecase
	storeClient StoreOwnershipResolver
	checker     authmw.PermissionChecker
}

func NewStoreReportHandler(analyticsUC *usecase.AnalyticsUsecase, storeClient StoreOwnershipResolver, checker authmw.PermissionChecker) *StoreReportHandler {
	return &StoreReportHandler{analyticsUC: analyticsUC, storeClient: storeClient, checker: checker}
}

// authorizeStoreOwner verifies the caller holds store.manage on the store's
// vendor. These routes are auth-only, so without this any logged-in user could
// read another store's reports by id. Writes the error response on failure.
func (h *StoreReportHandler) authorizeStoreOwner(c *gin.Context, storeID string) bool {
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

// GetStoreReport returns an aggregated performance summary for a store.
// GET /api/v1/stores/:id/reports?period=day|week|month
func (h *StoreReportHandler) GetStoreReport(c *gin.Context) {
	storeID := c.Param("id")
	if storeID == "" {
		response.BadRequest(c, "store id required")
		return
	}
	if !h.authorizeStoreOwner(c, storeID) {
		return
	}
	period := c.DefaultQuery("period", "month")
	report, err := h.analyticsUC.GetStoreReport(c.Request.Context(), storeID, period)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.Success(c, report)
}

// GetStoreRevenue returns daily revenue rows for a store.
// GET /api/v1/stores/:id/revenue?period=day|week|month
func (h *StoreReportHandler) GetStoreRevenue(c *gin.Context) {
	storeID := c.Param("id")
	if storeID == "" {
		response.BadRequest(c, "store id required")
		return
	}
	if !h.authorizeStoreOwner(c, storeID) {
		return
	}
	period := c.DefaultQuery("period", "month")
	rows, err := h.analyticsUC.GetStoreRevenue(c.Request.Context(), storeID, period)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.Success(c, rows)
}
