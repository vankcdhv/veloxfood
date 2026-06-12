package v1

import (
	"context"
	"net/http"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/order/internal/entity"
	"project/services/order/internal/infrastructure/grpcclient"
	"project/services/order/internal/usecase"

	"github.com/gin-gonic/gin"
)

// StoreOwnershipResolver resolves a store's vendor/owner so the handler can
// authorize owner actions. Satisfied by *grpcclient.StoreClient.
type StoreOwnershipResolver interface {
	GetStoreOwnership(ctx context.Context, storeID string) (*grpcclient.StoreOwnership, error)
}

// OwnerOrderHandler handles order management endpoints for store owners and staff.
type OwnerOrderHandler struct {
	lifecycleUC    usecase.OrderLifecycleUsecase
	userClient     *grpcclient.UserClient
	locationClient *grpcclient.LocationClient
	storeClient    StoreOwnershipResolver
	checker        authmw.PermissionChecker
}

func NewOwnerOrderHandler(
	lifecycleUC usecase.OrderLifecycleUsecase,
	userClient *grpcclient.UserClient,
	locationClient *grpcclient.LocationClient,
	storeClient StoreOwnershipResolver,
	checker authmw.PermissionChecker,
) *OwnerOrderHandler {
	return &OwnerOrderHandler{
		lifecycleUC:    lifecycleUC,
		userClient:     userClient,
		locationClient: locationClient,
		storeClient:    storeClient,
		checker:        checker,
	}
}

// authorizeStoreOwner verifies the authenticated user holds store.manage on the
// store's vendor before any owner action. Without this gate, any authenticated
// user could confirm/reject/advance another store's orders by guessing IDs.
// Writes the error response and returns false on failure.
func (h *OwnerOrderHandler) authorizeStoreOwner(c *gin.Context, storeID string) bool {
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

// ownerOrderView augments an order with the customer's contact info + a readable
// delivery path so the store owner knows who/where to fulfill (never raw UUIDs).
type ownerOrderView struct {
	*entity.Order
	CustomerName  string `json:"CustomerName"`
	CustomerPhone string `json:"CustomerPhone"`
	LocationPath  string `json:"LocationPath"`
}

func (h *OwnerOrderHandler) enrich(c *gin.Context, o *entity.Order) *ownerOrderView {
	ctx := c.Request.Context()
	v := &ownerOrderView{Order: o}
	if h.userClient != nil {
		u := h.userClient.GetUser(ctx, o.CustomerID)
		v.CustomerName, v.CustomerPhone = u.FullName, u.Phone
	}
	if h.locationClient != nil && o.Fulfillment == entity.FulfillmentDelivery && o.LocationID != nil {
		level := ""
		if o.LocationLevel != nil {
			level = *o.LocationLevel
		}
		v.LocationPath = h.locationClient.GetLocationPath(ctx, *o.LocationID, level)
	}
	return v
}

// ListStoreOrders GET /api/v1/stores/:storeId/orders
func (h *OwnerOrderHandler) ListStoreOrders(c *gin.Context) {
	storeID := c.Param("storeId")
	if !h.authorizeStoreOwner(c, storeID) {
		return
	}
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
	views := make([]*ownerOrderView, len(orders))
	for i, o := range orders {
		views[i] = h.enrich(c, o)
	}
	response.Paginated(c, views, total, page)
}

// GetOrder GET /api/v1/stores/:storeId/orders/:id
func (h *OwnerOrderHandler) GetOrder(c *gin.Context) {
	if !h.authorizeStoreOwner(c, c.Param("storeId")) {
		return
	}
	orderID := c.Param("id")
	order, err := h.lifecycleUC.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, h.enrich(c, order))
}

// ConfirmOrder POST /api/v1/stores/:storeId/orders/:id/confirm
func (h *OwnerOrderHandler) ConfirmOrder(c *gin.Context) {
	storeID := c.Param("storeId")
	if !h.authorizeStoreOwner(c, storeID) {
		return
	}
	changedBy := authmw.UserIDFromContext(c.Request.Context())
	orderID := c.Param("id")

	if err := h.lifecycleUC.AdvanceStatus(c.Request.Context(), orderID, storeID, entity.StatusConfirmed, changedBy); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"status": "CONFIRMED"})
}

// RejectOrder POST /api/v1/stores/:storeId/orders/:id/reject
func (h *OwnerOrderHandler) RejectOrder(c *gin.Context) {
	storeID := c.Param("storeId")
	if !h.authorizeStoreOwner(c, storeID) {
		return
	}
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
	storeID := c.Param("storeId")
	if !h.authorizeStoreOwner(c, storeID) {
		return
	}
	changedBy := authmw.UserIDFromContext(c.Request.Context())
	orderID := c.Param("id")

	var body struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.lifecycleUC.AdvanceStatus(c.Request.Context(), orderID, storeID, entity.OrderStatus(body.Status), changedBy); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"status": body.Status})
}

// VerifyPickupPIN POST /api/v1/stores/:storeId/orders/:id/pickup-verify
// Body: {"pin": "1234"}
func (h *OwnerOrderHandler) VerifyPickupPIN(c *gin.Context) {
	storeID := c.Param("storeId")
	if !h.authorizeStoreOwner(c, storeID) {
		return
	}
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
