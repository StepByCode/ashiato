package auth

import (
	"context"
	"net/http"
	"strings"

	fbauth "firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"

	"github.com/dokkiitech/ashiato/api/internal/domain"
	appctx "github.com/dokkiitech/ashiato/api/internal/middleware"
)

type Verifier interface {
	Verify(ctx context.Context, rawToken string) (domain.Principal, error)
}

type Syncer interface {
	SyncPrincipal(ctx context.Context, principal domain.Principal) (domain.Actor, error)
}

type Authenticator struct {
	verifier Verifier
	syncer   Syncer
}

type firebaseVerifier struct {
	client *fbauth.Client
}

func NewFirebaseVerifier(client *fbauth.Client) Verifier {
	return &firebaseVerifier{client: client}
}

func (f *firebaseVerifier) Verify(ctx context.Context, rawToken string) (domain.Principal, error) {
	token, err := f.client.VerifyIDToken(ctx, rawToken)
	if err != nil {
		return domain.Principal{}, domain.NewAppError(domain.ErrorCodeUnauthorized, "invalid firebase token", err)
	}

	subject := token.UID
	email, _ := token.Claims["email"].(string)
	name, _ := token.Claims["name"].(string)
	if name == "" {
		name, _ = token.Claims["displayName"].(string)
	}

	if subject == "" {
		return domain.Principal{}, domain.NewAppError(domain.ErrorCodeUnauthorized, "missing uid in token", nil)
	}
	if email == "" {
		return domain.Principal{}, domain.NewAppError(domain.ErrorCodeUnauthorized, "missing email in token", nil)
	}
	if name == "" {
		name = email
	}

	return domain.Principal{
		Subject: subject,
		Email:   strings.ToLower(email),
		Name:    name,
	}, nil
}

func NewAuthenticator(verifier Verifier, syncer Syncer) *Authenticator {
	return &Authenticator{verifier: verifier, syncer: syncer}
}

// isSimpleAPIPath returns true for endpoints defined in docs/backend-api-request.md
// that do not require authentication in the current phase (暫定で認証不要).
func isSimpleAPIPath(reqPath string) bool {
	if reqPath == "/api/v1/meeting" ||
		strings.HasPrefix(reqPath, "/api/v1/publicity/") {
		return true
	}
	if reqPath == "/api/v1/tasks" {
		return true
	}
	if strings.HasPrefix(reqPath, "/api/v1/tasks/") {
		suffix := strings.TrimPrefix(reqPath, "/api/v1/tasks/")
		parts := strings.SplitN(suffix, "/", 2)
		if len(parts) == 2 {
			switch parts[1] {
			case "owner", "url", "state":
				return true
			}
		}
	}
	if strings.HasPrefix(reqPath, "/api/v1/profile/") {
		return true
	}
	if reqPath == "/api/v1/invite" || reqPath == "/api/v1/invites" {
		return true
	}
	return false
}

func (a *Authenticator) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestPath := c.Request().URL.Path
			switch {
			case isSimpleAPIPath(requestPath):
				return next(c)
			case strings.HasPrefix(requestPath, "/api/"):
				return a.authorizeUser(next, c)
			default:
				return next(c)
			}
		}
	}
}

func (a *Authenticator) authorizeUser(next echo.HandlerFunc, c echo.Context) error {
	authorization := c.Request().Header.Get("Authorization")
	if authorization == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"code":    "unauthorized",
			"message": "missing bearer token",
		})
	}

	rawToken, ok := strings.CutPrefix(authorization, "Bearer ")
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"code":    "unauthorized",
			"message": "invalid authorization header",
		})
	}

	principal, err := a.verifier.Verify(c.Request().Context(), rawToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"code":    "unauthorized",
			"message": err.Error(),
		})
	}

	actor, err := a.syncer.SyncPrincipal(c.Request().Context(), principal)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"code":    "unauthorized",
			"message": err.Error(),
		})
	}

	ctx := appctx.WithActor(c.Request().Context(), actor)
	c.SetRequest(c.Request().WithContext(ctx))
	return next(c)
}
