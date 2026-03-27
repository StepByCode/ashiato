package simpleapi

import (
	"fmt"
	"net/http"
	"strconv"
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

const meetingCollection = "meeting_settings"

// RegisterMeetingRoutes registers Meeting API endpoints.
func RegisterMeetingRoutes(g *echo.Group, client *db.Client) {
	g.GET("/meeting", getMeetingHandler(client))
	g.PATCH("/meeting", patchMeetingHandler(client))
}

func meetingDocID(year, month int) string {
	if year == 0 || month == 0 {
		return "default"
	}
	return fmt.Sprintf("%d_%02d", year, month)
}

func parsePeriodParams(c echo.Context) (int, int) {
	yearStr := c.QueryParam("year")
	monthStr := c.QueryParam("month")
	year, _ := strconv.Atoi(yearStr)
	month, _ := strconv.Atoi(monthStr)
	return year, month
}

func getMeetingHandler(client *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		year, month := parsePeriodParams(c)
		docID := meetingDocID(year, month)
		ref := client.NewRef(meetingCollection).Child(docID)
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
		Year      int     `json:"year"`
		Month     int     `json:"month"`
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
		docID := meetingDocID(req.Year, req.Month)
		ref := client.NewRef(meetingCollection).Child(docID)

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
