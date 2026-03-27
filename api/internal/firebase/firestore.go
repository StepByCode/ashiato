package firebase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Firestore provides CRUD operations via the Firestore REST API.
type Firestore struct {
	app     *App
	baseURL string
}

// NewFirestore creates a Firestore client for the given App.
func NewFirestore(app *App) *Firestore {
	return &Firestore{
		app:     app,
		baseURL: fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/(default)/documents", app.ProjectID),
	}
}

// Document represents a Firestore document with typed fields.
type Document struct {
	Name       string                 `json:"name"`
	Fields     map[string]interface{} `json:"fields"`
	CreateTime string                 `json:"createTime,omitempty"`
	UpdateTime string                 `json:"updateTime,omitempty"`
}

// Get retrieves a single document by collection and document ID.
func (f *Firestore) Get(ctx context.Context, collection, docID string) (*Document, error) {
	url := fmt.Sprintf("%s/%s/%s", f.baseURL, collection, docID)
	return f.doRequest(ctx, http.MethodGet, url, nil)
}

// Set creates or overwrites a document with the given ID.
func (f *Firestore) Set(ctx context.Context, collection, docID string, fields map[string]interface{}) (*Document, error) {
	url := fmt.Sprintf("%s/%s?documentId=%s", f.baseURL, collection, docID)
	body := map[string]interface{}{"fields": fields}
	return f.doRequest(ctx, http.MethodPost, url, body)
}

// Patch updates specific fields of a document.
func (f *Firestore) Patch(ctx context.Context, collection, docID string, fields map[string]interface{}, fieldPaths []string) (*Document, error) {
	parts := make([]string, 0, len(fieldPaths))
	for _, fp := range fieldPaths {
		parts = append(parts, "updateMask.fieldPaths="+fp)
	}
	url := fmt.Sprintf("%s/%s/%s?%s", f.baseURL, collection, docID, strings.Join(parts, "&"))
	body := map[string]interface{}{"fields": fields}
	return f.doRequest(ctx, http.MethodPatch, url, body)
}

// ListResponse contains the result of a list operation.
type ListResponse struct {
	Documents []Document `json:"documents"`
}

// List retrieves all documents in a collection.
func (f *Firestore) List(ctx context.Context, collection string) ([]Document, error) {
	url := fmt.Sprintf("%s/%s", f.baseURL, collection)

	token, err := f.app.AccessToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("firestore list %s: %d %s", collection, resp.StatusCode, body)
	}

	var result ListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Documents, nil
}

// RunQuery executes a structured query against a collection.
func (f *Firestore) RunQuery(ctx context.Context, query map[string]interface{}) ([]Document, error) {
	url := f.baseURL + ":runQuery"

	token, err := f.app.AccessToken(ctx)
	if err != nil {
		return nil, err
	}

	jsonBody, _ := json.Marshal(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("firestore query: %d %s", resp.StatusCode, body)
	}

	var results []struct {
		Document *Document `json:"document"`
	}
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, err
	}

	docs := make([]Document, 0, len(results))
	for _, r := range results {
		if r.Document != nil {
			docs = append(docs, *r.Document)
		}
	}
	return docs, nil
}

// Delete removes a document.
func (f *Firestore) Delete(ctx context.Context, collection, docID string) error {
	url := fmt.Sprintf("%s/%s/%s", f.baseURL, collection, docID)

	token, err := f.app.AccessToken(ctx)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("firestore delete: %d %s", resp.StatusCode, body)
	}
	return nil
}

func (f *Firestore) doRequest(ctx context.Context, method, url string, body interface{}) (*Document, error) {
	token, err := f.app.AccessToken(ctx)
	if err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("firestore %s: %d %s", method, resp.StatusCode, respBody)
	}

	var doc Document
	if err := json.Unmarshal(respBody, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// ErrNotFound is returned when a document does not exist.
var ErrNotFound = fmt.Errorf("firestore: document not found")

// Helper functions to convert between Go types and Firestore value format.

// StringVal creates a Firestore string value.
func StringVal(s string) map[string]interface{} {
	return map[string]interface{}{"stringValue": s}
}

// TimestampVal creates a Firestore timestamp value.
func TimestampVal(t time.Time) map[string]interface{} {
	return map[string]interface{}{"timestampValue": t.Format(time.RFC3339Nano)}
}

// GetStringField extracts a string from Firestore document fields.
func GetStringField(fields map[string]interface{}, key string) string {
	v, ok := fields[key]
	if !ok {
		return ""
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return ""
	}
	s, _ := m["stringValue"].(string)
	return s
}

// GetTimestampField extracts a time.Time from Firestore document fields.
func GetTimestampField(fields map[string]interface{}, key string) *time.Time {
	v, ok := fields[key]
	if !ok {
		return nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	s, _ := m["timestampValue"].(string)
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return nil
	}
	return &t
}
