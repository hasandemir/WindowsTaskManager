package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/anomaly"
	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/storage"
)

func TestProviderOf(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "anthropic"},
		{"anthropic", "anthropic"},
		{"Anthropic", "anthropic"},
		{"  ANTHROPIC  ", "anthropic"},
		{"openai", "openai"},
		{"OpenAI", "openai"},
		{"openai-compatible", "openai"},
		{"openrouter", "openai"},
		{"groq", "openai"},
		{"deepseek", "openai"},
		{"ollama", "openai"},
		{"lmstudio", "openai"},
		{"lm-studio", "openai"},
		{"unknown", "unknown"},
	}
	for _, c := range cases {
		got := providerOf(&config.Config{AI: config.AIConfig{Provider: c.in}})
		if got != c.want {
			t.Errorf("providerOf(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEffectiveEndpoint(t *testing.T) {
	cases := []struct {
		provider string
		endpoint string
		want     string
	}{
		{"", "", "https://api.anthropic.com/v1/messages"},
		{"anthropic", "", "https://api.anthropic.com/v1/messages"},
		{"openai", "", "https://api.openai.com/v1/chat/completions"},
		{"openai", "http://localhost:11434/v1/chat/completions", "http://localhost:11434/v1/chat/completions"},
		{"groq", "", "https://api.openai.com/v1/chat/completions"}, // alias of openai
	}
	for _, c := range cases {
		got := effectiveEndpoint(&config.Config{AI: config.AIConfig{Provider: c.provider, Endpoint: c.endpoint}})
		if got != c.want {
			t.Errorf("effectiveEndpoint(%q,%q) = %q, want %q", c.provider, c.endpoint, got, c.want)
		}
	}
}

// TestAnthropicCall stands up a fake Anthropic server, runs Analyze, and
// verifies the request shape and the response unwrap.
func TestAnthropicCall(t *testing.T) {
	var receivedBody anthropicRequest
	var receivedKey, receivedVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("x-api-key")
		receivedVersion = r.Header.Get("anthropic-version")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"hello from claude"}]}`))
	}))
	defer srv.Close()

	cfg := &config.Config{AI: config.AIConfig{
		Enabled:              true,
		Provider:             "anthropic",
		APIKey:               "sk-ant-test",
		Model:                "claude-test",
		Endpoint:             srv.URL,
		MaxTokens:            128,
		MaxRequestsPerMinute: 60,
		Language:             "en",
	}}
	a := NewAdvisor(cfg, storage.NewStore(60, 10), func() []anomaly.Alert { return nil })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := a.Analyze(ctx, "what is up?")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if got.Answer != "hello from claude" {
		t.Errorf("answer = %q", got.Answer)
	}
	if receivedKey != "sk-ant-test" {
		t.Errorf("x-api-key = %q", receivedKey)
	}
	if receivedVersion != "2023-06-01" {
		t.Errorf("anthropic-version = %q", receivedVersion)
	}
	if receivedBody.Model != "claude-test" {
		t.Errorf("model = %q", receivedBody.Model)
	}
	if len(receivedBody.Messages) != 1 || receivedBody.Messages[0].Role != "user" {
		t.Errorf("messages = %+v", receivedBody.Messages)
	}
	if !strings.Contains(receivedBody.Messages[0].Content, "what is up?") {
		t.Errorf("user question not propagated: %q", receivedBody.Messages[0].Content)
	}
}

// TestOpenAICall covers the /v1/chat/completions path with a fake server,
// including the Bearer auth and extra-headers passthrough.
func TestOpenAICall(t *testing.T) {
	var receivedBody openaiRequest
	var receivedAuth, receivedReferer string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedReferer = r.Header.Get("HTTP-Referer")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello from gpt"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	cfg := &config.Config{AI: config.AIConfig{
		Enabled:              true,
		Provider:             "openai",
		APIKey:               "sk-test",
		Model:                "gpt-test",
		Endpoint:             srv.URL,
		MaxTokens:            128,
		MaxRequestsPerMinute: 60,
		Language:             "en",
		ExtraHeaders:         map[string]string{"HTTP-Referer": "http://wtm.local"},
	}}
	a := NewAdvisor(cfg, storage.NewStore(60, 10), func() []anomaly.Alert { return nil })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := a.Analyze(ctx, "diagnose")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if got.Answer != "hello from gpt" {
		t.Errorf("answer = %q", got.Answer)
	}
	if receivedAuth != "Bearer sk-test" {
		t.Errorf("Authorization = %q", receivedAuth)
	}
	if receivedReferer != "http://wtm.local" {
		t.Errorf("HTTP-Referer = %q", receivedReferer)
	}
	if receivedBody.Model != "gpt-test" {
		t.Errorf("model = %q", receivedBody.Model)
	}
	if len(receivedBody.Messages) != 2 {
		t.Fatalf("expected 2 messages (system+user), got %d", len(receivedBody.Messages))
	}
	if receivedBody.Messages[0].Role != "system" || receivedBody.Messages[1].Role != "user" {
		t.Errorf("messages = %+v", receivedBody.Messages)
	}
	if !strings.Contains(receivedBody.Messages[1].Content, "diagnose") {
		t.Errorf("user question not propagated: %q", receivedBody.Messages[1].Content)
	}
}

func TestAnalyzeDisabled(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Enabled: false, MaxRequestsPerMinute: 5}}
	a := NewAdvisor(cfg, storage.NewStore(60, 10), nil)
	if _, err := a.Analyze(context.Background(), "x"); err == nil {
		t.Error("expected error when disabled")
	}
}

func TestUnknownProviderError(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{
		Enabled: true, Provider: "ollama-rest", APIKey: "x", MaxRequestsPerMinute: 5,
	}}
	a := NewAdvisor(cfg, storage.NewStore(60, 10), nil)
	_, err := a.Analyze(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "unknown AI provider") {
		t.Errorf("expected unknown provider error, got %v", err)
	}
}
