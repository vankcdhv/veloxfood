package v1

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/pkg/storage"
	"project/services/review/internal/entity"
	"project/services/review/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// reviewListResponse is the storefront-facing shape for a page of reviews.
type reviewListResponse struct {
	Items []*entity.Review `json:"Items"`
	Total int64            `json:"Total"`
}

// ReviewPhotoUploader stores a review photo and returns its public URL.
type ReviewPhotoUploader interface {
	Put(ctx context.Context, objectKey, contentType string, r io.Reader, size int64) (string, error)
}

// CustomerReviewHandler handles customer-facing review operations.
type CustomerReviewHandler struct {
	reviewUC usecase.ReviewUsecase
	uploader ReviewPhotoUploader // nil when object storage is unavailable
}

func NewCustomerReviewHandler(reviewUC usecase.ReviewUsecase, uploader ReviewPhotoUploader) *CustomerReviewHandler {
	return &CustomerReviewHandler{reviewUC: reviewUC, uploader: uploader}
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

// ListStoreReviews GET /api/v1/stores/:id/reviews?page=1&page_size=20&rating=5
// rating 1–5 narrows to that star; omitted/0 returns all.
func (h *CustomerReviewHandler) ListStoreReviews(c *gin.Context) {
	storeID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	rating, _ := strconv.Atoi(c.DefaultQuery("rating", "0"))

	reviews, total, err := h.reviewUC.ListStoreReviews(c.Request.Context(), storeID, rating, page, pageSize)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, reviewListResponse{Items: reviews, Total: total})
}

// StoreRatingSummary GET /api/v1/stores/:id/reviews/summary
// Returns the store's average rating, review count and per-star histogram
// (STORE-targeted reviews).
func (h *CustomerReviewHandler) StoreRatingSummary(c *gin.Context) {
	summary, err := h.reviewUC.GetStoreRatingSummary(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, summary)
}

// UploadPhoto POST /api/v1/reviews/photos (multipart "image", ≤5MB JPEG/PNG/WebP)
// Stores the photo and returns its public URL for inclusion in photo_urls when
// the review is submitted.
func (h *CustomerReviewHandler) UploadPhoto(c *gin.Context) {
	if h.uploader == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "photo storage unavailable"})
		return
	}

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		response.BadRequest(c, "image file required")
		return
	}
	defer file.Close()
	if header.Size > storage.MaxImageBytes {
		response.BadRequest(c, "image exceeds 5MB")
		return
	}
	ct := header.Header.Get("Content-Type")
	if !storage.AllowedImageType(ct) {
		response.BadRequest(c, "image must be a JPEG, PNG or WebP image")
		return
	}

	customerID := authmw.UserIDFromContext(c.Request.Context())
	objectKey := fmt.Sprintf("reviews/%s/%s%s", customerID, uuid.NewString(), filepath.Ext(header.Filename))
	url, err := h.uploader.Put(c.Request.Context(), objectKey, ct, file, header.Size)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.Success(c, gin.H{"photo_url": url})
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
