package simpleapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"time"

	"firebase.google.com/go/v4/db"
	"github.com/labstack/echo/v4"

	"github.com/dokkiitech/ashiato/api/internal/discord"
	appctx "github.com/dokkiitech/ashiato/api/internal/middleware"
)

// InviteDeps holds dependencies for invite endpoints.
type InviteDeps struct {
	DBClient     *db.Client
	FirebaseAuth FirebaseUserCreator
	ResendAPIKey string
	FromEmail    string
	LoginURL     string
	Webhook      *discord.WebhookClient
	Logger       *slog.Logger
}

// FirebaseUserCreator is implemented by auth.FirebaseUserCreatorAdapter.
type FirebaseUserCreator interface {
	CreateFirebaseUser(ctx context.Context, email, password, displayName string) (string, error)
	DeleteFirebaseUser(ctx context.Context, uid string) error
}

type inviteRequest struct {
	Email string `json:"email"`
}

type inviteResponse struct {
	UID      string `json:"uid"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type inviteStatusResponse struct {
	UID                string `json:"uid"`
	Email              string `json:"email"`
	PasswordChangedAt  string `json:"passwordChangedAt,omitempty"`
	NeedsPasswordReset bool   `json:"needsPasswordReset"`
}

const invitesCollection = "simple_invites"

// RegisterInviteRoutes registers the invite endpoint.
func RegisterInviteRoutes(g *echo.Group, deps InviteDeps) {
	g.POST("/invite", inviteHandler(deps))
	g.GET("/invites", listInvitesHandler(deps.DBClient))
	g.GET("/invite/:uid/status", inviteStatusHandler(deps.DBClient))
	g.PATCH("/invite/:uid/password-changed", markInvitePasswordChangedHandler(deps.DBClient))
	g.DELETE("/invite/:uid", revokeInviteHandler(deps))
}

func inviteHandler(deps InviteDeps) echo.HandlerFunc {
	return func(c echo.Context) error {
		if _, ok := appctx.ActorFromContext(c.Request().Context()); !ok {
			return c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: ErrorBody{Code: "UNAUTHORIZED", Message: "sign-in is required"},
			})
		}

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
			"uid":               uid,
			"email":             email,
			"createdAt":         now,
			"passwordChangedAt": "",
		}
		_ = deps.DBClient.NewRef(invitesCollection).Child(uid).Set(ctx, record)

		// Create empty profile
		profile := ProfileResponse{
			UID:       uid,
			Email:     email,
			UpdatedAt: now,
		}
		_ = deps.DBClient.NewRef(profilesCollection).Child(uid).Set(ctx, profile)

		if deps.ResendAPIKey == "" {
			rollbackInvite(ctx, deps, uid)
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: ErrorBody{Code: "RESEND_NOT_CONFIGURED", Message: "招待メール送信の設定が不足しています"},
			})
		}

		if err := sendInviteEmail(ctx, deps.ResendAPIKey, deps.FromEmail, deps.LoginURL, email, password); err != nil {
			deps.Logger.Error("failed to send invite email", slog.Any("error", err), slog.String("email", email))
			rollbackInvite(ctx, deps, uid)
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: ErrorBody{Code: "INVITE_EMAIL_FAILED", Message: "招待メールの送信に失敗しました: " + err.Error()},
			})
		}

		// Notify Discord
		if deps.Webhook != nil {
			go func() {
				fields := []discord.WebhookEmbedField{
					{Name: "メールアドレス", Value: email, Inline: false},
					{Name: "初回パスワード", Value: password, Inline: false},
				}
				if _, sendErr := deps.Webhook.SendEmbed(context.Background(), "新しいメンバーを招待しました", "Backstage の招待を作成しました。", fields); sendErr != nil {
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
		if _, ok := appctx.ActorFromContext(c.Request().Context()); !ok {
			return c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: ErrorBody{Code: "UNAUTHORIZED", Message: "sign-in is required"},
			})
		}

		ref := client.NewRef(invitesCollection)
		var all map[string]map[string]interface{}
		if err := ref.Get(c.Request().Context(), &all); err != nil || len(all) == 0 {
			return c.JSON(http.StatusOK, map[string]interface{}{"invites": []interface{}{}})
		}
		invites := make([]map[string]interface{}, 0, len(all))
		for _, v := range all {
			invites = append(invites, v)
		}
		sort.Slice(invites, func(i, j int) bool {
			left, _ := invites[i]["createdAt"].(string)
			right, _ := invites[j]["createdAt"].(string)
			return left > right
		})
		return c.JSON(http.StatusOK, map[string]interface{}{"invites": invites})
	}
}

func inviteStatusHandler(client *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		uid := strings.TrimSpace(c.Param("uid"))
		actor, ok := appctx.ActorFromContext(c.Request().Context())
		if !ok || actor.Subject != uid {
			return c.JSON(http.StatusForbidden, ErrorResponse{
				Error: ErrorBody{Code: "FORBIDDEN", Message: "invite status access is limited to the signed-in user"},
			})
		}

		var invite inviteStatusResponse
		if err := client.NewRef(invitesCollection).Child(uid).Get(c.Request().Context(), &invite); err != nil || invite.UID == "" {
			return c.JSON(http.StatusOK, inviteStatusResponse{
				UID:                uid,
				NeedsPasswordReset: false,
			})
		}
		invite.NeedsPasswordReset = strings.TrimSpace(invite.PasswordChangedAt) == ""
		return c.JSON(http.StatusOK, invite)
	}
}

func markInvitePasswordChangedHandler(client *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		uid := strings.TrimSpace(c.Param("uid"))
		actor, ok := appctx.ActorFromContext(c.Request().Context())
		if !ok || actor.Subject != uid {
			return c.JSON(http.StatusForbidden, ErrorResponse{
				Error: ErrorBody{Code: "FORBIDDEN", Message: "password change update is limited to the signed-in user"},
			})
		}

		now := time.Now().Format(time.RFC3339)
		ref := client.NewRef(invitesCollection).Child(uid)
		var invite map[string]interface{}
		if err := ref.Get(c.Request().Context(), &invite); err != nil || len(invite) == 0 {
			return c.JSON(http.StatusOK, map[string]string{"status": "no_invite"})
		}
		if err := ref.Update(c.Request().Context(), map[string]interface{}{
			"passwordChangedAt": now,
		}); err != nil {
			return internalError(c)
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "passwordChangedAt": now})
	}
}

func revokeInviteHandler(deps InviteDeps) echo.HandlerFunc {
	return func(c echo.Context) error {
		actor, ok := appctx.ActorFromContext(c.Request().Context())
		if !ok {
			return c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: ErrorBody{Code: "UNAUTHORIZED", Message: "sign-in is required"},
			})
		}

		uid := strings.TrimSpace(c.Param("uid"))
		if uid == "" {
			return validationError(c, "uid", "is required")
		}
		if actor.Subject == uid {
			return c.JSON(http.StatusForbidden, ErrorResponse{
				Error: ErrorBody{Code: "FORBIDDEN", Message: "cannot revoke your own invite"},
			})
		}

		ctx := c.Request().Context()
		var invite map[string]interface{}
		if err := deps.DBClient.NewRef(invitesCollection).Child(uid).Get(ctx, &invite); err != nil || len(invite) == 0 {
			return notFoundError(c, "invite not found")
		}

		if err := rollbackInvite(ctx, deps, uid); err != nil {
			deps.Logger.Error("failed to revoke invite", slog.Any("error", err), slog.String("uid", uid))
			return internalErrorWithLog(c, "invite revoke failed", err, "uid", uid)
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "revoked"})
	}
}

func rollbackInvite(ctx context.Context, deps InviteDeps, uid string) error {
	var firstErr error
	if err := deps.FirebaseAuth.DeleteFirebaseUser(ctx, uid); err != nil && !strings.Contains(strings.ToLower(err.Error()), "no user record") {
		firstErr = err
	}

	if err := deps.DBClient.NewRef(invitesCollection).Child(uid).Delete(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := deps.DBClient.NewRef(profilesCollection).Child(uid).Delete(ctx); err != nil && firstErr == nil {
		firstErr = err
	}

	var users map[string]map[string]interface{}
	if err := deps.DBClient.NewRef("users").Get(ctx, &users); err == nil {
		for userID, user := range users {
			firebaseUID, _ := user["firebase_uid"].(string)
			if firebaseUID != uid {
				continue
			}
			if err := deps.DBClient.NewRef("users").Child(userID).Delete(ctx); err != nil && firstErr == nil {
				firstErr = err
			}

			var members map[string]map[string]interface{}
			if err := deps.DBClient.NewRef("organization_members").Get(ctx, &members); err == nil {
				for memberID, member := range members {
					memberUserID, _ := member["user_id"].(string)
					if memberUserID != userID {
						continue
					}
					if err := deps.DBClient.NewRef("organization_members").Child(memberID).Delete(ctx); err != nil && firstErr == nil {
						firstErr = err
					}
				}
			}
			break
		}
	}

	return firstErr
}

func sendInviteEmail(ctx context.Context, apiKey, fromEmail, loginURL, toEmail, password string) error {
	loginURL = strings.TrimSpace(loginURL)
	if loginURL == "" {
		loginURL = "https://backstage.stepbycode.work/login"
	}

	htmlBody := fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto; padding: 2rem;">
  <h2 style="color: #df6900;">Backstage へようこそ</h2>
  <p>StepByCode の運営ツール Backstage に招待されました。</p>
  <p>以下の情報でログインしてください:</p>
  <div style="background: #f5f5f5; padding: 1rem; border-radius: 8px; margin: 1rem 0;">
    <p style="margin: 0.25rem 0;"><strong>メールアドレス:</strong> %s</p>
    <p style="margin: 0.25rem 0;"><strong>パスワード:</strong> %s</p>
  </div>
  <p style="margin: 1rem 0;"><a href="%s" style="display: inline-block; background: #df6900; color: #fff; text-decoration: none; padding: 0.75rem 1rem; border-radius: 8px;">ログイン画面を開く</a></p>
  <p style="margin: 0.5rem 0; font-size: 0.875rem;">ボタンが開けない場合: <a href="%s">%s</a></p>
  <p style="color: #666; font-size: 0.875rem;">初回ログイン後、プロフィールの登録をお願いします。</p>
</div>`, toEmail, password, loginURL, loginURL, loginURL)
	textBody := fmt.Sprintf("Backstage に招待されました。\n\nメールアドレス: %s\n初回パスワード: %s\nログイン画面: %s\n\n初回ログイン後、プロフィールの登録をお願いします。", toEmail, password, loginURL)

	payload := map[string]interface{}{
		"from":    fromEmail,
		"to":      []string{toEmail},
		"subject": "【Backstage】招待のお知らせ",
		"html":    htmlBody,
		"text":    textBody,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send resend email: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("resend returned %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	return nil
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
