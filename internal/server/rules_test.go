package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
)

func TestRulesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "wtm.yaml")
	cfg := config.DefaultConfig()
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	s, _ := newTestServer(t, cfgPath, cfg)

	body := map[string]any{
		"rules": []map[string]any{
			{
				"name":             "chrome mem cap",
				"enabled":          true,
				"match":            "chrome.exe",
				"metric":           "memory_bytes",
				"op":               ">=",
				"threshold":        4000000000,
				"for_seconds":      30,
				"action":           "kill",
				"cooldown_seconds": 120,
			},
		},
	}
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/rules", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.handleRulesUpdate(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	reloaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Rules) != 1 {
		t.Fatalf("rules len=%d", len(reloaded.Rules))
	}
	r := reloaded.Rules[0]
	if r.Name != "chrome mem cap" || r.Match != "chrome.exe" || r.Action != "kill" {
		t.Errorf("rule=%+v", r)
	}
	if r.Threshold != 4000000000 {
		t.Errorf("threshold=%v", r.Threshold)
	}
	if r.For.Seconds() != 30 {
		t.Errorf("for=%v", r.For)
	}
	if r.Cooldown.Seconds() != 120 {
		t.Errorf("cooldown=%v", r.Cooldown)
	}

	// GET returns the rule.
	getReq := httptest.NewRequest("GET", "/api/v1/rules", nil)
	getRR := httptest.NewRecorder()
	s.handleRulesGet(getRR, getReq)
	var resp struct {
		Rules []ruleDTO `json:"rules"`
	}
	if err := json.Unmarshal(getRR.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Rules) != 1 || resp.Rules[0].ForSeconds != 30 {
		t.Errorf("dto=%+v", resp.Rules)
	}
}

func TestRulesValidation(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "wtm.yaml")
	cfg := config.DefaultConfig()
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	s, _ := newTestServer(t, cfgPath, cfg)

	bad := []string{
		`{"rules":[{"name":"","match":"x","metric":"cpu_percent"}]}`,                       // blank name
		`{"rules":[{"name":"a","match":"","metric":"cpu_percent"}]}`,                       // blank match
		`{"rules":[{"name":"a","match":"x","metric":"bogus"}]}`,                            // bad metric
		`{"rules":[{"name":"a","match":"x","metric":"cpu_percent","op":"bogus"}]}`,         // bad op
		`{"rules":[{"name":"a","match":"x","metric":"cpu_percent","action":"reboot"}]}`,    // bad action
		`{"rules":[{"name":"a","match":"x","metric":"cpu_percent","for_seconds":999999}]}`, // for too large
		`{"rules":[{"name":"a","match":"x","metric":"cpu_percent"},{"name":"a","match":"y","metric":"cpu_percent"}]}`, // dup name
	}
	for _, body := range bad {
		req := httptest.NewRequest("POST", "/api/v1/rules", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		s.handleRulesUpdate(rr, req)
		if rr.Code != 400 {
			t.Errorf("body=%s status=%d want 400", body, rr.Code)
		}
	}
}
