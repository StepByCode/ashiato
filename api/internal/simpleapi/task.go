package simpleapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
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

func RegisterTaskRoutes(g *echo.Group, pool *pgxpool.Pool) {
	g.GET("/tasks", getTasksHandler(pool))
	g.POST("/tasks", createTaskHandler(pool))
	g.PATCH("/tasks/:taskId/owner", patchTaskOwnerHandler(pool))
	g.PATCH("/tasks/:taskId/url", patchTaskURLHandler(pool))
	g.PATCH("/tasks/:taskId/state", patchTaskStateHandler(pool))
}

func scanTask(row pgx.Row) (*TaskResponse, error) {
	var t TaskResponse
	var createdAt, updatedAt time.Time
	err := row.Scan(&t.ID, &t.Title, &t.Owner, &t.State, &t.URL, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	t.CreatedAt = createdAt.Format(time.RFC3339)
	t.UpdatedAt = updatedAt.Format(time.RFC3339)
	return &t, nil
}

func getTasksHandler(pool *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		month := c.QueryParam("month")

		var rows pgx.Rows
		var err error
		if month != "" {
			// month format: "2026-02" → extract year and month
			var y, m int
			if _, parseErr := fmt.Sscanf(month, "%d-%d", &y, &m); parseErr != nil {
				return validationError(c, "month", "must be YYYY-MM format")
			}
			start := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
			end := start.AddDate(0, 1, 0)
			rows, err = pool.Query(ctx,
				`SELECT id, title, owner, state, url, created_at, updated_at
				 FROM simple_tasks
				 WHERE created_at >= $1 AND created_at < $2
				 ORDER BY created_at DESC`, start, end)
		} else {
			rows, err = pool.Query(ctx,
				`SELECT id, title, owner, state, url, created_at, updated_at
				 FROM simple_tasks
				 ORDER BY created_at DESC`)
		}
		if err != nil {
			return internalError(c)
		}
		defer rows.Close()

		tasks := make([]TaskResponse, 0)
		for rows.Next() {
			var t TaskResponse
			var createdAt, updatedAt time.Time
			if err := rows.Scan(&t.ID, &t.Title, &t.Owner, &t.State, &t.URL, &createdAt, &updatedAt); err != nil {
				return internalError(c)
			}
			t.CreatedAt = createdAt.Format(time.RFC3339)
			t.UpdatedAt = updatedAt.Format(time.RFC3339)
			tasks = append(tasks, t)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{"tasks": tasks})
	}
}

func createTaskHandler(pool *pgxpool.Pool) echo.HandlerFunc {
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

		row := pool.QueryRow(c.Request().Context(),
			`INSERT INTO simple_tasks (id, title, owner, state, url, created_at, updated_at)
			 VALUES ($1, $2, $3, 'in_progress', '', $4, $4)
			 RETURNING id, title, owner, state, url, created_at, updated_at`,
			id, title, req.Owner, now)

		task, err := scanTask(row)
		if err != nil {
			return internalError(c)
		}

		return c.JSON(http.StatusCreated, map[string]interface{}{"task": task})
	}
}

func patchTaskOwnerHandler(pool *pgxpool.Pool) echo.HandlerFunc {
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

		task, err := updateTaskField(c.Request().Context(), pool, taskID, "owner", req.Owner)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"task": task})
	}
}

func patchTaskURLHandler(pool *pgxpool.Pool) echo.HandlerFunc {
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

		task, err := updateTaskField(c.Request().Context(), pool, taskID, "url", req.URL)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"task": task})
	}
}

func patchTaskStateHandler(pool *pgxpool.Pool) echo.HandlerFunc {
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

		// Check current state for transition rules
		var currentState string
		err := pool.QueryRow(c.Request().Context(),
			`SELECT state FROM simple_tasks WHERE id = $1`, taskID).Scan(&currentState)
		if err != nil {
			if err == pgx.ErrNoRows {
				return notFoundError(c, "task not found")
			}
			return internalError(c)
		}

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

		task, err := updateTaskField(c.Request().Context(), pool, taskID, "state", req.State)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"task": task})
	}
}

// updateTaskField updates a single column and returns the updated task.
// Returns an echo error response on failure.
func updateTaskField(ctx context.Context, pool *pgxpool.Pool, taskID, column, value string) (*TaskResponse, error) {
	// Use parameterized column name via a safe allowlist
	var query string
	switch column {
	case "owner":
		query = `UPDATE simple_tasks SET owner = $1, updated_at = NOW() WHERE id = $2
				 RETURNING id, title, owner, state, url, created_at, updated_at`
	case "url":
		query = `UPDATE simple_tasks SET url = $1, updated_at = NOW() WHERE id = $2
				 RETURNING id, title, owner, state, url, created_at, updated_at`
	case "state":
		query = `UPDATE simple_tasks SET state = $1, updated_at = NOW() WHERE id = $2
				 RETURNING id, title, owner, state, url, created_at, updated_at`
	default:
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "invalid column")
	}

	row := pool.QueryRow(ctx, query, value, taskID)
	task, err := scanTask(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, echo.NewHTTPError(http.StatusNotFound, "task not found")
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	return task, nil
}
