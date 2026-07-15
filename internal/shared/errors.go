package shared

import (
	"errors"
	"net/http"
)

var ErrNotFound = errors.New("resource not found")

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAuthError(message string) *AppError {
	return &AppError{
		Code:       "auth_error",
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

func NewConflictError(message string) *AppError {
	return &AppError{
		Code:       "conflict_error",
		Message:    message,
		StatusCode: http.StatusConflict,
	}
}
