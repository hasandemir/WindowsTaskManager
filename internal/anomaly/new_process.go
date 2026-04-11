package anomaly

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// NewProcessDetector flags freshly spawned processes whose executable lives
// inside one of the configured suspicious paths. Each PID alerts at most once.
type NewProcessDetector struct {
	mu    sync.Mutex
	known map[uint32]struct{}
}

func NewNewProcessDetector() *NewProcessDetector {
	return &NewProcessDetector{known: make(map[uint32]struct{})}
}

func (d *NewProcessDetector) Name() string { return "new_process" }

func (d *NewProcessDetector) Analyze(ctx *AnalysisContext) {
	cfg := ctx.Cfg.Anomaly.NewProcess
	if !cfg.Enabled {
		return
	}
	patterns := expandPaths(cfg.SuspiciousPaths)

	d.mu.Lock()
	defer d.mu.Unlock()

	live := make(map[uint32]struct{}, len(ctx.Snapshot.Processes))
	for i := range ctx.Snapshot.Processes {
		p := &ctx.Snapshot.Processes[i]
		live[p.PID] = struct{}{}
		if _, seen := d.known[p.PID]; seen {
			continue
		}
		d.known[p.PID] = struct{}{}
		if isIgnoredProcess(ctx.Cfg, p.Name) {
			continue
		}
		if p.ExePath == "" {
			continue
		}
		lower := strings.ToLower(p.ExePath)
		matched := ""
		for _, pat := range patterns {
			if pat != "" && strings.HasPrefix(lower, pat) {
				matched = pat
				break
			}
		}
		if matched == "" {
			continue
		}
		ctx.RaiseAlert(Alert{
			Type:        d.Name(),
			Severity:    SeverityInfo,
			Title:       "New process from suspicious path",
			Description: fmt.Sprintf("%s spawned from %s", p.Name, p.ExePath),
			PID:         p.PID,
			ProcessName: p.Name,
			Action:      cfg.Action,
			Details: map[string]any{
				"exe_path":     p.ExePath,
				"matched_path": matched,
			},
		})
	}
	for pid := range d.known {
		if _, ok := live[pid]; !ok {
			delete(d.known, pid)
		}
	}
}

// expandPaths resolves environment variables (%TEMP%, %USERPROFILE%) in
// each pattern and returns lowercase results.
func expandPaths(patterns []string) []string {
	out := make([]string, 0, len(patterns))
	for _, p := range patterns {
		expanded := os.ExpandEnv(replaceWinVars(p))
		out = append(out, strings.ToLower(expanded))
	}
	return out
}

func replaceWinVars(s string) string {
	// Convert %FOO% syntax into ${FOO} so os.ExpandEnv handles it.
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '%' {
			end := strings.IndexByte(s[i+1:], '%')
			if end > 0 {
				b.WriteString("${")
				b.WriteString(s[i+1 : i+1+end])
				b.WriteString("}")
				i += end + 2
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
