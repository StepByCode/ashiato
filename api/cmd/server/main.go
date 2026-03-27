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

	"github.com/dokkiitech/ashiato/api/internal/app"
)

func main() {
	ctx := context.Background()

	httpHandler, cfg, logger, err := app.NewHTTPHandler(ctx)
	if err != nil {
		if logger != nil {
			logger.Error("failed to initialize API handler", slog.Any("error", err))
		}
		os.Exit(1)
	}

	httpServer := &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%s", cfg.Port),
		Handler:           httpHandler,
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
