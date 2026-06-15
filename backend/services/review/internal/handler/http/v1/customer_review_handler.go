package v1

import (
	"strconv"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/review/internal/entity"
	"project/services/review/internal/usecase"

	"github.com/gin-gonic/gin"
)

// reviewListResponse is the storefront-facing shape for a page of reviews.
type reviewListResponse struct {
	Items []*entity.Review `json:"Items"`
	Total int64            `json:"Total"`
}

// CustomerReviewHandler handles customer-facing review operations.
type CustomerReviewHandler struct {
	reviewUC usecase.ReviewUsecase
}

func NewCustomerReviewHandler(reviewUC usecase.ReviewUsecase) *CustomerReviewHandler {
	return &CustomerReviewHandler{reviewUC: reviewUC}
}

// CreateReview POST /api/v1/orders/:id/reviews
// Request body (snake_case): target_type, target_id, rating, comment, photo_urls?
func (h *CustomerReviewHandler) CreateReview(c *gin.Context) {
	orderID := c.Param("id")
	customerID := authmw.UserIDFromContext(c.Request.Context())

	var body struct {
		TargetType string   `json:"target_type" binding:"required,oneof=STORE ITEM SHIPPER"`
		TargetID   string   `json:"target_id"   binding:"required"`
		Rating     int      `json:"rating"      binding:"required,min=1,max=5"`
		Comment    string   `json:"comment"`
		PhotoURLs  []string `json:"photo_urls"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	rev, err := h.reviewUC.CreateReview(c.Request.Context(), usecase.CreateReviewRequest{
		OrderID:    orderID,
		CustomerID: customerID,
		TargetType: body.TargetType,
		TargetID:   body.TargetID,
		Rating:     body.Rating,
		Comment:    body.Comment,
		PhotoURLs:  body.PhotoURLs,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, rev)
}

// ListStoreReviews GET /api/v1/stores/:id/reviews?page=1&page_size=20
func (h *CustomerReviewHandler) ListStoreReviews(c *gin.Context) {
	storeID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	reviews, total, err := h.reviewUC.ListStoreReviews(c.Request.Context(), storeID, page, pageSize)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, reviewListResponse{Items: reviews, Total: total})
}

// StoreRatingSummary GET /api/v1/stores/:id/reviews/summary
// Returns the store's average rating + review count (STORE-targeted reviews).
func (h *CustomerReviewHandler) StoreRatingSummary(c *gin.Context) {
	avg, count, err := h.reviewUC.GetStoreRatingSummary(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"Avg": avg, "Count": count})
}

// ItemRatingSummaries GET /api/v1/stores/:id/reviews/item-summaries
// Returns per-menu-item avg+count for all ITEM reviews of the store.
func (h *CustomerReviewHandler) ItemRatingSummaries(c *gin.Context) {
	summaries, err := h.reviewUC.GetItemRatingSummaries(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, summaries)
}

// ListItemReviews GET /api/v1/stores/:id/reviews/items/:itemId?page=1&page_size=20
func (h *CustomerReviewHandler) ListItemReviews(c *gin.Context) {
	itemID := c.Param("itemId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	reviews, total, err := h.reviewUC.ListItemReviews(c.Request.Context(), itemID, page, pageSize)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, reviewListResponse{Items: reviews, Total: total})
}

// ReportReview POST /api/v1/reviews/:id/report
// Request body (snake_case): reason
func (h *CustomerReviewHandler) ReportReview(c *gin.Context) {
	reviewID := c.Param("id")
	var body struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	report, err := h.reviewUC.ReportReview(c.Request.Context(), reviewID, body.Reason)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, report)
}
