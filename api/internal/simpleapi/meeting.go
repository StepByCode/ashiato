package simpleapi

import (
	"net/http"
	"time"

	"firebase.google.com/go/v4/db"
	"github.com/labstack/echo/v4"
)

// MeetingResponse matches the spec in docs/backend-api-request.md §4.2.
type MeetingResponse struct {
	MeetingAt *string `json:"meetingAt,omitempty"`
	MeetURL   string  `json:"meetUrl"`
	UpdatedAt string  `json:"updatedAt"`
}

const (
	meetingCollection = "meeting_settings"
	meetingDocID      = "default"
)

// RegisterMeetingRoutes registers Meeting API endpoints.
func RegisterMeetingRoutes(g *echo.Group, client *db.Client) {
	g.GET("/meeting", getMeetingHandler(client))
	g.PATCH("/meeting", patchMeetingHandler(client))
}

func getMeetingHandler(client *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		ref := client.NewRef(meetingCollection).Child(meetingDocID)
		var resp MeetingResponse
		if err := ref.Get(ctx, &resp); err != nil || resp.UpdatedAt == "" {
			return c.JSON(http.StatusOK, MeetingResponse{
				MeetURL:   "",
				UpdatedAt: time.Now().Format(time.RFC3339),
			})
		}
		return c.JSON(http.StatusOK, resp)
	}
}

func patchMeetingHandler(client *db.Client) echo.HandlerFunc {
	type request struct {
		MeetingAt *string `json:"meetingAt"`
		MeetURL   *string `json:"meetUrl"`
	}

	return func(c echo.Context) error {
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}

		if req.MeetingAt != nil && *req.MeetingAt != "" {
			if _, err := time.Parse(time.RFC3339, *req.MeetingAt); err != nil {
				return validationError(c, "meetingAt", "must be ISO 8601 format")
			}
		}

		ctx := c.Request().Context()
		now := time.Now().Format(time.RFC3339)
		ref := client.NewRef(meetingCollection).Child(meetingDocID)

		// Try update existing
		updates := map[string]interface{}{
			"updatedAt": now,
		}
		if req.MeetingAt != nil {
			updates["meetingAt"] = *req.MeetingAt
		}
		if req.MeetURL != nil {
			updates["meetUrl"] = *req.MeetURL
		}

		// Check if doc exists
		var existing MeetingResponse
		if err := ref.Get(ctx, &existing); err != nil || existing.UpdatedAt == "" {
			// Create new
			data := MeetingResponse{
				MeetURL:   "",
				UpdatedAt: now,
			}
			if req.MeetingAt != nil {
				data.MeetingAt = req.MeetingAt
			}
			if req.MeetURL != nil {
				data.MeetURL = *req.MeetURL
			}
			if err := ref.Set(ctx, data); err != nil {
				return internalError(c)
			}
			return c.JSON(http.StatusOK, data)
		}

		if err := ref.Update(ctx, updates); err != nil {
			return internalError(c)
		}

		var resp MeetingResponse
		if err := ref.Get(ctx, &resp); err != nil {
			return internalError(c)
		}
		return c.JSON(http.StatusOK, resp)
	}
}
