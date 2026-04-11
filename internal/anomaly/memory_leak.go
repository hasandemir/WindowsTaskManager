package anomaly

import (
	"fmt"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/stats"
)

// MemoryLeakDetector flags processes whose working set grows linearly over a
// configurable window with high R² and average use above a threshold.
type MemoryLeakDetector struct{}

func NewMemoryLeakDetector() *MemoryLeakDetector { return &MemoryLeakDetector{} }

func (d *MemoryLeakDetector) Name() string { return "memory_leak" }

func (d *MemoryLeakDetector) Analyze(ctx *AnalysisContext) {
	cfg := ctx.Cfg.Anomaly.MemoryLeak
	if !cfg.Enabled {
		return
	}
	growthBps := config.ParseSize(cfg.MinGrowthRate) // bytes per minute
	memThresh := config.ParseSize(cfg.MemoryThreshold)
	if growthBps == 0 || memThresh == 0 {
		return
	}

	windowStart := ctx.Now.Add(-cfg.Window)

	for _, p := range ctx.Snapshot.Processes {
		if isIgnoredProcess(ctx.Cfg, p.Name) {
			continue
		}
		samples := ctx.Store.ProcessHistory(p.PID)
		if len(samples) < 5 {
			continue
		}
		xs := make([]float64, 0, len(samples))
		ys := make([]float64, 0, len(samples))
		for _, s := range samples {
			if s.Time.Before(windowStart) {
				continue
			}
			xs = append(xs, float64(s.Time.Unix()))
			ys = append(ys, float64(s.WorkingSet))
		}
		if len(xs) < 5 {
			continue
		}

		slope, _, r2 := stats.LinearRegression(xs, ys)
		if r2 < cfg.MinRSquared {
			continue
		}
		// slope is bytes/sec; convert to bytes/min for comparison
		growthPerMin := slope * 60.0
		if growthPerMin < float64(growthBps) {
			continue
		}
		if p.WorkingSet < memThresh {
			continue
		}
		ctx.RaiseAlert(Alert{
			Type:        d.Name(),
			Severity:    SeverityWarning,
			Title:       "Memory leak suspected",
			Description: fmt.Sprintf("%s (PID %d) growing %.1f MB/min (R²=%.2f)", p.Name, p.PID, growthPerMin/(1<<20), r2),
			PID:         p.PID,
			ProcessName: p.Name,
			Action:      cfg.Action,
			Details: map[string]any{
				"growth_bytes_per_min": growthPerMin,
				"r_squared":            r2,
				"working_set":          p.WorkingSet,
			},
		})
	}
}
