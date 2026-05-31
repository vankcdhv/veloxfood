package v1

import (
	"encoding/json"
	"net/http"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/order/internal/usecase"

	"github.com/gin-gonic/gin"
)

// CustomerCartHandler handles cart CRUD endpoints for authenticated customers.
type CustomerCartHandler struct {
	cartUC usecase.CartUsecase
}

func NewCustomerCartHandler(cartUC usecase.CartUsecase) *CustomerCartHandler {
	return &CustomerCartHandler{cartUC: cartUC}
}

// GetCart GET /api/v1/me/cart?store_id=<uuid>
func (h *CustomerCartHandler) GetCart(c *gin.Context) {
	customerID := authmw.UserIDFromContext(c.Request.Context())
	storeID := c.Query("store_id")
	if storeID == "" {
		response.BadRequest(c, "store_id query param required")
		return
	}

	cart, err := h.cartUC.GetCart(c.Request.Context(), customerID, storeID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	if cart == nil {
		response.Success(c, gin.H{"cart": nil, "items": []any{}})
		return
	}
	response.Success(c, cart)
}

// AddOrUpdateItem PUT /api/v1/me/cart/items
func (h *CustomerCartHandler) AddOrUpdateItem(c *gin.Context) {
	customerID := authmw.UserIDFromContext(c.Request.Context())

	var body struct {
		StoreID         string          `json:"store_id" binding:"required"`
		MenuItemID      string          `json:"menu_item_id" binding:"required"`
		NameSnapshot    string          `json:"name_snapshot"`
		PriceSnapshot   int64           `json:"price_snapshot"`
		Qty             int             `json:"qty" binding:"required,min=1"`
		CutoffID        string          `json:"cutoff_id"`
		Date            string          `json:"date"`
		OptionsSnapshot json.RawMessage `json:"options_snapshot"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	cart, err := h.cartUC.AddOrUpdateItem(c.Request.Context(), usecase.AddCartItemRequest{
		CustomerID:      customerID,
		StoreID:         body.StoreID,
		MenuItemID:      body.MenuItemID,
		NameSnapshot:    body.NameSnapshot,
		PriceSnapshot:   body.PriceSnapshot,
		Qty:             body.Qty,
		CutoffID:        body.CutoffID,
		Date:            body.Date,
		OptionsSnapshot: body.OptionsSnapshot,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, cart)
}

// RemoveItem DELETE /api/v1/me/cart/items/:menuItemId?store_id=<uuid>
func (h *CustomerCartHandler) RemoveItem(c *gin.Context) {
	customerID := authmw.UserIDFromContext(c.Request.Context())
	menuItemID := c.Param("menuItemId")
	storeID := c.Query("store_id")
	if storeID == "" {
		response.BadRequest(c, "store_id query param required")
		return
	}

	if err := h.cartUC.RemoveItem(c.Request.Context(), customerID, storeID, menuItemID); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ClearCart DELETE /api/v1/me/cart?store_id=<uuid>
func (h *CustomerCartHandler) ClearCart(c *gin.Context) {
	customerID := authmw.UserIDFromContext(c.Request.Context())
	storeID := c.Query("store_id")
	if storeID == "" {
		response.BadRequest(c, "store_id query param required")
		return
	}

	if err := h.cartUC.ClearCart(c.Request.Context(), customerID, storeID); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
