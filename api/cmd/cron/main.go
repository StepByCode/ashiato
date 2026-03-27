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

// JST は Asia/Tokyo タイムゾーン。サーバーは UTC で動作するため、
// 「日本時間の1日」を正しく判定するために JST 基準で現在月を計算する。
var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

// main provisions workflow periods for all organizations.
// It calculates the target month as current month + 2 (JST basis) and creates
// initial meeting (planned) and announcement (draft) entries if they do not exist.
//
// This command is intended to be run as a CronJob on the 1st of each month.
// サーバーが UTC で動作するため、JST 1日 0:00 に実行したい場合は
// crontab に "0 15 L * *" (前月末 15:00 UTC) のように設定する。
// より安全には "0 15 28-31 * *" で毎月末の15:00 UTCに実行し、
// 翌月がJST基準で正しく計算される。
//
// 推奨 crontab (UTC): 0 15 1 * *
// → JST 翌日 0:00 = JST 1日に相当（UTCの1日15時 = JST 2日0時だが、
//   実際には JST 基準で now を変換するため、UTC 1日 0時に実行しても
//   JST では1日9時となり正しく動作する）
//
// 最もシンプルな設定:
//   crontab (UTC): 0 0 1 * *
//   → UTC 1日 0:00 = JST 1日 9:00 → JST基準で「今月」が正しい
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

	// JST 基準で現在月を取得し、+2ヶ月を対象とする。
	// サーバーが UTC でも JST 変換により正しい月を算出。
	// 例: UTC 2026-01-01 00:00 → JST 2026-01-01 09:00 → target = 2026-03
	nowJST := time.Now().In(jst)
	target := nowJST.AddDate(0, 2, 0)
	targetYear := int32(target.Year())
	targetMonth := int32(target.Month())

	logger.Info("provisioning workflow period",
		slog.String("now_utc", time.Now().UTC().Format(time.RFC3339)),
		slog.String("now_jst", nowJST.Format(time.RFC3339)),
		slog.Int("target_year", int(targetYear)),
		slog.Int("target_month", int(targetMonth)),
	)

	if err := service.ProvisionAllOrganizations(ctx, targetYear, targetMonth); err != nil {
		logger.Error("failed to provision workflow periods", slog.Any("error", err))
		os.Exit(1)
	}

	fmt.Printf("successfully provisioned workflow period %d/%02d\n", targetYear, targetMonth)
}
