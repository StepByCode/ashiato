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
	Content string         `json:"content,omitempty"`
	Embeds  []WebhookEmbed `json:"embeds,omitempty"`
}

type WebhookEmbed struct {
	Title       string             `json:"title,omitempty"`
	Description string             `json:"description,omitempty"`
	Color       int                `json:"color,omitempty"`
	Fields      []WebhookEmbedField `json:"fields,omitempty"`
}

type WebhookEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type webhookResponse struct {
	ID string `json:"id"`
}

// Send posts a message to Discord via webhook and returns the message ID.
func (w *WebhookClient) Send(ctx context.Context, content string) (string, error) {
	return w.sendPayload(ctx, webhookPayload{Content: content})
}

func (w *WebhookClient) SendEmbed(ctx context.Context, title, description string, fields []WebhookEmbedField) (string, error) {
	return w.sendPayload(ctx, webhookPayload{
		Embeds: []WebhookEmbed{
			{
				Title:       title,
				Description: description,
				Color:       0xDF6900,
				Fields:      fields,
			},
		},
	})
}

func (w *WebhookClient) sendPayload(ctx context.Context, payload webhookPayload) (string, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal webhook payload: %w", err)
	}

	// ?wait=true makes Discord return the created message (with ID).
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.webhookURL+"?wait=true", bytes.NewReader(bodyBytes))
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
