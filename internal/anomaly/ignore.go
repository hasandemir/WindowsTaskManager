package anomaly

import (
	"strings"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
)

// isIgnoredProcess reports whether the given process name matches any entry
// in cfg.Anomaly.IgnoreProcesses. Matching is case-insensitive substring so
// "svchost" in the list catches "svchost.exe", "SVCHOST.EXE", etc. Detectors
// that iterate every process should call this early to avoid drowning the
// alert store in noise from routine Windows background work.
func isIgnoredProcess(cfg *config.Config, name string) bool {
	if cfg == nil || len(cfg.Anomaly.IgnoreProcesses) == 0 || name == "" {
		return false
	}
	lower := strings.ToLower(name)
	for _, pat := range cfg.Anomaly.IgnoreProcesses {
		pat = strings.TrimSpace(pat)
		if pat == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(pat)) {
			return true
		}
	}
	return false
}
