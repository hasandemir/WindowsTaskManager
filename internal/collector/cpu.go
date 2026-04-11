//go:build windows

package collector

import (
	"runtime"
	"sync"

	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
	"github.com/ersinkoc/WindowsTaskManager/internal/winapi"
)

// CPUCollector samples system-wide CPU usage from GetSystemTimes plus per-core
// usage from NtQuerySystemInformation. Percentages are computed from deltas
// against the previous sample.
type CPUCollector struct {
	mu sync.Mutex

	numLogical int
	cpuName    string
	freqMHz    uint32

	prevIdle   uint64
	prevKernel uint64
	prevUser   uint64
	hasPrev    bool

	prevPerCore []winapi.SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION
}

// NewCPUCollector creates a CPU collector. cpuName/freqMHz come from the
// registry collector at startup; the collector will fall back to defaults
// if either is empty.
func NewCPUCollector(name string, freqMHz uint32) *CPUCollector {
	return &CPUCollector{
		numLogical: runtime.NumCPU(),
		cpuName:    name,
		freqMHz:    freqMHz,
	}
}

// Collect returns a populated CPUMetrics. The first call returns zeros for
// percent (no delta yet) but populates static fields.
func (c *CPUCollector) Collect() metrics.CPUMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := metrics.CPUMetrics{
		NumLogical: c.numLogical,
		Name:       c.cpuName,
		FreqMHz:    c.freqMHz,
	}

	idle, kernel, user, err := winapi.GetSystemTimes()
	if err == nil {
		idleT := idle.Ticks()
		kernelT := kernel.Ticks()
		userT := user.Ticks()
		if c.hasPrev {
			deltaIdle := idleT - c.prevIdle
			deltaKernel := kernelT - c.prevKernel
			deltaUser := userT - c.prevUser
			deltaTotal := deltaKernel + deltaUser
			if deltaTotal > 0 {
				busy := deltaTotal - deltaIdle
				out.TotalPercent = clamp01(float64(busy)/float64(deltaTotal)) * 100.0
			}
		}
		c.prevIdle = idleT
		c.prevKernel = kernelT
		c.prevUser = userT
		c.hasPrev = true
	}

	infos, perr := winapi.QueryProcessorPerformance(c.numLogical)
	if perr == nil {
		out.PerCore = make([]float64, c.numLogical)
		if len(c.prevPerCore) == len(infos) {
			for i := range infos {
				deltaIdle := infos[i].IdleTime - c.prevPerCore[i].IdleTime
				deltaKernel := infos[i].KernelTime - c.prevPerCore[i].KernelTime
				deltaUser := infos[i].UserTime - c.prevPerCore[i].UserTime
				deltaTotal := deltaKernel + deltaUser
				if deltaTotal > 0 {
					busy := deltaTotal - deltaIdle
					out.PerCore[i] = clamp01(float64(busy)/float64(deltaTotal)) * 100.0
				}
			}
		}
		c.prevPerCore = infos
	}

	return out
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
