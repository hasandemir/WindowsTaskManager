package server

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ersinkoc/WindowsTaskManager/internal/ai"
	"github.com/ersinkoc/WindowsTaskManager/internal/config"
)

func TestAIExecuteRejectsUnissuedSuggestion(t *testing.T) {
	s, _ := newTestServer(t, "", config.DefaultConfig())

	body, _ := json.Marshal(aiExecuteRequest{
		ID:   "missing-1",
		Type: "protect",
		Name: "node.exe",
	})
	req := authedJSONRequest("POST", "/api/v1/ai/execute", body, s.csrfToken)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != 400 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "suggestion not found or expired") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestAIExecuteConsumesIssuedSuggestion(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "wtm.yaml")
	cfg := config.DefaultConfig()
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("seed save: %v", err)
	}

	s, _ := newTestServer(t, cfgPath, cfg)
	suggestion := ai.Suggestion{
		ID:   "issued-1",
		Type: "protect",
		Name: "node.exe",
	}
	s.rememberAISuggestions([]ai.Suggestion{suggestion})

	body, _ := json.Marshal(aiExecuteRequest{
		ID:   suggestion.ID,
		Type: suggestion.Type,
		Name: suggestion.Name,
	})
	req := authedJSONRequest("POST", "/api/v1/ai/execute", body, s.csrfToken)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	reloaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	found := false
	for _, name := range reloaded.Controller.ProtectedProcesses {
		if strings.EqualFold(name, "node.exe") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("protected_processes=%v want node.exe", reloaded.Controller.ProtectedProcesses)
	}

	replayReq := authedJSONRequest("POST", "/api/v1/ai/execute", body, s.csrfToken)
	replayRR := httptest.NewRecorder()
	s.router.ServeHTTP(replayRR, replayReq)
	if replayRR.Code != 400 {
		t.Fatalf("replay status=%d body=%s", replayRR.Code, replayRR.Body.String())
	}
}
