package simpleapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// ErrorResponse matches the spec in docs/backend-api-request.md §3.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []FieldDetail `json:"details,omitempty"`
}

type FieldDetail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func validationError(c echo.Context, field, reason string) error {
	return c.JSON(http.StatusBadRequest, ErrorResponse{
		Error: ErrorBody{
			Code:    "VALIDATION_ERROR",
			Message: field + " " + reason,
			Details: []FieldDetail{{Field: field, Reason: reason}},
		},
	})
}

func notFoundError(c echo.Context, message string) error {
	return c.JSON(http.StatusNotFound, ErrorResponse{
		Error: ErrorBody{
			Code:    "NOT_FOUND",
			Message: message,
		},
	})
}

func conflictError(c echo.Context, message string) error {
	return c.JSON(http.StatusConflict, ErrorResponse{
		Error: ErrorBody{
			Code:    "STATE_CONFLICT",
			Message: message,
		},
	})
}

func internalError(c echo.Context) error {
	return c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: ErrorBody{
			Code:    "INTERNAL_ERROR",
			Message: "internal server error",
		},
	})
}
