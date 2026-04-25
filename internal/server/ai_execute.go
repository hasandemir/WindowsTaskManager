package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/ai"
	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/controller"
)

type aiExecuteInputError struct {
	message string
}

func (e *aiExecuteInputError) Error() string { return e.message }

type issuedAISuggestion struct {
	suggestion ai.Suggestion
	expiresAt  time.Time
}

type aiSuggestionStore struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[string]issuedAISuggestion
}

func newAISuggestionStore(ttl time.Duration) *aiSuggestionStore {
	return &aiSuggestionStore{
		ttl:   ttl,
		items: make(map[string]issuedAISuggestion),
	}
}

func (s *aiSuggestionStore) remember(items []ai.Suggestion) {
	if s == nil || len(items) == 0 {
		return
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		s.items[id] = issuedAISuggestion{
			suggestion: cloneAISuggestion(item),
			expiresAt:  now.Add(s.ttl),
		}
	}
}

func (s *aiSuggestionStore) consume(expected ai.Suggestion) error {
	if s == nil {
		return nil
	}
	id := strings.TrimSpace(expected.ID)
	if id == "" {
		return &aiExecuteInputError{message: "suggestion id required"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.pruneLocked(now)
	issued, ok := s.items[id]
	if !ok {
		return &aiExecuteInputError{message: "suggestion not found or expired"}
	}
	if !sameAISuggestion(issued.suggestion, expected) {
		return &aiExecuteInputError{message: "suggestion payload mismatch"}
	}
	delete(s.items, id)
	return nil
}

func (s *aiSuggestionStore) pruneLocked(now time.Time) {
	for id, item := range s.items {
		if !item.expiresAt.After(now) {
			delete(s.items, id)
		}
	}
}

func cloneAISuggestion(in ai.Suggestion) ai.Suggestion {
	out := in
	if in.Rule != nil {
		rule := *in.Rule
		out.Rule = &rule
	}
	if in.Policy != nil {
		policy := *in.Policy
		out.Policy = &policy
	}
	return out
}

func sameAISuggestion(left, right ai.Suggestion) bool {
	if strings.TrimSpace(left.ID) != strings.TrimSpace(right.ID) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(left.Type), strings.TrimSpace(right.Type)) {
		return false
	}
	if left.PID != right.PID {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(left.Name), strings.TrimSpace(right.Name)) {
		return false
	}
	return sameRuleSuggestion(left.Rule, right.Rule)
}

func sameRuleSuggestion(left, right *ai.RuleSuggestion) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Name == right.Name &&
		left.Enabled == right.Enabled &&
		left.Match == right.Match &&
		left.Metric == right.Metric &&
		left.Op == right.Op &&
		left.Threshold == right.Threshold &&
		left.For == right.For &&
		left.ForSeconds == right.ForSeconds &&
		left.Action == right.Action &&
		left.Cooldown == right.Cooldown &&
		left.CooldownSeconds == right.CooldownSeconds
}

// aiExecuteRequest is the body of POST /api/v1/ai/execute. The dashboard
// posts the full Suggestion object from the AI advisor verbatim, so this
// struct mirrors Suggestion (including the opaque ID) rather than pruning
// fields — readJSON uses DisallowUnknownFields and would 400 otherwise.
type aiExecuteRequest struct {
	ID      string             `json:"id,omitempty"` // opaque; used for UI dedup
	Type    string             `json:"type"`         // kill | suspend | protect | ignore | add_rule
	PID     uint32             `json:"pid,omitempty"`
	Confirm bool               `json:"confirm,omitempty"`
	Name    string             `json:"name,omitempty"`
	Rule    *ai.RuleSuggestion `json:"rule,omitempty"`
	Reason  string             `json:"reason,omitempty"`
	Policy  *ai.AutoPolicy     `json:"policy,omitempty"` // ignored server-side; used by UI for dry-run labels
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
	suggestion := ai.Suggestion{
		ID:     body.ID,
		Type:   body.Type,
		PID:    body.PID,
		Name:   body.Name,
		Reason: body.Reason,
		Rule:   body.Rule,
		Policy: body.Policy,
	}
	if err := s.consumeAISuggestion(suggestion); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := s.ExecuteAISuggestion(suggestion, body.Confirm); err != nil {
		var inputErr *aiExecuteInputError
		switch {
		case errors.As(err, &inputErr):
			writeError(w, http.StatusBadRequest, "bad_request", inputErr.Error())
		case errors.Is(err, controller.ErrProtected),
			errors.Is(err, controller.ErrCritical),
			errors.Is(err, controller.ErrSelf),
			errors.Is(err, controller.ErrConfirmNeeded),
			errors.Is(err, controller.ErrNotFound):
			s.controllerError(w, err)
		default:
			writeError(w, http.StatusBadRequest, "save_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "type": suggestion.Type, "pid": suggestion.PID, "name": suggestion.Name})
}

func (s *Server) rememberAISuggestions(items []ai.Suggestion) {
	if s == nil || s.aiExec == nil {
		return
	}
	s.aiExec.remember(items)
}

func (s *Server) consumeAISuggestion(item ai.Suggestion) error {
	if s == nil || s.aiExec == nil {
		return nil
	}
	return s.aiExec.consume(item)
}

func (s *Server) ExecuteAISuggestion(suggestion ai.Suggestion, confirm bool) error {
	switch strings.ToLower(strings.TrimSpace(suggestion.Type)) {
	case "kill":
		if suggestion.PID == 0 {
			return &aiExecuteInputError{message: "pid required"}
		}
		return s.controller.Kill(suggestion.PID, confirm)
	case "suspend":
		if suggestion.PID == 0 {
			return &aiExecuteInputError{message: "pid required"}
		}
		return s.controller.Suspend(suggestion.PID, confirm)
	case "protect":
		if strings.TrimSpace(suggestion.Name) == "" {
			return &aiExecuteInputError{message: "name required"}
		}
		return s.mutateConfig(func(c *config.Config) error {
			c.Controller.ProtectedProcesses = appendUniqueFold(c.Controller.ProtectedProcesses, suggestion.Name)
			return nil
		})
	case "ignore":
		if strings.TrimSpace(suggestion.Name) == "" {
			return &aiExecuteInputError{message: "name required"}
		}
		return s.mutateConfig(func(c *config.Config) error {
			c.Anomaly.IgnoreProcesses = appendUniqueFold(c.Anomaly.IgnoreProcesses, suggestion.Name)
			return nil
		})
	case "add_rule":
		rule, err := aiRuleToConfig(suggestion.Rule)
		if err != nil {
			return &aiExecuteInputError{message: err.Error()}
		}
		return s.mutateConfig(func(c *config.Config) error {
			for _, existing := range c.Rules {
				if strings.EqualFold(existing.Name, rule.Name) {
					return fmt.Errorf("rule %q already exists", rule.Name)
				}
			}
			c.Rules = append(c.Rules, rule)
			return nil
		})
	default:
		return &aiExecuteInputError{message: "type must be kill|suspend|protect|ignore|add_rule"}
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
	current := cloneConfig(s.cfg)
	s.mu.RUnlock()

	next := current

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
