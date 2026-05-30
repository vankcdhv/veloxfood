package v1

import (
	"path/filepath"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

const maxShipperPhotoBytes = 5 << 20 // 5 MB per photo

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
	if header.Size > maxShipperPhotoBytes {
		return photo{}, &fieldError{field: field, msg: field + " exceeds 5MB"}
	}
	f, err := header.Open()
	if err != nil {
		return photo{}, &fieldError{field: field, msg: "cannot read " + field}
	}
	ct := header.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
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
