package simpleapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"firebase.google.com/go/v4/db"
	"github.com/labstack/echo/v4"

	"github.com/dokkiitech/ashiato/api/internal/discord"
	appctx2 "github.com/dokkiitech/ashiato/api/internal/middleware"
	appctx "github.com/dokkiitech/ashiato/api/internal/middleware"
)

// MeetingResponse matches the spec in docs/backend-api-request.md §4.2.
type MeetingResponse struct {
	MeetingAt *string `json:"meetingAt,omitempty"`
	MeetURL   string  `json:"meetUrl"`
	UpdatedAt string  `json:"updatedAt"`
}

const meetingCollection = "meeting_settings"

// RegisterMeetingRoutes registers Meeting API endpoints.
func RegisterMeetingRoutes(g *echo.Group, client *db.Client, webhook *discord.WebhookClient) {
	g.GET("/meeting", getMeetingHandler(client))
	g.PATCH("/meeting", patchMeetingHandler(client))
	g.POST("/meeting/share", shareMeetingHandler(client, webhook))
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
			if err != nil {
				slog.WarnContext(ctx, "meeting fetch failed; returning default", "trace_id", appctx.TraceIDFromContext(ctx), "doc_id", docID, "error", err)
			}
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
			if err != nil {
				slog.WarnContext(ctx, "meeting fetch before save failed; creating document", "trace_id", appctx.TraceIDFromContext(ctx), "doc_id", docID, "error", err)
			}
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
				return internalErrorWithLog(c, "meeting create failed", err, "doc_id", docID, "year", req.Year, "month", req.Month)
			}
			return c.JSON(http.StatusOK, data)
		}

		if err := ref.Update(ctx, updates); err != nil {
			return internalErrorWithLog(c, "meeting update failed", err, "doc_id", docID, "year", req.Year, "month", req.Month)
		}

		var resp MeetingResponse
		if err := ref.Get(ctx, &resp); err != nil {
			return internalErrorWithLog(c, "meeting fetch after update failed", err, "doc_id", docID, "year", req.Year, "month", req.Month)
		}
		return c.JSON(http.StatusOK, resp)
	}
}

func shareMeetingHandler(client *db.Client, webhook *discord.WebhookClient) echo.HandlerFunc {
	type request struct {
		Year      int     `json:"year"`
		Month     int     `json:"month"`
		MeetingAt *string `json:"meetingAt"`
		MeetURL   *string `json:"meetUrl"`
	}

	return func(c echo.Context) error {
		actor, ok := appctx2.ActorFromContext(c.Request().Context())
		if !ok {
			return c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: ErrorBody{Code: "UNAUTHORIZED", Message: "sign-in is required"},
			})
		}

		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}

		var meeting MeetingResponse
		docID := meetingDocID(req.Year, req.Month)
		if err := client.NewRef(meetingCollection).Child(docID).Get(c.Request().Context(), &meeting); err != nil || meeting.UpdatedAt == "" {
			meeting = MeetingResponse{}
		}
		if req.MeetingAt != nil {
			meeting.MeetingAt = req.MeetingAt
		}
		if req.MeetURL != nil {
			meeting.MeetURL = *req.MeetURL
		}
		if meeting.MeetingAt == nil && strings.TrimSpace(meeting.MeetURL) == "" {
			return notFoundError(c, "meeting not found")
		}

		meetingText := "未設定"
		if meeting.MeetingAt != nil && *meeting.MeetingAt != "" {
			if parsed, err := time.Parse(time.RFC3339, *meeting.MeetingAt); err == nil {
				meetingText = parsed.In(time.FixedZone("JST", 9*60*60)).Format("2006/01/02 (Mon) 15:04")
			}
		}

		meetURL := "未設定"
		if strings.TrimSpace(meeting.MeetURL) != "" {
			meetURL = strings.TrimSpace(meeting.MeetURL)
		}

		sharerName := actor.Name
		var profile ProfileResponse
		if err := client.NewRef(profilesCollection).Child(actor.Subject).Get(c.Request().Context(), &profile); err == nil {
			if strings.TrimSpace(profile.Name) != "" {
				sharerName = strings.TrimSpace(profile.Name)
			}
		}

		fields := []discord.WebhookEmbedField{
			{Name: "対象月", Value: fmt.Sprintf("%d年%d月", req.Year, req.Month), Inline: true},
			{Name: "定例日時", Value: meetingText, Inline: true},
			{Name: "Meet URL", Value: meetURL, Inline: false},
			{Name: "共有者", Value: sharerName, Inline: false},
		}
		if _, err := webhook.SendEmbed(c.Request().Context(), "定例の予定を共有", "Backstage から定例予定を共有しました。", fields); err != nil {
			return internalErrorWithLog(c, "meeting share failed", err, "year", req.Year, "month", req.Month)
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
