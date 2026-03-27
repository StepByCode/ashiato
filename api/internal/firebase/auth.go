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
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthVerifier verifies Firebase Auth ID tokens.
type AuthVerifier struct {
	projectID string
	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	keysExpAt time.Time
}

// NewAuthVerifier creates a verifier for Firebase Auth ID tokens.
func NewAuthVerifier(projectID string) *AuthVerifier {
	return &AuthVerifier{projectID: projectID}
}

// DecodedToken contains verified claims from a Firebase ID token.
type DecodedToken struct {
	UID   string
	Email string
	Name  string
}

const googleCertsURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"

// VerifyIDToken validates a Firebase Auth ID token and returns the decoded claims.
func (v *AuthVerifier) VerifyIDToken(ctx context.Context, idToken string) (*DecodedToken, error) {
	keys, err := v.fetchPublicKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase auth: fetch keys: %w", err)
	}

	// Parse without verification first to get the key ID
	parser := jwt.NewParser()
	unverified, _, err := parser.ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("firebase auth: parse token: %w", err)
	}

	kid, ok := unverified.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("firebase auth: missing kid header")
	}

	pubKey, ok := keys[kid]
	if !ok {
		return nil, fmt.Errorf("firebase auth: unknown key id: %s", kid)
	}

	// Verify the token
	token, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	},
		jwt.WithIssuer(fmt.Sprintf("https://securetoken.google.com/%s", v.projectID)),
		jwt.WithAudience(v.projectID),
	)
	if err != nil {
		return nil, fmt.Errorf("firebase auth: verify token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("firebase auth: invalid claims")
	}

	uid, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)

	if uid == "" {
		return nil, fmt.Errorf("firebase auth: missing sub claim")
	}

	return &DecodedToken{
		UID:   uid,
		Email: email,
		Name:  name,
	}, nil
}

func (v *AuthVerifier) fetchPublicKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.keys != nil && time.Now().Before(v.keysExpAt) {
		return v.keys, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleCertsURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch google certs: %d", resp.StatusCode)
	}

	var certs map[string]string
	if err := json.Unmarshal(body, &certs); err != nil {
		return nil, err
	}

	keys := make(map[string]*rsa.PublicKey, len(certs))
	for kid, certPEM := range certs {
		block, _ := pem.Decode([]byte(certPEM))
		if block == nil {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		rsaKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			continue
		}
		keys[kid] = rsaKey
	}

	v.keys = keys
	// Cache keys for 1 hour (Google rotates roughly every 6h)
	v.keysExpAt = time.Now().Add(time.Hour)
	return v.keys, nil
}
