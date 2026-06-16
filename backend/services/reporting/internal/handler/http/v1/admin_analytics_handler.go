package v1

import (
	"project/pkg/pagination"
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

// ListRecentOrders returns recent order_facts rows, paginated.
// GET /api/v1/admin/orders/recent?page=1&page_size=20
//
// Each item shape (field names unchanged):
//
//	{"OrderID","Code","StoreID","GrandTotal","Status","CreatedAt"}
func (h *AdminAnalyticsHandler) ListRecentOrders(c *gin.Context) {
	page, pageSize := pagination.Parse(c)
	offset := pagination.Offset(page, pageSize)

	facts, total, err := h.analyticsUC.ListRecentOrders(c.Request.Context(), pageSize, offset)
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
	response.Paginated(c, items, total, page)
}
