package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
)

// ruleDTO is the wire shape. We use `for_seconds` and `cooldown_seconds` so
// clients don't have to parse Go duration strings.
type ruleDTO struct {
	Name            string  `json:"name"`
	Enabled         bool    `json:"enabled"`
	Match           string  `json:"match"`
	Metric          string  `json:"metric"`
	Op              string  `json:"op"`
	Threshold       float64 `json:"threshold"`
	ForSeconds      int     `json:"for_seconds"`
	Action          string  `json:"action"`
	CooldownSeconds int     `json:"cooldown_seconds"`
}

func rulesToDTO(rs []config.Rule) []ruleDTO {
	out := make([]ruleDTO, 0, len(rs))
	for _, r := range rs {
		out = append(out, ruleDTO{
			Name:            r.Name,
			Enabled:         r.Enabled,
			Match:           r.Match,
			Metric:          r.Metric,
			Op:              r.Op,
			Threshold:       r.Threshold,
			ForSeconds:      int(r.For / time.Second),
			Action:          r.Action,
			CooldownSeconds: int(r.Cooldown / time.Second),
		})
	}
	return out
}

func rulesFromDTO(in []ruleDTO) ([]config.Rule, error) {
	out := make([]config.Rule, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for i, d := range in {
		name := strings.TrimSpace(d.Name)
		if name == "" {
			return nil, ruleErr(i, "name required")
		}
		if _, dup := seen[name]; dup {
			return nil, ruleErr(i, "duplicate rule name: "+name)
		}
		seen[name] = struct{}{}
		if strings.TrimSpace(d.Match) == "" {
			return nil, ruleErr(i, "match pattern required")
		}
		metric := strings.ToLower(strings.TrimSpace(d.Metric))
		switch metric {
		case "cpu_percent", "memory_bytes", "private_bytes", "thread_count":
		default:
			return nil, ruleErr(i, "metric must be cpu_percent, memory_bytes, private_bytes, or thread_count")
		}
		op := strings.TrimSpace(d.Op)
		if op == "" {
			op = ">="
		}
		if op != ">" && op != ">=" && op != "<" && op != "<=" {
			return nil, ruleErr(i, "op must be one of > >= < <=")
		}
		action := strings.ToLower(strings.TrimSpace(d.Action))
		if action == "" {
			action = "alert"
		}
		if action != "alert" && action != "kill" && action != "suspend" {
			return nil, ruleErr(i, "action must be alert, kill, or suspend")
		}
		if d.ForSeconds < 0 || d.ForSeconds > 86400 {
			return nil, ruleErr(i, "for_seconds must be 0..86400")
		}
		cool := d.CooldownSeconds
		if cool < 0 {
			cool = 0
		}
		out = append(out, config.Rule{
			Name:      name,
			Enabled:   d.Enabled,
			Match:     d.Match,
			Metric:    metric,
			Op:        op,
			Threshold: d.Threshold,
			For:       time.Duration(d.ForSeconds) * time.Second,
			Action:    action,
			Cooldown:  time.Duration(cool) * time.Second,
		})
	}
	return out, nil
}

type ruleParseError struct {
	idx int
	msg string
}

func (e *ruleParseError) Error() string { return e.msg }

func ruleErr(i int, msg string) error { return &ruleParseError{idx: i, msg: msg} }

func (s *Server) handleRulesGet(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	rs := append([]config.Rule{}, s.cfg.Rules...)
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"rules": rulesToDTO(rs)})
}

func (s *Server) handleRulesUpdate(w http.ResponseWriter, r *http.Request) {
	if s.cfgPath == "" {
		writeError(w, http.StatusServiceUnavailable, "no_config", "config file path not set")
		return
	}
	var body struct {
		Rules []ruleDTO `json:"rules"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	rules, err := rulesFromDTO(body.Rules)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_rule", err.Error())
		return
	}

	s.mu.RLock()
	current := cloneConfig(s.cfg)
	s.mu.RUnlock()

	next := current
	next.Rules = rules
	if err := next.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_config", err.Error())
		return
	}
	if err := config.Save(s.cfgPath, &next); err != nil {
		writeError(w, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	s.SetConfig(&next)
	if s.onCfgApply != nil {
		s.onCfgApply(&next)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"rules": rulesToDTO(next.Rules),
	})
}
