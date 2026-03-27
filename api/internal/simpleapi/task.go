package simpleapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"firebase.google.com/go/v4/db"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	appctx "github.com/dokkiitech/ashiato/api/internal/middleware"
)

// TaskResponse matches the spec in docs/backend-api-request.md §4.1.
type TaskResponse struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	AssigneeID   string `json:"assigneeId,omitempty"`
	AssigneeName string `json:"assigneeName,omitempty"`
	State        string `json:"state"`
	URL          string `json:"url"`
	Year         int    `json:"year,omitempty"`
	Month        int    `json:"month,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

var validStates = map[string]bool{
	"in_progress": true, "done": true, "approved": true,
}

const tasksCollection = "simple_tasks"

var requiredTasks = []struct {
	Title              string
	AssigneeRequired   bool
	FixedWithoutAssign bool
}{
	{Title: "イベント名", AssigneeRequired: false, FixedWithoutAssign: true},
	{Title: "connpassURL", AssigneeRequired: true},
	{Title: "Place", AssigneeRequired: true},
}

func tasksCollectionForPeriod(year, month int) string {
	if year == 0 || month == 0 {
		return tasksCollection
	}
	return fmt.Sprintf("%s_%d_%02d", tasksCollection, year, month)
}

func requiredTaskID(title string) string {
	return "required-" + strings.ToLower(strings.ReplaceAll(title, " ", "-"))
}

func taskAllowsEmptyAssignee(title string) bool {
	for _, required := range requiredTasks {
		if required.Title == title {
			return !required.AssigneeRequired
		}
	}
	return false
}

func lookupMemberName(ctx context.Context, client *db.Client, assigneeID string) string {
	if assigneeID == "" {
		return ""
	}
	var user struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := client.NewRef("users").Child(assigneeID).Get(ctx, &user); err != nil {
		return ""
	}
	if user.Name != "" {
		return user.Name
	}
	return user.Email
}

func ensureRequiredTasks(ctx context.Context, client *db.Client, year, month int) error {
	collection := tasksCollectionForPeriod(year, month)
	ref := client.NewRef(collection)
	var all map[string]TaskResponse
	if err := ref.Get(ctx, &all); err != nil && !strings.Contains(err.Error(), "unexpected end of JSON input") {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	for _, required := range requiredTasks {
		id := requiredTaskID(required.Title)
		if existing, ok := all[id]; ok && existing.ID != "" {
			continue
		}

		task := TaskResponse{
			ID:        id,
			Title:     required.Title,
			State:     "in_progress",
			URL:       "",
			Year:      year,
			Month:     month,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := ref.Child(id).Set(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

// RegisterTaskRoutes registers Task API endpoints on the given Echo group.
func RegisterTaskRoutes(g *echo.Group, client *db.Client) {
	g.GET("/tasks", getTasksHandler(client))
	g.POST("/tasks", createTaskHandler(client))
	g.PATCH("/tasks/:taskId/assignee", patchTaskAssigneeHandler(client))
	g.PATCH("/tasks/:taskId/url", patchTaskURLHandler(client))
	g.PATCH("/tasks/:taskId/state", patchTaskStateHandler(client))
}

func getTasksHandler(client *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		year, month := parsePeriodParams(c)
		collection := tasksCollectionForPeriod(year, month)

		if year != 0 && month != 0 {
			if err := ensureRequiredTasks(ctx, client, year, month); err != nil {
				return internalErrorWithLog(c, "failed to ensure required tasks", err, "collection", collection, "year", year, "month", month)
			}
		}

		ref := client.NewRef(collection)
		var all map[string]TaskResponse
		if err := ref.Get(ctx, &all); err != nil || len(all) == 0 {
			if err != nil {
				slog.WarnContext(ctx, "task list fetch failed; returning empty list", "trace_id", appctx.TraceIDFromContext(ctx), "collection", collection, "year", year, "month", month, "error", err)
			}
			return c.JSON(http.StatusOK, map[string]interface{}{"tasks": []TaskResponse{}})
		}

		tasks := make([]TaskResponse, 0, len(all))
		for _, t := range all {
			if t.AssigneeID != "" && t.AssigneeName == "" {
				t.AssigneeName = lookupMemberName(ctx, client, t.AssigneeID)
			}
			tasks = append(tasks, t)
		}
		sort.Slice(tasks, func(i, j int) bool {
			leftRequired := strings.HasPrefix(tasks[i].ID, "required-")
			rightRequired := strings.HasPrefix(tasks[j].ID, "required-")
			if leftRequired != rightRequired {
				return leftRequired
			}
			return tasks[i].CreatedAt < tasks[j].CreatedAt
		})

		return c.JSON(http.StatusOK, map[string]interface{}{"tasks": tasks})
	}
}

func createTaskHandler(client *db.Client) echo.HandlerFunc {
	type request struct {
		Title      string `json:"title"`
		AssigneeID string `json:"assigneeId"`
		Year       int    `json:"year"`
		Month      int    `json:"month"`
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
		if req.AssigneeID == "" && !taskAllowsEmptyAssignee(title) {
			return validationError(c, "assigneeId", "is required")
		}

		collection := tasksCollectionForPeriod(req.Year, req.Month)
		id := "task-" + uuid.New().String()[:8]
		now := time.Now().Format(time.RFC3339)
		task := TaskResponse{
			ID:           id,
			Title:        title,
			AssigneeID:   req.AssigneeID,
			AssigneeName: lookupMemberName(c.Request().Context(), client, req.AssigneeID),
			State:        "in_progress",
			URL:          "",
			Year:         req.Year,
			Month:        req.Month,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := client.NewRef(collection).Child(id).Set(c.Request().Context(), task); err != nil {
			return internalErrorWithLog(c, "task create failed", err, "collection", collection, "task_id", id, "year", req.Year, "month", req.Month)
		}

		return c.JSON(http.StatusCreated, map[string]interface{}{"task": task})
	}
}

func patchTaskAssigneeHandler(client *db.Client) echo.HandlerFunc {
	type request struct {
		AssigneeID string `json:"assigneeId"`
	}
	return func(c echo.Context) error {
		taskID := c.Param("taskId")
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}

		task, _ := findTask(c.Request().Context(), client, taskID)
		if task == nil {
			return notFoundError(c, "task not found")
		}
		if req.AssigneeID == "" && !taskAllowsEmptyAssignee(task.Title) {
			return validationError(c, "assigneeId", "is required")
		}
		return updateTaskAssignee(c, client, taskID, req.AssigneeID)
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

		// Search for the task across all collections (current and period-specific).
		task, ref := findTask(ctx, client, taskID)
		if task == nil {
			return notFoundError(c, "task not found")
		}

		if task.State == "approved" {
			return conflictError(c, "cannot transition from approved state")
		}

		allowed := (task.State == "in_progress" && req.State == "done") ||
			(task.State == "done" && req.State == "in_progress") ||
			(task.State == "done" && req.State == "approved")
		if !allowed {
			return conflictError(c, fmt.Sprintf("cannot transition from %s to %s", task.State, req.State))
		}

		now := time.Now().Format(time.RFC3339)
		updates := map[string]interface{}{
			"state":     req.State,
			"updatedAt": now,
		}
		if err := ref.Update(ctx, updates); err != nil {
			return internalErrorWithLog(c, "task state update failed", err, "task_id", taskID)
		}

		var updated TaskResponse
		if err := ref.Get(ctx, &updated); err != nil {
			return internalErrorWithLog(c, "task fetch after state update failed", err, "task_id", taskID)
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"task": updated})
	}
}

func updateTaskField(c echo.Context, client *db.Client, taskID, field, value string) error {
	ctx := c.Request().Context()
	now := time.Now().Format(time.RFC3339)

	// Search for the task across collections.
	task, ref := findTask(ctx, client, taskID)
	if task == nil {
		return notFoundError(c, "task not found")
	}

	updates := map[string]interface{}{
		field:       value,
		"updatedAt": now,
	}
	if err := ref.Update(ctx, updates); err != nil {
		return internalErrorWithLog(c, "task field update failed", err, "task_id", taskID, "field", field)
	}

	var updated TaskResponse
	if err := ref.Get(ctx, &updated); err != nil {
		return internalErrorWithLog(c, "task fetch after field update failed", err, "task_id", taskID, "field", field)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"task": updated})
}

func updateTaskAssignee(c echo.Context, client *db.Client, taskID, assigneeID string) error {
	ctx := c.Request().Context()
	now := time.Now().Format(time.RFC3339)

	task, ref := findTask(ctx, client, taskID)
	if task == nil {
		return notFoundError(c, "task not found")
	}

	updates := map[string]interface{}{
		"assigneeId":   assigneeID,
		"assigneeName": lookupMemberName(ctx, client, assigneeID),
		"updatedAt":    now,
	}
	if err := ref.Update(ctx, updates); err != nil {
		return internalErrorWithLog(c, "task assignee update failed", err, "task_id", taskID)
	}

	var updated TaskResponse
	if err := ref.Get(ctx, &updated); err != nil {
		return internalErrorWithLog(c, "task fetch after assignee update failed", err, "task_id", taskID)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"task": updated})
}

// findTask searches for a task first in the default collection, then looks it up
// by checking if the ID exists. For period-based tasks, the task ID contains
// the info needed. We check the default collection first for backward compat.
func findTask(ctx context.Context, client *db.Client, taskID string) (*TaskResponse, *db.Ref) {
	// Try default collection first.
	ref := client.NewRef(tasksCollection).Child(taskID)
	var task TaskResponse
	if err := ref.Get(ctx, &task); err == nil && task.ID != "" {
		return &task, ref
	}

	// Search in period-based collections by scanning the root for matching collections.
	// Since Firebase RTDB doesn't support listing collections easily, we check
	// if the task has year/month info embedded. For now, scan a reasonable range.
	now := time.Now()
	for offset := -2; offset <= 3; offset++ {
		t := now.AddDate(0, offset, 0)
		collection := tasksCollectionForPeriod(t.Year(), int(t.Month()))
		ref = client.NewRef(collection).Child(taskID)
		if err := ref.Get(ctx, &task); err == nil && task.ID != "" {
			return &task, ref
		}
	}

	return nil, nil
}
