package simpleapi

import (
	"net/http"
	"time"
	"unicode/utf8"

	"cloud.google.com/go/firestore"
	"github.com/labstack/echo/v4"
)

// TemplateResponse matches the spec in docs/backend-api-request.md §4.3.
type TemplateResponse struct {
	Text      string `json:"text" firestore:"text"`
	UpdatedAt string `json:"updatedAt" firestore:"updatedAt"`
}

// ChannelResponse matches the spec in docs/backend-api-request.md §4.4.
type ChannelResponse struct {
	ID        string `json:"id" firestore:"id"`
	Name      string `json:"name" firestore:"name"`
	Note      string `json:"note" firestore:"note"`
	State     string `json:"state" firestore:"state"`
	UpdatedAt string `json:"updatedAt" firestore:"updatedAt"`
}

const (
	templateCollection = "publicity_template"
	templateDocID      = "default"
	channelsCollection = "publicity_channels"
)

// RegisterPublicityRoutes registers Publicity API endpoints.
func RegisterPublicityRoutes(g *echo.Group, client *firestore.Client) {
	g.GET("/publicity/template", getTemplateHandler(client))
	g.PATCH("/publicity/template", patchTemplateHandler(client))
	g.GET("/publicity/channels", getChannelsHandler(client))
	g.PATCH("/publicity/channels/:channelId/state", patchChannelStateHandler(client))
}

func getTemplateHandler(client *firestore.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		doc, err := client.Collection(templateCollection).Doc(templateDocID).Get(c.Request().Context())
		if err != nil {
			return c.JSON(http.StatusOK, TemplateResponse{
				Text:      "",
				UpdatedAt: time.Now().Format(time.RFC3339),
			})
		}

		var resp TemplateResponse
		if err := doc.DataTo(&resp); err != nil {
			return internalError(c)
		}
		return c.JSON(http.StatusOK, resp)
	}
}

func patchTemplateHandler(client *firestore.Client) echo.HandlerFunc {
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
		data := TemplateResponse{Text: req.Text, UpdatedAt: now}

		ref := client.Collection(templateCollection).Doc(templateDocID)
		if _, err := ref.Set(ctx, data); err != nil {
			return internalError(c)
		}
		return c.JSON(http.StatusOK, data)
	}
}

func getChannelsHandler(client *firestore.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		docs, err := client.Collection(channelsCollection).Documents(c.Request().Context()).GetAll()
		if err != nil || len(docs) == 0 {
			return c.JSON(http.StatusOK, map[string]interface{}{"channels": defaultChannels()})
		}

		channels := make([]ChannelResponse, 0, len(docs))
		for _, doc := range docs {
			var ch ChannelResponse
			if err := doc.DataTo(&ch); err != nil {
				continue
			}
			channels = append(channels, ch)
		}
		if len(channels) == 0 {
			channels = defaultChannels()
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"channels": channels})
	}
}

func patchChannelStateHandler(client *firestore.Client) echo.HandlerFunc {
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
		ref := client.Collection(channelsCollection).Doc(channelID)

		if _, err := ref.Get(ctx); err != nil {
			if isNotFound(err) {
				return notFoundError(c, "channel not found")
			}
			return internalError(c)
		}

		if _, err := ref.Update(ctx, []firestore.Update{
			{Path: "state", Value: req.State},
			{Path: "updatedAt", Value: now},
		}); err != nil {
			return internalError(c)
		}

		doc, err := ref.Get(ctx)
		if err != nil {
			return internalError(c)
		}
		var ch ChannelResponse
		if err := doc.DataTo(&ch); err != nil {
			return internalError(c)
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
