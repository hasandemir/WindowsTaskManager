package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/ai"
	"github.com/ersinkoc/WindowsTaskManager/internal/config"
)

// aiExecuteRequest is the body of POST /api/v1/ai/execute. The dashboard
// posts the full Suggestion object from the AI advisor verbatim, so this
// struct mirrors Suggestion (including the opaque ID) rather than pruning
// fields — readJSON uses DisallowUnknownFields and would 400 otherwise.
type aiExecuteRequest struct {
	ID     string             `json:"id,omitempty"`   // opaque — ignored server-side, used for UI dedup
	Type   string             `json:"type"`           // kill | suspend | protect | ignore | add_rule
	PID    uint32             `json:"pid,omitempty"`  // required for kill/suspend
	Name   string             `json:"name,omitempty"` // required for protect/ignore
	Rule   *ai.RuleSuggestion `json:"rule,omitempty"` // required for add_rule
	Reason string             `json:"reason,omitempty"`
	Policy *ai.AutoPolicy     `json:"policy,omitempty"` // ignored server-side; used by UI for dry-run labels
}

// aiRuleToConfig validates and converts the LLM-emitted rule wire shape into
// a config.Rule. Same validation as rulesFromDTO so suggestions can't sneak
// past the checks the manual rules table enforces.
func aiRuleToConfig(d *ai.RuleSuggestion) (config.Rule, error) {
	if d == nil {
		return config.Rule{}, fmt.Errorf("rule required")
	}
	name := strings.TrimSpace(d.Name)
	if name == "" {
		return config.Rule{}, fmt.Errorf("rule name required")
	}
	match := strings.TrimSpace(d.Match)
	if match == "" {
		return config.Rule{}, fmt.Errorf("rule match pattern required")
	}
	metric := strings.ToLower(strings.TrimSpace(d.Metric))
	switch metric {
	case "cpu_percent", "memory_bytes", "private_bytes", "thread_count":
	default:
		return config.Rule{}, fmt.Errorf("metric must be cpu_percent, memory_bytes, private_bytes, or thread_count")
	}
	op := strings.TrimSpace(d.Op)
	if op == "" {
		op = ">="
	}
	if op != ">" && op != ">=" && op != "<" && op != "<=" {
		return config.Rule{}, fmt.Errorf("op must be one of > >= < <=")
	}
	action := strings.ToLower(strings.TrimSpace(d.Action))
	if action == "" {
		action = "alert"
	}
	if action != "alert" && action != "kill" && action != "suspend" {
		return config.Rule{}, fmt.Errorf("action must be alert, kill, or suspend")
	}
	forSec := d.ForSeconds
	if forSec == 0 {
		forSec = d.For
	}
	if forSec < 0 || forSec > 86400 {
		return config.Rule{}, fmt.Errorf("for_seconds must be 0..86400")
	}
	coolSec := d.CooldownSeconds
	if coolSec == 0 {
		coolSec = d.Cooldown
	}
	if coolSec < 0 {
		coolSec = 0
	}
	return config.Rule{
		Name:      name,
		Enabled:   d.Enabled,
		Match:     match,
		Metric:    metric,
		Op:        op,
		Threshold: d.Threshold,
		For:       time.Duration(forSec) * time.Second,
		Action:    action,
		Cooldown:  time.Duration(coolSec) * time.Second,
	}, nil
}

// handleAIExecute dispatches an approved AI suggestion. Every mutation goes
// through the same safety checks the regular endpoints use — nothing executes
// just because the AI asked for it. The user has already approved the action
// in the UI before this endpoint is called.
func (s *Server) handleAIExecute(w http.ResponseWriter, r *http.Request) {
	var body aiExecuteRequest
	if !readJSON(w, r, &body) {
		return
	}

	switch strings.ToLower(strings.TrimSpace(body.Type)) {
	case "kill":
		if body.PID == 0 {
			writeError(w, http.StatusBadRequest, "bad_request", "pid required")
			return
		}
		if err := s.controller.Kill(body.PID, true); err != nil {
			s.controllerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "type": "kill", "pid": body.PID})

	case "suspend":
		if body.PID == 0 {
			writeError(w, http.StatusBadRequest, "bad_request", "pid required")
			return
		}
		if err := s.controller.Suspend(body.PID); err != nil {
			s.controllerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "type": "suspend", "pid": body.PID})

	case "protect":
		if strings.TrimSpace(body.Name) == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "name required")
			return
		}
		if err := s.mutateConfig(func(c *config.Config) error {
			c.Controller.ProtectedProcesses = appendUniqueFold(c.Controller.ProtectedProcesses, body.Name)
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "save_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "type": "protect", "name": body.Name})

	case "ignore":
		if strings.TrimSpace(body.Name) == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "name required")
			return
		}
		if err := s.mutateConfig(func(c *config.Config) error {
			c.Anomaly.IgnoreProcesses = appendUniqueFold(c.Anomaly.IgnoreProcesses, body.Name)
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "save_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "type": "ignore", "name": body.Name})

	case "add_rule":
		rule, err := aiRuleToConfig(body.Rule)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if err := s.mutateConfig(func(c *config.Config) error {
			for _, existing := range c.Rules {
				if strings.EqualFold(existing.Name, rule.Name) {
					return fmt.Errorf("rule %q already exists", rule.Name)
				}
			}
			c.Rules = append(c.Rules, rule)
			return nil
		}); err != nil {
			writeError(w, http.StatusBadRequest, "save_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "type": "add_rule", "rule": rule.Name})

	default:
		writeError(w, http.StatusBadRequest, "bad_type", "type must be kill|suspend|protect|ignore|add_rule")
	}
}

// handleConfigProtectToggle adds or removes a process name from the protected
// list. Toggle via the `protect` boolean. Called from the process row action.
func (s *Server) handleConfigProtectToggle(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string `json:"name"`
		Protect bool   `json:"protect"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name required")
		return
	}
	if err := s.mutateConfig(func(c *config.Config) error {
		if body.Protect {
			c.Controller.ProtectedProcesses = appendUniqueFold(c.Controller.ProtectedProcesses, body.Name)
		} else {
			c.Controller.ProtectedProcesses = removeFold(c.Controller.ProtectedProcesses, body.Name)
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": body.Name, "protect": body.Protect})
}

// handleConfigIgnoreToggle adds or removes a process name from the anomaly
// ignore list. Toggle via the `ignore` boolean.
func (s *Server) handleConfigIgnoreToggle(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		Ignore bool   `json:"ignore"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name required")
		return
	}
	if err := s.mutateConfig(func(c *config.Config) error {
		if body.Ignore {
			c.Anomaly.IgnoreProcesses = appendUniqueFold(c.Anomaly.IgnoreProcesses, body.Name)
		} else {
			c.Anomaly.IgnoreProcesses = removeFold(c.Anomaly.IgnoreProcesses, body.Name)
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": body.Name, "ignore": body.Ignore})
}

// mutateConfig clones the current config, lets the caller mutate it, runs
// Validate, saves to disk and applies in-process. Centralises the "modify
// one field" pattern so protect/ignore/add_rule all go through the same hot
// reload path the AI config update uses.
func (s *Server) mutateConfig(mutate func(*config.Config) error) error {
	if s.cfgPath == "" {
		return fmt.Errorf("config file path not set")
	}

	s.mu.RLock()
	current := *s.cfg
	s.mu.RUnlock()

	next := current
	next.Controller.ProtectedProcesses = append([]string(nil), current.Controller.ProtectedProcesses...)
	next.Anomaly.IgnoreProcesses = append([]string(nil), current.Anomaly.IgnoreProcesses...)
	next.Rules = append([]config.Rule(nil), current.Rules...)

	if err := mutate(&next); err != nil {
		return err
	}
	if err := next.Validate(); err != nil {
		return err
	}
	if err := config.Save(s.cfgPath, &next); err != nil {
		return err
	}
	s.SetConfig(&next)
	if s.onCfgApply != nil {
		s.onCfgApply(&next)
	}
	return nil
}

func appendUniqueFold(list []string, item string) []string {
	item = strings.TrimSpace(item)
	if item == "" {
		return list
	}
	for _, x := range list {
		if strings.EqualFold(x, item) {
			return list
		}
	}
	return append(list, item)
}

func removeFold(list []string, item string) []string {
	out := list[:0]
	for _, x := range list {
		if !strings.EqualFold(x, item) {
			out = append(out, x)
		}
	}
	return out
}
