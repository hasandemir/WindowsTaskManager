package anomaly

import (
	"fmt"

	"github.com/ersinkoc/WindowsTaskManager/internal/stats"
)

// NetworkAnomalyDetector flags either a connection-count blowup or a system-wide
// connection cap breach. Connection counts are smoothed using Welford's mean
// to compute a sigma-based threshold per-process.
type NetworkAnomalyDetector struct {
	stats map[uint32]*stats.Welford // pid -> connection count stats
}

func NewNetworkAnomalyDetector() *NetworkAnomalyDetector {
	return &NetworkAnomalyDetector{stats: make(map[uint32]*stats.Welford)}
}

func (d *NetworkAnomalyDetector) Name() string { return "network_anomaly" }

func (d *NetworkAnomalyDetector) Analyze(ctx *AnalysisContext) {
	cfg := ctx.Cfg.Anomaly.NetworkAnomaly
	if !cfg.Enabled || ctx.Snapshot.PortBindings == nil {
		return
	}

	perPID := make(map[uint32]int, 64)
	totalConns := 0
	for _, pb := range ctx.Snapshot.PortBindings {
		if pb.PID == 0 {
			continue
		}
		perPID[pb.PID]++
		totalConns++
	}

	if totalConns >= cfg.MaxSystemConnections {
		ctx.RaiseAlert(Alert{
			Type:        d.Name() + "_system",
			Severity:    SeverityCritical,
			Title:       "System connection cap exceeded",
			Description: fmt.Sprintf("%d active connections (limit %d)", totalConns, cfg.MaxSystemConnections),
			Action:      cfg.Action,
			Details: map[string]any{
				"total_connections": totalConns,
				"limit":             cfg.MaxSystemConnections,
			},
		})
	} else {
		ctx.ClearAlert(d.Name()+"_system", 0)
	}

	live := make(map[uint32]struct{}, len(perPID))
	for pid, count := range perPID {
		live[pid] = struct{}{}
		w, ok := d.stats[pid]
		if !ok {
			w = stats.NewWelford()
			d.stats[pid] = w
		}
		fcount := float64(count)
		if w.Count() >= 10 && w.IsAnomaly(fcount, float64(cfg.ConnectionSigma)) && fcount > w.Mean()+1 {
			procName := ctx.Snapshot.ProcessName(pid)
			ctx.RaiseAlert(Alert{
				Type:        d.Name(),
				Severity:    SeverityWarning,
				Title:       "Connection burst",
				Description: fmt.Sprintf("%s (PID %d) has %d connections (mean %.1f)", procName, pid, count, w.Mean()),
				PID:         pid,
				ProcessName: procName,
				Action:      cfg.Action,
				Details: map[string]any{
					"connections": count,
					"mean":        w.Mean(),
					"std_dev":     w.StdDev(),
				},
			})
		}
		w.Add(fcount)
	}
	for pid := range d.stats {
		if _, ok := live[pid]; !ok {
			delete(d.stats, pid)
		}
	}
}
