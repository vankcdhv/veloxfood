package v1

import (
	"net/http"
	"strconv"

	"project/pkg/response"
	"project/services/reporting/internal/usecase"

	"github.com/gin-gonic/gin"
)

// AdminAnalyticsHandler serves platform-wide analytics to admins.
type AdminAnalyticsHandler struct {
	analyticsUC *usecase.AnalyticsUsecase
}

func NewAdminAnalyticsHandler(analyticsUC *usecase.AnalyticsUsecase) *AdminAnalyticsHandler {
	return &AdminAnalyticsHandler{analyticsUC: analyticsUC}
}

// GetAnalytics returns aggregated platform metrics for a period.
// GET /api/v1/admin/analytics?period=day|week|month
//
// Response shape (PascalCase — consumed by admin/page.tsx):
//
//	{
//	  "OrdersTotal":    42,
//	  "RevenueTotal":   6720000,
//	  "NewUsersTotal":  5,
//	  "NewVendors":     1,
//	  "CancelledTotal": 3,
//	  "AvgOrderValue":  160000.0
//	}
func (h *AdminAnalyticsHandler) GetAnalytics(c *gin.Context) {
	period := c.DefaultQuery("period", "day")
	summary, err := h.analyticsUC.GetAnalytics(c.Request.Context(), period)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.Success(c, summary)
}

// ListRecentOrders returns the most recent order_facts rows.
// GET /api/v1/admin/orders/recent?limit=20
//
// Each item in the array:
//
//	{
//	  "OrderID":       "uuid",
//	  "StoreID":       "uuid",
//	  "GrandTotal":    160000,
//	  "Status":        "COMPLETED",
//	  "CreatedAt":     "RFC3339"
//	}
func (h *AdminAnalyticsHandler) ListRecentOrders(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	facts, err := h.analyticsUC.ListRecentOrders(c.Request.Context(), limit)
	if err != nil {
		response.InternalError(c)
		return
	}

	// Project to a lean response shape for the FE.
	type orderItem struct {
		OrderID    string `json:"OrderID"`
		Code       string `json:"Code"`
		StoreID    string `json:"StoreID"`
		GrandTotal int64  `json:"GrandTotal"`
		Status     string `json:"Status"`
		CreatedAt  string `json:"CreatedAt"`
	}
	items := make([]orderItem, 0, len(facts))
	for _, f := range facts {
		items = append(items, orderItem{
			OrderID:    f.OrderID,
			Code:       f.Code,
			StoreID:    f.StoreID,
			GrandTotal: f.GrandTotal,
			Status:     f.Status,
			CreatedAt:  f.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": 200, "message": "success", "data": items})
}
