package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	FirebaseCredentialsJSON string
	DefaultOrgName          string
	DefaultOrgSlug          string
	OwnerEmails             map[string]struct{}
	BotSharedToken          string
	Port                    string
}

func Load() (Config, error) {
	cfg := Config{
		FirebaseCredentialsJSON: strings.TrimSpace(os.Getenv("FIREBASE_CREDENTIALS_JSON")),
		DefaultOrgName:          strings.TrimSpace(os.Getenv("DEFAULT_ORG_NAME")),
		DefaultOrgSlug:          strings.TrimSpace(os.Getenv("DEFAULT_ORG_SLUG")),
		OwnerEmails:             parseOwnerEmails(strings.TrimSpace(os.Getenv("OWNER_EMAILS"))),
		BotSharedToken:          strings.TrimSpace(os.Getenv("BOT_SHARED_TOKEN")),
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
	case cfg.BotSharedToken == "":
		return Config{}, errors.New("BOT_SHARED_TOKEN is required")
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
