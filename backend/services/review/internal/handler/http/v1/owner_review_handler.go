package v1

import (
	"project/pkg/response"
	"project/services/review/internal/usecase"

	"github.com/gin-gonic/gin"
)

// OwnerReviewHandler handles store-owner reply operations.
type OwnerReviewHandler struct {
	reviewUC usecase.ReviewUsecase
}

func NewOwnerReviewHandler(reviewUC usecase.ReviewUsecase) *OwnerReviewHandler {
	return &OwnerReviewHandler{reviewUC: reviewUC}
}

// ReplyToReview POST /api/v1/reviews/:id/reply
// Authorisation: caller must have store.manage permission for the review's store vendor.
// Request body (snake_case): content
func (h *OwnerReviewHandler) ReplyToReview(c *gin.Context) {
	reviewID := c.Param("id")
	var body struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	reply, err := h.reviewUC.ReplyToReview(c.Request.Context(), reviewID, body.Content)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, reply)
}
