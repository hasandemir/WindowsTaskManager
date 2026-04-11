package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/anomaly"
	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/storage"
)

// Advisor wraps one or more LLM provider backends with caching and rate
// limiting. It implements the server.AIAdvisor interface
// (Enabled / Status / Analyze).
//
// Currently supported providers:
//
//   - "anthropic"            -> Anthropic Messages API ( /v1/messages )
//   - "openai"               -> any OpenAI-compatible /v1/chat/completions
//     endpoint (OpenAI, OpenRouter, Groq, DeepSeek,
//     Together, Mistral, Ollama, LM Studio, ...)
//
// The endpoint, model, api_key and extra_headers fields can all be
// overridden in the config so the user can point this at any provider
// without code changes.
type Advisor struct {
	mu         sync.RWMutex
	cfg        *config.Config
	store      *storage.Store
	alertsRef  func() []anomaly.Alert
	rl         *TokenBucket
	cache      *Cache
	httpClient *http.Client

	lastErr        string
	lastReqAt      time.Time
	totalReqs      uint64
	totalCacheHits uint64
}

// NewAdvisor builds a new advisor. alertSource is a closure that returns
// the current active alerts for the prompt context.
func NewAdvisor(cfg *config.Config, store *storage.Store, alertSource func() []anomaly.Alert) *Advisor {
	return &Advisor{
		cfg:        cfg,
		store:      store,
		alertsRef:  alertSource,
		rl:         NewTokenBucket(cfg.AI.MaxRequestsPerMinute),
		cache:      NewCache(60*time.Second, 64),
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// SetConfig hot-swaps the active config.
func (a *Advisor) SetConfig(cfg *config.Config) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg = cfg
	a.rl = NewTokenBucket(cfg.AI.MaxRequestsPerMinute)
}

// Enabled reports whether the advisor is configured to run.
func (a *Advisor) Enabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg.AI.Enabled && a.cfg.AI.APIKey != ""
}

// Status returns a JSON-friendly status snapshot.
func (a *Advisor) Status() map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return map[string]any{
		"enabled":          a.cfg.AI.Enabled,
		"configured":       a.cfg.AI.APIKey != "",
		"provider":         providerOf(a.cfg),
		"endpoint":         effectiveEndpoint(a.cfg),
		"model":            a.cfg.AI.Model,
		"language":         a.cfg.AI.Language,
		"max_per_minute":   a.cfg.AI.MaxRequestsPerMinute,
		"tokens_available": a.rl.Available(),
		"cache_size":       a.cache.Size(),
		"total_requests":   a.totalReqs,
		"cache_hits":       a.totalCacheHits,
		"last_request":     a.lastReqAt.Format(time.RFC3339),
		"last_error":       a.lastErr,
	}
}

// Analyze runs an LLM call. The user-supplied question is appended to a
// snapshot+alerts context. Cached responses are returned without making
// an HTTP call. The returned AnalyzeResult carries both the prose answer
// (with any <actions> block stripped) and the parsed structured actions.
func (a *Advisor) Analyze(ctx context.Context, userQuestion string) (*AnalyzeResult, error) {
	a.mu.RLock()
	cfg := a.cfg
	a.mu.RUnlock()
	if !cfg.AI.Enabled || cfg.AI.APIKey == "" {
		return nil, errors.New("AI advisor disabled")
	}

	snap := a.store.Latest()
	var alerts []anomaly.Alert
	if a.alertsRef != nil {
		alerts = a.alertsRef()
	}
	prompt := BuildPrompt(cfg.AI.Language, snap, alerts, cfg.AI.IncludeProcessTree, cfg.AI.IncludePortMap, userQuestion)

	if cached, ok := a.cache.Get(prompt); ok {
		a.mu.Lock()
		a.totalCacheHits++
		a.mu.Unlock()
		cleaned, actions := parseActionsBlock(cached)
		return &AnalyzeResult{Answer: cleaned, Actions: actions}, nil
	}

	if !a.rl.Take() {
		return nil, errors.New("AI rate limit exceeded; try again later")
	}

	answer, err := a.callProvider(ctx, cfg, prompt)
	a.mu.Lock()
	a.totalReqs++
	a.lastReqAt = time.Now()
	if err != nil {
		a.lastErr = err.Error()
	} else {
		a.lastErr = ""
	}
	a.mu.Unlock()
	if err != nil {
		return nil, err
	}

	a.cache.Set(prompt, answer)
	cleaned, actions := parseActionsBlock(answer)
	return &AnalyzeResult{Answer: cleaned, Actions: actions}, nil
}

// callProvider dispatches the LLM call based on cfg.AI.Provider.
func (a *Advisor) callProvider(ctx context.Context, cfg *config.Config, prompt string) (string, error) {
	switch providerOf(cfg) {
	case "anthropic":
		return a.callAnthropic(ctx, cfg, prompt)
	case "openai":
		return a.callOpenAI(ctx, cfg, prompt)
	default:
		return "", fmt.Errorf("unknown AI provider %q (use 'anthropic' or 'openai')", cfg.AI.Provider)
	}
}

// providerOf normalizes the provider name. Empty string defaults to anthropic
// for backwards compatibility with older configs.
func providerOf(cfg *config.Config) string {
	p := strings.ToLower(strings.TrimSpace(cfg.AI.Provider))
	switch p {
	case "":
		return "anthropic"
	case "openai-compatible", "openai_compatible", "openrouter", "groq",
		"deepseek", "together", "mistral", "ollama", "lmstudio", "lm-studio":
		return "openai"
	}
	return p
}

// effectiveEndpoint returns the URL the advisor will actually hit, factoring
// in the provider default when the user left endpoint blank.
func effectiveEndpoint(cfg *config.Config) string {
	if cfg.AI.Endpoint != "" {
		return cfg.AI.Endpoint
	}
	switch providerOf(cfg) {
	case "anthropic":
		return "https://api.anthropic.com/v1/messages"
	case "openai":
		return "https://api.openai.com/v1/chat/completions"
	}
	return ""
}
