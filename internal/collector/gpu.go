//go:build windows

package collector

import (
	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
	"github.com/ersinkoc/WindowsTaskManager/internal/winapi"
)

// GPUCollector returns best-effort GPU information from the registry. Live
// utilization/VRAM/temperature require D3DKMT or vendor SDKs (NVML, ADL),
// neither of which is allowed under our pure-Go constraint, so the
// Available flag stays false until those are wired up.
type GPUCollector struct {
	cachedName string
}

func NewGPUCollector() *GPUCollector { return &GPUCollector{} }

func (g *GPUCollector) Collect() metrics.GPUMetrics {
	if g.cachedName == "" {
		g.cachedName = readGPUName()
	}
	return metrics.GPUMetrics{
		Name:      g.cachedName,
		Available: false,
	}
}

// readGPUName reads the primary display adapter name from the registry.
func readGPUName() string {
	const path = `SYSTEM\CurrentControlSet\Control\Video\{00000000-0000-0000-0000-000000000000}\0000`
	if name, err := winapi.RegReadString(winapi.HKEY_LOCAL_MACHINE, path, "DriverDesc"); err == nil && name != "" {
		return name
	}
	// Fallback: scan adapter 0000 under the standard root.
	const root = `HARDWARE\DEVICEMAP\VIDEO`
	if device, err := winapi.RegReadString(winapi.HKEY_LOCAL_MACHINE, root, `\Device\Video0`); err == nil && device != "" {
		// device is e.g. \Registry\Machine\System\... ; we cannot follow that
		// without ZwOpenKey, so we stop here and return a placeholder.
		_ = device
	}
	return "Unknown GPU"
}
