package v1

import (
	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/store/internal/usecase"

	"github.com/gin-gonic/gin"
)

// AdminStoreHandler handles global-admin operations on the store service.
// Routes are protected by the store.approve global permission (wired in router).
type AdminStoreHandler struct {
	storeUC usecase.StoreUsecase
	hoursUC usecase.HoursUsecase
}

func NewAdminStoreHandler(storeUC usecase.StoreUsecase, hoursUC usecase.HoursUsecase) *AdminStoreHandler {
	return &AdminStoreHandler{storeUC: storeUC, hoursUC: hoursUC}
}

// ListStores GET /admin/stores?sale_status=&q=&limit=20&offset=0 — admin overview
// of every store (includes OwnerUserName), paginated.
func (h *AdminStoreHandler) ListStores(c *gin.Context) {
	limit, offset := parseLimitOffset(c, 20)
	stores, total, err := h.storeUC.ListStoresEnriched(c.Request.Context(), c.Query("sale_status"), c.Query("q"), limit, offset)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Paginated(c, stores, total, offset/limit+1)
}

// ListHoursChange GET /admin/stores/:id/hours-change?status= — pending requests to review.
func (h *AdminStoreHandler) ListHoursChange(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		status = "pending"
	}
	reqs, err := h.hoursUC.ListChangeRequests(c.Request.Context(), c.Param("id"), status)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, reqs)
}

// CreateStore POST /admin/stores — manual creation by admin (e.g. onboarding bypass).
func (h *AdminStoreHandler) CreateStore(c *gin.Context) {
	var body struct {
		VendorID    string `json:"vendor_id" binding:"required"`
		OwnerUserID string `json:"owner_user_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	store, err := h.storeUC.CreateStore(c.Request.Context(), body.VendorID, body.OwnerUserID, body.Name)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, store)
}

// UpdateStore PATCH /admin/stores/:id — admin overrides any store field.
func (h *AdminStoreHandler) UpdateStore(c *gin.Context) {
	var body struct {
		Name         string `json:"name" binding:"required"`
		BusinessType string `json:"business_type"`
		Address      string `json:"address"`
		Phone        string `json:"phone"`
		PrepMinutes  int    `json:"prep_minutes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	store, err := h.storeUC.UpdateStore(c.Request.Context(), c.Param("id"), body.Name, body.BusinessType, body.Address, body.Phone, body.PrepMinutes)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, store)
}

// ApproveHoursChange POST /admin/stores/:id/hours-change/:req/approve
func (h *AdminStoreHandler) ApproveHoursChange(c *gin.Context) {
	ctx := c.Request.Context()
	adminID := authmw.UserIDFromContext(ctx)
	if err := h.hoursUC.ApproveHoursChange(ctx, c.Param("req"), adminID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// RejectHoursChange POST /admin/stores/:id/hours-change/:req/reject
func (h *AdminStoreHandler) RejectHoursChange(c *gin.Context) {
	ctx := c.Request.Context()
	adminID := authmw.UserIDFromContext(ctx)
	if err := h.hoursUC.RejectHoursChange(ctx, c.Param("req"), adminID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
