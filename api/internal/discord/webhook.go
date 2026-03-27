package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WebhookClient sends messages via Discord Webhook API.
type WebhookClient struct {
	webhookURL string
	httpClient *http.Client
}

func NewWebhookClient(webhookURL string) *WebhookClient {
	return &WebhookClient{
		webhookURL: webhookURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type webhookPayload struct {
	Content string `json:"content"`
}

type webhookResponse struct {
	ID string `json:"id"`
}

// Send posts a message to Discord via webhook and returns the message ID.
func (w *WebhookClient) Send(ctx context.Context, content string) (string, error) {
	payload, err := json.Marshal(webhookPayload{Content: content})
	if err != nil {
		return "", fmt.Errorf("marshal webhook payload: %w", err)
	}

	// ?wait=true makes Discord return the created message (with ID).
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.webhookURL+"?wait=true", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send webhook request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("discord webhook returned %d: %s", resp.StatusCode, string(body))
	}

	var result webhookResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode webhook response: %w", err)
	}
	return result.ID, nil
}
