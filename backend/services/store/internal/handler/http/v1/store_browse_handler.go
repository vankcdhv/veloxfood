package v1

import (
	"net/http"
	"time"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/store/internal/usecase"

	"github.com/gin-gonic/gin"
)

// StoreBrowseHandler handles public (any-auth) store browsing endpoints.
type StoreBrowseHandler struct {
	storeUC   usecase.StoreUsecase
	catalogUC usecase.CatalogUsecase
	shipFeeUC usecase.ShipFeeUsecase
	hoursUC   usecase.HoursUsecase
}

func NewStoreBrowseHandler(
	storeUC usecase.StoreUsecase,
	catalogUC usecase.CatalogUsecase,
	shipFeeUC usecase.ShipFeeUsecase,
	hoursUC usecase.HoursUsecase,
) *StoreBrowseHandler {
	return &StoreBrowseHandler{
		storeUC:   storeUC,
		catalogUC: catalogUC,
		shipFeeUC: shipFeeUC,
		hoursUC:   hoursUC,
	}
}

// ListStores GET /api/v1/stores
func (h *StoreBrowseHandler) ListStores(c *gin.Context) {
	stores, err := h.storeUC.ListStoresEnriched(c.Request.Context(), c.Query("sale_status"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, stores)
}

// GetStore GET /api/v1/stores/:id
func (h *StoreBrowseHandler) GetStore(c *gin.Context) {
	store, err := h.storeUC.GetStoreEnriched(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, store)
}

// GetMenu GET /api/v1/stores/:id/menu
func (h *StoreBrowseHandler) GetMenu(c *gin.Context) {
	items, err := h.catalogUC.ListMenuItems(c.Request.Context(), c.Param("id"), c.Query("category_id"), "on")
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// ListCategories GET /api/v1/stores/:id/categories — lets the storefront group
// the (flat) menu and the vendor console list its categories.
func (h *StoreBrowseHandler) ListCategories(c *gin.Context) {
	cats, err := h.catalogUC.ListCategories(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, cats)
}

// GetShipFee GET /api/v1/stores/:id/ship-fee?room_id=
func (h *StoreBrowseHandler) GetShipFee(c *gin.Context) {
	roomID := c.Query("room_id")
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room_id is required"})
		return
	}
	fee, err := h.shipFeeUC.ResolveFee(c.Request.Context(), c.Param("id"), roomID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"unit_ship_fee": fee})
}

// GetStoreHours GET /api/v1/stores/:id/hours
// Returns the current approved operating hours and ship cutoffs for a store.
// Readable by any authenticated user (customers + vendors).
func (h *StoreBrowseHandler) GetStoreHours(c *gin.Context) {
	ctx := c.Request.Context()
	storeID := c.Param("id")

	oh, err := h.hoursUC.ListOperatingHours(ctx, storeID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	sc, err := h.hoursUC.ListShipCutoffs(ctx, storeID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{
		"operating_hours": oh,
		"ship_cutoffs":    sc,
	})
}

// GetStoreSlots GET /api/v1/stores/:id/slots?date=YYYY-MM-DD
// Returns the store's ship cutoffs (ca giao) plus the remaining slot quota per
// (menu item, cutoff) for the given date (default today). Powers the customer
// session picker + "còn N suất" badges.
func (h *StoreBrowseHandler) GetStoreSlots(c *gin.Context) {
	ctx := c.Request.Context()
	storeID := c.Param("id")

	date := time.Now().UTC().Truncate(24 * time.Hour)
	if ds := c.Query("date"); ds != "" {
		d, err := time.Parse("2006-01-02", ds)
		if err != nil {
			response.BadRequest(c, "invalid date (want YYYY-MM-DD)")
			return
		}
		date = d
	}

	cutoffs, err := h.hoursUC.ListShipCutoffs(ctx, storeID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	quotas, err := h.catalogUC.ListStoreSlotQuotas(ctx, storeID, date)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	type slotQuota struct {
		MenuItemID string `json:"MenuItemID"`
		CutoffID   string `json:"CutoffID"`
		Quota      int    `json:"Quota"`
		SoldCount  int    `json:"SoldCount"`
		Remaining  int    `json:"Remaining"`
	}
	out := make([]slotQuota, 0, len(quotas))
	for _, q := range quotas {
		rem := q.Quota - q.SoldCount
		if rem < 0 {
			rem = 0
		}
		out = append(out, slotQuota{q.MenuItemID, q.CutoffID, q.Quota, q.SoldCount, rem})
	}

	response.Success(c, gin.H{"cutoffs": cutoffs, "quotas": out})
}

// ListMyStores GET /api/v1/stores/mine
// Returns only the stores owned by the authenticated user (vendor console view).
func (h *StoreBrowseHandler) ListMyStores(c *gin.Context) {
	ownerUserID := authmw.UserIDFromContext(c.Request.Context())
	stores, err := h.storeUC.ListMyStores(c.Request.Context(), ownerUserID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, stores)
}
