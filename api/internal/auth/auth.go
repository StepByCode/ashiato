package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"github.com/dokkiitech/ashiato/api/internal/config"
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
	botToken string
}

func NewVerifier(ctx context.Context, cfg config.Config) (Verifier, error) {
	switch cfg.AuthMode {
	case config.AuthModeStub:
		return &stubVerifier{secret: []byte(cfg.DevJWTSecret)}, nil
	case config.AuthModeOIDC:
		provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuerURL)
		if err != nil {
			return nil, err
		}
		return &oidcVerifier{
			verifier: provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID}),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported auth mode: %s", cfg.AuthMode)
	}
}

func NewAuthenticator(verifier Verifier, syncer Syncer, botToken string) *Authenticator {
	return &Authenticator{verifier: verifier, syncer: syncer, botToken: botToken}
}

func (a *Authenticator) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestPath := c.Request().URL.Path
			switch {
			case strings.HasPrefix(requestPath, "/internal/"):
				return a.authorizeBot(next, c)
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

func (a *Authenticator) authorizeBot(next echo.HandlerFunc, c echo.Context) error {
	received := c.Request().Header.Get("X-Bot-Token")
	if received == "" || received != a.botToken {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"code":    "unauthorized",
			"message": "invalid bot token",
		})
	}
	return next(c)
}

type stubVerifier struct {
	secret []byte
}

func (s *stubVerifier) Verify(_ context.Context, rawToken string) (domain.Principal, error) {
	token, err := jwt.Parse(rawToken, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unsupported signing method")
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return domain.Principal{}, errors.New("invalid stub token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return domain.Principal{}, errors.New("invalid stub claims")
	}

	return principalFromMapClaims(claims)
}

type oidcVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func (o *oidcVerifier) Verify(ctx context.Context, rawToken string) (domain.Principal, error) {
	idToken, err := o.verifier.Verify(ctx, rawToken)
	if err != nil {
		return domain.Principal{}, errors.New("invalid oidc token")
	}

	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return domain.Principal{}, errors.New("invalid oidc claims")
	}

	return principalFromClaimsMap(claims)
}

func principalFromMapClaims(claims jwt.MapClaims) (domain.Principal, error) {
	plain := map[string]any{}
	for key, value := range claims {
		plain[key] = value
	}
	return principalFromClaimsMap(plain)
}

func principalFromClaimsMap(claims map[string]any) (domain.Principal, error) {
	subject := strings.TrimSpace(asString(claims["sub"]))
	email := strings.TrimSpace(asString(claims["email"]))
	name := strings.TrimSpace(asString(claims["name"]))
	if name == "" {
		name = strings.TrimSpace(asString(claims["preferred_username"]))
	}

	switch {
	case subject == "":
		return domain.Principal{}, errors.New("missing sub claim")
	case email == "":
		return domain.Principal{}, errors.New("missing email claim")
	case name == "":
		return domain.Principal{}, errors.New("missing name claim")
	}

	return domain.Principal{
		Subject: subject,
		Email:   strings.ToLower(email),
		Name:    name,
	}, nil
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}
