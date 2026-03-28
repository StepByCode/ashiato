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

	"github.com/dokkiitech/ashiato/api/internal/discord"
	"github.com/dokkiitech/ashiato/api/internal/domain"
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
	{Title: "connpass URL", AssigneeRequired: true},
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

type taskUserDoc struct {
	FirebaseUID string `json:"firebase_uid"`
	Name        string `json:"name"`
	Email       string `json:"email"`
}

func lookupProfileName(ctx context.Context, client *db.Client, firebaseUID string) string {
	if firebaseUID == "" {
		return ""
	}
	var profile struct {
		Name string `json:"name"`
	}
	if err := client.NewRef(profilesCollection).Child(firebaseUID).Get(ctx, &profile); err != nil {
		return ""
	}
	return strings.TrimSpace(profile.Name)
}

func actorDisplayName(ctx context.Context, client *db.Client, actor domain.Actor) string {
	if name := lookupProfileName(ctx, client, actor.Subject); name != "" {
		return name
	}
	if strings.TrimSpace(actor.Name) != "" && !strings.EqualFold(strings.TrimSpace(actor.Name), strings.TrimSpace(actor.Email)) {
		return strings.TrimSpace(actor.Name)
	}
	return strings.TrimSpace(actor.Email)
}

func notifyTaskDone(ctx context.Context, webhook *discord.WebhookClient, actorName string, task TaskResponse) error {
	if webhook == nil {
		return nil
	}
	fields := []discord.WebhookEmbedField{
		{Name: "タスク", Value: task.Title, Inline: false},
		{Name: "実行者", Value: actorName, Inline: true},
	}
	if task.Year != 0 && task.Month != 0 {
		fields = append(fields, discord.WebhookEmbedField{
			Name: "対象月", Value: fmt.Sprintf("%d年%d月", task.Year, task.Month), Inline: true,
		})
	}
	fields = append(fields, discord.WebhookEmbedField{
		Name: "次の対応", Value: "確認の後、Approveをしてください。", Inline: false,
	})
	_, err := webhook.SendEmbed(ctx, "作成タスクが Done になりました", fmt.Sprintf("%sさんが%sをDoneにしました。", actorName, task.Title), fields)
	return err
}

func notifyTaskApproved(ctx context.Context, webhook *discord.WebhookClient, actorName string, task TaskResponse) error {
	if webhook == nil {
		return nil
	}
	fields := []discord.WebhookEmbedField{
		{Name: "タスク", Value: task.Title, Inline: false},
		{Name: "承認者", Value: actorName, Inline: true},
	}
	if task.AssigneeName != "" {
		fields = append(fields, discord.WebhookEmbedField{Name: "担当者", Value: task.AssigneeName, Inline: true})
	}
	if task.Year != 0 && task.Month != 0 {
		fields = append(fields, discord.WebhookEmbedField{
			Name: "対象月", Value: fmt.Sprintf("%d年%d月", task.Year, task.Month), Inline: true,
		})
	}
	_, err := webhook.SendEmbed(ctx, "作成タスクが Approve されました", fmt.Sprintf("%sさんが%sをApproveしました。", actorName, task.Title), fields)
	return err
}

func ensureRequiredTasks(ctx context.Context, client *db.Client, year, month int) (map[string]TaskResponse, error) {
	collection := tasksCollectionForPeriod(year, month)
	ref := client.NewRef(collection)
	var all map[string]TaskResponse
	if err := ref.Get(ctx, &all); err != nil && !strings.Contains(err.Error(), "unexpected end of JSON input") {
		return nil, err
	}
	if all == nil {
		all = map[string]TaskResponse{}
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
			return nil, err
		}
		all[id] = task
	}
	return all, nil
}

func resolveAssigneeNames(ctx context.Context, client *db.Client, tasks []TaskResponse) {
	needed := map[string]struct{}{}
	for _, task := range tasks {
		if task.AssigneeID != "" {
			needed[task.AssigneeID] = struct{}{}
		}
	}
	if len(needed) == 0 {
		return
	}

	var users map[string]taskUserDoc
	if err := client.NewRef("users").Get(ctx, &users); err != nil || len(users) == 0 {
		return
	}

	var profiles map[string]struct {
		Name string `json:"name"`
	}
	_ = client.NewRef(profilesCollection).Get(ctx, &profiles)

	names := map[string]string{}
	for userID := range needed {
		user, ok := users[userID]
		if !ok {
			continue
		}
		if user.FirebaseUID != "" {
			if profile, ok := profiles[user.FirebaseUID]; ok && strings.TrimSpace(profile.Name) != "" {
				names[userID] = strings.TrimSpace(profile.Name)
				continue
			}
		}
		if strings.TrimSpace(user.Name) != "" {
			names[userID] = strings.TrimSpace(user.Name)
		}
	}

	for index := range tasks {
		if tasks[index].AssigneeID == "" {
			continue
		}
		tasks[index].AssigneeName = names[tasks[index].AssigneeID]
	}
}

// RegisterTaskRoutes registers Task API endpoints on the given Echo group.
func RegisterTaskRoutes(g *echo.Group, client *db.Client, webhook *discord.WebhookClient) {
	g.GET("/tasks", getTasksHandler(client))
	g.POST("/tasks", createTaskHandler(client))
	g.PATCH("/tasks/:taskId/assignee", patchTaskAssigneeHandler(client))
	g.PATCH("/tasks/:taskId/url", patchTaskURLHandler(client))
	g.PATCH("/tasks/:taskId/state", patchTaskStateHandler(client, webhook))
}

func getTasksHandler(client *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		year, month := parsePeriodParams(c)
		collection := tasksCollectionForPeriod(year, month)

		var all map[string]TaskResponse
		if year != 0 && month != 0 {
			var err error
			all, err = ensureRequiredTasks(ctx, client, year, month)
			if err != nil {
				return internalErrorWithLog(c, "failed to ensure required tasks", err, "collection", collection, "year", year, "month", month)
			}
		}

		if all == nil {
			ref := client.NewRef(collection)
			if err := ref.Get(ctx, &all); err != nil || len(all) == 0 {
				if err != nil {
					slog.WarnContext(ctx, "task list fetch failed; returning empty list", "trace_id", appctx.TraceIDFromContext(ctx), "collection", collection, "year", year, "month", month, "error", err)
				}
				return c.JSON(http.StatusOK, map[string]interface{}{"tasks": []TaskResponse{}})
			}
		}
		if len(all) == 0 {
			return c.JSON(http.StatusOK, map[string]interface{}{"tasks": []TaskResponse{}})
		}

		tasks := make([]TaskResponse, 0, len(all))
		for _, t := range all {
			tasks = append(tasks, t)
		}
		resolveAssigneeNames(ctx, client, tasks)
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
			ID:         id,
			Title:      title,
			AssigneeID: req.AssigneeID,
			State:      "in_progress",
			URL:        "",
			Year:       req.Year,
			Month:      req.Month,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if err := client.NewRef(collection).Child(id).Set(c.Request().Context(), task); err != nil {
			return internalErrorWithLog(c, "task create failed", err, "collection", collection, "task_id", id, "year", req.Year, "month", req.Month)
		}
		single := []TaskResponse{task}
		resolveAssigneeNames(c.Request().Context(), client, single)
		task = single[0]

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

func patchTaskStateHandler(client *db.Client, webhook *discord.WebhookClient) echo.HandlerFunc {
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
		actor, ok := appctx.ActorFromContext(ctx)
		if !ok {
			return c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: ErrorBody{Code: "UNAUTHORIZED", Message: "sign-in is required"},
			})
		}

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
		if task.State != "done" && req.State == "done" {
			if err := notifyTaskDone(ctx, webhook, actorDisplayName(ctx, client, actor), updated); err != nil {
				return internalErrorWithLog(c, "task done notification failed", err, "task_id", taskID)
			}
		}
		if task.State != "approved" && req.State == "approved" {
			if err := notifyTaskApproved(ctx, webhook, actorDisplayName(ctx, client, actor), updated); err != nil {
				return internalErrorWithLog(c, "task approve notification failed", err, "task_id", taskID)
			}
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
		"assigneeName": "",
		"updatedAt":    now,
	}
	if err := ref.Update(ctx, updates); err != nil {
		return internalErrorWithLog(c, "task assignee update failed", err, "task_id", taskID)
	}

	var updated TaskResponse
	if err := ref.Get(ctx, &updated); err != nil {
		return internalErrorWithLog(c, "task fetch after assignee update failed", err, "task_id", taskID)
	}
	single := []TaskResponse{updated}
	resolveAssigneeNames(ctx, client, single)
	updated = single[0]

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
