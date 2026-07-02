package v1

import (
	"net/http"
	"strconv"
	"time"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/store/internal/repository"
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

// ListStores GET /api/v1/stores?sale_status=&q=&limit=12&offset=0
// Paginated (infinite-scroll friendly): returns a {items,total,page} envelope.
func (h *StoreBrowseHandler) ListStores(c *gin.Context) {
	ctx := c.Request.Context()
	limit, offset := parseLimitOffset(c, 12)
	stores, total, err := h.storeUC.ListStoresEnriched(ctx, c.Query("sale_status"), c.Query("q"), limit, offset)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	now := time.Now()
	for _, s := range stores {
		if openRes, err := h.hoursUC.IsOpenNow(ctx, s.ID, s.SaleStatus, now); err == nil {
			s.OpenNow = openRes.OpenNow
			s.OpenTimeToday = openRes.OpenTimeToday
			s.CloseTimeToday = openRes.CloseTimeToday
		}
	}
	response.Paginated(c, stores, total, offset/limit+1)
}

// GetStore GET /api/v1/stores/:id
func (h *StoreBrowseHandler) GetStore(c *gin.Context) {
	ctx := c.Request.Context()
	store, err := h.storeUC.GetStoreEnriched(ctx, c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}

	now := time.Now()
	if openRes, err := h.hoursUC.IsOpenNow(ctx, store.ID, store.SaleStatus, now); err == nil {
		store.OpenNow = openRes.OpenNow
		store.OpenTimeToday = openRes.OpenTimeToday
		store.CloseTimeToday = openRes.CloseTimeToday
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

// GetShipFee GET /api/v1/stores/:id/ship-fee?level=<BUILDING|FLOOR|ROOM>&location_id=<id>
// Back-compat: if "level" is absent, falls back to "room_id" query param treated as ROOM.
func (h *StoreBrowseHandler) GetShipFee(c *gin.Context) {
	level := c.Query("level")
	locationID := c.Query("location_id")

	// Backward compatibility: legacy callers pass room_id without level.
	if locationID == "" && level == "" {
		locationID = c.Query("room_id")
		level = "ROOM"
	}

	if locationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "location_id (or room_id) is required"})
		return
	}
	if level == "" {
		level = "ROOM"
	}

	fee, err := h.shipFeeUC.ResolveFee(c.Request.Context(), c.Param("id"), level, locationID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"unit_ship_fee": fee})
}

// GetStoreHours GET /api/v1/stores/:id/hours
// Returns the current approved operating hours for a store.
// Readable by any authenticated user (customers + vendors).
func (h *StoreBrowseHandler) GetStoreHours(c *gin.Context) {
	ctx := c.Request.Context()
	storeID := c.Param("id")

	oh, err := h.hoursUC.ListOperatingHours(ctx, storeID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"operating_hours": oh})
}

// ListMyStores GET /api/v1/stores/mine
// Returns only the stores owned by the authenticated user (vendor console view).
func (h *StoreBrowseHandler) ListMyStores(c *gin.Context) {
	ctx := c.Request.Context()
	ownerUserID := authmw.UserIDFromContext(ctx)
	stores, err := h.storeUC.ListMyStores(ctx, ownerUserID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	// Enrich open-now status so the owner console shows the real state, not
	// always "Ngoài giờ" (mirrors the public ListStores enrichment).
	now := time.Now()
	for _, s := range stores {
		if openRes, err := h.hoursUC.IsOpenNow(ctx, s.ID, s.SaleStatus, now); err == nil {
			s.OpenNow = openRes.OpenNow
			s.OpenTimeToday = openRes.OpenTimeToday
			s.CloseTimeToday = openRes.CloseTimeToday
		}
	}
	response.Success(c, stores)
}

// SearchMenuItems GET /api/v1/menu-items/search
//
//	?q=<text>&limit=12&offset=0
//	&sort=price_asc|price_desc          (default: relevance)
//	&price_min=<vnd>&price_max=<vnd>
//	&store_id=<uuid>
//
// Public endpoint — no auth required. Returns sellable menu items whose name
// matches q accent-insensitively across all active stores, paginated for
// infinite scroll. Empty q → empty page. Each row carries OpenNow so the
// client can filter/badge closed stores.
func (h *StoreBrowseHandler) SearchMenuItems(c *gin.Context) {
	ctx := c.Request.Context()
	filter := repository.SearchMenuItemsFilter{
		Q:       c.Query("q"),
		Sort:    c.Query("sort"),
		StoreID: c.Query("store_id"),
	}
	if n, err := strconv.ParseInt(c.Query("price_min"), 10, 64); err == nil && n > 0 {
		filter.PriceMin = n
	}
	if n, err := strconv.ParseInt(c.Query("price_max"), 10, 64); err == nil && n > 0 {
		filter.PriceMax = n
	}

	limit, offset := parseLimitOffset(c, 12)
	items, total, err := h.catalogUC.SearchMenuItems(ctx, filter, limit, offset)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	// Annotate open-now per distinct store on this page (hours + sale status).
	now := time.Now()
	openByStore := make(map[string]bool)
	for i := range items {
		open, seen := openByStore[items[i].StoreID]
		if !seen {
			if openRes, err := h.hoursUC.IsOpenNow(ctx, items[i].StoreID, items[i].SaleStatus, now); err == nil {
				open = openRes.OpenNow
			}
			openByStore[items[i].StoreID] = open
		}
		items[i].OpenNow = open
	}
	response.Paginated(c, items, total, offset/limit+1)
}

// parseLimitOffset reads limit/offset query params with a default limit and
// offset 0. limit is capped at 100; non-positive or invalid values fall back.
func parseLimitOffset(c *gin.Context, defaultLimit int) (limit, offset int) {
	limit = defaultLimit
	if n, err := strconv.Atoi(c.Query("limit")); err == nil && n > 0 {
		if n > 100 {
			n = 100
		}
		limit = n
	}
	if n, err := strconv.Atoi(c.Query("offset")); err == nil && n > 0 {
		offset = n
	}
	return
}
