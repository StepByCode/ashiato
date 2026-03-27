package simpleapi

import (
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/labstack/echo/v4"
)

// MeetingResponse matches the spec in docs/backend-api-request.md §4.2.
type MeetingResponse struct {
	MeetingAt *string `json:"meetingAt" firestore:"meetingAt,omitempty"`
	MeetURL   string  `json:"meetUrl" firestore:"meetUrl"`
	UpdatedAt string  `json:"updatedAt" firestore:"updatedAt"`
}

const (
	meetingCollection = "meeting_settings"
	meetingDocID      = "default"
)

// RegisterMeetingRoutes registers Meeting API endpoints.
func RegisterMeetingRoutes(g *echo.Group, client *firestore.Client) {
	g.GET("/meeting", getMeetingHandler(client))
	g.PATCH("/meeting", patchMeetingHandler(client))
}

func getMeetingHandler(client *firestore.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		doc, err := client.Collection(meetingCollection).Doc(meetingDocID).Get(ctx)
		if err != nil {
			// Return empty defaults if document doesn't exist
			return c.JSON(http.StatusOK, MeetingResponse{
				MeetURL:   "",
				UpdatedAt: time.Now().Format(time.RFC3339),
			})
		}

		var resp MeetingResponse
		if err := doc.DataTo(&resp); err != nil {
			return internalError(c)
		}
		return c.JSON(http.StatusOK, resp)
	}
}

func patchMeetingHandler(client *firestore.Client) echo.HandlerFunc {
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
		ref := client.Collection(meetingCollection).Doc(meetingDocID)

		updates := []firestore.Update{
			{Path: "updatedAt", Value: now},
		}
		if req.MeetingAt != nil {
			updates = append(updates, firestore.Update{Path: "meetingAt", Value: *req.MeetingAt})
		}
		if req.MeetURL != nil {
			updates = append(updates, firestore.Update{Path: "meetUrl", Value: *req.MeetURL})
		}

		// Upsert: try update, fallback to set
		_, err := ref.Update(ctx, updates)
		if err != nil {
			// Document may not exist yet — create with defaults
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
			if _, setErr := ref.Set(ctx, data); setErr != nil {
				return internalError(c)
			}
			return c.JSON(http.StatusOK, data)
		}

		doc, err := ref.Get(ctx)
		if err != nil {
			return internalError(c)
		}
		var resp MeetingResponse
		if err := doc.DataTo(&resp); err != nil {
			return internalError(c)
		}
		return c.JSON(http.StatusOK, resp)
	}
}
