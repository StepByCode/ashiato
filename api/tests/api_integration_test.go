package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/dokkiitech/ashiato/api/internal/auth"
	"github.com/dokkiitech/ashiato/api/internal/config"
	"github.com/dokkiitech/ashiato/api/internal/handler"
	"github.com/dokkiitech/ashiato/api/internal/logging"
	"github.com/dokkiitech/ashiato/api/internal/oapi"
	"github.com/dokkiitech/ashiato/api/internal/repository"
	"github.com/dokkiitech/ashiato/api/internal/testutil"
	"github.com/dokkiitech/ashiato/api/internal/usecase"
)

func TestAPIWorkflow(t *testing.T) {
	ctx, dsn := testutil.StartPostgres(t)

	cfg := config.Config{
		DatabaseURL:    dsn,
		AuthMode:       config.AuthModeStub,
		DevJWTSecret:   "integration-secret",
		DefaultOrgName: "StepByCode",
		DefaultOrgSlug: "stepbycode",
		OwnerEmails: map[string]struct{}{
			"owner@example.com": {},
		},
		BotSharedToken: "bot-secret",
	}

	store, err := repository.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("repository.Open: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	app := newTestApp(ctx, t, cfg, store)

	ownerToken := stubJWT(t, cfg.DevJWTSecret, "owner-sub", "owner@example.com", "Owner User")
	editorToken := stubJWT(t, cfg.DevJWTSecret, "editor-sub", "editor@example.com", "Editor User")

	ownerMe := requestJSON[oapi.MeResponse](t, app, http.MethodGet, "/api/v1/me", ownerToken, nil, "")
	editorMe := requestJSON[oapi.MeResponse](t, app, http.MethodGet, "/api/v1/me", editorToken, nil, "")

	taskPayload := oapi.CreateTaskRequest{
		Title:  "Publish event page",
		Status: oapi.Done,
		ApproverIds: []openapi_types.UUID{
			ownerMe.User.Id,
			editorMe.User.Id,
		},
		Year:  2026,
		Month: 4,
	}
	task := requestJSON[oapi.Task](t, app, http.MethodPost, "/api/v1/tasks", ownerToken, taskPayload, "")

	task = requestJSON[oapi.Task](t, app, http.MethodPost, "/api/v1/tasks/"+task.Id.String()+"/approve", ownerToken, nil, "")
	if task.ApprovalState != oapi.PartiallyApproved {
		t.Fatalf("expected partially approved, got %s", task.ApprovalState)
	}

	repeatedApprove := request(t, app, http.MethodPost, "/api/v1/tasks/"+task.Id.String()+"/approve", ownerToken, nil, "")
	if repeatedApprove.Code != http.StatusConflict {
		t.Fatalf("expected 409 on repeated approval, got %d", repeatedApprove.Code)
	}

	task = requestJSON[oapi.Task](t, app, http.MethodPost, "/api/v1/tasks/"+task.Id.String()+"/approve", editorToken, nil, "")
	if task.ApprovalState != oapi.Approved {
		t.Fatalf("expected approved state, got %s", task.ApprovalState)
	}

	announcementPayload := oapi.PutAnnouncementRequest{
		Body:           "Ashiato release draft",
		PublishChannel: strPtr("stepbycode-events"),
	}
	announcement := requestJSON[oapi.Announcement](t, app, http.MethodPut, "/api/v1/announcements/2026/4", ownerToken, announcementPayload, "")
	if announcement.Status != oapi.Draft {
		t.Fatalf("expected draft announcement, got %s", announcement.Status)
	}

	publishPayload := oapi.PublishAnnouncementRequest{PublishChannel: "stepbycode-events"}
	announcement = requestJSON[oapi.Announcement](t, app, http.MethodPost, "/api/v1/announcements/"+announcement.Id.String()+"/publish", ownerToken, publishPayload, "")
	if announcement.Status != oapi.PublishRequested {
		t.Fatalf("expected publish_requested, got %s", announcement.Status)
	}

	queue := requestJSON[oapi.PublishRequestsResponse](t, app, http.MethodGet, "/internal/v1/announcement-publish-requests", "", nil, cfg.BotSharedToken)
	if len(queue.Requests) != 1 {
		t.Fatalf("expected one publish request, got %d", len(queue.Requests))
	}

	completePayload := oapi.CompletePublishRequest{DiscordMessageId: "discord-msg-1"}
	announcement = requestJSON[oapi.Announcement](t, app, http.MethodPost, "/internal/v1/announcement-publish-requests/"+announcement.Id.String()+"/complete", "", completePayload, cfg.BotSharedToken)
	if announcement.Status != oapi.Published {
		t.Fatalf("expected published announcement, got %s", announcement.Status)
	}
	if announcement.DiscordMessageId == nil || *announcement.DiscordMessageId != "discord-msg-1" {
		t.Fatalf("unexpected discord message id: %+v", announcement.DiscordMessageId)
	}

	periods := requestJSON[oapi.WorkflowPeriodsResponse](t, app, http.MethodGet, "/api/v1/workflow-periods", ownerToken, nil, "")
	if len(periods.Periods) == 0 {
		t.Fatal("expected workflow periods")
	}
}

func newTestApp(ctx context.Context, t *testing.T, cfg config.Config, store *repository.Store) *echo.Echo {
	t.Helper()

	logger := logging.NewLogger()
	verifier, err := auth.NewVerifier(ctx, cfg)
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	service := usecase.NewService(store, logger, cfg)
	authenticator := auth.NewAuthenticator(verifier, service, cfg.BotSharedToken)
	server := handler.NewServer(service)

	e := echo.New()
	e.Use(echomiddleware.Recover())
	e.Use(logging.RequestMiddleware(logger))
	e.Use(authenticator.Middleware())
	oapi.RegisterHandlers(e, oapi.NewStrictHandler(server, nil))
	return e
}

func stubJWT(t *testing.T, secret, sub, email, name string) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   sub,
		"email": email,
		"name":  name,
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}
	return signed
}

func requestJSON[T any](t *testing.T, app *echo.Echo, method, path, bearer string, body any, botToken string) T {
	t.Helper()
	rec := request(t, app, method, path, bearer, body, botToken)
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}

	var result T
	if rec.Body.Len() == 0 {
		return result
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	return result
}

func request(t *testing.T, app *echo.Echo, method, path, bearer string, body any, botToken string) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if botToken != "" {
		req.Header.Set("X-Bot-Token", botToken)
	}

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func strPtr(value string) *string {
	return &value
}
