package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const maxJSONBodyBytes int64 = 1 << 20 // 1 MiB

// writeJSON sends a JSON-encoded payload.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// Connection likely already closed; nothing useful to do here.
		_ = err
	}
}

// writeError emits a structured error envelope.
func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": msg,
		},
	})
}

// readJSON decodes a request body or returns false after writing an error.
func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "missing request body")
		return false
	}
	body := http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	defer body.Close()
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return false
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain a single JSON value")
		return false
	}
	return true
}

// parseUint32Param extracts a uint32 path parameter or writes a 400.
func parseUint32Param(w http.ResponseWriter, r *http.Request, name string) (uint32, bool) {
	raw := Param(r, name)
	if raw == "" {
		writeError(w, http.StatusBadRequest, "missing_param", name+" required")
		return 0, false
	}
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_param", name+" must be uint32")
		return 0, false
	}
	return uint32(n), true
}

// formatBytes converts a byte count into a short human-readable string.
func formatBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for n/div >= unit && exp < 4 {
		div *= unit
		exp++
	}
	suffixes := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(n)/float64(div), suffixes[exp])
}
