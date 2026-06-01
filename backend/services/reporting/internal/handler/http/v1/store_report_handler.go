package v1

import (
	"project/pkg/response"
	"project/services/reporting/internal/usecase"

	"github.com/gin-gonic/gin"
)

// StoreReportHandler serves per-store analytics to authenticated store owners.
type StoreReportHandler struct {
	analyticsUC *usecase.AnalyticsUsecase
}

func NewStoreReportHandler(analyticsUC *usecase.AnalyticsUsecase) *StoreReportHandler {
	return &StoreReportHandler{analyticsUC: analyticsUC}
}

// GetStoreReport returns an aggregated performance summary for a store.
// GET /api/v1/stores/:id/reports?period=day|week|month
func (h *StoreReportHandler) GetStoreReport(c *gin.Context) {
	storeID := c.Param("id")
	if storeID == "" {
		response.BadRequest(c, "store id required")
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
	period := c.DefaultQuery("period", "month")
	rows, err := h.analyticsUC.GetStoreRevenue(c.Request.Context(), storeID, period)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.Success(c, rows)
}
