package anomaly

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
)

// spawnStormWhitelist is a hard-coded parent exclusion list: shells, browsers,
// dev tools, and terminal hosts that legitimately fork many children under
// normal use. Having it in code (not just config) means a first-time user
// isn't buried in spawn-storm alerts before they've even tuned anything.
var spawnStormWhitelist = map[string]struct{}{
	// Shells / terminals
	"cmd.exe":             {},
	"powershell.exe":      {},
	"pwsh.exe":            {},
	"bash.exe":            {},
	"wsl.exe":             {},
	"wslhost.exe":         {},
	"windowsterminal.exe": {},
	"openconsole.exe":     {},
	"conhost.exe":         {},
	// Browsers (all Chromium-based browsers spawn ~1 child per tab)
	"chrome.exe":    {},
	"msedge.exe":    {},
	"brave.exe":     {},
	"firefox.exe":   {},
	"zen.exe":       {},
	"opera.exe":     {},
	"vivaldi.exe":   {},
	"librewolf.exe": {},
	"arc.exe":       {},
	// Dev / build tools
	"node.exe":   {},
	"bun.exe":    {},
	"deno.exe":   {},
	"python.exe": {},
	"python3.exe": {},
	"go.exe":     {},
	"cargo.exe":  {},
	"docker.exe": {},
	"code.exe":   {},
	"windsurf.exe": {},
}

func isSpawnStormWhitelisted(name string) bool {
	_, ok := spawnStormWhitelist[strings.ToLower(name)]
	return ok
}

// SpawnStormDetector flags processes spawning excessive children rapidly.
// It tracks per-parent child counts within the last 60 seconds.
type SpawnStormDetector struct {
	mu           sync.Mutex
	parentEvents map[uint32][]time.Time // parent PID -> recent child create times
	knownPIDs    map[uint32]struct{}
}

func NewSpawnStormDetector() *SpawnStormDetector {
	return &SpawnStormDetector{
		parentEvents: make(map[uint32][]time.Time),
		knownPIDs:    make(map[uint32]struct{}),
	}
}

func (d *SpawnStormDetector) Name() string { return "spawn_storm" }

func (d *SpawnStormDetector) Analyze(ctx *AnalysisContext) {
	cfg := ctx.Cfg.Anomaly.SpawnStorm
	if !cfg.Enabled {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	cutoff := ctx.Now.Add(-1 * time.Minute)

	// Discover newly created processes since the last analysis.
	currentPIDs := make(map[uint32]struct{}, len(ctx.Snapshot.Processes))
	for i := range ctx.Snapshot.Processes {
		p := &ctx.Snapshot.Processes[i]
		currentPIDs[p.PID] = struct{}{}
		if _, seen := d.knownPIDs[p.PID]; seen {
			continue
		}
		d.knownPIDs[p.PID] = struct{}{}
		if p.ParentPID == 0 {
			continue
		}
		d.parentEvents[p.ParentPID] = append(d.parentEvents[p.ParentPID], ctx.Now)
	}

	// Drop disappeared PIDs to keep the map bounded.
	for pid := range d.knownPIDs {
		if _, ok := currentPIDs[pid]; !ok {
			delete(d.knownPIDs, pid)
		}
	}

	// Trim per-parent events older than 1 minute and check thresholds.
	for parentPID, times := range d.parentEvents {
		fresh := times[:0]
		for _, t := range times {
			if t.After(cutoff) {
				fresh = append(fresh, t)
			}
		}
		if len(fresh) == 0 {
			delete(d.parentEvents, parentPID)
			continue
		}
		d.parentEvents[parentPID] = fresh

		if len(fresh) >= cfg.MaxChildrenPerMinute {
			parent := findProcess(ctx.Snapshot.Processes, parentPID)
			if parent == nil {
				continue
			}
			if isIgnoredProcess(ctx.Cfg, parent.Name) {
				continue
			}
			if isSpawnStormWhitelisted(parent.Name) {
				continue
			}
			ctx.RaiseAlert(Alert{
				Type:        d.Name(),
				Severity:    SeverityCritical,
				Title:       "Process spawn storm",
				Description: fmt.Sprintf("%s (PID %d) spawned %d children in 60s", parent.Name, parent.PID, len(fresh)),
				PID:         parent.PID,
				ProcessName: parent.Name,
				Action:      cfg.Action,
				Details: map[string]any{
					"children_per_minute": len(fresh),
					"threshold":           cfg.MaxChildrenPerMinute,
				},
			})
		}
	}
}

func findProcess(list []metrics.ProcessInfo, pid uint32) *metrics.ProcessInfo {
	for i := range list {
		if list[i].PID == pid {
			return &list[i]
		}
	}
	return nil
}
