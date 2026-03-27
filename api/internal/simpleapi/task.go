package simpleapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TaskResponse matches the spec in docs/backend-api-request.md §4.1.
type TaskResponse struct {
	ID        string `json:"id" firestore:"id"`
	Title     string `json:"title" firestore:"title"`
	Owner     string `json:"owner" firestore:"owner"`
	State     string `json:"state" firestore:"state"`
	URL       string `json:"url" firestore:"url"`
	CreatedAt string `json:"createdAt" firestore:"createdAt"`
	UpdatedAt string `json:"updatedAt" firestore:"updatedAt"`
}

var validOwners = map[string]bool{
	"kido": true, "kitahara": true, "sogo": true, "nakai": true,
}

var validStates = map[string]bool{
	"in_progress": true, "done": true, "approved": true,
}

const tasksCollection = "simple_tasks"

// RegisterTaskRoutes registers Task API endpoints on the given Echo group.
func RegisterTaskRoutes(g *echo.Group, client *firestore.Client) {
	g.GET("/tasks", getTasksHandler(client))
	g.POST("/tasks", createTaskHandler(client))
	g.PATCH("/tasks/:taskId/owner", patchTaskOwnerHandler(client))
	g.PATCH("/tasks/:taskId/url", patchTaskURLHandler(client))
	g.PATCH("/tasks/:taskId/state", patchTaskStateHandler(client))
}

func getTasksHandler(client *firestore.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		month := c.QueryParam("month")

		q := client.Collection(tasksCollection).OrderBy("createdAt", firestore.Desc)

		docs, err := q.Documents(ctx).GetAll()
		if err != nil {
			return internalError(c)
		}

		tasks := make([]TaskResponse, 0, len(docs))
		for _, doc := range docs {
			var t TaskResponse
			if err := doc.DataTo(&t); err != nil {
				continue
			}
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

func createTaskHandler(client *firestore.Client) echo.HandlerFunc {
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

		if _, err := client.Collection(tasksCollection).Doc(id).Set(c.Request().Context(), task); err != nil {
			return internalError(c)
		}

		return c.JSON(http.StatusCreated, map[string]interface{}{"task": task})
	}
}

func patchTaskOwnerHandler(client *firestore.Client) echo.HandlerFunc {
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

func patchTaskURLHandler(client *firestore.Client) echo.HandlerFunc {
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

func patchTaskStateHandler(client *firestore.Client) echo.HandlerFunc {
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
		doc, err := client.Collection(tasksCollection).Doc(taskID).Get(ctx)
		if err != nil {
			if isNotFound(err) {
				return notFoundError(c, "task not found")
			}
			return internalError(c)
		}

		var current TaskResponse
		if err := doc.DataTo(&current); err != nil {
			return internalError(c)
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

func updateTaskField(c echo.Context, client *firestore.Client, taskID, field, value string) error {
	ctx := c.Request().Context()
	now := time.Now().Format(time.RFC3339)

	ref := client.Collection(tasksCollection).Doc(taskID)
	_, err := ref.Update(ctx, []firestore.Update{
		{Path: field, Value: value},
		{Path: "updatedAt", Value: now},
	})
	if err != nil {
		if isNotFound(err) {
			return notFoundError(c, "task not found")
		}
		return internalError(c)
	}

	doc, err := ref.Get(ctx)
	if err != nil {
		return internalError(c)
	}
	var task TaskResponse
	if err := doc.DataTo(&task); err != nil {
		return internalError(c)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"task": task})
}

func isNotFound(err error) bool {
	return status.Code(err) == codes.NotFound || errors.Is(err, context.Canceled)
}
