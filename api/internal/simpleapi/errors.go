package simpleapi

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	appctx "github.com/dokkiitech/ashiato/api/internal/middleware"
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

func internalErrorWithLog(c echo.Context, message string, err error, fields ...any) error {
	attrs := []any{
		"trace_id", appctx.TraceIDFromContext(c.Request().Context()),
		"path", c.Request().URL.Path,
		"method", c.Request().Method,
		"error", err,
	}
	attrs = append(attrs, fields...)
	slog.ErrorContext(c.Request().Context(), message, attrs...)
	return internalError(c)
}
