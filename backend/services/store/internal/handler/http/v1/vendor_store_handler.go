package v1

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/store/internal/usecase"

	"github.com/gin-gonic/gin"
)

// VendorStoreHandler handles vendor-owner operations on their own store.
// Every mutating route checks that the authenticated user holds store.manage
// permission scoped to the store's vendor_id.
type VendorStoreHandler struct {
	storeUC   usecase.StoreUsecase
	catalogUC usecase.CatalogUsecase
	shipFeeUC usecase.ShipFeeUsecase
	hoursUC   usecase.HoursUsecase
	checker   authmw.PermissionChecker
	uploader  usecase.FileUploader
}

func NewVendorStoreHandler(
	storeUC usecase.StoreUsecase,
	catalogUC usecase.CatalogUsecase,
	shipFeeUC usecase.ShipFeeUsecase,
	hoursUC usecase.HoursUsecase,
	checker authmw.PermissionChecker,
	uploader usecase.FileUploader,
) *VendorStoreHandler {
	return &VendorStoreHandler{
		storeUC:   storeUC,
		catalogUC: catalogUC,
		shipFeeUC: shipFeeUC,
		hoursUC:   hoursUC,
		checker:   checker,
		uploader:  uploader,
	}
}

// authorizeStoreOwner loads the store and checks store.manage permission scoped to
// its vendor_id. Returns (vendorID, true) on success; writes error response and
// returns ("", false) on failure.
func (h *VendorStoreHandler) authorizeStoreOwner(c *gin.Context, storeID string) (string, bool) {
	ctx := c.Request.Context()
	store, err := h.storeUC.GetStore(ctx, storeID)
	if err != nil {
		response.HandleError(c, err)
		return "", false
	}

	userID := authmw.UserIDFromContext(ctx)
	ok, err := h.checker.HasVendorPermission(ctx, userID, store.VendorID, "store.manage")
	if err != nil {
		response.InternalError(c)
		return "", false
	}
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions for this store"})
		return "", false
	}
	return store.VendorID, true
}

// ListShipFeeRules GET /stores/:id/ship-fees — vendor lists its own fee rules.
func (h *VendorStoreHandler) ListShipFeeRules(c *gin.Context) {
	storeID := c.Param("id")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	rules, err := h.shipFeeUC.ListShipFeeRules(c.Request.Context(), storeID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, rules)
}

// ListHoursChange GET /stores/:id/hours-change?status= — vendor sees the status
// of its own operating-hours change requests.
func (h *VendorStoreHandler) ListHoursChange(c *gin.Context) {
	storeID := c.Param("id")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	reqs, err := h.hoursUC.ListChangeRequests(c.Request.Context(), storeID, c.Query("status"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, reqs)
}

// ─── Store settings ───────────────────────────────────────────────────────────

// UpdateStore PATCH /stores/:id
func (h *VendorStoreHandler) UpdateStore(c *gin.Context) {
	storeID := c.Param("id")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	var body struct {
		Name         string `json:"name"`
		BusinessType string `json:"business_type"`
		Address      string `json:"address"`
		Phone        string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	store, err := h.storeUC.UpdateStore(c.Request.Context(), storeID, body.Name, body.BusinessType, body.Address, body.Phone)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, store)
}

// UpdateSaleStatus PATCH /stores/:id/sale-status
func (h *VendorStoreHandler) UpdateSaleStatus(c *gin.Context) {
	storeID := c.Param("id")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	var body struct {
		SaleStatus string `json:"sale_status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.storeUC.UpdateSaleStatus(c.Request.Context(), storeID, body.SaleStatus); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// UpdatePickup PATCH /stores/:id/pickup
func (h *VendorStoreHandler) UpdatePickup(c *gin.Context) {
	storeID := c.Param("id")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.storeUC.UpdatePickup(c.Request.Context(), storeID, body.Enabled); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ─── Categories ───────────────────────────────────────────────────────────────

// CreateCategory POST /stores/:id/categories
func (h *VendorStoreHandler) CreateCategory(c *gin.Context) {
	storeID := c.Param("id")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	var body struct {
		Name      string `json:"name" binding:"required"`
		SortOrder int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	cat, err := h.catalogUC.CreateCategory(c.Request.Context(), storeID, body.Name, body.SortOrder)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, cat)
}

// UpdateCategory PATCH /stores/:id/categories/:catId
func (h *VendorStoreHandler) UpdateCategory(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	var body struct {
		Name      string `json:"name" binding:"required"`
		SortOrder int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	cat, err := h.catalogUC.UpdateCategory(c.Request.Context(), c.Param("catId"), body.Name, body.SortOrder)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, cat)
}

// DeleteCategory DELETE /stores/:id/categories/:catId
func (h *VendorStoreHandler) DeleteCategory(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	if err := h.catalogUC.DeleteCategory(c.Request.Context(), c.Param("catId")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ─── Menu items ───────────────────────────────────────────────────────────────

// CreateMenuItem POST /stores/:id/menu
func (h *VendorStoreHandler) CreateMenuItem(c *gin.Context) {
	storeID := c.Param("id")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	var body struct {
		CategoryID  string `json:"category_id"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Price       int64  `json:"price" binding:"required"`
		Tags        string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	item, err := h.catalogUC.CreateMenuItem(c.Request.Context(), storeID, body.CategoryID, body.Name, body.Description, body.Price, body.Tags)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, item)
}

// UpdateMenuItem PATCH /stores/:id/menu/:itemId
func (h *VendorStoreHandler) UpdateMenuItem(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	var body struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		Price       int64   `json:"price" binding:"required"`
		Tags        string  `json:"tags"`
		ImageURL    *string `json:"image_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	item, err := h.catalogUC.UpdateMenuItem(c.Request.Context(), c.Param("itemId"), body.Name, body.Description, body.Price, body.Tags, body.ImageURL)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, item)
}

// ToggleMenuItemStatus PATCH /stores/:id/menu/:itemId/status
func (h *VendorStoreHandler) ToggleMenuItemStatus(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	var body struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.catalogUC.ToggleMenuItemStatus(c.Request.Context(), c.Param("itemId"), body.Status); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// UploadMenuItemImage POST /stores/:id/menu/:itemId/image (multipart)
func (h *VendorStoreHandler) UploadMenuItemImage(c *gin.Context) {
	storeID := c.Param("id")
	itemID := c.Param("itemId")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		response.BadRequest(c, "image file required")
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	objectKey := fmt.Sprintf("stores/%s/menu/%s%s", storeID, itemID, ext)
	url, err := h.uploader.Put(c.Request.Context(), objectKey, header.Header.Get("Content-Type"), file.(io.Reader), header.Size)
	if err != nil {
		response.InternalError(c)
		return
	}

	if err := h.catalogUC.SetMenuItemImage(c.Request.Context(), itemID, url); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"image_url": url})
}

// DeleteMenuItem DELETE /stores/:id/menu/:itemId
func (h *VendorStoreHandler) DeleteMenuItem(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	if err := h.catalogUC.DeleteMenuItem(c.Request.Context(), c.Param("itemId")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ─── Option groups ────────────────────────────────────────────────────────────

// ListOptionGroups GET /stores/:id/option-groups
func (h *VendorStoreHandler) ListOptionGroups(c *gin.Context) {
	storeID := c.Param("id")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	groups, err := h.catalogUC.ListOptionGroups(c.Request.Context(), storeID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, groups)
}

// CreateOptionGroup POST /stores/:id/option-groups
func (h *VendorStoreHandler) CreateOptionGroup(c *gin.Context) {
	storeID := c.Param("id")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	var body struct {
		Name      string `json:"name" binding:"required"`
		MinSelect int    `json:"min_select"`
		MaxSelect int    `json:"max_select"`
		Required  bool   `json:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	og, err := h.catalogUC.CreateOptionGroup(c.Request.Context(), storeID, body.Name, body.MinSelect, body.MaxSelect, body.Required)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, og)
}

// UpdateOptionGroup PATCH /stores/:id/option-groups/:ogId
func (h *VendorStoreHandler) UpdateOptionGroup(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	var body struct {
		Name      string `json:"name" binding:"required"`
		MinSelect int    `json:"min_select"`
		MaxSelect int    `json:"max_select"`
		Required  bool   `json:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	og, err := h.catalogUC.UpdateOptionGroup(c.Request.Context(), c.Param("ogId"), body.Name, body.MinSelect, body.MaxSelect, body.Required)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, og)
}

// DeleteOptionGroup DELETE /stores/:id/option-groups/:ogId
func (h *VendorStoreHandler) DeleteOptionGroup(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	if err := h.catalogUC.DeleteOptionGroup(c.Request.Context(), c.Param("ogId")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ─── Options ─────────────────────────────────────────────────────────────────

// CreateOption POST /stores/:id/option-groups/:ogId/options
func (h *VendorStoreHandler) CreateOption(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	var body struct {
		Name       string `json:"name" binding:"required"`
		ExtraPrice int64  `json:"extra_price"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	opt, err := h.catalogUC.CreateOption(c.Request.Context(), c.Param("ogId"), body.Name, body.ExtraPrice)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, opt)
}

// ListOptions GET /stores/:id/option-groups/:ogId/options
func (h *VendorStoreHandler) ListOptions(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	opts, err := h.catalogUC.ListOptions(c.Request.Context(), c.Param("ogId"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, opts)
}

// UpdateOption PATCH /stores/:id/option-groups/:ogId/options/:optId
func (h *VendorStoreHandler) UpdateOption(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	var body struct {
		Name       string `json:"name" binding:"required"`
		ExtraPrice int64  `json:"extra_price"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	opt, err := h.catalogUC.UpdateOption(c.Request.Context(), c.Param("optId"), body.Name, body.ExtraPrice)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, opt)
}

// DeleteOption DELETE /stores/:id/option-groups/:ogId/options/:optId
func (h *VendorStoreHandler) DeleteOption(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	if err := h.catalogUC.DeleteOption(c.Request.Context(), c.Param("optId")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// AttachOptionGroup POST /stores/:id/menu/:itemId/option-groups
func (h *VendorStoreHandler) AttachOptionGroup(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	var body struct {
		OptionGroupID string `json:"option_group_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.catalogUC.AttachOptionGroup(c.Request.Context(), c.Param("itemId"), body.OptionGroupID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// DetachOptionGroup DELETE /stores/:id/menu/:itemId/option-groups/:ogId
func (h *VendorStoreHandler) DetachOptionGroup(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	if err := h.catalogUC.DetachOptionGroup(c.Request.Context(), c.Param("itemId"), c.Param("ogId")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ListOptionGroupsForItem GET /stores/:id/menu/:itemId/option-groups
func (h *VendorStoreHandler) ListOptionGroupsForItem(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	groups, err := h.catalogUC.ListOptionGroupsForItem(c.Request.Context(), c.Param("itemId"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, groups)
}

// ─── Combos ───────────────────────────────────────────────────────────────────

// ListCombos GET /stores/:id/combos
func (h *VendorStoreHandler) ListCombos(c *gin.Context) {
	storeID := c.Param("id")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	combos, err := h.catalogUC.ListCombos(c.Request.Context(), storeID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, combos)
}

// CreateCombo POST /stores/:id/combos
func (h *VendorStoreHandler) CreateCombo(c *gin.Context) {
	storeID := c.Param("id")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	var body struct {
		Name  string `json:"name" binding:"required"`
		Price int64  `json:"price" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	combo, err := h.catalogUC.CreateCombo(c.Request.Context(), storeID, body.Name, body.Price)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, combo)
}

// UpdateCombo PATCH /stores/:id/combos/:comboId
func (h *VendorStoreHandler) UpdateCombo(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	var body struct {
		Name  string `json:"name" binding:"required"`
		Price int64  `json:"price" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	combo, err := h.catalogUC.UpdateCombo(c.Request.Context(), c.Param("comboId"), body.Name, body.Price)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, combo)
}

// DeleteCombo DELETE /stores/:id/combos/:comboId
func (h *VendorStoreHandler) DeleteCombo(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	if err := h.catalogUC.DeleteCombo(c.Request.Context(), c.Param("comboId")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// AddComboItem POST /stores/:id/combos/:comboId/items
func (h *VendorStoreHandler) AddComboItem(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	var body struct {
		MenuItemID string `json:"menu_item_id" binding:"required"`
		Quantity   int    `json:"quantity" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.catalogUC.AddComboItem(c.Request.Context(), c.Param("comboId"), body.MenuItemID, body.Quantity); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, gin.H{"ok": true})
}

// ListComboItems GET /stores/:id/combos/:comboId/items
func (h *VendorStoreHandler) ListComboItems(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	items, err := h.catalogUC.ListComboItems(c.Request.Context(), c.Param("comboId"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// RemoveComboItem DELETE /stores/:id/combos/:comboId/items/:menuItemId
func (h *VendorStoreHandler) RemoveComboItem(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	if err := h.catalogUC.RemoveComboItem(c.Request.Context(), c.Param("comboId"), c.Param("menuItemId")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ─── Ship fee rules ───────────────────────────────────────────────────────────

// CreateShipFeeRule POST /stores/:id/ship-fees
// scope: "building" | "floor" | "room". unit_fee >= 0 (0 = free delivery).
// When scope is "floor" or "room", a building-scope rule must already exist for
// the ancestor building; the usecase returns ErrBuildingRuleRequired otherwise.
func (h *VendorStoreHandler) CreateShipFeeRule(c *gin.Context) {
	storeID := c.Param("id")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	var body struct {
		Scope   string `json:"scope" binding:"required"`
		RefID   string `json:"ref_id" binding:"required"`
		// unit_fee is NOT marked binding:"required" so that 0 (free) is accepted.
		UnitFee int64 `json:"unit_fee"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	rule, err := h.shipFeeUC.CreateShipFeeRule(c.Request.Context(), storeID, body.Scope, body.RefID, body.UnitFee)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, rule)
}

// UpdateShipFeeRule PATCH /stores/:id/ship-fees/:ruleId
func (h *VendorStoreHandler) UpdateShipFeeRule(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	var body struct {
		UnitFee int64 `json:"unit_fee" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	rule, err := h.shipFeeUC.UpdateShipFeeRule(c.Request.Context(), c.Param("ruleId"), body.UnitFee)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, rule)
}

// DeleteShipFeeRule DELETE /stores/:id/ship-fees/:ruleId
func (h *VendorStoreHandler) DeleteShipFeeRule(c *gin.Context) {
	if _, ok := h.authorizeStoreOwner(c, c.Param("id")); !ok {
		return
	}
	if err := h.shipFeeUC.DeleteShipFeeRule(c.Request.Context(), c.Param("ruleId")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ─── Slot quota ───────────────────────────────────────────────────────────────

// SetSlotQuota POST /stores/:id/menu/:itemId/quota
// Sets (upserts) the per-(item × cutoff × date) slot quota.
// Vendor must hold store.manage permission scoped to this store's vendor_id.
func (h *VendorStoreHandler) SetSlotQuota(c *gin.Context) {
	storeID := c.Param("id")
	itemID := c.Param("itemId")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	var body struct {
		Date     string `json:"date" binding:"required"`
		CutoffID string `json:"cutoff_id" binding:"required"`
		Quota    int    `json:"quota" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	date, err := time.Parse("2006-01-02", body.Date)
	if err != nil {
		response.BadRequest(c, "date must be YYYY-MM-DD")
		return
	}
	if err := h.catalogUC.SetSlotQuota(c.Request.Context(), itemID, date, body.CutoffID, body.Quota); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ListSlotQuotas GET /stores/:id/menu/:itemId/quota?date=YYYY-MM-DD
// Returns all per-cutoff quota rows for the item on the given date (defaults to today UTC).
func (h *VendorStoreHandler) ListSlotQuotas(c *gin.Context) {
	storeID := c.Param("id")
	itemID := c.Param("itemId")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	var date time.Time
	if dateStr := c.Query("date"); dateStr == "" {
		date = time.Now().UTC().Truncate(24 * time.Hour)
	} else {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			response.BadRequest(c, "date must be YYYY-MM-DD")
			return
		}
	}
	quotas, err := h.catalogUC.ListSlotQuotas(c.Request.Context(), itemID, date)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, quotas)
}

// ─── Hours change requests ────────────────────────────────────────────────────

// SubmitHoursChange POST /stores/:id/hours-change
func (h *VendorStoreHandler) SubmitHoursChange(c *gin.Context) {
	storeID := c.Param("id")
	if _, ok := h.authorizeStoreOwner(c, storeID); !ok {
		return
	}
	// The client posts { "payload": { operating_hours, ship_cutoffs, note } };
	// store the inner object so approval can read operating_hours/ship_cutoffs
	// at the top level (not double-wrapped under "payload").
	var body struct {
		Payload map[string]any `json:"payload"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if body.Payload == nil {
		response.BadRequest(c, "payload is required")
		return
	}
	req, err := h.hoursUC.SubmitHoursChange(c.Request.Context(), storeID, body.Payload)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, req)
}
