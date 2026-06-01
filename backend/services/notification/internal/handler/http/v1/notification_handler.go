package v1

import (
	"fmt"
	"net/http"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/notification/internal/usecase"

	"github.com/gin-gonic/gin"
)

// NotificationHandler handles the notification-centre HTTP API.
type NotificationHandler struct {
	uc *usecase.NotificationUsecase
}

func NewNotificationHandler(uc *usecase.NotificationUsecase) *NotificationHandler {
	return &NotificationHandler{uc: uc}
}

// ListNotifications GET /api/v1/me/notifications?skip=0&limit=20
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())
	skip, limit := parsePaging(c)

	result, err := h.uc.List(c.Request.Context(), userID, skip, limit)
	if err != nil {
		response.InternalError(c)
		return
	}

	// Build PascalCase response items for FE.
	type item struct {
		ID        string  `json:"ID"`
		UserID    string  `json:"UserID"`
		Type      string  `json:"Type"`
		Title     string  `json:"Title"`
		Body      string  `json:"Body"`
		Data      any     `json:"Data"`
		Channel   string  `json:"Channel"`
		ReadAt    *string `json:"ReadAt"`
		CreatedAt string  `json:"CreatedAt"`
	}
	items := make([]item, len(result.Items))
	for i, n := range result.Items {
		var readAt *string
		if n.ReadAt != nil {
			s := n.ReadAt.UTC().Format("2006-01-02T15:04:05Z")
			readAt = &s
		}
		items[i] = item{
			ID:        n.ID,
			UserID:    n.UserID,
			Type:      n.Type,
			Title:     n.Title,
			Body:      n.Body,
			Data:      n.Data,
			Channel:   string(n.Channel),
			ReadAt:    readAt,
			CreatedAt: n.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
	}

	c.JSON(http.StatusOK, response.Response{
		Status:  http.StatusOK,
		Message: "success",
		Data: gin.H{
			"Items": items,
			"Total": result.Total,
		},
	})
}

// UnreadCount GET /api/v1/me/notifications/unread-count
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())
	count, err := h.uc.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c)
		return
	}
	c.JSON(http.StatusOK, response.Response{
		Status:  http.StatusOK,
		Message: "success",
		Data:    gin.H{"Count": count},
	})
}

// MarkRead PATCH /api/v1/me/notifications/:id/read
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())
	id := c.Param("id")

	found, err := h.uc.MarkRead(c.Request.Context(), id, userID)
	if err != nil {
		response.InternalError(c)
		return
	}
	if !found {
		response.NotFound(c, "notification not found")
		return
	}
	response.Success(c, gin.H{"Read": true})
}

// MarkAllRead POST /api/v1/me/notifications/read-all
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())
	if err := h.uc.MarkAllRead(c.Request.Context(), userID); err != nil {
		response.InternalError(c)
		return
	}
	response.Success(c, gin.H{"Read": true})
}

// RegisterDeviceToken POST /api/v1/me/device-tokens
func (h *NotificationHandler) RegisterDeviceToken(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())

	var body struct {
		FCMToken string `json:"fcm_token" binding:"required"`
		Platform string `json:"platform" binding:"required,oneof=ios android web"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.uc.RegisterToken(c.Request.Context(), userID, body.FCMToken, body.Platform); err != nil {
		response.InternalError(c)
		return
	}
	response.Success(c, gin.H{"Registered": true})
}

// DeleteDeviceToken DELETE /api/v1/me/device-tokens/:id
func (h *NotificationHandler) DeleteDeviceToken(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())
	id := c.Param("id")

	found, err := h.uc.DeleteToken(c.Request.Context(), id, userID)
	if err != nil {
		response.InternalError(c)
		return
	}
	if !found {
		response.NotFound(c, "device token not found")
		return
	}
	response.Success(c, gin.H{"Deleted": true})
}

// parsePaging extracts skip/limit from query params with safe defaults.
func parsePaging(c *gin.Context) (skip, limit int) {
	skip = 0
	limit = 20
	if s := c.Query("skip"); s != "" {
		if n, err := scanInt(s); err == nil && n >= 0 {
			skip = n
		}
	}
	if l := c.Query("limit"); l != "" {
		if n, err := scanInt(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	return
}

func scanInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
