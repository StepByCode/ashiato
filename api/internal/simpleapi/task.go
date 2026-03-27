package simpleapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/dokkiitech/ashiato/api/internal/firebase"
)

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
	"kido":     true,
	"kitahara": true,
	"sogo":     true,
	"nakai":    true,
}

var validStates = map[string]bool{
	"in_progress": true,
	"done":        true,
	"approved":    true,
}

const tasksCollection = "simple_tasks"

func RegisterTaskRoutes(g *echo.Group, fs *firebase.Firestore) {
	g.GET("/tasks", getTasksHandler(fs))
	g.POST("/tasks", createTaskHandler(fs))
	g.PATCH("/tasks/:taskId/owner", patchTaskOwnerHandler(fs))
	g.PATCH("/tasks/:taskId/url", patchTaskURLHandler(fs))
	g.PATCH("/tasks/:taskId/state", patchTaskStateHandler(fs))
}

func docToTask(doc *firebase.Document) TaskResponse {
	f := doc.Fields
	return TaskResponse{
		ID:        firebase.GetStringField(f, "id"),
		Title:     firebase.GetStringField(f, "title"),
		Owner:     firebase.GetStringField(f, "owner"),
		State:     firebase.GetStringField(f, "state"),
		URL:       firebase.GetStringField(f, "url"),
		CreatedAt: firebase.GetStringField(f, "createdAt"),
		UpdatedAt: firebase.GetStringField(f, "updatedAt"),
	}
}

func taskFields(id, title, owner, state, url string, createdAt, updatedAt time.Time) map[string]interface{} {
	return map[string]interface{}{
		"id":        firebase.StringVal(id),
		"title":     firebase.StringVal(title),
		"owner":     firebase.StringVal(owner),
		"state":     firebase.StringVal(state),
		"url":       firebase.StringVal(url),
		"createdAt": firebase.StringVal(createdAt.Format(time.RFC3339)),
		"updatedAt": firebase.StringVal(updatedAt.Format(time.RFC3339)),
	}
}

func getTasksHandler(fs *firebase.Firestore) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		month := c.QueryParam("month")

		docs, err := fs.List(ctx, tasksCollection)
		if err != nil {
			return internalError(c)
		}

		tasks := make([]TaskResponse, 0, len(docs))
		for i := range docs {
			t := docToTask(&docs[i])

			// Filter by month if specified
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

func createTaskHandler(fs *firebase.Firestore) echo.HandlerFunc {
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
		now := time.Now()
		fields := taskFields(id, title, req.Owner, "in_progress", "", now, now)

		_, err := fs.Set(c.Request().Context(), tasksCollection, id, fields)
		if err != nil {
			return internalError(c)
		}

		task := TaskResponse{
			ID:        id,
			Title:     title,
			Owner:     req.Owner,
			State:     "in_progress",
			URL:       "",
			CreatedAt: now.Format(time.RFC3339),
			UpdatedAt: now.Format(time.RFC3339),
		}

		return c.JSON(http.StatusCreated, map[string]interface{}{"task": task})
	}
}

func patchTaskOwnerHandler(fs *firebase.Firestore) echo.HandlerFunc {
	type request struct {
		Owner string `json:"owner"`
	}

	return func(c echo.Context) error {
		taskID := c.Param("taskId")
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}
		if !validOwners[req.Owner] {
			return validationError(c, "owner", "is required")
		}

		task, err := updateTaskField(c, fs, taskID, "owner", req.Owner)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"task": task})
	}
}

func patchTaskURLHandler(fs *firebase.Firestore) echo.HandlerFunc {
	type request struct {
		URL string `json:"url"`
	}

	return func(c echo.Context) error {
		taskID := c.Param("taskId")
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}
		if len(req.URL) > 2048 {
			return validationError(c, "url", "must be at most 2048 characters")
		}

		task, err := updateTaskField(c, fs, taskID, "url", req.URL)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"task": task})
	}
}

func patchTaskStateHandler(fs *firebase.Firestore) echo.HandlerFunc {
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

		// Get current task to check state transition
		ctx := c.Request().Context()
		doc, err := fs.Get(ctx, tasksCollection, taskID)
		if err != nil {
			if errors.Is(err, firebase.ErrNotFound) {
				return notFoundError(c, "task not found")
			}
			return internalError(c)
		}
		currentState := firebase.GetStringField(doc.Fields, "state")

		if currentState == "approved" {
			return conflictError(c, "cannot transition from approved state")
		}

		// Allowed transitions: in_progress->done, done->in_progress, done->approved
		allowed := false
		switch {
		case currentState == "in_progress" && req.State == "done":
			allowed = true
		case currentState == "done" && req.State == "in_progress":
			allowed = true
		case currentState == "done" && req.State == "approved":
			allowed = true
		}
		if !allowed {
			return conflictError(c, fmt.Sprintf("cannot transition from %s to %s", currentState, req.State))
		}

		task, err := updateTaskField(c, fs, taskID, "state", req.State)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"task": task})
	}
}

func updateTaskField(c echo.Context, fs *firebase.Firestore, taskID, field, value string) (*TaskResponse, error) {
	ctx := c.Request().Context()

	now := time.Now()
	fields := map[string]interface{}{
		field:       firebase.StringVal(value),
		"updatedAt": firebase.StringVal(now.Format(time.RFC3339)),
	}

	doc, err := fs.Patch(ctx, tasksCollection, taskID, fields, []string{field, "updatedAt"})
	if err != nil {
		if errors.Is(err, firebase.ErrNotFound) {
			return nil, notFoundError(c, "task not found")
		}
		return nil, internalError(c)
	}

	task := docToTask(doc)
	return &task, nil
}
