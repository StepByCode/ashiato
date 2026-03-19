package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/dokkiitech/ashiato/api/internal/config"
)

func TestStubVerifierVerify(t *testing.T) {
	verifier, err := NewVerifier(context.Background(), config.Config{
		AuthMode:     config.AuthModeStub,
		DevJWTSecret: "secret",
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "stub-user",
		"email": "stub@example.com",
		"name":  "Stub User",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	rawToken, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	principal, err := verifier.Verify(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if principal.Subject != "stub-user" || principal.Email != "stub@example.com" || principal.Name != "Stub User" {
		t.Fatalf("unexpected principal: %+v", principal)
	}
}

func TestOIDCVerifierVerify(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":   server.URL,
			"jwks_uri": server.URL + "/keys",
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{
				{
					"kty": "RSA",
					"alg": "RS256",
					"use": "sig",
					"kid": "test-key",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
				},
			},
		})
	})

	verifier, err := NewVerifier(context.Background(), config.Config{
		AuthMode:      config.AuthModeOIDC,
		OIDCIssuerURL: server.URL,
		OIDCClientID:  "ashiato-client",
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":   server.URL,
		"aud":   "ashiato-client",
		"sub":   "oidc-user",
		"email": "oidc@example.com",
		"name":  "OIDC User",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	})
	token.Header["kid"] = "test-key"

	rawToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	principal, err := verifier.Verify(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if principal.Subject != "oidc-user" || principal.Email != "oidc@example.com" || principal.Name != "OIDC User" {
		t.Fatalf("unexpected principal: %+v", principal)
	}
}
