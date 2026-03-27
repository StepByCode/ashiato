package simpleapi

import (
	"net/http"
	"strings"
	"time"

	"firebase.google.com/go/v4/db"
	"github.com/labstack/echo/v4"
)

// ProfileResponse is the user profile stored in the database.
type ProfileResponse struct {
	UID       string `json:"uid"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Position  string `json:"position"`
	UpdatedAt string `json:"updatedAt"`
}

const profilesCollection = "simple_profiles"

// RegisterProfileRoutes registers profile API endpoints.
func RegisterProfileRoutes(g *echo.Group, client *db.Client) {
	g.GET("/profile/:uid", getProfileHandler(client))
	g.PATCH("/profile/:uid", patchProfileHandler(client))
}

func getProfileHandler(client *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		uid := c.Param("uid")
		ref := client.NewRef(profilesCollection).Child(uid)
		var profile ProfileResponse
		if err := ref.Get(c.Request().Context(), &profile); err != nil || profile.UID == "" {
			return c.JSON(http.StatusNotFound, ErrorResponse{
				Error: ErrorBody{Code: "NOT_FOUND", Message: "profile not found"},
			})
		}
		return c.JSON(http.StatusOK, profile)
	}
}

func patchProfileHandler(client *db.Client) echo.HandlerFunc {
	type request struct {
		Name     *string `json:"name"`
		Position *string `json:"position"`
	}

	return func(c echo.Context) error {
		uid := c.Param("uid")
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}

		ctx := c.Request().Context()
		now := time.Now().Format(time.RFC3339)
		ref := client.NewRef(profilesCollection).Child(uid)

		// Check if profile exists; if not, create it
		var existing ProfileResponse
		_ = ref.Get(ctx, &existing)

		updates := map[string]interface{}{
			"uid":       uid,
			"updatedAt": now,
		}
		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				return validationError(c, "name", "is required")
			}
			updates["name"] = name
		}
		if req.Position != nil {
			updates["position"] = strings.TrimSpace(*req.Position)
		}

		if existing.UID == "" {
			// First-time setup
			data := ProfileResponse{
				UID:       uid,
				UpdatedAt: now,
			}
			if req.Name != nil {
				data.Name = strings.TrimSpace(*req.Name)
			}
			if req.Position != nil {
				data.Position = strings.TrimSpace(*req.Position)
			}
			if err := ref.Set(ctx, data); err != nil {
				return internalError(c)
			}
			return c.JSON(http.StatusOK, data)
		}

		if err := ref.Update(ctx, updates); err != nil {
			return internalError(c)
		}

		var profile ProfileResponse
		if err := ref.Get(ctx, &profile); err != nil {
			return internalError(c)
		}
		return c.JSON(http.StatusOK, profile)
	}
}
