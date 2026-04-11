//go:build windows

package controller

import (
	"os"
	"testing"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
)

func TestSafetyRejectsSelfPID(t *testing.T) {
	s := NewSafety(config.DefaultConfig())
	err := s.Check(metrics.ProcessInfo{
		PID:  uint32(os.Getpid()),
		Name: "wtm.exe",
	}, true)
	if err != ErrSelf {
		t.Fatalf("err=%v want %v", err, ErrSelf)
	}
}
