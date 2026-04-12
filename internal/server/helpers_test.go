package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestReadJSONRejectsTrailingData(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"name":"ok"}{"extra":true}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	var body struct {
		Name string `json:"name"`
	}
	if readJSON(rr, req, &body) {
		t.Fatal("readJSON accepted multiple JSON values")
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestLocalOnlyMiddlewareAllowsIPv6Loopback(t *testing.T) {
	called := false
	handler := localOnlyMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.RemoteAddr = "[::1]:19876"
	rr := httptest.NewRecorder()
	handler(rr, req)

	if !called {
		t.Fatal("loopback IPv6 request was rejected")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d want %d", rr.Code, http.StatusNoContent)
	}
}

func TestMutationGuardRejectsMissingOriginAndToken(t *testing.T) {
	handler := mutationGuardMiddleware("secret")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules", strings.NewReader(`{}`))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Host = "127.0.0.1:8080"
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want %d", rr.Code, http.StatusForbidden)
	}
}

func TestMutationGuardAllowsLoopbackOriginWithToken(t *testing.T) {
	handler := mutationGuardMiddleware("secret")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules", strings.NewReader(`{}`))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Host = "127.0.0.1:8080"
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	req.Header.Set("X-WTM-CSRF", "secret")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d want %d", rr.Code, http.StatusNoContent)
	}
}

func TestServeStaticInjectsCSRFMToken(t *testing.T) {
	s := &Server{
		staticFS: fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte(`<meta name="wtm-csrf-token" content="__WTM_CSRF_TOKEN__" />`)},
		},
		csrfToken: "abc123",
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	s.serveStatic(rr, req)

	if !strings.Contains(rr.Body.String(), `content="abc123"`) {
		t.Fatalf("csrf token was not injected: %q", rr.Body.String())
	}
}
