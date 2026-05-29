package response

import (
	"errors"
	"net/http"

	"project/pkg/apperror"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type PaginatedData struct {
	Items interface{} `json:"items"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Status:  http.StatusOK,
		Message: "success",
		Data:    data,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Status:  http.StatusCreated,
		Message: "created",
		Data:    data,
	})
}

func Paginated(c *gin.Context, items interface{}, total int64, page int) {
	c.JSON(http.StatusOK, Response{
		Status:  http.StatusOK,
		Message: "success",
		Data: PaginatedData{
			Items: items,
			Total: total,
			Page:  page,
		},
	})
}

func BadRequest(c *gin.Context, err string) {
	c.JSON(http.StatusBadRequest, Response{
		Status:  http.StatusBadRequest,
		Message: "bad request",
		Error:   err,
	})
}

func NotFound(c *gin.Context, err string) {
	c.JSON(http.StatusNotFound, Response{
		Status:  http.StatusNotFound,
		Message: "not found",
		Error:   err,
	})
}

func Conflict(c *gin.Context, err string) {
	c.JSON(http.StatusConflict, Response{
		Status:  http.StatusConflict,
		Message: "conflict",
		Error:   err,
	})
}

func InternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, Response{
		Status:  http.StatusInternalServerError,
		Message: "internal server error",
		Error:   "something went wrong",
	})
}

func HandleError(c *gin.Context, err error) {
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		c.JSON(appErr.Code, Response{
			Status: appErr.Code,
			Error:  appErr.Message,
		})
		return
	}
	InternalError(c)
}
