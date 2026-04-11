//go:build windows

package collector

import (
	"strings"

	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
	"github.com/ersinkoc/WindowsTaskManager/internal/winapi"
)

// DiskCollector returns the list of fixed drives plus their capacity.
// IO/sec counters are left at zero — implementing the Win32 disk
// performance API requires elevation and is left for a future iteration.
type DiskCollector struct{}

func NewDiskCollector() *DiskCollector { return &DiskCollector{} }

func (d *DiskCollector) Collect() metrics.DiskMetrics {
	roots, err := winapi.GetLogicalDriveStrings()
	if err != nil {
		return metrics.DiskMetrics{}
	}
	out := metrics.DiskMetrics{Drives: make([]metrics.DriveInfo, 0, len(roots))}
	for _, root := range roots {
		t := winapi.GetDriveType(root)
		if t != winapi.DRIVE_FIXED && t != winapi.DRIVE_REMOVABLE && t != winapi.DRIVE_REMOTE {
			continue
		}
		_, total, free, err := winapi.GetDiskFreeSpaceEx(root)
		if err != nil || total == 0 {
			continue
		}
		used := total - free
		label, fs := winapi.GetVolumeInformation(root)
		di := metrics.DriveInfo{
			Letter:     strings.TrimRight(root, "\\"),
			Label:      label,
			FSType:     fs,
			TotalBytes: total,
			FreeBytes:  free,
			UsedBytes:  used,
			UsedPct:    float64(used) / float64(total) * 100.0,
		}
		out.Drives = append(out.Drives, di)
	}
	return out
}
