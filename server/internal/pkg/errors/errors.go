package errors

import "net/http"

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"error"`
}

func (e *AppError) Error() string {
	return e.Message
}

var (
	ErrNotFound   = &AppError{Code: http.StatusNotFound, Message: "url not found"}
	ErrBadRequest = &AppError{Code: http.StatusBadRequest, Message: "bad request"}
	ErrInternal   = &AppError{Code: http.StatusInternalServerError, Message: "internal server error"}
)
