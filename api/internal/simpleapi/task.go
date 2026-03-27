package simpleapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"firebase.google.com/go/v4/db"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// TaskResponse matches the spec in docs/backend-api-request.md §4.1.
type TaskResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Owner     string `json:"owner"`
	State     string `json:"state"`
	URL       string `json:"url"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

var validOwners = map[string]bool{
	"kido": true, "kitahara": true, "sogo": true, "nakai": true,
}

var validStates = map[string]bool{
	"in_progress": true, "done": true, "approved": true,
}

const tasksCollection = "simple_tasks"

// RegisterTaskRoutes registers Task API endpoints on the given Echo group.
func RegisterTaskRoutes(g *echo.Group, client *db.Client) {
	g.GET("/tasks", getTasksHandler(client))
	g.POST("/tasks", createTaskHandler(client))
	g.PATCH("/tasks/:taskId/owner", patchTaskOwnerHandler(client))
	g.PATCH("/tasks/:taskId/url", patchTaskURLHandler(client))
	g.PATCH("/tasks/:taskId/state", patchTaskStateHandler(client))
}

func getTasksHandler(client *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		month := c.QueryParam("month")

		ref := client.NewRef(tasksCollection)
		var all map[string]TaskResponse
		if err := ref.OrderByChild("createdAt").Get(ctx, &all); err != nil {
			return internalError(c)
		}

		tasks := make([]TaskResponse, 0, len(all))
		for _, t := range all {
			// Filter by month if specified (format: "2026-02")
			if month != "" {
				var y, m int
				if _, parseErr := fmt.Sscanf(month, "%d-%d", &y, &m); parseErr != nil {
					return validationError(c, "month", "must be YYYY-MM format")
				}
				createdAt, err := time.Parse(time.RFC3339, t.CreatedAt)
				if err != nil {
					continue
				}
				if createdAt.Year() != y || int(createdAt.Month()) != m {
					continue
				}
			}
			tasks = append(tasks, t)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{"tasks": tasks})
	}
}

func createTaskHandler(client *db.Client) echo.HandlerFunc {
	type request struct {
		Title string `json:"title"`
		Owner string `json:"owner"`
	}

	return func(c echo.Context) error {
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}

		title := strings.TrimSpace(req.Title)
		if title == "" {
			return validationError(c, "title", "is required")
		}
		if !validOwners[req.Owner] {
			return validationError(c, "owner", "is required")
		}

		id := "task-" + uuid.New().String()[:8]
		now := time.Now().Format(time.RFC3339)
		task := TaskResponse{
			ID:        id,
			Title:     title,
			Owner:     req.Owner,
			State:     "in_progress",
			URL:       "",
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := client.NewRef(tasksCollection).Child(id).Set(c.Request().Context(), task); err != nil {
			return internalError(c)
		}

		return c.JSON(http.StatusCreated, map[string]interface{}{"task": task})
	}
}

func patchTaskOwnerHandler(client *db.Client) echo.HandlerFunc {
	type request struct {
		Owner string `json:"owner"`
	}
	return func(c echo.Context) error {
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}
		if !validOwners[req.Owner] {
			return validationError(c, "owner", "is required")
		}
		return updateTaskField(c, client, c.Param("taskId"), "owner", req.Owner)
	}
}

func patchTaskURLHandler(client *db.Client) echo.HandlerFunc {
	type request struct {
		URL string `json:"url"`
	}
	return func(c echo.Context) error {
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}
		if len(req.URL) > 2048 {
			return validationError(c, "url", "must be at most 2048 characters")
		}
		return updateTaskField(c, client, c.Param("taskId"), "url", req.URL)
	}
}

func patchTaskStateHandler(client *db.Client) echo.HandlerFunc {
	type request struct {
		State string `json:"state"`
	}
	return func(c echo.Context) error {
		taskID := c.Param("taskId")
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}
		if !validStates[req.State] {
			return validationError(c, "state", "must be in_progress, done, or approved")
		}

		ctx := c.Request().Context()
		ref := client.NewRef(tasksCollection).Child(taskID)
		var current TaskResponse
		if err := ref.Get(ctx, &current); err != nil {
			return internalError(c)
		}
		if current.ID == "" {
			return notFoundError(c, "task not found")
		}

		if current.State == "approved" {
			return conflictError(c, "cannot transition from approved state")
		}

		allowed := (current.State == "in_progress" && req.State == "done") ||
			(current.State == "done" && req.State == "in_progress") ||
			(current.State == "done" && req.State == "approved")
		if !allowed {
			return conflictError(c, fmt.Sprintf("cannot transition from %s to %s", current.State, req.State))
		}

		return updateTaskField(c, client, taskID, "state", req.State)
	}
}

func updateTaskField(c echo.Context, client *db.Client, taskID, field, value string) error {
	ctx := c.Request().Context()
	now := time.Now().Format(time.RFC3339)

	ref := client.NewRef(tasksCollection).Child(taskID)

	// Check existence first
	var existing TaskResponse
	if err := ref.Get(ctx, &existing); err != nil {
		return internalError(c)
	}
	if existing.ID == "" {
		return notFoundError(c, "task not found")
	}

	updates := map[string]interface{}{
		field:       value,
		"updatedAt": now,
	}
	if err := ref.Update(ctx, updates); err != nil {
		return internalError(c)
	}

	var task TaskResponse
	if err := ref.Get(ctx, &task); err != nil {
		return internalError(c)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"task": task})
}
