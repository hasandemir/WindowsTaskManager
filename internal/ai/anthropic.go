package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
)

// anthropicRequest mirrors the Anthropic Messages API request body.
type anthropicRequest struct {
	Model     string             `json:"model"`
	System    string             `json:"system"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// callAnthropic invokes Anthropic's /v1/messages endpoint (or any compatible
// endpoint set via cfg.AI.Endpoint).
func (a *Advisor) callAnthropic(ctx context.Context, cfg *config.Config, prompt string) (string, error) {
	reqBody := anthropicRequest{
		Model:     cfg.AI.Model,
		System:    SystemPrompt(cfg.AI.Language),
		MaxTokens: cfg.AI.MaxTokens,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	endpoint := normalizeAnthropicEndpoint(cfg.AI.Endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.AI.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	for k, v := range cfg.AI.ExtraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic call: %w", err)
	}
	defer resp.Body.Close()

	body, err := readProviderBody(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("anthropic %d: %s", resp.StatusCode, truncateForError(string(body)))
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse response: %w (body: %s)", err, truncateForError(string(body)))
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("anthropic %s: %s", parsed.Error.Type, parsed.Error.Message)
	}
	// Prefer explicit text blocks, but fall back to any non-empty Text field —
	// some Anthropic-compatible providers (z.ai, others) use slightly different
	// content types like "message" or omit the Type altogether.
	for _, c := range parsed.Content {
		if c.Type == "text" && c.Text != "" {
			return c.Text, nil
		}
	}
	for _, c := range parsed.Content {
		if c.Text != "" {
			return c.Text, nil
		}
	}
	return "", fmt.Errorf("anthropic: no text content in response: %s", truncateForError(string(body)))
}

// normalizeAnthropicEndpoint ensures the endpoint points at a /messages URL.
// Users often paste a base URL (e.g. https://api.z.ai/api/anthropic) and expect
// it to just work; append /v1/messages when missing.
func normalizeAnthropicEndpoint(endpoint string) string {
	if endpoint == "" {
		return "https://api.anthropic.com/v1/messages"
	}
	trimmed := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(trimmed, "/messages") {
		return trimmed
	}
	return trimmed + "/v1/messages"
}
