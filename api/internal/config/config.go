package config

import (
	"errors"
	"os"
	"strings"
)

const (
	AuthModeStub = "stub"
	AuthModeOIDC = "oidc"
)

type Config struct {
	DatabaseURL      string
	AuthMode         string
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	DevJWTSecret     string
	DefaultOrgName   string
	DefaultOrgSlug   string
	OwnerEmails      map[string]struct{}
	BotSharedToken   string
	Port             string
	// Firebase configuration for Simple API
	FirebaseSAKeyPath string // path to service account JSON key file
	FirebaseProjectID string // Firebase project ID (can be auto-detected from SA key)
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:      strings.TrimSpace(os.Getenv("DATABASE_URL")),
		AuthMode:         strings.TrimSpace(os.Getenv("AUTH_MODE")),
		OIDCIssuerURL:    strings.TrimSpace(os.Getenv("OIDC_ISSUER_URL")),
		OIDCClientID:     strings.TrimSpace(os.Getenv("OIDC_CLIENT_ID")),
		OIDCClientSecret: strings.TrimSpace(os.Getenv("OIDC_CLIENT_SECRET")),
		DevJWTSecret:     strings.TrimSpace(os.Getenv("DEV_JWT_SECRET")),
		DefaultOrgName:   strings.TrimSpace(os.Getenv("DEFAULT_ORG_NAME")),
		DefaultOrgSlug:   strings.TrimSpace(os.Getenv("DEFAULT_ORG_SLUG")),
		OwnerEmails:      parseOwnerEmails(strings.TrimSpace(os.Getenv("OWNER_EMAILS"))),
		BotSharedToken:    strings.TrimSpace(os.Getenv("BOT_SHARED_TOKEN")),
		Port:              defaultValue(strings.TrimSpace(os.Getenv("PORT")), "8080"),
		FirebaseSAKeyPath: strings.TrimSpace(os.Getenv("FIREBASE_SA_KEY_PATH")),
		FirebaseProjectID: strings.TrimSpace(os.Getenv("FIREBASE_PROJECT_ID")),
	}

	if cfg.AuthMode == "" {
		cfg.AuthMode = AuthModeStub
	}
	if cfg.DefaultOrgName == "" {
		cfg.DefaultOrgName = "StepByCode"
	}
	if cfg.DefaultOrgSlug == "" {
		cfg.DefaultOrgSlug = "stepbycode"
	}

	switch {
	case cfg.DatabaseURL == "":
		return Config{}, errors.New("DATABASE_URL is required")
	case cfg.BotSharedToken == "":
		return Config{}, errors.New("BOT_SHARED_TOKEN is required")
	case cfg.AuthMode != AuthModeStub && cfg.AuthMode != AuthModeOIDC:
		return Config{}, errors.New("AUTH_MODE must be stub or oidc")
	case cfg.AuthMode == AuthModeStub && cfg.DevJWTSecret == "":
		return Config{}, errors.New("DEV_JWT_SECRET is required when AUTH_MODE=stub")
	case cfg.AuthMode == AuthModeOIDC && (cfg.OIDCIssuerURL == "" || cfg.OIDCClientID == ""):
		return Config{}, errors.New("OIDC_ISSUER_URL and OIDC_CLIENT_ID are required when AUTH_MODE=oidc")
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
