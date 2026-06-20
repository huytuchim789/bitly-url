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
	ErrNotFound     = &AppError{Code: http.StatusNotFound, Message: "url not found"}
	ErrBadRequest   = &AppError{Code: http.StatusBadRequest, Message: "bad request"}
	ErrInternal     = &AppError{Code: http.StatusInternalServerError, Message: "internal server error"}
	ErrGone         = &AppError{Code: http.StatusGone, Message: "url has expired"}
	ErrForbidden    = &AppError{Code: http.StatusForbidden, Message: "redirect target not allowed"}
	ErrRateLimited  = &AppError{Code: http.StatusTooManyRequests, Message: "rate limit exceeded"}
	ErrShortCode    = &AppError{Code: http.StatusConflict, Message: "short code already taken"}
)
