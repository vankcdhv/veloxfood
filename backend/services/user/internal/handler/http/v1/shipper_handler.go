package v1

import (
	"path/filepath"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/pkg/storage"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

// ShipperHandler handles shipper self-registration.
type ShipperHandler struct {
	registerUC usecase.ShipperRegisterUsecase
}

func NewShipperHandler(registerUC usecase.ShipperRegisterUsecase) *ShipperHandler {
	return &ShipperHandler{registerUC: registerUC}
}

// Register POST /api/v1/shipper/register — multipart form with id_document + portrait photos.
func (h *ShipperHandler) Register(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())

	idDoc, err := readPhoto(c, "id_document")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	defer idDoc.close()

	portrait, err := readPhoto(c, "portrait")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	defer portrait.close()

	out, err := h.registerUC.Register(c.Request.Context(), userID, usecase.ShipperRegisterInput{
		IDDocument: idDoc.file,
		Portrait:   portrait.file,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, out)
}

// GetMine GET /api/v1/shipper/me — caller's own application status (or registered:false).
func (h *ShipperHandler) GetMine(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())
	out, err := h.registerUC.GetMine(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	if out == nil {
		response.Success(c, gin.H{"registered": false})
		return
	}
	response.Success(c, gin.H{
		"registered":  true,
		"status":      out.Status,
		"applied_at":  out.AppliedAt,
		"approved_at": out.ApprovedAt,
	})
}

// photo bundles an opened multipart file + cleanup.
type photo struct {
	file  usecase.UploadFile
	close func()
}

func readPhoto(c *gin.Context, field string) (photo, error) {
	header, err := c.FormFile(field)
	if err != nil {
		return photo{}, &fieldError{field: field, msg: field + " is required"}
	}
	if header.Size > storage.MaxImageBytes {
		return photo{}, &fieldError{field: field, msg: field + " exceeds 5MB"}
	}
	ct := header.Header.Get("Content-Type")
	if !storage.AllowedImageType(ct) {
		return photo{}, &fieldError{field: field, msg: field + " must be a JPEG, PNG or WebP image"}
	}
	f, err := header.Open()
	if err != nil {
		return photo{}, &fieldError{field: field, msg: "cannot read " + field}
	}
	return photo{
		file: usecase.UploadFile{
			Reader:      f,
			Size:        header.Size,
			ContentType: ct,
			Ext:         filepath.Ext(header.Filename),
		},
		close: func() { _ = f.Close() },
	}, nil
}

type fieldError struct {
	field string
	msg   string
}

func (e *fieldError) Error() string { return e.msg }
