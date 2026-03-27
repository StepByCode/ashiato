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

	"github.com/dokkiitech/ashiato/api/internal/auth"
	"github.com/dokkiitech/ashiato/api/internal/config"
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
	store, err := repository.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect database", slog.Any("error", err))
		os.Exit(1)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		logger.Error("failed to apply migrations", slog.Any("error", err))
		os.Exit(1)
	}

	verifier, err := auth.NewVerifier(ctx, cfg)
	if err != nil {
		logger.Error("failed to initialize auth verifier", slog.Any("error", err))
		os.Exit(1)
	}

	service := usecase.NewService(store, logger, cfg)
	authenticator := auth.NewAuthenticator(verifier, service, cfg.BotSharedToken)
	server := handler.NewServer(service)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(echomiddleware.Recover())
	e.Use(logging.RequestMiddleware(logger))
	e.Use(authenticator.Middleware())
	e.HTTPErrorHandler = jsonHTTPErrorHandler

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	strictHandler := oapi.NewStrictHandler(server, nil)
	oapi.RegisterHandlers(e, strictHandler)

	// Simple API endpoints (docs/backend-api-request.md) backed by Firebase/Firestore.
	// Uses GOOGLE_APPLICATION_CREDENTIALS for authentication (Application Default Credentials).
	fbApp, err := firebase.NewApp(ctx, nil)
	if err != nil {
		logger.Error("failed to initialize Firebase app", slog.Any("error", err))
		os.Exit(1)
	}
	fsClient, err := fbApp.Firestore(ctx)
	if err != nil {
		logger.Error("failed to initialize Firestore client", slog.Any("error", err))
		os.Exit(1)
	}
	defer fsClient.Close()
	logger.Info("firebase/firestore initialized")

	simpleGroup := e.Group("/api/v1")
	simpleapi.RegisterTaskRoutes(simpleGroup, fsClient)
	simpleapi.RegisterMeetingRoutes(simpleGroup, fsClient)
	simpleapi.RegisterPublicityRoutes(simpleGroup, fsClient)

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
