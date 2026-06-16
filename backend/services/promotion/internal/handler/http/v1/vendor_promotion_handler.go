package v1

import (
	"net/http"
	"time"

	authmw "project/pkg/auth/middleware"
	"project/pkg/pagination"
	"project/pkg/response"
	"project/services/promotion/internal/usecase"

	"github.com/gin-gonic/gin"
)

// VendorPromotionHandler handles store-owner operations on promotions.
// Every mutating route verifies store ownership via the Store gRPC service
// then checks store.manage permission scoped to the store's vendor_id.
type VendorPromotionHandler struct {
	promoUC usecase.PromotionUsecase
	checker authmw.PermissionChecker
}

func NewVendorPromotionHandler(
	promoUC usecase.PromotionUsecase,
	checker authmw.PermissionChecker,
) *VendorPromotionHandler {
	return &VendorPromotionHandler{promoUC: promoUC, checker: checker}
}

// authorizeStoreOwner calls the Store service to resolve vendor_id, then checks
// store.manage permission. Returns (vendorID, true) on success; writes the error
// response and returns ("", false) on failure.
func (h *VendorPromotionHandler) authorizeStoreOwner(c *gin.Context, storeID string) (string, bool) {
	ctx := c.Request.Context()

	ownership, err := h.promoUC.GetStoreOwnership(ctx, storeID)
	if err != nil {
		response.InternalError(c)
		return "", false
	}
	if !ownership.Found {
		response.NotFound(c, "store not found")
		return "", false
	}

	userID := authmw.UserIDFromContext(ctx)
	ok, err := h.checker.HasVendorPermission(ctx, userID, ownership.VendorID, "store.manage")
	if err != nil {
		response.InternalError(c)
		return "", false
	}
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions for this store"})
		return "", false
	}
	return ownership.VendorID, true
}

// ListPromotions GET /stores/:storeId/promotions?page=1&page_size=20
func (h *VendorPromotionHandler) ListPromotions(c *gin.Context) {
	storeID := c.Param("storeId")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	page, pageSize := pagination.Parse(c)
	offset := pagination.Offset(page, pageSize)
	promotions, total, err := h.promoUC.ListPromotions(c.Request.Context(), storeID, pageSize, offset)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Paginated(c, promotions, total, page)
}

// CreatePromotion POST /stores/:storeId/promotions
func (h *VendorPromotionHandler) CreatePromotion(c *gin.Context) {
	storeID := c.Param("storeId")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}

	// Request bodies are snake_case (repo convention for inbound DTOs); the
	// response entity stays PascalCase.
	var body struct {
		Code        string `json:"code"         binding:"required"`
		Type        string `json:"type"         binding:"required"`
		ValueKind   string `json:"value_kind"   binding:"required"`
		Value       int64  `json:"value"        binding:"required,min=0"`
		MinOrder    int64  `json:"min_order"`
		MaxDiscount *int64 `json:"max_discount"`
		StartsAt    string `json:"starts_at"    binding:"required"`
		EndsAt      string `json:"ends_at"      binding:"required"`
		UsageLimit  *int   `json:"usage_limit"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	startsAt, err := time.Parse(time.RFC3339, body.StartsAt)
	if err != nil {
		response.BadRequest(c, "starts_at must be RFC3339")
		return
	}
	endsAt, err := time.Parse(time.RFC3339, body.EndsAt)
	if err != nil {
		response.BadRequest(c, "ends_at must be RFC3339")
		return
	}

	p, err := h.promoUC.CreatePromotion(c.Request.Context(), usecase.CreatePromotionRequest{
		StoreID:     storeID,
		Code:        body.Code,
		Type:        body.Type,
		ValueKind:   body.ValueKind,
		Value:       body.Value,
		MinOrder:    body.MinOrder,
		MaxDiscount: body.MaxDiscount,
		StartsAt:    startsAt,
		EndsAt:      endsAt,
		UsageLimit:  body.UsageLimit,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, p)
}

// UpdatePromotion PATCH /stores/:storeId/promotions/:promoId
// Code is immutable; all other fields are optional.
func (h *VendorPromotionHandler) UpdatePromotion(c *gin.Context) {
	storeID := c.Param("storeId")
	promoID := c.Param("promoId")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}

	var body struct {
		Value       int64  `json:"value"`
		MinOrder    int64  `json:"min_order"`
		MaxDiscount *int64 `json:"max_discount"`
		StartsAt    string `json:"starts_at"`
		EndsAt      string `json:"ends_at"`
		UsageLimit  *int   `json:"usage_limit"`
		Status      string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	req := usecase.UpdatePromotionRequest{
		Value:       body.Value,
		MinOrder:    body.MinOrder,
		MaxDiscount: body.MaxDiscount,
		UsageLimit:  body.UsageLimit,
		Status:      body.Status,
	}
	if body.StartsAt != "" {
		t, err := time.Parse(time.RFC3339, body.StartsAt)
		if err != nil {
			response.BadRequest(c, "starts_at must be RFC3339")
			return
		}
		req.StartsAt = t
	}
	if body.EndsAt != "" {
		t, err := time.Parse(time.RFC3339, body.EndsAt)
		if err != nil {
			response.BadRequest(c, "ends_at must be RFC3339")
			return
		}
		req.EndsAt = t
	}

	p, err := h.promoUC.UpdatePromotion(c.Request.Context(), promoID, storeID, req)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, p)
}

// DeletePromotion DELETE /stores/:storeId/promotions/:promoId
func (h *VendorPromotionHandler) DeletePromotion(c *gin.Context) {
	storeID := c.Param("storeId")
	promoID := c.Param("promoId")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	if err := h.promoUC.DeletePromotion(c.Request.Context(), promoID, storeID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
