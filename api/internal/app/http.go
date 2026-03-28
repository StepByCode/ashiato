package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	firebase "firebase.google.com/go/v4"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"google.golang.org/api/option"

	"github.com/dokkiitech/ashiato/api/internal/auth"
	"github.com/dokkiitech/ashiato/api/internal/config"
	"github.com/dokkiitech/ashiato/api/internal/corsutil"
	"github.com/dokkiitech/ashiato/api/internal/discord"
	"github.com/dokkiitech/ashiato/api/internal/handler"
	"github.com/dokkiitech/ashiato/api/internal/logging"
	"github.com/dokkiitech/ashiato/api/internal/oapi"
	"github.com/dokkiitech/ashiato/api/internal/repository"
	"github.com/dokkiitech/ashiato/api/internal/simpleapi"
	"github.com/dokkiitech/ashiato/api/internal/usecase"
)

func NewHTTPHandler(ctx context.Context) (http.Handler, config.Config, *slog.Logger, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, config.Config{}, nil, err
	}

	logger := logging.NewLogger()

	var fbApp *firebase.App
	if cfg.FirebaseCredentialsJSON != "" {
		opt := option.WithCredentialsJSON([]byte(cfg.FirebaseCredentialsJSON))
		fbApp, err = firebase.NewApp(ctx, nil, opt)
	} else {
		logger.Info("FIREBASE_CREDENTIALS_JSON not set; using Application Default Credentials")
		fbApp, err = firebase.NewApp(ctx, nil)
	}
	if err != nil {
		return nil, config.Config{}, logger, fmt.Errorf("initialize Firebase app: %w", err)
	}

	authClient, err := fbApp.Auth(ctx)
	if err != nil {
		return nil, config.Config{}, logger, fmt.Errorf("initialize Firebase Auth client: %w", err)
	}

	dbClient, err := fbApp.DatabaseWithURL(ctx, cfg.FirebaseDatabaseURL)
	if err != nil {
		return nil, config.Config{}, logger, fmt.Errorf("initialize Realtime Database client: %w", err)
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
	allowedOrigins := corsutil.BuildAllowedOrigins(cfg.CORSAllowedOrigins)
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins: allowedOrigins,
		AllowOriginFunc: func(origin string) (bool, error) {
			return corsutil.IsOriginAllowed(origin, allowedOrigins), nil
		},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))
	e.Use(logging.RequestMiddleware(logger))
	e.Use(authenticator.Middleware())
	e.HTTPErrorHandler = jsonHTTPErrorHandler

	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

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

	apiGroup := e.Group("/api/v1")
	handler.RegisterMemberRoutes(apiGroup, service)

	simpleGroup := e.Group("/api/v1")
	simpleapi.RegisterTaskRoutes(simpleGroup, dbClient, webhook)
	simpleapi.RegisterMeetingRoutes(simpleGroup, dbClient, webhook)
	simpleapi.RegisterPublicityRoutes(simpleGroup, dbClient, webhook)
	simpleapi.RegisterProfileRoutes(simpleGroup, dbClient)
	simpleapi.RegisterWorkflowPeriodRoutes(simpleGroup, dbClient)
	simpleapi.RegisterInviteRoutes(simpleGroup, simpleapi.InviteDeps{
		DBClient:     dbClient,
		FirebaseAuth: userCreator,
		ResendAPIKey: cfg.ResendAPIKey,
		FromEmail:    cfg.InviteFromEmail,
		Webhook:      webhook,
		Logger:       logger,
	})

	return e, cfg, logger, nil
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
