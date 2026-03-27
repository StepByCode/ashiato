package simpleapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"firebase.google.com/go/v4/db"
	"github.com/labstack/echo/v4"

	"github.com/dokkiitech/ashiato/api/internal/discord"
)

// InviteDeps holds dependencies for invite endpoints.
type InviteDeps struct {
	DBClient       *db.Client
	FirebaseAuth   FirebaseUserCreator
	ResendAPIKey   string
	FromEmail      string
	Webhook        *discord.WebhookClient
	Logger         *slog.Logger
}

// FirebaseUserCreator is implemented by auth.FirebaseUserCreatorAdapter.
type FirebaseUserCreator interface {
	CreateFirebaseUser(ctx context.Context, email, password, displayName string) (string, error)
}

type inviteRequest struct {
	Email string `json:"email"`
}

type inviteResponse struct {
	UID      string `json:"uid"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

const invitesCollection = "simple_invites"

// RegisterInviteRoutes registers the invite endpoint.
func RegisterInviteRoutes(g *echo.Group, deps InviteDeps) {
	g.POST("/invite", inviteHandler(deps))
	g.GET("/invites", listInvitesHandler(deps.DBClient))
}

func inviteHandler(deps InviteDeps) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req inviteRequest
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}

		email := strings.TrimSpace(strings.ToLower(req.Email))
		if email == "" || !strings.Contains(email, "@") {
			return validationError(c, "email", "is required and must be valid")
		}

		ctx := c.Request().Context()

		// Generate password
		password, err := generatePassword(12)
		if err != nil {
			deps.Logger.Error("failed to generate password", slog.Any("error", err))
			return internalError(c)
		}

		// Create Firebase user
		uid, err := deps.FirebaseAuth.CreateFirebaseUser(ctx, email, password, email)
		if err != nil {
			deps.Logger.Error("failed to create firebase user", slog.Any("error", err), slog.String("email", email))
			return c.JSON(http.StatusConflict, ErrorResponse{
				Error: ErrorBody{Code: "CONFLICT", Message: "user already exists or creation failed: " + err.Error()},
			})
		}

		// Store invite record
		now := time.Now().Format(time.RFC3339)
		record := map[string]interface{}{
			"uid":       uid,
			"email":     email,
			"createdAt": now,
		}
		_ = deps.DBClient.NewRef(invitesCollection).Child(uid).Set(ctx, record)

		// Create empty profile
		profile := ProfileResponse{
			UID:       uid,
			Email:     email,
			UpdatedAt: now,
		}
		_ = deps.DBClient.NewRef(profilesCollection).Child(uid).Set(ctx, profile)

		// Send invite email via Resend HTTP API
		if deps.ResendAPIKey != "" {
			go func() {
				sendInviteEmail(deps.ResendAPIKey, deps.FromEmail, email, password, deps.Logger)
			}()
		}

		// Notify Discord
		if deps.Webhook != nil {
			go func() {
				msg := fmt.Sprintf("🆕 新しいメンバーを招待しました: %s", email)
				if _, sendErr := deps.Webhook.Send(context.Background(), msg); sendErr != nil {
					deps.Logger.Error("failed to send discord notification", slog.Any("error", sendErr))
				}
			}()
		}

		return c.JSON(http.StatusCreated, inviteResponse{
			UID:      uid,
			Email:    email,
			Password: password,
		})
	}
}

func listInvitesHandler(client *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		ref := client.NewRef(invitesCollection)
		var all map[string]map[string]interface{}
		if err := ref.Get(c.Request().Context(), &all); err != nil || len(all) == 0 {
			return c.JSON(http.StatusOK, map[string]interface{}{"invites": []interface{}{}})
		}
		invites := make([]map[string]interface{}, 0, len(all))
		for _, v := range all {
			invites = append(invites, v)
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"invites": invites})
	}
}

func sendInviteEmail(apiKey, fromEmail, toEmail, password string, logger *slog.Logger) {
	htmlBody := fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto; padding: 2rem;">
  <h2 style="color: #df6900;">Ashiato へようこそ</h2>
  <p>StepByCode の運営ツール Ashiato に招待されました。</p>
  <p>以下の情報でログインしてください:</p>
  <div style="background: #f5f5f5; padding: 1rem; border-radius: 8px; margin: 1rem 0;">
    <p style="margin: 0.25rem 0;"><strong>メールアドレス:</strong> %s</p>
    <p style="margin: 0.25rem 0;"><strong>パスワード:</strong> %s</p>
  </div>
  <p style="color: #666; font-size: 0.875rem;">初回ログイン後、プロフィールの登録をお願いします。</p>
</div>`, toEmail, password)

	payload := map[string]interface{}{
		"from":    fromEmail,
		"to":      []string{toEmail},
		"subject": "【Ashiato】招待のお知らせ",
		"html":    htmlBody,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		logger.Error("failed to create resend request", slog.Any("error", err))
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("failed to send resend email", slog.Any("error", err), slog.String("email", toEmail))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		logger.Error("resend returned error", slog.Int("status", resp.StatusCode), slog.String("email", toEmail))
		return
	}
	logger.Info("invite email sent", slog.String("email", toEmail))
}

func generatePassword(length int) (string, error) {
	const charset = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!@#$"
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}
