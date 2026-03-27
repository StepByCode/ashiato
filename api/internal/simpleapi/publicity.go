package simpleapi

import (
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
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

func RegisterPublicityRoutes(g *echo.Group, pool *pgxpool.Pool) {
	g.GET("/publicity/template", getTemplateHandler(pool))
	g.PATCH("/publicity/template", patchTemplateHandler(pool))
	g.GET("/publicity/channels", getChannelsHandler(pool))
	g.PATCH("/publicity/channels/:channelId/state", patchChannelStateHandler(pool))
}

func getTemplateHandler(pool *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		var text string
		var updatedAt time.Time

		err := pool.QueryRow(c.Request().Context(),
			`SELECT text, updated_at FROM publicity_template WHERE id = 1`).
			Scan(&text, &updatedAt)
		if err != nil {
			return internalError(c)
		}

		return c.JSON(http.StatusOK, TemplateResponse{
			Text:      text,
			UpdatedAt: updatedAt.Format(time.RFC3339),
		})
	}
}

func patchTemplateHandler(pool *pgxpool.Pool) echo.HandlerFunc {
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

		var text string
		var updatedAt time.Time

		err := pool.QueryRow(c.Request().Context(),
			`UPDATE publicity_template SET text = $1, updated_at = NOW() WHERE id = 1
			 RETURNING text, updated_at`, req.Text).
			Scan(&text, &updatedAt)
		if err != nil {
			return internalError(c)
		}

		return c.JSON(http.StatusOK, TemplateResponse{
			Text:      text,
			UpdatedAt: updatedAt.Format(time.RFC3339),
		})
	}
}

func getChannelsHandler(pool *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		rows, err := pool.Query(c.Request().Context(),
			`SELECT id, name, note, state, updated_at FROM publicity_channels ORDER BY id`)
		if err != nil {
			return internalError(c)
		}
		defer rows.Close()

		channels := make([]ChannelResponse, 0)
		for rows.Next() {
			var ch ChannelResponse
			var updatedAt time.Time
			if err := rows.Scan(&ch.ID, &ch.Name, &ch.Note, &ch.State, &updatedAt); err != nil {
				return internalError(c)
			}
			ch.UpdatedAt = updatedAt.Format(time.RFC3339)
			channels = append(channels, ch)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{"channels": channels})
	}
}

func patchChannelStateHandler(pool *pgxpool.Pool) echo.HandlerFunc {
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

		var ch ChannelResponse
		var updatedAt time.Time

		err := pool.QueryRow(c.Request().Context(),
			`UPDATE publicity_channels SET state = $1, updated_at = NOW() WHERE id = $2
			 RETURNING id, name, note, state, updated_at`, req.State, channelID).
			Scan(&ch.ID, &ch.Name, &ch.Note, &ch.State, &updatedAt)
		if err != nil {
			if err == pgx.ErrNoRows {
				return notFoundError(c, "channel not found")
			}
			return internalError(c)
		}
		ch.UpdatedAt = updatedAt.Format(time.RFC3339)

		return c.JSON(http.StatusOK, map[string]interface{}{"channel": ch})
	}
}
