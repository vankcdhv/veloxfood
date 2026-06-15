package v1

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/delivery/internal/entity"
	"project/services/delivery/internal/usecase"

	"github.com/gin-gonic/gin"
)

// ShipperDeliveryHandler handles all shipper-facing delivery endpoints.
type ShipperDeliveryHandler struct {
	deliveryUC usecase.DeliveryUsecase
	incidentUC usecase.IncidentUsecase
	uploader   usecase.FileUploader
}

func NewShipperDeliveryHandler(deliveryUC usecase.DeliveryUsecase, incidentUC usecase.IncidentUsecase, uploader usecase.FileUploader) *ShipperDeliveryHandler {
	return &ShipperDeliveryHandler{deliveryUC: deliveryUC, incidentUC: incidentUC, uploader: uploader}
}

// ListAvailable returns all AVAILABLE deliveries with resolved room paths.
// GET /api/v1/deliveries/available
func (h *ShipperDeliveryHandler) ListAvailable(c *gin.Context) {
	items, err := h.deliveryUC.ListAvailable(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Response{
			Status: http.StatusInternalServerError, Message: "internal server error", Error: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, response.Response{Status: http.StatusOK, Message: "ok", Data: items})
}

// ClaimDelivery atomically claims a delivery for the authenticated shipper.
// POST /api/v1/deliveries/:orderId/claim
func (h *ShipperDeliveryHandler) ClaimDelivery(c *gin.Context) {
	orderID := c.Param("orderId")
	shipperID := authmw.UserIDFromContext(c.Request.Context())

	result, err := h.deliveryUC.ClaimDelivery(c.Request.Context(), orderID, shipperID)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrDeliveryNotFound):
			c.JSON(http.StatusNotFound, response.Response{
				Status: http.StatusNotFound, Message: "delivery not found",
			})
		case errors.Is(err, usecase.ErrAlreadyClaimed):
			c.JSON(http.StatusConflict, response.Response{
				Status: http.StatusConflict, Message: "already claimed",
			})
		case errors.Is(err, usecase.ErrBatchFull):
			c.JSON(http.StatusConflict, response.Response{
				Status: http.StatusConflict, Message: "batch is full",
			})
		default:
			c.JSON(http.StatusInternalServerError, response.Response{
				Status: http.StatusInternalServerError, Message: "internal server error", Error: err.Error(),
			})
		}
		return
	}
	c.JSON(http.StatusCreated, response.Response{Status: http.StatusCreated, Message: "claimed", Data: result})
}

// UpdateStatus advances the delivery status (PICKED_UP | DELIVERING | DELIVERED).
// PATCH /api/v1/deliveries/:orderId/status
func (h *ShipperDeliveryHandler) UpdateStatus(c *gin.Context) {
	orderID := c.Param("orderId")
	shipperID := authmw.UserIDFromContext(c.Request.Context())

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Response{
			Status: http.StatusBadRequest, Message: "bad request", Error: err.Error(),
		})
		return
	}

	next := entity.DeliveryStatus(req.Status)
	switch next {
	case entity.DeliveryPickedUp, entity.DeliveryDelivering, entity.DeliveryDelivered:
		// valid shipper-driven statuses
	default:
		c.JSON(http.StatusBadRequest, response.Response{
			Status: http.StatusBadRequest, Message: "invalid status",
			Error: "status must be PICKED_UP, DELIVERING, or DELIVERED",
		})
		return
	}

	if err := h.deliveryUC.UpdateStatus(c.Request.Context(), orderID, shipperID, next); err != nil {
		switch {
		case errors.Is(err, usecase.ErrDeliveryNotFound):
			c.JSON(http.StatusNotFound, response.Response{
				Status: http.StatusNotFound, Message: "delivery not found",
			})
		case errors.Is(err, usecase.ErrNotDeliveryOwner):
			c.JSON(http.StatusForbidden, response.Response{
				Status: http.StatusForbidden, Message: "not the claiming shipper",
			})
		case errors.Is(err, usecase.ErrInvalidTransition):
			c.JSON(http.StatusBadRequest, response.Response{
				Status: http.StatusBadRequest, Message: "invalid status transition",
			})
		default:
			c.JSON(http.StatusInternalServerError, response.Response{
				Status: http.StatusInternalServerError, Message: "internal server error", Error: err.Error(),
			})
		}
		return
	}
	c.JSON(http.StatusOK, response.Response{Status: http.StatusOK, Message: "status updated"})
}

// MyDeliveries returns the shipper's delivery history and total earnings.
// GET /api/v1/me/deliveries
func (h *ShipperDeliveryHandler) MyDeliveries(c *gin.Context) {
	shipperID := authmw.UserIDFromContext(c.Request.Context())
	result, err := h.deliveryUC.MyDeliveries(c.Request.Context(), shipperID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Response{
			Status: http.StatusInternalServerError, Message: "internal server error", Error: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, response.Response{Status: http.StatusOK, Message: "ok", Data: result})
}

// ReportIncident creates an incident for a delivery the shipper owns.
// POST /api/v1/deliveries/:orderId/incident
// UploadIncidentPhoto stores an incident evidence photo and returns its URL,
// which the client then submits as photo_url in the incident report.
// POST /api/v1/deliveries/:orderId/incident-photo (multipart, field "image")
func (h *ShipperDeliveryHandler) UploadIncidentPhoto(c *gin.Context) {
	orderID := c.Param("orderId")
	shipperID := authmw.UserIDFromContext(c.Request.Context())

	// Only the assigned shipper may upload evidence for this delivery — the
	// object key is deterministic per order, so without this any shipper could
	// overwrite another's incident photo.
	owns, err := h.deliveryUC.OwnsDelivery(c.Request.Context(), orderID, shipperID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Response{
			Status: http.StatusInternalServerError, Message: "internal server error", Error: err.Error(),
		})
		return
	}
	if !owns {
		c.JSON(http.StatusForbidden, response.Response{
			Status: http.StatusForbidden, Message: "forbidden", Error: "not the delivery owner",
		})
		return
	}

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Response{
			Status: http.StatusBadRequest, Message: "bad request", Error: "image file required",
		})
		return
	}
	defer file.Close()

	// Reject oversized uploads (max 5MB) and non-image content types.
	const maxIncidentPhotoBytes = 5 << 20
	if header.Size > maxIncidentPhotoBytes {
		c.JSON(http.StatusBadRequest, response.Response{
			Status: http.StatusBadRequest, Message: "bad request", Error: "image too large (max 5MB)",
		})
		return
	}
	contentType := header.Header.Get("Content-Type")
	if !allowedImageType(contentType) {
		c.JSON(http.StatusBadRequest, response.Response{
			Status: http.StatusBadRequest, Message: "bad request", Error: "unsupported image type",
		})
		return
	}

	ext := filepath.Ext(header.Filename)
	objectKey := fmt.Sprintf("deliveries/%s/incident%s", orderID, ext)
	url, err := h.uploader.Put(c.Request.Context(), objectKey, header.Header.Get("Content-Type"), file.(io.Reader), header.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Response{
			Status: http.StatusInternalServerError, Message: "internal server error", Error: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, response.Response{Status: http.StatusOK, Message: "ok", Data: gin.H{"photo_url": url}})
}

// allowedImageType reports whether the multipart content type is an accepted
// incident-evidence image format.
func allowedImageType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

// GetOrderShipper returns the shipper assigned to the caller's order, so the
// customer can rate the delivery. GET /api/v1/deliveries/:orderId/shipper
func (h *ShipperDeliveryHandler) GetOrderShipper(c *gin.Context) {
	orderID := c.Param("orderId")
	customerID := authmw.UserIDFromContext(c.Request.Context())
	shipperID, err := h.deliveryUC.GetOrderShipper(c.Request.Context(), orderID, customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Response{
			Status: http.StatusInternalServerError, Message: "internal server error", Error: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, response.Response{Status: http.StatusOK, Message: "ok", Data: gin.H{"shipper_id": shipperID}})
}

func (h *ShipperDeliveryHandler) ReportIncident(c *gin.Context) {
	orderID := c.Param("orderId")
	shipperID := authmw.UserIDFromContext(c.Request.Context())

	var req struct {
		Type     string  `json:"type" binding:"required,oneof=WRONG_ADDRESS CUSTOMER_ABSENT ITEM_DAMAGED OTHER"`
		Note     string  `json:"note" binding:"required"`
		PhotoURL *string `json:"photo_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Response{
			Status: http.StatusBadRequest, Message: "bad request", Error: err.Error(),
		})
		return
	}

	inc, err := h.incidentUC.ReportIncident(c.Request.Context(), usecase.ReportIncidentRequest{
		OrderID:   orderID,
		ShipperID: shipperID,
		Type:      req.Type,
		Note:      req.Note,
		PhotoURL:  req.PhotoURL,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrDeliveryNotFound):
			c.JSON(http.StatusNotFound, response.Response{
				Status: http.StatusNotFound, Message: "delivery not found",
			})
		case errors.Is(err, usecase.ErrNotDeliveryOwner):
			c.JSON(http.StatusForbidden, response.Response{
				Status: http.StatusForbidden, Message: "not the claiming shipper",
			})
		default:
			c.JSON(http.StatusInternalServerError, response.Response{
				Status: http.StatusInternalServerError, Message: "internal server error", Error: err.Error(),
			})
		}
		return
	}
	c.JSON(http.StatusCreated, response.Response{Status: http.StatusCreated, Message: "incident reported", Data: inc})
}
