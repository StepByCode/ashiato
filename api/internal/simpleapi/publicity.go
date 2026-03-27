package simpleapi

import (
	"errors"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/labstack/echo/v4"

	"github.com/dokkiitech/ashiato/api/internal/firebase"
)

type TemplateResponse struct {
	Text      string `json:"text"`
	UpdatedAt string `json:"updatedAt"`
}

type ChannelResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Note      string `json:"note"`
	State     string `json:"state"`
	UpdatedAt string `json:"updatedAt"`
}

const (
	templateCollection = "publicity_template"
	templateDocID      = "default"
	channelsCollection = "publicity_channels"
)

func RegisterPublicityRoutes(g *echo.Group, fs *firebase.Firestore) {
	g.GET("/publicity/template", getTemplateHandler(fs))
	g.PATCH("/publicity/template", patchTemplateHandler(fs))
	g.GET("/publicity/channels", getChannelsHandler(fs))
	g.PATCH("/publicity/channels/:channelId/state", patchChannelStateHandler(fs))
}

func getTemplateHandler(fs *firebase.Firestore) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		doc, err := fs.Get(ctx, templateCollection, templateDocID)
		if err != nil {
			now := time.Now().Format(time.RFC3339)
			return c.JSON(http.StatusOK, TemplateResponse{Text: "", UpdatedAt: now})
		}

		return c.JSON(http.StatusOK, TemplateResponse{
			Text:      firebase.GetStringField(doc.Fields, "text"),
			UpdatedAt: firebase.GetStringField(doc.Fields, "updatedAt"),
		})
	}
}

func patchTemplateHandler(fs *firebase.Firestore) echo.HandlerFunc {
	type request struct {
		Text string `json:"text"`
	}

	return func(c echo.Context) error {
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}

		if utf8.RuneCountInString(req.Text) > 140 {
			return validationError(c, "text", "must be at most 140 characters")
		}

		ctx := c.Request().Context()
		now := time.Now().Format(time.RFC3339)
		fields := map[string]interface{}{
			"text":      firebase.StringVal(req.Text),
			"updatedAt": firebase.StringVal(now),
		}

		// Upsert pattern
		_, getErr := fs.Get(ctx, templateCollection, templateDocID)
		if getErr != nil {
			doc, err := fs.Set(ctx, templateCollection, templateDocID, fields)
			if err != nil {
				return internalError(c)
			}
			return c.JSON(http.StatusOK, TemplateResponse{
				Text:      firebase.GetStringField(doc.Fields, "text"),
				UpdatedAt: firebase.GetStringField(doc.Fields, "updatedAt"),
			})
		}

		doc, err := fs.Patch(ctx, templateCollection, templateDocID, fields, []string{"text", "updatedAt"})
		if err != nil {
			return internalError(c)
		}

		return c.JSON(http.StatusOK, TemplateResponse{
			Text:      firebase.GetStringField(doc.Fields, "text"),
			UpdatedAt: firebase.GetStringField(doc.Fields, "updatedAt"),
		})
	}
}

func getChannelsHandler(fs *firebase.Firestore) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		docs, err := fs.List(ctx, channelsCollection)
		if err != nil {
			// Return default channels if collection doesn't exist yet
			return c.JSON(http.StatusOK, map[string]interface{}{"channels": defaultChannels()})
		}

		if len(docs) == 0 {
			return c.JSON(http.StatusOK, map[string]interface{}{"channels": defaultChannels()})
		}

		channels := make([]ChannelResponse, 0, len(docs))
		for i := range docs {
			f := docs[i].Fields
			channels = append(channels, ChannelResponse{
				ID:        firebase.GetStringField(f, "id"),
				Name:      firebase.GetStringField(f, "name"),
				Note:      firebase.GetStringField(f, "note"),
				State:     firebase.GetStringField(f, "state"),
				UpdatedAt: firebase.GetStringField(f, "updatedAt"),
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{"channels": channels})
	}
}

func patchChannelStateHandler(fs *firebase.Firestore) echo.HandlerFunc {
	type request struct {
		State string `json:"state"`
	}

	validChannelStates := map[string]bool{
		"in_progress": true,
		"done":        true,
	}

	return func(c echo.Context) error {
		channelID := c.Param("channelId")
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}

		if !validChannelStates[req.State] {
			return validationError(c, "state", "must be in_progress or done")
		}

		ctx := c.Request().Context()
		now := time.Now().Format(time.RFC3339)

		// Check channel exists
		_, err := fs.Get(ctx, channelsCollection, channelID)
		if err != nil {
			if errors.Is(err, firebase.ErrNotFound) {
				return notFoundError(c, "channel not found")
			}
			return internalError(c)
		}

		fields := map[string]interface{}{
			"state":     firebase.StringVal(req.State),
			"updatedAt": firebase.StringVal(now),
		}

		doc, err := fs.Patch(ctx, channelsCollection, channelID, fields, []string{"state", "updatedAt"})
		if err != nil {
			return internalError(c)
		}

		ch := ChannelResponse{
			ID:        firebase.GetStringField(doc.Fields, "id"),
			Name:      firebase.GetStringField(doc.Fields, "name"),
			Note:      firebase.GetStringField(doc.Fields, "note"),
			State:     firebase.GetStringField(doc.Fields, "state"),
			UpdatedAt: firebase.GetStringField(doc.Fields, "updatedAt"),
		}

		return c.JSON(http.StatusOK, map[string]interface{}{"channel": ch})
	}
}

func defaultChannels() []ChannelResponse {
	now := time.Now().Format(time.RFC3339)
	return []ChannelResponse{
		{ID: "x", Name: "X", Note: "", State: "in_progress", UpdatedAt: now},
		{ID: "instagram", Name: "Instagram", Note: "", State: "in_progress", UpdatedAt: now},
		{ID: "facebook", Name: "Facebook", Note: "", State: "in_progress", UpdatedAt: now},
	}
}
