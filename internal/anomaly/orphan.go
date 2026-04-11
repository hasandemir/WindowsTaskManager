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

// Hard-coded floor for orphan significance. Windows orphans are usually
// harmless (installers, updater helpers, tray detaches) — only alert on
// orphans that are genuinely chewing resources.
const (
	orphanMinCPUPercent = 10.0
	orphanMinMemBytes   = uint64(1) << 30 // 1 GB
)

func (d *OrphanDetector) Analyze(ctx *AnalysisContext) {
	cfg := ctx.Cfg.Anomaly.Orphan
	if !cfg.Enabled {
		return
	}
	memThresh := config.ParseSize(cfg.ResourceThresholdMemory)
	if memThresh < orphanMinMemBytes {
		memThresh = orphanMinMemBytes
	}
	cpuThresh := float64(cfg.ResourceThresholdCPU)
	if cpuThresh < orphanMinCPUPercent {
		cpuThresh = orphanMinCPUPercent
	}

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
		if isIgnoredProcess(ctx.Cfg, p.Name) {
			continue
		}
		if p.CPUPercent < cpuThresh && p.WorkingSet < memThresh {
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
