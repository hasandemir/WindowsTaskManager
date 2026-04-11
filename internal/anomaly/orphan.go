package anomaly

import (
	"fmt"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
)

// OrphanDetector flags processes whose parent has died but which still
// consume resources above configurable thresholds.
type OrphanDetector struct{}

func NewOrphanDetector() *OrphanDetector { return &OrphanDetector{} }

func (d *OrphanDetector) Name() string { return "orphan" }

func (d *OrphanDetector) Analyze(ctx *AnalysisContext) {
	cfg := ctx.Cfg.Anomaly.Orphan
	if !cfg.Enabled {
		return
	}
	memThresh := config.ParseSize(cfg.ResourceThresholdMemory)

	pidSet := make(map[uint32]metrics.ProcessInfo, len(ctx.Snapshot.Processes))
	for _, p := range ctx.Snapshot.Processes {
		pidSet[p.PID] = p
	}

	for _, p := range ctx.Snapshot.Processes {
		if p.ParentPID == 0 {
			continue
		}
		if _, alive := pidSet[p.ParentPID]; alive {
			continue
		}
		if p.CPUPercent < float64(cfg.ResourceThresholdCPU) && p.WorkingSet < memThresh {
			continue
		}
		ctx.RaiseAlert(Alert{
			Type:        d.Name(),
			Severity:    SeverityInfo,
			Title:       "Orphan process",
			Description: fmt.Sprintf("%s (PID %d) parent %d is gone", p.Name, p.PID, p.ParentPID),
			PID:         p.PID,
			ProcessName: p.Name,
			Action:      cfg.Action,
			Details: map[string]any{
				"parent_pid":  p.ParentPID,
				"cpu_percent": p.CPUPercent,
				"working_set": p.WorkingSet,
			},
		})
	}
}
