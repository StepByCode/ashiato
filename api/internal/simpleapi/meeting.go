package simpleapi

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type MeetingResponse struct {
	MeetingAt *string `json:"meetingAt"`
	MeetURL   string  `json:"meetUrl"`
	UpdatedAt string  `json:"updatedAt"`
}

func RegisterMeetingRoutes(g *echo.Group, pool *pgxpool.Pool) {
	g.GET("/meeting", getMeetingHandler(pool))
	g.PATCH("/meeting", patchMeetingHandler(pool))
}

func getMeetingHandler(pool *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		var meetingAt *time.Time
		var meetURL string
		var updatedAt time.Time

		err := pool.QueryRow(c.Request().Context(),
			`SELECT meeting_at, meet_url, updated_at FROM meeting_settings WHERE id = 1`).
			Scan(&meetingAt, &meetURL, &updatedAt)
		if err != nil {
			return internalError(c)
		}

		resp := MeetingResponse{
			MeetURL:   meetURL,
			UpdatedAt: updatedAt.Format(time.RFC3339),
		}
		if meetingAt != nil {
			s := meetingAt.Format(time.RFC3339)
			resp.MeetingAt = &s
		}

		return c.JSON(http.StatusOK, resp)
	}
}

func patchMeetingHandler(pool *pgxpool.Pool) echo.HandlerFunc {
	type request struct {
		MeetingAt *string `json:"meetingAt"`
		MeetURL   *string `json:"meetUrl"`
	}

	return func(c echo.Context) error {
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}

		// Parse meetingAt if provided
		var meetingAt *time.Time
		if req.MeetingAt != nil {
			t, err := time.Parse(time.RFC3339, *req.MeetingAt)
			if err != nil {
				return validationError(c, "meetingAt", "must be ISO 8601 format")
			}
			meetingAt = &t
		}

		// Build update query dynamically based on provided fields
		ctx := c.Request().Context()

		if req.MeetingAt != nil && req.MeetURL != nil {
			_, err := pool.Exec(ctx,
				`UPDATE meeting_settings SET meeting_at = $1, meet_url = $2, updated_at = NOW() WHERE id = 1`,
				meetingAt, *req.MeetURL)
			if err != nil {
				return internalError(c)
			}
		} else if req.MeetingAt != nil {
			_, err := pool.Exec(ctx,
				`UPDATE meeting_settings SET meeting_at = $1, updated_at = NOW() WHERE id = 1`,
				meetingAt)
			if err != nil {
				return internalError(c)
			}
		} else if req.MeetURL != nil {
			_, err := pool.Exec(ctx,
				`UPDATE meeting_settings SET meet_url = $1, updated_at = NOW() WHERE id = 1`,
				*req.MeetURL)
			if err != nil {
				return internalError(c)
			}
		}

		// Return updated meeting
		var retMeetingAt *time.Time
		var retMeetURL string
		var retUpdatedAt time.Time

		err := pool.QueryRow(ctx,
			`SELECT meeting_at, meet_url, updated_at FROM meeting_settings WHERE id = 1`).
			Scan(&retMeetingAt, &retMeetURL, &retUpdatedAt)
		if err != nil {
			return internalError(c)
		}

		resp := MeetingResponse{
			MeetURL:   retMeetURL,
			UpdatedAt: retUpdatedAt.Format(time.RFC3339),
		}
		if retMeetingAt != nil {
			s := retMeetingAt.Format(time.RFC3339)
			resp.MeetingAt = &s
		}

		return c.JSON(http.StatusOK, resp)
	}
}
