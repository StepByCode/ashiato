package simpleapi

import (
	"fmt"
	"net/http"
	"time"
	"unicode/utf8"

	"firebase.google.com/go/v4/db"
	"github.com/labstack/echo/v4"
)

// TemplateResponse matches the spec in docs/backend-api-request.md §4.3.
type TemplateResponse struct {
	Text      string `json:"text"`
	UpdatedAt string `json:"updatedAt"`
}

// ChannelResponse matches the spec in docs/backend-api-request.md §4.4.
type ChannelResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Note      string `json:"note"`
	State     string `json:"state"`
	UpdatedAt string `json:"updatedAt"`
}

const (
	templateCollection = "publicity_template"
	channelsCollection = "publicity_channels"
)

func templateDocID(year, month int) string {
	if year == 0 || month == 0 {
		return "default"
	}
	return fmt.Sprintf("%d_%02d", year, month)
}

func channelsCollectionForPeriod(year, month int) string {
	if year == 0 || month == 0 {
		return channelsCollection
	}
	return fmt.Sprintf("%s_%d_%02d", channelsCollection, year, month)
}

// RegisterPublicityRoutes registers Publicity API endpoints.
func RegisterPublicityRoutes(g *echo.Group, client *db.Client) {
	g.GET("/publicity/template", getTemplateHandler(client))
	g.PATCH("/publicity/template", patchTemplateHandler(client))
	g.GET("/publicity/channels", getChannelsHandler(client))
	g.PATCH("/publicity/channels/:channelId/state", patchChannelStateHandler(client))
}

func getTemplateHandler(client *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		year, month := parsePeriodParams(c)
		docID := templateDocID(year, month)
		ref := client.NewRef(templateCollection).Child(docID)
		var resp TemplateResponse
		if err := ref.Get(c.Request().Context(), &resp); err != nil || resp.UpdatedAt == "" {
			return c.JSON(http.StatusOK, TemplateResponse{
				Text:      "",
				UpdatedAt: time.Now().Format(time.RFC3339),
			})
		}
		return c.JSON(http.StatusOK, resp)
	}
}

func patchTemplateHandler(client *db.Client) echo.HandlerFunc {
	type request struct {
		Text  string `json:"text"`
		Year  int    `json:"year"`
		Month int    `json:"month"`
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
		docID := templateDocID(req.Year, req.Month)
		data := TemplateResponse{Text: req.Text, UpdatedAt: now}

		ref := client.NewRef(templateCollection).Child(docID)
		if err := ref.Set(ctx, data); err != nil {
			return internalError(c)
		}
		return c.JSON(http.StatusOK, data)
	}
}

func getChannelsHandler(client *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		year, month := parsePeriodParams(c)
		collection := channelsCollectionForPeriod(year, month)
		ref := client.NewRef(collection)
		var all map[string]ChannelResponse
		if err := ref.Get(c.Request().Context(), &all); err != nil || len(all) == 0 {
			// Initialize default channels for this period.
			defaults := defaultChannels()
			for _, ch := range defaults {
				_ = client.NewRef(collection).Child(ch.ID).Set(c.Request().Context(), ch)
			}
			return c.JSON(http.StatusOK, map[string]interface{}{"channels": defaults})
		}

		channels := make([]ChannelResponse, 0, len(all))
		for _, ch := range all {
			channels = append(channels, ch)
		}
		if len(channels) == 0 {
			channels = defaultChannels()
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"channels": channels})
	}
}

func patchChannelStateHandler(client *db.Client) echo.HandlerFunc {
	type request struct {
		State string `json:"state"`
		Year  int    `json:"year"`
		Month int    `json:"month"`
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
		collection := channelsCollectionForPeriod(req.Year, req.Month)
		ref := client.NewRef(collection).Child(channelID)

		var existing ChannelResponse
		if err := ref.Get(ctx, &existing); err != nil || existing.ID == "" {
			return notFoundError(c, "channel not found")
		}

		updates := map[string]interface{}{
			"state":     req.State,
			"updatedAt": now,
		}
		if err := ref.Update(ctx, updates); err != nil {
			return internalError(c)
		}

		var ch ChannelResponse
		if err := ref.Get(ctx, &ch); err != nil {
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
