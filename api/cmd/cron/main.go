package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"

	"github.com/dokkiitech/ashiato/api/internal/config"
	"github.com/dokkiitech/ashiato/api/internal/discord"
	"github.com/dokkiitech/ashiato/api/internal/logging"
	"github.com/dokkiitech/ashiato/api/internal/repository"
	"github.com/dokkiitech/ashiato/api/internal/usecase"
)

// main provisions workflow periods for all organizations.
// It calculates the target month as current month + 2 and creates initial
// meeting (planned) and announcement (draft) entries if they do not exist.
//
// This command is intended to be run as a CronJob on the 1st of each month.
//
// Example: if run on 2026-01-01, it provisions 2026-03 (March).
func main() {
	ctx := context.Background()
	logger := logging.NewLogger()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	var fbApp *firebase.App
	if cfg.FirebaseCredentialsJSON != "" {
		opt := option.WithCredentialsJSON([]byte(cfg.FirebaseCredentialsJSON))
		fbApp, err = firebase.NewApp(ctx, nil, opt)
	} else {
		fbApp, err = firebase.NewApp(ctx, nil)
	}
	if err != nil {
		logger.Error("failed to initialize Firebase app", slog.Any("error", err))
		os.Exit(1)
	}

	dbClient, err := fbApp.DatabaseWithURL(ctx, cfg.FirebaseDatabaseURL)
	if err != nil {
		logger.Error("failed to initialize Realtime Database client", slog.Any("error", err))
		os.Exit(1)
	}

	store := repository.New(dbClient)
	webhook := discord.NewWebhookClient(cfg.DiscordWebhookURL)
	service := usecase.NewService(store, webhook, logger, cfg, nil)

	// Calculate target month: current + 2.
	now := time.Now()
	target := now.AddDate(0, 2, 0)
	targetYear := int32(target.Year())
	targetMonth := int32(target.Month())

	logger.Info("provisioning workflow period",
		slog.Int("target_year", int(targetYear)),
		slog.Int("target_month", int(targetMonth)),
	)

	if err := service.ProvisionAllOrganizations(ctx, targetYear, targetMonth); err != nil {
		logger.Error("failed to provision workflow periods", slog.Any("error", err))
		os.Exit(1)
	}

	fmt.Printf("successfully provisioned workflow period %d/%02d\n", targetYear, targetMonth)
}
