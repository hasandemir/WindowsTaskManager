package anomaly

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type runawayState struct {
	since time.Time
}

// RunawayCPUDetector flags processes that sustain >= cpuThreshold for
// duration >= durationThreshold (warning) or criticalDuration (critical).
type RunawayCPUDetector struct {
	mu     sync.Mutex
	states map[uint32]*runawayState
}

func NewRunawayCPUDetector() *RunawayCPUDetector {
	return &RunawayCPUDetector{states: make(map[uint32]*runawayState)}
}

func (d *RunawayCPUDetector) Name() string { return "runaway_cpu" }

func (d *RunawayCPUDetector) Analyze(ctx *AnalysisContext) {
	cfg := ctx.Cfg.Anomaly.RunawayCPU
	if !cfg.Enabled {
		return
	}
	whitelist := make(map[string]struct{}, len(cfg.HighCPUWhitelist))
	for _, w := range cfg.HighCPUWhitelist {
		whitelist[strings.ToLower(w)] = struct{}{}
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	live := make(map[uint32]struct{}, len(ctx.Snapshot.Processes))
	for i := range ctx.Snapshot.Processes {
		p := &ctx.Snapshot.Processes[i]
		live[p.PID] = struct{}{}
		if _, skip := whitelist[strings.ToLower(p.Name)]; skip {
			continue
		}
		if p.CPUPercent < float64(cfg.CPUThreshold) {
			delete(d.states, p.PID)
			ctx.ClearAlert(d.Name(), p.PID)
			continue
		}
		st, ok := d.states[p.PID]
		if !ok {
			d.states[p.PID] = &runawayState{since: ctx.Now}
			continue
		}
		dur := ctx.Now.Sub(st.since)
		if dur < cfg.DurationThreshold {
			continue
		}
		sev := SeverityWarning
		if dur >= cfg.CriticalDuration {
			sev = SeverityCritical
		}
		ctx.RaiseAlert(Alert{
			Type:        d.Name(),
			Severity:    sev,
			Title:       "Runaway CPU usage",
			Description: fmt.Sprintf("%s (PID %d) at %.0f%% CPU for %s", p.Name, p.PID, p.CPUPercent, dur.Truncate(time.Second)),
			PID:         p.PID,
			ProcessName: p.Name,
			Action:      cfg.Action,
			Details: map[string]any{
				"cpu_percent":      p.CPUPercent,
				"duration_seconds": dur.Seconds(),
			},
		})
	}

	for pid := range d.states {
		if _, ok := live[pid]; !ok {
			delete(d.states, pid)
		}
	}
}
