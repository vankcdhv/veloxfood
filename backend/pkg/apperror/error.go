package apperror

import "net/http"

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

func New(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, message)
}

func NotFound(message string) *Error {
	return New(http.StatusNotFound, message)
}

func Conflict(message string) *Error {
	return New(http.StatusConflict, message)
}

func Unauthorized(message string) *Error {
	return New(http.StatusUnauthorized, message)
}

func Forbidden(message string) *Error {
	return New(http.StatusForbidden, message)
}

func Internal(message string) *Error {
	return New(http.StatusInternalServerError, message)
}

func Gone(message string) *Error {
	return New(http.StatusGone, message)
}
