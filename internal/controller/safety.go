//go:build windows

package controller

import (
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
)

// Common errors returned by the controller surface.
var (
	ErrProtected     = errors.New("process is protected")
	ErrCritical      = errors.New("process is marked critical by Windows")
	ErrConfirmNeeded = errors.New("operation requires explicit confirmation")
	ErrNotFound      = errors.New("process not found")
	ErrSelf          = errors.New("operation not allowed on the running Windows Task Manager process")
)

// Safety enforces the kill/suspend/limit policies declared in config.
type Safety struct {
	mu      sync.RWMutex
	cfg     *config.Config
	selfPID uint32
}

func NewSafety(cfg *config.Config) *Safety {
	return &Safety{
		cfg:     cfg,
		selfPID: uint32(os.Getpid()),
	}
}

// SetConfig swaps the active config (called on hot reload).
func (s *Safety) SetConfig(cfg *config.Config) {
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
}

// Check verifies whether destructive action is allowed against `info`.
// `forceConfirmed` indicates the user has provided explicit confirmation
// for confirm-required cases.
func (s *Safety) Check(info metrics.ProcessInfo, forceConfirmed bool) error {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()
	if cfg == nil {
		return ErrProtected
	}
	if info.PID == s.selfPID {
		return ErrSelf
	}
	if info.PID == 0 || info.PID == 4 {
		return ErrCritical
	}
	if info.IsCritical {
		return ErrCritical
	}
	name := strings.ToLower(info.Name)
	for _, p := range cfg.Controller.ProtectedProcesses {
		if strings.EqualFold(p, name) || strings.EqualFold(p, info.Name) {
			return ErrProtected
		}
	}
	if isSystemPath(info.ExePath) && cfg.Controller.ConfirmKillSystem && !forceConfirmed {
		return ErrConfirmNeeded
	}
	return nil
}

// isSystemPath reports whether an executable lives under %SystemRoot%.
func isSystemPath(path string) bool {
	if path == "" {
		return false
	}
	p := strings.ToLower(path)
	return strings.HasPrefix(p, `c:\windows\`) || strings.HasPrefix(p, `c:\windows\system32\`)
}
