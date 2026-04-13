package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/anomaly"
	"github.com/ersinkoc/WindowsTaskManager/internal/config"
)

// EventBackgroundAnalysis is emitted when a background AI run completes.
const EventBackgroundAnalysis = "ai.background"

// BackgroundRun records one completed background analysis cycle.
type BackgroundRun struct {
	Trigger        string       `json:"trigger"`
	AlertID        string       `json:"alert_id,omitempty"`
	AlertType      string       `json:"alert_type,omitempty"`
	AlertTitle     string       `json:"alert_title,omitempty"`
	AlertPID       uint32       `json:"alert_pid,omitempty"`
	AlertProcess   string       `json:"alert_process,omitempty"`
	StartedAt      time.Time    `json:"started_at"`
	FinishedAt     time.Time    `json:"finished_at"`
	ReservedTokens int          `json:"reserved_tokens"`
	Cached         bool         `json:"cached"`
	Error          string       `json:"error,omitempty"`
	Answer         string       `json:"answer,omitempty"`
	Actions        []Suggestion `json:"actions,omitempty"`
	AutoCandidates int          `json:"auto_candidates,omitempty"`
}

// BackgroundBudget reports the scheduler's active guardrails.
type BackgroundBudget struct {
	CyclesLastHour          int       `json:"cycles_last_hour"`
	MaxCyclesPerHour        int       `json:"max_cycles_per_hour"`
	ReservedTokensToday     int       `json:"reserved_tokens_today"`
	MaxReservedTokensPerDay int       `json:"max_reserved_tokens_per_day"`
	MinIntervalSeconds      int       `json:"min_interval_seconds"`
	CooldownUntil           time.Time `json:"cooldown_until,omitempty"`
}

// BackgroundState is the JSON shape returned by /api/v1/ai/watch.
type BackgroundState struct {
	Enabled               bool             `json:"enabled"`
	Configured            bool             `json:"configured"`
	AutoAnalyzeOnCritical bool             `json:"auto_analyze_on_critical"`
	AutoActionEnabled     bool             `json:"auto_action_enabled"`
	AutoActionDryRun      bool             `json:"auto_action_dry_run"`
	InFlight              bool             `json:"in_flight"`
	LastEventAt           time.Time        `json:"last_event_at,omitempty"`
	LastSkipReason        string           `json:"last_skip_reason,omitempty"`
	LastError             string           `json:"last_error,omitempty"`
	LastRun               *BackgroundRun   `json:"last_run,omitempty"`
	RecentRuns            []BackgroundRun  `json:"recent_runs,omitempty"`
	Budget                BackgroundBudget `json:"budget"`
}

type backgroundTracker struct {
	inFlight            bool
	lastEventAt         time.Time
	lastStartedAt       time.Time
	lastSkipReason      string
	lastError           string
	cooldownUntil       time.Time
	cycleStarts         []time.Time
	reservedDay         string
	reservedTokensToday int
	historyLimit        int
	lastRun             *BackgroundRun
	recentRuns          []BackgroundRun
}

func (b *backgroundTracker) applyConfig(cfg *config.Config) {
	limit := 1
	if cfg != nil {
		limit = cfg.AI.Scheduler.HistoryLimit
	}
	if limit < 1 {
		limit = 1
	}
	b.historyLimit = limit
	if len(b.recentRuns) > limit {
		b.recentRuns = append([]BackgroundRun(nil), b.recentRuns[len(b.recentRuns)-limit:]...)
	}
}

func (b *backgroundTracker) snapshot(cfg *config.Config, configured bool) BackgroundState {
	out := BackgroundState{
		Configured:     configured,
		InFlight:       b.inFlight,
		LastEventAt:    b.lastEventAt,
		LastSkipReason: b.lastSkipReason,
		LastError:      b.lastError,
		LastRun:        cloneRun(b.lastRun),
		RecentRuns:     append([]BackgroundRun(nil), b.recentRuns...),
	}
	if cfg != nil {
		out.Enabled = cfg.AI.Scheduler.Enabled
		out.AutoAnalyzeOnCritical = cfg.AI.AutoAnalyzeOnCritical
		out.AutoActionEnabled = cfg.AI.AutoAction.Enabled
		out.AutoActionDryRun = cfg.AI.AutoAction.DryRun
		out.Budget = BackgroundBudget{
			CyclesLastHour:          countRecentCycles(b.cycleStarts, time.Now()),
			MaxCyclesPerHour:        cfg.AI.Scheduler.MaxCyclesPerHour,
			ReservedTokensToday:     b.reservedTokensToday,
			MaxReservedTokensPerDay: cfg.AI.Scheduler.MaxReservedTokensPerDay,
			MinIntervalSeconds:      int(cfg.AI.Scheduler.MinInterval / time.Second),
			CooldownUntil:           b.cooldownUntil,
		}
	}
	return out
}

func cloneRun(in *BackgroundRun) *BackgroundRun {
	if in == nil {
		return nil
	}
	cp := *in
	cp.Actions = append([]Suggestion(nil), in.Actions...)
	return &cp
}

func countRecentCycles(starts []time.Time, now time.Time) int {
	n := 0
	for _, ts := range starts {
		if now.Sub(ts) <= time.Hour {
			n++
		}
	}
	return n
}

func (a *Advisor) handleRaisedAlert(data any) {
	alert, ok := data.(anomaly.Alert)
	if !ok {
		return
	}
	a.maybeScheduleBackground(alert)
}

func (a *Advisor) maybeScheduleBackground(alert anomaly.Alert) {
	a.mu.RLock()
	cfg := a.cfg
	configured := a.cfg != nil && a.cfg.AI.Enabled && a.cfg.AI.APIKey != ""
	a.mu.RUnlock()

	now := time.Now()
	if cfg == nil {
		a.recordBackgroundSkip(now, "no_config")
		return
	}
	if !cfg.AI.Scheduler.Enabled {
		a.recordBackgroundSkip(now, "scheduler_disabled")
		return
	}
	if !cfg.AI.AutoAnalyzeOnCritical {
		a.recordBackgroundSkip(now, "auto_analyze_disabled")
		return
	}
	if !configured {
		a.recordBackgroundSkip(now, "ai_not_configured")
		return
	}
	if alert.Severity != anomaly.SeverityCritical {
		a.recordBackgroundSkip(now, "non_critical_alert")
		return
	}

	reserved := cfg.AI.MaxTokens
	if reserved < 1 {
		reserved = 1
	}

	a.bgMu.Lock()
	defer a.bgMu.Unlock()

	a.bg.applyConfig(cfg)
	a.bg.lastEventAt = now
	a.bg.cycleStarts = trimCycleStarts(a.bg.cycleStarts, now)
	resetReservedDay(&a.bg, now)

	switch {
	case a.bg.inFlight:
		a.bg.lastSkipReason = "background_inflight"
		return
	case !a.bg.cooldownUntil.IsZero() && now.Before(a.bg.cooldownUntil):
		a.bg.lastSkipReason = "error_cooldown"
		return
	case cfg.AI.Scheduler.MinInterval > 0 && !a.bg.lastStartedAt.IsZero() && now.Sub(a.bg.lastStartedAt) < cfg.AI.Scheduler.MinInterval:
		a.bg.lastSkipReason = "min_interval"
		return
	case cfg.AI.Scheduler.MaxCyclesPerHour > 0 && len(a.bg.cycleStarts) >= cfg.AI.Scheduler.MaxCyclesPerHour:
		a.bg.lastSkipReason = "cycle_limit"
		return
	case cfg.AI.Scheduler.MaxReservedTokensPerDay > 0 && a.bg.reservedTokensToday+reserved > cfg.AI.Scheduler.MaxReservedTokensPerDay:
		a.bg.lastSkipReason = "token_budget"
		return
	}

	a.bg.inFlight = true
	a.bg.lastSkipReason = ""
	a.bg.lastStartedAt = now
	a.bg.cycleStarts = append(a.bg.cycleStarts, now)
	a.bg.reservedTokensToday += reserved

	go a.runBackgroundAnalysis(cloneConfig(cfg), alert, reserved, now)
}

func cloneConfig(cfg *config.Config) *config.Config {
	if cfg == nil {
		return nil
	}
	cp := *cfg
	cp.Controller.ProtectedProcesses = append([]string(nil), cfg.Controller.ProtectedProcesses...)
	cp.Anomaly.IgnoreProcesses = append([]string(nil), cfg.Anomaly.IgnoreProcesses...)
	cp.Rules = append([]config.Rule(nil), cfg.Rules...)
	if cfg.AI.ExtraHeaders != nil {
		cp.AI.ExtraHeaders = make(map[string]string, len(cfg.AI.ExtraHeaders))
		for k, v := range cfg.AI.ExtraHeaders {
			cp.AI.ExtraHeaders[k] = v
		}
	}
	if cfg.WellKnownPorts != nil {
		cp.WellKnownPorts = make(map[uint16]string, len(cfg.WellKnownPorts))
		for k, v := range cfg.WellKnownPorts {
			cp.WellKnownPorts[k] = v
		}
	}
	return &cp
}

func trimCycleStarts(starts []time.Time, now time.Time) []time.Time {
	out := starts[:0]
	for _, ts := range starts {
		if now.Sub(ts) <= time.Hour {
			out = append(out, ts)
		}
	}
	return out
}

func resetReservedDay(bg *backgroundTracker, now time.Time) {
	day := now.Format("2006-01-02")
	if bg.reservedDay == day {
		return
	}
	bg.reservedDay = day
	bg.reservedTokensToday = 0
}

func (a *Advisor) recordBackgroundSkip(now time.Time, reason string) {
	a.bgMu.Lock()
	a.bg.lastEventAt = now
	a.bg.lastSkipReason = reason
	a.bgMu.Unlock()
}

func (a *Advisor) runBackgroundAnalysis(cfg *config.Config, alert anomaly.Alert, reserved int, startedAt time.Time) {
	run := BackgroundRun{
		Trigger:        "critical_alert",
		AlertID:        alert.ID,
		AlertType:      alert.Type,
		AlertTitle:     alert.Title,
		AlertPID:       alert.PID,
		AlertProcess:   alert.ProcessName,
		StartedAt:      startedAt,
		ReservedTokens: reserved,
	}

	ctx, cancel := context.WithTimeout(a.backgroundContext(), 45*time.Second)
	defer cancel()

	result, err := a.analyzeWithConfig(ctx, cfg, buildBackgroundPrompt(alert))
	run.FinishedAt = time.Now()
	if err != nil {
		run.Error = err.Error()
	} else {
		run.Answer = result.Answer
		run.Actions = append([]Suggestion(nil), result.Actions...)
		run.Cached = result.Cached
	}

	a.bgMu.Lock()

	if run.Error == "" {
		run.Actions = a.applyAutoPolicyLocked(cfg, run.Actions)
		run.AutoCandidates = countAutoCandidates(run.Actions)
	}

	a.bg.inFlight = false
	if run.Cached && a.bg.reservedTokensToday >= reserved {
		a.bg.reservedTokensToday -= reserved
	}
	if run.Error != "" {
		a.bg.lastError = run.Error
		if cfg != nil && cfg.AI.Scheduler.CooldownAfterError > 0 {
			a.bg.cooldownUntil = time.Now().Add(cfg.AI.Scheduler.CooldownAfterError)
		}
	} else {
		a.bg.lastError = ""
		a.bg.cooldownUntil = time.Time{}
	}
	a.bg.lastRun = cloneRun(&run)
	a.bg.recentRuns = append(a.bg.recentRuns, run)
	if limit := a.bg.historyLimit; limit > 0 && len(a.bg.recentRuns) > limit {
		a.bg.recentRuns = append([]BackgroundRun(nil), a.bg.recentRuns[len(a.bg.recentRuns)-limit:]...)
	}
	a.bgMu.Unlock()

	if a.emitter != nil {
		a.emitter.Emit(EventBackgroundAnalysis, cloneRun(&run))
	}
}

func (a *Advisor) applyAutoPolicyLocked(cfg *config.Config, actions []Suggestion) []Suggestion {
	if len(actions) == 0 {
		return actions
	}
	out := make([]Suggestion, 0, len(actions))
	for _, sug := range actions {
		sug.Policy = evaluateAutoPolicy(cfg, a.bg.recentRuns, sug)
		out = append(out, sug)
	}
	return out
}

func evaluateAutoPolicy(cfg *config.Config, priorRuns []BackgroundRun, sug Suggestion) *AutoPolicy {
	if cfg == nil || !cfg.AI.AutoAction.Enabled {
		return &AutoPolicy{Status: "disabled", Reason: "auto_action_disabled"}
	}
	if !cfg.AI.AutoAction.DryRun {
		return &AutoPolicy{Status: "blocked", Reason: "live_auto_execute_not_implemented"}
	}
	switch sug.Type {
	case "kill", "suspend":
		return &AutoPolicy{Status: "blocked", Reason: "destructive_actions_not_supported"}
	}
	if !containsFold(cfg.AI.AutoAction.AllowedActions, sug.Type) {
		return &AutoPolicy{Status: "blocked", Reason: "action_not_allowlisted"}
	}

	repeats := countSuggestionRepeats(priorRuns, sug.ID) + 1
	required := cfg.AI.AutoAction.RequireRepeatCycles
	if required < 1 {
		required = 1
	}
	if repeats < required {
		return &AutoPolicy{
			Status:              "needs_repeat",
			Reason:              "waiting_for_repeat_cycles",
			RepeatCount:         repeats,
			RequiredRepeatCount: required,
		}
	}
	return &AutoPolicy{
		Status:              "dry_run_eligible",
		Reason:              "policy_passed_dry_run_only",
		RepeatCount:         repeats,
		RequiredRepeatCount: required,
	}
}

func countSuggestionRepeats(runs []BackgroundRun, id string) int {
	if id == "" {
		return 0
	}
	total := 0
	for _, run := range runs {
		for _, sug := range run.Actions {
			if sug.ID == id {
				total++
			}
		}
	}
	return total
}

func countAutoCandidates(actions []Suggestion) int {
	n := 0
	for _, sug := range actions {
		if sug.Policy != nil && sug.Policy.Status == "dry_run_eligible" {
			n++
		}
	}
	return n
}

func containsFold(items []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), target) {
			return true
		}
	}
	return false
}

func buildBackgroundPrompt(alert anomaly.Alert) string {
	var b strings.Builder
	b.WriteString("A new CRITICAL anomaly alert was raised by the background watcher.\n")
	b.WriteString("Focus on this alert first, but use the full snapshot for context.\n")
	b.WriteString("Give a concise health assessment, explain why the alert matters, and suggest the safest next step.\n")
	b.WriteString("Prefer protect/ignore/add_rule over kill/suspend unless the target is clearly non-system and directly implicated.\n\n")
	fmt.Fprintf(&b, "Alert type: %s\n", alert.Type)
	fmt.Fprintf(&b, "Title: %s\n", alert.Title)
	if alert.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", alert.Description)
	}
	if alert.PID != 0 {
		fmt.Fprintf(&b, "PID: %d\n", alert.PID)
	}
	if alert.ProcessName != "" {
		fmt.Fprintf(&b, "Process: %s\n", alert.ProcessName)
	}
	return b.String()
}
