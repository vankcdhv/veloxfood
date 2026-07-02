package v1

import (
	"errors"
	"time"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/store/internal/usecase"

	"github.com/gin-gonic/gin"
)

// FavoriteHandler exposes the customer's bookmarked stores and menu items.
// All routes require authentication; ownership is implicit (JWT user id).
type FavoriteHandler struct {
	favoriteUC usecase.FavoriteUsecase
	hoursUC    usecase.HoursUsecase
}

func NewFavoriteHandler(favoriteUC usecase.FavoriteUsecase, hoursUC usecase.HoursUsecase) *FavoriteHandler {
	return &FavoriteHandler{favoriteUC: favoriteUC, hoursUC: hoursUC}
}

// List GET /api/v1/me/favorites?type=store|item
// type=store → enriched store cards (with OpenNow); type=item → menu-item rows.
func (h *FavoriteHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	userID := authmw.UserIDFromContext(ctx)

	switch c.DefaultQuery("type", "store") {
	case "store", "STORE":
		stores, err := h.favoriteUC.Stores(ctx, userID)
		if err != nil {
			response.HandleError(c, err)
			return
		}
		now := time.Now()
		views := make([]*usecase.StoreView, len(stores))
		for i, s := range stores {
			v := &usecase.StoreView{Store: s}
			if openRes, err := h.hoursUC.IsOpenNow(ctx, s.ID, s.SaleStatus, now); err == nil {
				v.OpenNow = openRes.OpenNow
				v.OpenTimeToday = openRes.OpenTimeToday
				v.CloseTimeToday = openRes.CloseTimeToday
			}
			views[i] = v
		}
		response.Success(c, views)
	case "item", "ITEM":
		items, err := h.favoriteUC.Items(ctx, userID)
		if err != nil {
			response.HandleError(c, err)
			return
		}
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
		response.Success(c, items)
	default:
		response.BadRequest(c, "type must be store or item")
	}
}

// ListIDs GET /api/v1/me/favorites/ids?type=store|item — bookmarked target ids
// only, for cheap heart-state hydration on lists.
func (h *FavoriteHandler) ListIDs(c *gin.Context) {
	ctx := c.Request.Context()
	ids, err := h.favoriteUC.IDs(ctx, authmw.UserIDFromContext(ctx), c.DefaultQuery("type", "store"))
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.Success(c, ids)
}

// Add PUT /api/v1/me/favorites/:type/:id — idempotent bookmark.
func (h *FavoriteHandler) Add(c *gin.Context) {
	ctx := c.Request.Context()
	if err := h.favoriteUC.Add(ctx, authmw.UserIDFromContext(ctx), c.Param("type"), c.Param("id")); err != nil {
		h.handleErr(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Remove DELETE /api/v1/me/favorites/:type/:id — idempotent unbookmark.
func (h *FavoriteHandler) Remove(c *gin.Context) {
	ctx := c.Request.Context()
	if err := h.favoriteUC.Remove(ctx, authmw.UserIDFromContext(ctx), c.Param("type"), c.Param("id")); err != nil {
		h.handleErr(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *FavoriteHandler) handleErr(c *gin.Context, err error) {
	if errors.Is(err, usecase.ErrInvalidFavoriteTarget) {
		response.BadRequest(c, "type must be store or item and id must be a UUID")
		return
	}
	response.HandleError(c, err)
}
