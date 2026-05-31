package v1

import (
	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/order/internal/entity"
	"project/services/order/internal/usecase"

	"github.com/gin-gonic/gin"
)

// OwnerOrderHandler handles order management endpoints for store owners and staff.
type OwnerOrderHandler struct {
	lifecycleUC usecase.OrderLifecycleUsecase
}

func NewOwnerOrderHandler(lifecycleUC usecase.OrderLifecycleUsecase) *OwnerOrderHandler {
	return &OwnerOrderHandler{lifecycleUC: lifecycleUC}
}

// ListStoreOrders GET /api/v1/stores/:storeId/orders
func (h *OwnerOrderHandler) ListStoreOrders(c *gin.Context) {
	storeID := c.Param("storeId")
	page, pageSize := parsePagination(c)

	var status entity.OrderStatus
	if s := c.Query("status"); s != "" {
		status = entity.OrderStatus(s)
	}

	orders, total, err := h.lifecycleUC.ListStoreOrders(c.Request.Context(), storeID, status, page, pageSize)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Paginated(c, orders, total, page)
}

// GetOrder GET /api/v1/stores/:storeId/orders/:id
func (h *OwnerOrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	order, err := h.lifecycleUC.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, order)
}

// ConfirmOrder POST /api/v1/stores/:storeId/orders/:id/confirm
func (h *OwnerOrderHandler) ConfirmOrder(c *gin.Context) {
	changedBy := authmw.UserIDFromContext(c.Request.Context())
	orderID := c.Param("id")

	if err := h.lifecycleUC.AdvanceStatus(c.Request.Context(), orderID, entity.StatusConfirmed, changedBy); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"status": "CONFIRMED"})
}

// RejectOrder POST /api/v1/stores/:storeId/orders/:id/reject
func (h *OwnerOrderHandler) RejectOrder(c *gin.Context) {
	storeID := c.Param("storeId")
	orderID := c.Param("id")

	if err := h.lifecycleUC.RejectByStore(c.Request.Context(), orderID, storeID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"status": "REJECTED"})
}

// AdvanceStatus PATCH /api/v1/stores/:storeId/orders/:id/status
// Body: {"status": "PREPARING"|"READY"|"DELIVERING"|"DELIVERED"}
func (h *OwnerOrderHandler) AdvanceStatus(c *gin.Context) {
	changedBy := authmw.UserIDFromContext(c.Request.Context())
	orderID := c.Param("id")

	var body struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.lifecycleUC.AdvanceStatus(c.Request.Context(), orderID, entity.OrderStatus(body.Status), changedBy); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"status": body.Status})
}

// VerifyPickupPIN POST /api/v1/stores/:storeId/orders/:id/pickup-verify
// Body: {"pin": "1234"}
func (h *OwnerOrderHandler) VerifyPickupPIN(c *gin.Context) {
	storeID := c.Param("storeId")
	orderID := c.Param("id")

	var body struct {
		PIN string `json:"pin" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.lifecycleUC.VerifyPickupPIN(c.Request.Context(), orderID, storeID, body.PIN); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"completed": true})
}
