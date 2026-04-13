package anomaly

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
)

// ruleState tracks how long a (rule, pid) pair has been matching, and the
// last time we executed the action so we can respect cooldown.
type ruleState struct {
	matchingSince time.Time
	lastActionAt  time.Time
}

// RulesDetector evaluates user-defined rules from config. On every tick it
// walks the process list, checks each rule's match+when, and when a rule has
// been satisfied for at least `for` seconds, fires the configured action via
// ctx.Actuator. Each (rule,pid) respects a cooldown so a flapping process
// doesn't get killed in a loop.
type RulesDetector struct {
	mu     sync.Mutex
	states map[string]*ruleState // key = rule.Name + "/" + pid
}

func NewRulesDetector() *RulesDetector {
	return &RulesDetector{states: make(map[string]*ruleState)}
}

func (d *RulesDetector) Name() string { return "rules" }

func (d *RulesDetector) Analyze(ctx *AnalysisContext) {
	rules := ctx.Cfg.Rules
	if len(rules) == 0 {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	live := make(map[string]struct{}, len(ctx.Snapshot.Processes)*2)

	for ri := range rules {
		r := &rules[ri]
		if !r.Enabled || strings.TrimSpace(r.Match) == "" {
			continue
		}
		match := strings.ToLower(strings.TrimSpace(r.Match))
		for pi := range ctx.Snapshot.Processes {
			p := &ctx.Snapshot.Processes[pi]
			if !strings.Contains(strings.ToLower(p.Name), match) {
				continue
			}
			if isIgnoredProcess(ctx.Cfg, p.Name) {
				continue
			}
			value, ok := ruleMetricValue(r.Metric, p)
			if !ok {
				continue
			}
			hit := compareRule(value, r.Op, r.Threshold)
			key := r.Name + "/" + uint32ToA(p.PID)
			live[key] = struct{}{}

			if !hit {
				delete(d.states, key)
				ctx.ClearAlert("rule:"+r.Name, p.PID)
				continue
			}

			st, existed := d.states[key]
			if !existed {
				st = &ruleState{matchingSince: ctx.Now}
				d.states[key] = st
			}
			if ctx.Now.Sub(st.matchingSince) < r.For {
				continue // still waiting out the duration window
			}

			cooldown := r.Cooldown
			if cooldown <= 0 {
				cooldown = time.Minute
			}
			action := strings.ToLower(strings.TrimSpace(r.Action))
			if action == "" {
				action = "alert"
			}

			d.raiseRuleAlert(ctx, r, p, value)

			// For kill/suspend, respect cooldown.
			if action == "alert" {
				continue
			}
			if !st.lastActionAt.IsZero() && ctx.Now.Sub(st.lastActionAt) < cooldown {
				continue
			}
			if ctx.Actuator == nil {
				continue
			}
			switch action {
			case "kill":
				if err := ctx.Actuator.Kill(p.PID, false); err != nil {
					log.Printf("rule %q kill PID %d failed: %v", r.Name, p.PID, err)
				} else {
					st.lastActionAt = ctx.Now
				}
			case "suspend":
				if err := ctx.Actuator.Suspend(p.PID, false); err != nil {
					log.Printf("rule %q suspend PID %d failed: %v", r.Name, p.PID, err)
				} else {
					st.lastActionAt = ctx.Now
				}
			}
		}
	}

	// GC states for (rule,pid) that no longer exist on this tick.
	for key := range d.states {
		if _, ok := live[key]; !ok {
			delete(d.states, key)
		}
	}
}

func (d *RulesDetector) raiseRuleAlert(ctx *AnalysisContext, r *config.Rule, p *metrics.ProcessInfo, value float64) {
	sev := SeverityWarning
	action := strings.ToLower(r.Action)
	if action == "kill" || action == "suspend" {
		sev = SeverityCritical
	}
	ctx.RaiseAlert(Alert{
		Type:        "rule:" + r.Name,
		Severity:    sev,
		Title:       "Rule triggered: " + r.Name,
		Description: fmt.Sprintf("%s (PID %d) %s=%.0f %s %.0f → %s", p.Name, p.PID, r.Metric, value, ruleOpOrDefault(r.Op), r.Threshold, action),
		PID:         p.PID,
		ProcessName: p.Name,
		Action:      action,
		Details: map[string]any{
			"metric":    r.Metric,
			"value":     value,
			"threshold": r.Threshold,
			"for":       r.For.String(),
			"rule":      r.Name,
		},
	})
}

func ruleMetricValue(metric string, p *metrics.ProcessInfo) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(metric)) {
	case "cpu_percent", "cpu":
		return p.CPUPercent, true
	case "memory_bytes", "memory", "working_set":
		return float64(p.WorkingSet), true
	case "private_bytes":
		return float64(p.PrivateBytes), true
	case "thread_count", "threads":
		return float64(p.ThreadCount), true
	}
	return 0, false
}

func compareRule(value float64, op string, threshold float64) bool {
	switch ruleOpOrDefault(op) {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	}
	return false
}

func ruleOpOrDefault(op string) string {
	op = strings.TrimSpace(op)
	if op == "" {
		return ">="
	}
	return op
}
