package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"google.golang.org/api/option"

	"github.com/dokkiitech/ashiato/api/internal/auth"
	"github.com/dokkiitech/ashiato/api/internal/config"
	"github.com/dokkiitech/ashiato/api/internal/discord"
	"github.com/dokkiitech/ashiato/api/internal/handler"
	"github.com/dokkiitech/ashiato/api/internal/logging"
	"github.com/dokkiitech/ashiato/api/internal/oapi"
	"github.com/dokkiitech/ashiato/api/internal/repository"
	"github.com/dokkiitech/ashiato/api/internal/simpleapi"
	"github.com/dokkiitech/ashiato/api/internal/usecase"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger := logging.NewLogger()

	// Initialize Firebase App.
	// Use explicit credentials JSON when provided; otherwise fall back to
	// Application Default Credentials (ADC) available in GCP environments.
	var fbApp *firebase.App
	if cfg.FirebaseCredentialsJSON != "" {
		opt := option.WithCredentialsJSON([]byte(cfg.FirebaseCredentialsJSON))
		fbApp, err = firebase.NewApp(ctx, nil, opt)
	} else {
		logger.Info("FIREBASE_CREDENTIALS_JSON not set; using Application Default Credentials")
		fbApp, err = firebase.NewApp(ctx, nil)
	}
	if err != nil {
		logger.Error("failed to initialize Firebase app", slog.Any("error", err))
		os.Exit(1)
	}

	// Firebase Auth client for token verification.
	authClient, err := fbApp.Auth(ctx)
	if err != nil {
		logger.Error("failed to initialize Firebase Auth client", slog.Any("error", err))
		os.Exit(1)
	}

	// Realtime Database client for data storage.
	dbClient, err := fbApp.DatabaseWithURL(ctx, cfg.FirebaseDatabaseURL)
	if err != nil {
		logger.Error("failed to initialize Realtime Database client", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("firebase initialized (auth + realtime database)")

	store := repository.New(dbClient)
	webhook := discord.NewWebhookClient(cfg.DiscordWebhookURL)
	verifier := auth.NewFirebaseVerifier(authClient)
	userCreator := auth.NewFirebaseUserCreator(authClient)
	service := usecase.NewService(store, webhook, logger, cfg, userCreator)
	authenticator := auth.NewAuthenticator(verifier, service)
	server := handler.NewServer(service)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(echomiddleware.Recover())
	e.Use(logging.RequestMiddleware(logger))
	e.Use(authenticator.Middleware())
	e.HTTPErrorHandler = jsonHTTPErrorHandler

	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Internal endpoint: provision workflow period (protected by CRON_SECRET).
	e.POST("/internal/v1/workflow-periods/provision", func(c echo.Context) error {
		if cfg.CronSecret == "" {
			return c.JSON(http.StatusForbidden, map[string]string{"code": "forbidden", "message": "cron secret not configured"})
		}
		secret := c.Request().Header.Get("X-Cron-Secret")
		if secret != cfg.CronSecret {
			return c.JSON(http.StatusForbidden, map[string]string{"code": "forbidden", "message": "invalid cron secret"})
		}

		var body struct {
			Year  int32 `json:"year"`
			Month int32 `json:"month"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"code": "bad_request", "message": "invalid request body"})
		}

		if err := service.ProvisionAllOrganizations(c.Request().Context(), body.Year, body.Month); err != nil {
			logger.Error("failed to provision workflow period", "year", body.Year, "month", body.Month, "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"code": "internal", "message": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "message": fmt.Sprintf("provisioned %d/%02d", body.Year, body.Month)})
	})

	strictHandler := oapi.NewStrictHandler(server, nil)
	oapi.RegisterHandlers(e, strictHandler)

	// Member management (requires authentication via middleware).
	apiGroup := e.Group("/api/v1")
	handler.RegisterMemberRoutes(apiGroup, service)

	// Simple API endpoints (docs/backend-api-request.md) backed by Realtime Database.
	simpleGroup := e.Group("/api/v1")
	simpleapi.RegisterTaskRoutes(simpleGroup, dbClient)
	simpleapi.RegisterMeetingRoutes(simpleGroup, dbClient)
	simpleapi.RegisterPublicityRoutes(simpleGroup, dbClient)
	simpleapi.RegisterProfileRoutes(simpleGroup, dbClient)
	simpleapi.RegisterWorkflowPeriodRoutes(simpleGroup, dbClient)

	// Invite endpoint (creates Firebase user, sends Resend email, notifies Discord).
	simpleapi.RegisterInviteRoutes(simpleGroup, simpleapi.InviteDeps{
		DBClient:     dbClient,
		FirebaseAuth: auth.NewFirebaseUserCreator(authClient),
		ResendAPIKey: cfg.ResendAPIKey,
		FromEmail:    cfg.InviteFromEmail,
		Webhook:      webhook,
		Logger:       logger,
	})

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("api server starting", slog.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", slog.Any("error", err))
	}
}

func jsonHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	message := "internal server error"

	var echoErr *echo.HTTPError
	if errors.As(err, &echoErr) {
		code = echoErr.Code
		message = fmt.Sprint(echoErr.Message)
	}

	if !c.Response().Committed {
		_ = c.JSON(code, map[string]string{
			"code":    http.StatusText(code),
			"message": message,
		})
	}
}
