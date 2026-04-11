package anomaly

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// hungState tracks how long a process has shown zero activity.
type hungState struct {
	since        time.Time
	lastIO       uint64
	lastCPUSlice float64
}

// HungProcessDetector flags processes with no CPU and no IO progress for
// the configured window.
type HungProcessDetector struct {
	mu     sync.Mutex
	states map[uint32]*hungState
}

func NewHungProcessDetector() *HungProcessDetector {
	return &HungProcessDetector{states: make(map[uint32]*hungState)}
}

func (d *HungProcessDetector) Name() string { return "hung_process" }

func (d *HungProcessDetector) Analyze(ctx *AnalysisContext) {
	cfg := ctx.Cfg.Anomaly.HungProcess
	if !cfg.Enabled {
		return
	}

	whitelist := make(map[string]struct{}, len(cfg.IdleWhitelist))
	for _, w := range cfg.IdleWhitelist {
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
		if isIgnoredProcess(ctx.Cfg, p.Name) {
			continue
		}
		ioSum := p.IOReadBytes + p.IOWriteBytes
		st, ok := d.states[p.PID]
		if !ok {
			d.states[p.PID] = &hungState{since: ctx.Now, lastIO: ioSum, lastCPUSlice: p.CPUPercent}
			continue
		}
		busy := p.CPUPercent > 0.5 || ioSum != st.lastIO
		if busy {
			st.since = ctx.Now
			st.lastIO = ioSum
			st.lastCPUSlice = p.CPUPercent
			ctx.ClearAlert(d.Name(), p.PID)
			continue
		}
		idleFor := ctx.Now.Sub(st.since)
		if idleFor < cfg.ZeroActivityThreshold {
			continue
		}
		sev := SeverityWarning
		if idleFor >= cfg.CriticalHungThreshold {
			sev = SeverityCritical
		}
		ctx.RaiseAlert(Alert{
			Type:        d.Name(),
			Severity:    sev,
			Title:       "Hung process detected",
			Description: fmt.Sprintf("%s (PID %d) has been idle for %s", p.Name, p.PID, idleFor.Truncate(time.Second)),
			PID:         p.PID,
			ProcessName: p.Name,
			Action:      cfg.Action,
			Details: map[string]any{
				"idle_for_seconds": idleFor.Seconds(),
			},
		})
	}

	for pid := range d.states {
		if _, ok := live[pid]; !ok {
			delete(d.states, pid)
		}
	}
}
