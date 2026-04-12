package server

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/storage"
)

func TestHandleInfoIncludesSelfPID(t *testing.T) {
	s, _ := newTestServer(t, "", config.DefaultConfig())
	s.store = storage.NewStore(60, 10)

	req := httptest.NewRequest("GET", "/api/v1/info", nil)
	rr := httptest.NewRecorder()
	s.handleInfo(rr, req)

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if got := int(body["self_pid"].(float64)); got != os.Getpid() {
		t.Fatalf("self_pid=%d want %d", got, os.Getpid())
	}
	if got := body["version"].(string); got != "test-version" {
		t.Fatalf("version=%q want %q", got, "test-version")
	}
}
