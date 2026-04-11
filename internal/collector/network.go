//go:build windows

package collector

import (
	"sync"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
	"github.com/ersinkoc/WindowsTaskManager/internal/winapi"
)

// netPrev caches an interface's prior counter snapshot for delta calc.
type netPrev struct {
	inOctets   uint64
	outOctets  uint64
	inPackets  uint64
	outPackets uint64
	inErrors   uint64
	outErrors  uint64
	sampleTime time.Time
}

// NetworkCollector samples per-interface throughput.
type NetworkCollector struct {
	mu   sync.Mutex
	prev map[uint32]netPrev
}

func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{prev: make(map[uint32]netPrev)}
}

func (n *NetworkCollector) Collect() metrics.NetworkMetrics {
	rows, err := winapi.GetIfTable2()
	if err != nil || len(rows) == 0 {
		return metrics.NetworkMetrics{}
	}

	now := time.Now()
	out := metrics.NetworkMetrics{
		Interfaces: make([]metrics.InterfaceInfo, 0, len(rows)),
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	live := make(map[uint32]netPrev, len(rows))

	for _, r := range rows {
		// Skip software/loopback interfaces (Type 24 == IF_TYPE_SOFTWARE_LOOPBACK).
		if r.Type == 24 {
			live[r.Index] = netPrev{
				inOctets: r.InOctets, outOctets: r.OutOctets,
				inPackets: r.InUcastPkts, outPackets: r.OutUcastPkts,
				inErrors: r.InErrors, outErrors: r.OutErrors, sampleTime: now,
			}
			continue
		}

		ii := metrics.InterfaceInfo{
			Name:      preferAlias(r.Alias, r.Description),
			Type:      ifTypeName(r.Type),
			Status:    operStatusName(r.OperStatus),
			SpeedMbps: pickSpeedMbps(r.SpeedRx, r.SpeedTx),
		}

		prev, hasPrev := n.prev[r.Index]
		if hasPrev {
			elapsed := now.Sub(prev.sampleTime).Seconds()
			if elapsed > 0 {
				ii.InBPS = bpsDelta(r.InOctets, prev.inOctets, elapsed)
				ii.OutBPS = bpsDelta(r.OutOctets, prev.outOctets, elapsed)
				ii.InPPS = bpsDelta(r.InUcastPkts, prev.inPackets, elapsed)
				ii.OutPPS = bpsDelta(r.OutUcastPkts, prev.outPackets, elapsed)
				ii.InErrors = saturatingSub(r.InErrors, prev.inErrors)
				ii.OutErrors = saturatingSub(r.OutErrors, prev.outErrors)
			}
		}
		live[r.Index] = netPrev{
			inOctets: r.InOctets, outOctets: r.OutOctets,
			inPackets: r.InUcastPkts, outPackets: r.OutUcastPkts,
			inErrors: r.InErrors, outErrors: r.OutErrors,
			sampleTime: now,
		}

		out.Interfaces = append(out.Interfaces, ii)
		out.TotalDownBPS += ii.InBPS
		out.TotalUpBPS += ii.OutBPS
	}

	n.prev = live
	return out
}

func preferAlias(alias, desc string) string {
	if alias != "" {
		return alias
	}
	return desc
}

func bpsDelta(curr, prev uint64, elapsed float64) uint64 {
	if curr <= prev {
		return 0
	}
	return uint64(float64(curr-prev) / elapsed)
}

func saturatingSub(curr, prev uint64) uint64 {
	if curr <= prev {
		return 0
	}
	return curr - prev
}

func pickSpeedMbps(rx, tx uint64) uint64 {
	speed := rx
	if tx > speed {
		speed = tx
	}
	return speed / 1_000_000
}

// ifTypeName maps IANAifType numbers to readable strings.
func ifTypeName(t uint32) string {
	switch t {
	case 6:
		return "ethernet"
	case 9:
		return "token-ring"
	case 23:
		return "ppp"
	case 24:
		return "loopback"
	case 71:
		return "wifi"
	case 131:
		return "tunnel"
	case 144:
		return "ieee1394"
	}
	return "other"
}

func operStatusName(s uint32) string {
	switch s {
	case 1:
		return "up"
	case 2:
		return "down"
	case 3:
		return "testing"
	case 4:
		return "unknown"
	case 5:
		return "dormant"
	case 6:
		return "not-present"
	case 7:
		return "lower-layer-down"
	}
	return "unknown"
}
