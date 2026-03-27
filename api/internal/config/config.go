package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	FirebaseCredentialsJSON string
	FirebaseDatabaseURL     string
	DiscordWebhookURL       string
	DefaultOrgName          string
	DefaultOrgSlug          string
	OwnerEmails             map[string]struct{}
	Port                    string
	ResendAPIKey            string
	InviteFromEmail         string
	CronSecret              string
	CORSAllowedOrigins      []string
}

func Load() (Config, error) {
	cfg := Config{
		FirebaseCredentialsJSON: strings.TrimSpace(os.Getenv("FIREBASE_CREDENTIALS_JSON")),
		FirebaseDatabaseURL:     strings.TrimSpace(os.Getenv("FIREBASE_DATABASE_URL")),
		DiscordWebhookURL:       strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL")),
		DefaultOrgName:          strings.TrimSpace(os.Getenv("DEFAULT_ORG_NAME")),
		DefaultOrgSlug:          strings.TrimSpace(os.Getenv("DEFAULT_ORG_SLUG")),
		OwnerEmails:             parseOwnerEmails(strings.TrimSpace(os.Getenv("OWNER_EMAILS"))),
		Port:                    defaultValue(strings.TrimSpace(os.Getenv("PORT")), "9999"),
		ResendAPIKey:            strings.TrimSpace(os.Getenv("RESEND_API_KEY")),
		InviteFromEmail:         defaultValue(strings.TrimSpace(os.Getenv("INVITE_FROM_EMAIL")), "no-reply@stepbycode.work"),
		CronSecret:              strings.TrimSpace(os.Getenv("CRON_SECRET")),
		CORSAllowedOrigins:      parseCSV(strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))),
	}

	if cfg.DefaultOrgName == "" {
		cfg.DefaultOrgName = "StepByCode"
	}
	if cfg.DefaultOrgSlug == "" {
		cfg.DefaultOrgSlug = "stepbycode"
	}

	switch {
	case cfg.FirebaseDatabaseURL == "":
		return Config{}, errors.New("FIREBASE_DATABASE_URL is required")
	case cfg.DiscordWebhookURL == "":
		return Config{}, errors.New("DISCORD_WEBHOOK_URL is required")
	}

	return cfg, nil
}

func defaultValue(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	var result []string
	for _, part := range strings.Split(raw, ",") {
		v := strings.TrimSpace(part)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

func parseOwnerEmails(raw string) map[string]struct{} {
	values := map[string]struct{}{}
	if raw == "" {
		return values
	}
	for _, part := range strings.Split(raw, ",") {
		email := strings.ToLower(strings.TrimSpace(part))
		if email == "" {
			continue
		}
		values[email] = struct{}{}
	}
	return values
}
