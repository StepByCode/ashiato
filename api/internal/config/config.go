package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	FirebaseCredentialsJSON string
	DiscordWebhookURL       string
	DefaultOrgName          string
	DefaultOrgSlug          string
	OwnerEmails             map[string]struct{}
	Port                    string
}

func Load() (Config, error) {
	cfg := Config{
		FirebaseCredentialsJSON: strings.TrimSpace(os.Getenv("FIREBASE_CREDENTIALS_JSON")),
		DiscordWebhookURL:       strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL")),
		DefaultOrgName:          strings.TrimSpace(os.Getenv("DEFAULT_ORG_NAME")),
		DefaultOrgSlug:          strings.TrimSpace(os.Getenv("DEFAULT_ORG_SLUG")),
		OwnerEmails:             parseOwnerEmails(strings.TrimSpace(os.Getenv("OWNER_EMAILS"))),
		Port:                    defaultValue(strings.TrimSpace(os.Getenv("PORT")), "8080"),
	}

	if cfg.DefaultOrgName == "" {
		cfg.DefaultOrgName = "StepByCode"
	}
	if cfg.DefaultOrgSlug == "" {
		cfg.DefaultOrgSlug = "stepbycode"
	}

	switch {
	case cfg.FirebaseCredentialsJSON == "":
		return Config{}, errors.New("FIREBASE_CREDENTIALS_JSON is required")
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
