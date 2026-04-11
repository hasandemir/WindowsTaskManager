package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
