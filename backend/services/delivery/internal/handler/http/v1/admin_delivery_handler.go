package v1

import (
	"errors"
	"net/http"

	"project/pkg/response"
	"project/services/delivery/internal/repository"
	"project/services/delivery/internal/usecase"

	"github.com/gin-gonic/gin"
)

// AdminDeliveryHandler handles admin-only delivery and incident listing endpoints.
type AdminDeliveryHandler struct {
	deliveryRepo repository.DeliveryRepository
	incidentUC   usecase.IncidentUsecase
}

func NewAdminDeliveryHandler(deliveryRepo repository.DeliveryRepository, incidentUC usecase.IncidentUsecase) *AdminDeliveryHandler {
	return &AdminDeliveryHandler{deliveryRepo: deliveryRepo, incidentUC: incidentUC}
}

// ListDeliveries returns all deliveries for admin overview.
// GET /api/v1/admin/deliveries
func (h *AdminDeliveryHandler) ListDeliveries(c *gin.Context) {
	rows, err := h.deliveryRepo.ListAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Response{
			Status: http.StatusInternalServerError, Message: "internal server error", Error: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, response.Response{Status: http.StatusOK, Message: "ok", Data: rows})
}

// ListIncidents returns all delivery incidents for admin review.
// GET /api/v1/admin/incidents
func (h *AdminDeliveryHandler) ListIncidents(c *gin.Context) {
	rows, err := h.incidentUC.ListAllIncidents(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Response{
			Status: http.StatusInternalServerError, Message: "internal server error", Error: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, response.Response{Status: http.StatusOK, Message: "ok", Data: rows})
}

// ResolveIncident marks an incident as resolved (admin "đã xử lý").
// PATCH /api/v1/admin/incidents/:id/resolve
func (h *AdminDeliveryHandler) ResolveIncident(c *gin.Context) {
	id := c.Param("id")
	if err := h.incidentUC.ResolveIncident(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, response.Response{
				Status: http.StatusNotFound, Message: "incident not found", Error: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, response.Response{
			Status: http.StatusInternalServerError, Message: "internal server error", Error: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, response.Response{Status: http.StatusOK, Message: "ok", Data: gin.H{"status": "RESOLVED"}})
}
