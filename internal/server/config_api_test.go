package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/ai"
	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
	"github.com/ersinkoc/WindowsTaskManager/internal/storage"
)

func TestConfigUpdateRoundTripViaRouter(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "wtm.yaml")
	cfg := config.DefaultConfig()
	cfg.AI.APIKey = "sk-secret-1234"
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("seed save: %v", err)
	}

	s, applied := newTestServer(t, cfgPath, cfg)
	body := configUpdateDTO{
		Server: &configServerUpdateDTO{
			OpenBrowser: boolPtr(false),
		},
		Monitoring: &configMonitoringUpdateDTO{
			IntervalMS:         intPtr(1500),
			HistoryDurationSec: intPtr(900),
			MaxProcesses:       intPtr(512),
		},
		Controller: &configControllerUpdateDTO{
			ConfirmKillSystem: boolPtr(false),
		},
		Notifications: &configNotificationsUpdateDTO{
			TrayBalloon:         boolPtr(false),
			BalloonRateLimitSec: intPtr(45),
			BalloonMinSeverity:  strPtr("warning"),
		},
		UI: &configUIUpdateDTO{
			Theme:                strPtr("light"),
			DefaultSort:          strPtr("memory"),
			DefaultSortOrder:     strPtr("asc"),
			SparklinePoints:      intPtr(90),
			ProcessTablePageSize: intPtr(150),
			RefreshRateMS:        intPtr(1250),
		},
	}
	buf, _ := json.Marshal(body)
	req := authedJSONRequest(http.MethodPut, "/api/v1/config", buf, s.csrfToken)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(*applied) == 0 {
		t.Fatal("OnCfgApply not invoked")
	}

	reloaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Server.OpenBrowser {
		t.Fatal("server.open_browser should be false after update")
	}
	if reloaded.Monitoring.Interval != 1500*time.Millisecond {
		t.Fatalf("interval=%v want 1500ms", reloaded.Monitoring.Interval)
	}
	if reloaded.Monitoring.HistoryDuration != 15*time.Minute {
		t.Fatalf("history_duration=%v want 15m", reloaded.Monitoring.HistoryDuration)
	}
	if reloaded.Controller.ConfirmKillSystem {
		t.Fatal("confirm_kill_system should be false after update")
	}
	if reloaded.Notifications.BalloonMinSeverity != "warning" {
		t.Fatalf("balloon_min_severity=%q", reloaded.Notifications.BalloonMinSeverity)
	}
	if reloaded.UI.DefaultSort != "memory" || reloaded.UI.DefaultSortOrder != "asc" {
		t.Fatalf("unexpected UI sort config: %+v", reloaded.UI)
	}
	if reloaded.AI.APIKey != "sk-secret-1234" {
		t.Fatalf("AI API key should be preserved, got %q", reloaded.AI.APIKey)
	}
}

func TestConnectionsRouteSupportsPIDFilter(t *testing.T) {
	s, _ := newTestServer(t, "", config.DefaultConfig())
	store := storage.NewStore(60, 10)
	store.SetLatest(&metrics.SystemSnapshot{
		Timestamp: time.Now(),
		PortBindings: []metrics.PortBinding{
			{PID: 100, Protocol: "tcp", LocalPort: 443},
			{PID: 200, Protocol: "udp", LocalPort: 5353},
		},
	})
	s.store = store

	req := httptest.NewRequest(http.MethodGet, "/api/v1/connections?pid=200", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Host = "127.0.0.1:19876"
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var rows []metrics.PortBinding
	if err := json.Unmarshal(rr.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 1 || rows[0].PID != 200 {
		t.Fatalf("rows=%+v want one PID 200 row", rows)
	}
}

func TestBrowserRouteSmoke(t *testing.T) {
	s, _ := newTestServer(t, "", config.DefaultConfig())
	s.staticFS = fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte(`<html><head><meta name="wtm-csrf-token" content="__WTM_CSRF_TOKEN__" /></head></html>`)},
		"app.js":     &fstest.MapFile{Data: []byte(`console.log("wtm");`)},
	}
	store := storage.NewStore(60, 10)
	store.SetLatest(&metrics.SystemSnapshot{
		Timestamp: time.Now(),
		CPU:       metrics.CPUMetrics{NumLogical: 8},
	})
	s.store = store

	indexReq := httptest.NewRequest(http.MethodGet, "/", nil)
	indexReq.RemoteAddr = "127.0.0.1:12345"
	indexReq.Host = "127.0.0.1:19876"
	indexRR := httptest.NewRecorder()
	s.router.ServeHTTP(indexRR, indexReq)
	if indexRR.Code != http.StatusOK {
		t.Fatalf("index status=%d", indexRR.Code)
	}
	if !bytes.Contains(indexRR.Body.Bytes(), []byte(s.csrfToken)) {
		t.Fatalf("index did not include csrf token: %q", indexRR.Body.String())
	}

	systemReq := httptest.NewRequest(http.MethodGet, "/api/v1/system", nil)
	systemReq.RemoteAddr = "127.0.0.1:12345"
	systemReq.Host = "127.0.0.1:19876"
	systemRR := httptest.NewRecorder()
	s.router.ServeHTTP(systemRR, systemReq)
	if systemRR.Code != http.StatusOK {
		t.Fatalf("system status=%d body=%s", systemRR.Code, systemRR.Body.String())
	}
}

func TestAIChatRouteSmoke(t *testing.T) {
	s, _ := newTestServer(t, "", config.DefaultConfig())
	advisorCalls := 0
	s.advisor = chatStubAdvisor{
		enabled: true,
		chatFn: func(message string) (string, error) {
			advisorCalls++
			if message != "Explain the spike" {
				t.Fatalf("message=%q want Explain the spike", message)
			}
			return "Spike came from node.exe.", nil
		},
	}

	req := authedJSONRequest(http.MethodPost, "/api/v1/ai/chat", []byte(`{"message":"Explain the spike"}`), s.csrfToken)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if advisorCalls != 1 {
		t.Fatalf("advisorCalls=%d want 1", advisorCalls)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["answer"] != "Spike came from node.exe." {
		t.Fatalf("answer=%v", body["answer"])
	}
}

func TestAIChatRouteSanitizesProviderErrors(t *testing.T) {
	s, _ := newTestServer(t, "", config.DefaultConfig())
	s.advisor = chatStubAdvisor{
		enabled: true,
		chatFn: func(string) (string, error) {
			return "", context.DeadlineExceeded
		},
	}

	req := authedJSONRequest(http.MethodPost, "/api/v1/ai/chat", []byte(`{"message":"hello"}`), s.csrfToken)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got := payload["error"]["message"]
	if got != "AI provider request failed" {
		t.Fatalf("message=%q want generic provider failure", got)
	}
}

func TestConfigGetRedactsAIExtraHeaders(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.AI.ExtraHeaders = map[string]string{
		"Authorization": "Bearer my-secret",
		"X-Title":       "WTM",
	}
	s, _ := newTestServer(t, "", cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Host = "127.0.0.1:19876"
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var got config.Config
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(got.AI.ExtraHeaders["Authorization"], "****") {
		t.Fatalf("authorization header not redacted: %q", got.AI.ExtraHeaders["Authorization"])
	}
	if got.AI.ExtraHeaders["X-Title"] != "WTM" {
		t.Fatalf("non-sensitive header mutated: %q", got.AI.ExtraHeaders["X-Title"])
	}
}

func authedJSONRequest(method, target string, body []byte, csrfToken string) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Host = "127.0.0.1:19876"
	req.Header.Set("Origin", "http://127.0.0.1:19876")
	req.Header.Set("X-WTM-CSRF", csrfToken)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func boolPtr(v bool) *bool { return &v }
func intPtr(v int) *int    { return &v }
func strPtr(v string) *string {
	return &v
}

type chatStubAdvisor struct {
	enabled bool
	chatFn  func(message string) (string, error)
}

func (s chatStubAdvisor) Enabled() bool          { return s.enabled }
func (s chatStubAdvisor) Status() map[string]any { return map[string]any{"enabled": s.enabled} }
func (s chatStubAdvisor) Analyze(context.Context, string) (*ai.AnalyzeResult, error) {
	return &ai.AnalyzeResult{Answer: "unused"}, nil
}
func (s chatStubAdvisor) Chat(_ context.Context, message string) (*ai.AnalyzeResult, error) {
	answer, err := s.chatFn(message)
	if err != nil {
		return nil, err
	}
	return &ai.AnalyzeResult{Answer: answer}, nil
}
func (s chatStubAdvisor) BackgroundState() ai.BackgroundState { return ai.BackgroundState{} }
