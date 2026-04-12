package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterPrefersLiteralRoutesOverParams(t *testing.T) {
	router := NewRouter()
	router.GET("/api/v1/processes/:pid", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "matched param route", http.StatusBadRequest)
	})
	router.GET("/api/v1/processes/limits", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/processes/limits", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
