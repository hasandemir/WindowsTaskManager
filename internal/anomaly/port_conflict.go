package anomaly

import (
	"fmt"
	"time"
)

// PortConflictDetector flags TCP endpoints stuck in TIME_WAIT or CLOSE_WAIT
// past their respective thresholds. The "Since" field on PortBinding tells
// us how long the tuple has held its current state.
type PortConflictDetector struct{}

func NewPortConflictDetector() *PortConflictDetector { return &PortConflictDetector{} }

func (d *PortConflictDetector) Name() string { return "port_conflict" }

func (d *PortConflictDetector) Analyze(ctx *AnalysisContext) {
	cfg := ctx.Cfg.Anomaly.PortConflict
	if !cfg.Enabled || ctx.Snapshot.PortBindings == nil {
		return
	}
	now := ctx.Now.Unix()

	for _, pb := range ctx.Snapshot.PortBindings {
		var threshold time.Duration
		switch pb.State {
		case "time-wait":
			threshold = cfg.TimeWaitThreshold
		case "close-wait":
			threshold = cfg.CloseWaitThreshold
		default:
			continue
		}
		age := time.Duration(now-pb.Since) * time.Second
		if age < threshold {
			continue
		}
		ctx.RaiseAlert(Alert{
			Type:        d.Name(),
			Severity:    SeverityWarning,
			Title:       "Stuck TCP state",
			Description: fmt.Sprintf("%s on %s:%d held %s for %s", pb.Process, pb.LocalAddr, pb.LocalPort, pb.State, age),
			PID:         pb.PID,
			ProcessName: pb.Process,
			Action:      cfg.Action,
			Details: map[string]any{
				"protocol":    pb.Protocol,
				"local_port":  pb.LocalPort,
				"state":       pb.State,
				"age_seconds": age.Seconds(),
			},
		})
	}
}
