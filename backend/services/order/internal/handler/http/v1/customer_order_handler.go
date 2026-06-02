package v1

import (
	"encoding/json"
	"fmt"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/order/internal/entity"
	"project/services/order/internal/usecase"

	"github.com/gin-gonic/gin"
)

// CustomerOrderHandler handles order placement and customer-facing order management.
type CustomerOrderHandler struct {
	placeOrderUC usecase.PlaceOrderUsecase
	lifecycleUC  usecase.OrderLifecycleUsecase
}

func NewCustomerOrderHandler(
	placeOrderUC usecase.PlaceOrderUsecase,
	lifecycleUC usecase.OrderLifecycleUsecase,
) *CustomerOrderHandler {
	return &CustomerOrderHandler{placeOrderUC: placeOrderUC, lifecycleUC: lifecycleUC}
}

// PlaceOrder POST /api/v1/orders
func (h *CustomerOrderHandler) PlaceOrder(c *gin.Context) {
	customerID := authmw.UserIDFromContext(c.Request.Context())

	var body struct {
		StoreID string `json:"store_id" binding:"required"`
		LocationID string `json:"location_id"`
		// LocationLevel is required for DELIVERY and must be one of BUILDING, FLOOR, ROOM.
		// Absent / empty is treated as ROOM for back-compat.
		LocationLevel string              `json:"location_level" binding:"omitempty,oneof=BUILDING FLOOR ROOM"`
		Fulfillment   string              `json:"fulfillment" binding:"required,oneof=DELIVERY PICKUP"`
		PaymentMethod string              `json:"payment_method" binding:"required,oneof=COD MOMO WALLET"`
		VoucherCodes  []string            `json:"voucher_codes"`
		Items         []placeOrderItemBody `json:"items" binding:"required,min=1"`
		// DesiredTime is the customer's preferred receive time (RFC3339). Empty = ASAP.
		DesiredTime string `json:"desired_time"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	items := make([]usecase.PlaceOrderItem, len(body.Items))
	for i, it := range body.Items {
		items[i] = usecase.PlaceOrderItem{
			MenuItemID:      it.MenuItemID,
			Qty:             it.Qty,
			OptionsSnapshot: it.OptionsSnapshot,
		}
	}

	result, err := h.placeOrderUC.PlaceOrder(c.Request.Context(), usecase.PlaceOrderRequest{
		CustomerID:    customerID,
		StoreID:       body.StoreID,
		LocationID:    body.LocationID,
		LocationLevel: body.LocationLevel,
		Fulfillment:   entity.Fulfillment(body.Fulfillment),
		PaymentMethod: entity.PaymentMethod(body.PaymentMethod),
		VoucherCodes:  body.VoucherCodes,
		Items:         items,
		DesiredTime:   body.DesiredTime,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Created(c, gin.H{
		"order_id":    result.OrderID,
		"code":        result.Code,
		"grand_total": result.GrandTotal,
		"pay_url":     result.PayURL,
	})
}

// placeOrderItemBody is the request shape for a single item when placing an order.
type placeOrderItemBody struct {
	MenuItemID      string          `json:"menu_item_id" binding:"required"`
	Qty             int             `json:"qty" binding:"required,min=1"`
	OptionsSnapshot json.RawMessage `json:"options_snapshot"`
}

// ListMyOrders GET /api/v1/orders
func (h *CustomerOrderHandler) ListMyOrders(c *gin.Context) {
	customerID := authmw.UserIDFromContext(c.Request.Context())
	page, pageSize := parsePagination(c)

	orders, total, err := h.lifecycleUC.ListCustomerOrders(c.Request.Context(), customerID, page, pageSize)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Paginated(c, orders, total, page)
}

// GetMyOrder GET /api/v1/orders/:id
func (h *CustomerOrderHandler) GetMyOrder(c *gin.Context) {
	customerID := authmw.UserIDFromContext(c.Request.Context())
	orderID := c.Param("id")

	order, err := h.lifecycleUC.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	if order.CustomerID != customerID {
		response.NotFound(c, "order not found")
		return
	}
	response.Success(c, order)
}

// CancelOrder POST /api/v1/orders/:id/cancel
func (h *CustomerOrderHandler) CancelOrder(c *gin.Context) {
	customerID := authmw.UserIDFromContext(c.Request.Context())
	orderID := c.Param("id")

	if err := h.lifecycleUC.CancelByCustomer(c.Request.Context(), orderID, customerID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"cancelled": true})
}

// Reorder POST /api/v1/orders/:id/reorder
func (h *CustomerOrderHandler) Reorder(c *gin.Context) {
	customerID := authmw.UserIDFromContext(c.Request.Context())
	orderID := c.Param("id")

	cart, err := h.lifecycleUC.Reorder(c.Request.Context(), orderID, customerID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, cart)
}

// parsePagination extracts page/page_size from query params with safe defaults.
func parsePagination(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20
	if p := c.Query("page"); p != "" {
		if n, err := parseInt(p); err == nil && n > 0 {
			page = n
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if n, err := parseInt(ps); err == nil && n > 0 && n <= 100 {
			pageSize = n
		}
	}
	return
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
