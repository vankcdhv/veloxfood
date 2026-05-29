package v1

import (
	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/user/internal/entity"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

// CardAdminHandler handles admin-only card identifier endpoints.
type CardAdminHandler struct {
	cardUC usecase.CardUsecase
}

func NewCardAdminHandler(cardUC usecase.CardUsecase) *CardAdminHandler {
	return &CardAdminHandler{cardUC: cardUC}
}

type bindCardRequest struct {
	Kind       entity.CardKind `json:"kind" binding:"required"`
	Identifier string          `json:"identifier" binding:"required"`
}

// BindCard handles POST /api/v1/admin/users/:user_id/cards.
func (h *CardAdminHandler) BindCard(c *gin.Context) {
	var req bindCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := c.Param("user_id")
	if userID == "" {
		response.BadRequest(c, "user_id is required")
		return
	}

	adminID := authmw.UserIDFromContext(c.Request.Context())
	card, err := h.cardUC.BindCard(c.Request.Context(), adminID, userID, req.Kind, req.Identifier)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, card)
}

// RevokeCard handles DELETE /api/v1/admin/users/:user_id/cards/:card_id.
func (h *CardAdminHandler) RevokeCard(c *gin.Context) {
	cardID := c.Param("card_id")
	if cardID == "" {
		response.BadRequest(c, "card_id is required")
		return
	}

	adminID := authmw.UserIDFromContext(c.Request.Context())
	if err := h.cardUC.RevokeCard(c.Request.Context(), adminID, cardID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}

// ListCards handles GET /api/v1/admin/users/:user_id/cards?include_revoked=false.
func (h *CardAdminHandler) ListCards(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		response.BadRequest(c, "user_id is required")
		return
	}

	includeRevoked := c.Query("include_revoked") == "true"
	cards, err := h.cardUC.ListCardsByUser(c.Request.Context(), userID, includeRevoked)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, cards)
}
