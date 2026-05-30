package v1

import (
	"project/pkg/response"
	"project/services/promotion/internal/usecase"

	"github.com/gin-gonic/gin"
)

// PublicPromotionHandler exposes endpoints available to any authenticated user.
type PublicPromotionHandler struct {
	promoUC usecase.PromotionUsecase
}

func NewPublicPromotionHandler(promoUC usecase.PromotionUsecase) *PublicPromotionHandler {
	return &PublicPromotionHandler{promoUC: promoUC}
}

// ValidatePromotion POST /promotions/validate
// Dry-run discount preview — does NOT reserve the promotion.
// Response shape: { Applicable bool, ItemDiscount int64, ShipDiscount int64, ErrorReason string }
func (h *PublicPromotionHandler) ValidatePromotion(c *gin.Context) {
	var body struct {
		Code      string `json:"code"       binding:"required"`
		StoreID   string `json:"store_id"   binding:"required"`
		Subtotal  int64  `json:"subtotal"   binding:"required"`
		ItemCount int32  `json:"item_count"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.promoUC.ValidatePromotion(c.Request.Context(), usecase.ValidateRequest{
		Code:      body.Code,
		StoreID:   body.StoreID,
		Subtotal:  body.Subtotal,
		ItemCount: body.ItemCount,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, result)
}
