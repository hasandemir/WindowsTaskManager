package storage

import (
	"testing"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
)

func TestUpdateLatestDoesNotAppendHistory(t *testing.T) {
	store := NewStore(60, 10)
	ts := time.Unix(1710000000, 0)
	snap := &metrics.SystemSnapshot{
		Timestamp: ts,
		CPU:       metrics.CPUMetrics{TotalPercent: 10},
		Processes: []metrics.ProcessInfo{
			{PID: 42, Name: "demo.exe", CPUPercent: 2.5, WorkingSet: 1024},
		},
	}

	store.SetLatest(snap)
	store.UpdateLatest(func(latest *metrics.SystemSnapshot) {
		latest.ProcessTree = []*metrics.ProcessNode{{Process: latest.Processes[0]}}
		latest.PortBindings = []metrics.PortBinding{{PID: 42, Process: "demo.exe", LocalPort: 8080}}
	})

	if got := len(store.SystemHistory()); got != 1 {
		t.Fatalf("system history len=%d want 1", got)
	}
	if got := len(store.ProcessHistory(42)); got != 1 {
		t.Fatalf("process history len=%d want 1", got)
	}
	latest := store.Latest()
	if latest == nil {
		t.Fatal("latest snapshot is nil")
	}
	if len(latest.ProcessTree) != 1 {
		t.Fatalf("process tree len=%d want 1", len(latest.ProcessTree))
	}
	if len(latest.PortBindings) != 1 {
		t.Fatalf("port bindings len=%d want 1", len(latest.PortBindings))
	}
}
