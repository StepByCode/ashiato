package firebase

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// App holds Firebase project configuration and cached access tokens.
type App struct {
	ProjectID        string
	serviceEmail     string
	privateKey       *rsa.PrivateKey
	mu               sync.Mutex
	cachedToken      string
	cachedTokenExpAt time.Time
}

// ServiceAccount represents the JSON key file for a GCP service account.
type ServiceAccount struct {
	ProjectID   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

// NewApp creates a Firebase App from a service account JSON blob.
func NewApp(saJSON []byte) (*App, error) {
	var sa ServiceAccount
	if err := json.Unmarshal(saJSON, &sa); err != nil {
		return nil, fmt.Errorf("firebase: parse service account: %w", err)
	}
	if sa.ProjectID == "" || sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, fmt.Errorf("firebase: missing required fields in service account JSON")
	}

	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		return nil, fmt.Errorf("firebase: failed to decode PEM private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("firebase: parse private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("firebase: private key is not RSA")
	}

	return &App{
		ProjectID:    sa.ProjectID,
		serviceEmail: sa.ClientEmail,
		privateKey:   rsaKey,
	}, nil
}

// AccessToken returns a cached or freshly minted access token for Google APIs.
func (a *App) AccessToken(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cachedToken != "" && time.Now().Before(a.cachedTokenExpAt) {
		return a.cachedToken, nil
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   a.serviceEmail,
		"sub":   a.serviceEmail,
		"aud":   "https://oauth2.googleapis.com/token",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
		"scope": "https://www.googleapis.com/auth/datastore",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(a.privateKey)
	if err != nil {
		return "", fmt.Errorf("firebase: sign jwt: %w", err)
	}

	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant_type:jwt-bearer"},
		"assertion":  {signed},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("firebase: token exchange: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("firebase: token exchange failed (%d): %s", resp.StatusCode, body)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("firebase: parse token response: %w", err)
	}

	a.cachedToken = result.AccessToken
	a.cachedTokenExpAt = now.Add(time.Duration(result.ExpiresIn-60) * time.Second)
	return a.cachedToken, nil
}
