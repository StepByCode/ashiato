package simpleapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/dokkiitech/ashiato/api/internal/firebase"
)

type MeetingResponse struct {
	MeetingAt *string `json:"meetingAt"`
	MeetURL   string  `json:"meetUrl"`
	UpdatedAt string  `json:"updatedAt"`
}

const (
	meetingCollection = "meeting_settings"
	meetingDocID      = "default"
)

func RegisterMeetingRoutes(g *echo.Group, fs *firebase.Firestore) {
	g.GET("/meeting", getMeetingHandler(fs))
	g.PATCH("/meeting", patchMeetingHandler(fs))
}

func docToMeeting(doc *firebase.Document) MeetingResponse {
	f := doc.Fields
	resp := MeetingResponse{
		MeetURL:   firebase.GetStringField(f, "meetUrl"),
		UpdatedAt: firebase.GetStringField(f, "updatedAt"),
	}
	if s := firebase.GetStringField(f, "meetingAt"); s != "" {
		resp.MeetingAt = &s
	}
	return resp
}

func getMeetingHandler(fs *firebase.Firestore) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		doc, err := fs.Get(ctx, meetingCollection, meetingDocID)
		if err != nil {
			// If document doesn't exist yet, return empty defaults
			now := time.Now().Format(time.RFC3339)
			return c.JSON(http.StatusOK, MeetingResponse{
				MeetURL:   "",
				UpdatedAt: now,
			})
		}

		return c.JSON(http.StatusOK, docToMeeting(doc))
	}
}

func patchMeetingHandler(fs *firebase.Firestore) echo.HandlerFunc {
	type request struct {
		MeetingAt *string `json:"meetingAt"`
		MeetURL   *string `json:"meetUrl"`
	}

	return func(c echo.Context) error {
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}

		// Validate meetingAt if provided
		if req.MeetingAt != nil && *req.MeetingAt != "" {
			if _, err := time.Parse(time.RFC3339, *req.MeetingAt); err != nil {
				return validationError(c, "meetingAt", "must be ISO 8601 format")
			}
		}

		ctx := c.Request().Context()
		now := time.Now().Format(time.RFC3339)
		fields := map[string]interface{}{
			"updatedAt": firebase.StringVal(now),
		}
		fieldPaths := []string{"updatedAt"}

		if req.MeetingAt != nil {
			fields["meetingAt"] = firebase.StringVal(*req.MeetingAt)
			fieldPaths = append(fieldPaths, "meetingAt")
		}
		if req.MeetURL != nil {
			fields["meetUrl"] = firebase.StringVal(*req.MeetURL)
			fieldPaths = append(fieldPaths, "meetUrl")
		}

		// Ensure document exists first (upsert pattern)
		doc, getErr := fs.Get(ctx, meetingCollection, meetingDocID)
		if getErr != nil {
			// Create with defaults + provided fields
			allFields := map[string]interface{}{
				"meetingAt": firebase.StringVal(""),
				"meetUrl":   firebase.StringVal(""),
				"updatedAt": firebase.StringVal(now),
			}
			for k, v := range fields {
				allFields[k] = v
			}
			doc, err := fs.Set(ctx, meetingCollection, meetingDocID, allFields)
			if err != nil {
				return internalError(c)
			}
			return c.JSON(http.StatusOK, docToMeeting(doc))
		}

		doc, err := fs.Patch(ctx, meetingCollection, meetingDocID, fields, fieldPaths)
		if err != nil {
			return internalError(c)
		}

		return c.JSON(http.StatusOK, docToMeeting(doc))
	}
}
