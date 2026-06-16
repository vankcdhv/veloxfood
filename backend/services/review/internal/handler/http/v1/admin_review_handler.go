package v1

import (
	"project/pkg/pagination"
	"project/pkg/response"
	"project/services/review/internal/usecase"

	"github.com/gin-gonic/gin"
)

// AdminReviewHandler handles admin moderation operations.
type AdminReviewHandler struct {
	reviewUC usecase.ReviewUsecase
}

func NewAdminReviewHandler(reviewUC usecase.ReviewUsecase) *AdminReviewHandler {
	return &AdminReviewHandler{reviewUC: reviewUC}
}

// ListReported GET /api/v1/admin/reviews/reported?page=1&page_size=20
func (h *AdminReviewHandler) ListReported(c *gin.Context) {
	page, pageSize := pagination.Parse(c)

	reports, total, err := h.reviewUC.ListReportedReviews(c.Request.Context(), page, pageSize)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Paginated(c, reports, total, page)
}

// HideReview PATCH /api/v1/admin/reviews/:id/hide
func (h *AdminReviewHandler) HideReview(c *gin.Context) {
	reviewID := c.Param("id")
	if err := h.reviewUC.HideReview(c.Request.Context(), reviewID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// RestoreReview PATCH /api/v1/admin/reviews/:id/restore
func (h *AdminReviewHandler) RestoreReview(c *gin.Context) {
	reviewID := c.Param("id")
	if err := h.reviewUC.RestoreReview(c.Request.Context(), reviewID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
