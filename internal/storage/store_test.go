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

func TestLatestReturnsDetachedClone(t *testing.T) {
	store := NewStore(60, 10)
	ts := time.Unix(1710001000, 0)
	original := &metrics.SystemSnapshot{
		Timestamp: ts,
		CPU: metrics.CPUMetrics{
			TotalPercent: 20,
			PerCore:      []float64{10, 30},
		},
		Disk: metrics.DiskMetrics{
			Drives: []metrics.DriveInfo{{Letter: "C:", UsedPct: 42}},
		},
		Network: metrics.NetworkMetrics{
			Interfaces: []metrics.InterfaceInfo{{Name: "Ethernet", InBPS: 100}},
		},
		Processes: []metrics.ProcessInfo{
			{PID: 7, Name: "agent.exe", CPUPercent: 8.5},
		},
		ProcessTree: []*metrics.ProcessNode{{
			Process: metrics.ProcessInfo{PID: 7, Name: "agent.exe"},
			Children: []*metrics.ProcessNode{{
				Process: metrics.ProcessInfo{PID: 8, Name: "worker.exe"},
			}},
		}},
		PortBindings: []metrics.PortBinding{{PID: 7, Process: "agent.exe", LocalPort: 9000}},
	}

	store.SetLatest(original)

	original.Processes[0].Name = "mutated.exe"
	original.ProcessTree[0].Children[0].Process.Name = "mutated-child.exe"
	original.PortBindings[0].Process = "mutated.exe"

	latestA := store.Latest()
	if latestA == nil {
		t.Fatal("latest snapshot is nil")
	}
	if latestA.Processes[0].Name != "agent.exe" {
		t.Fatalf("stored process name=%q want agent.exe", latestA.Processes[0].Name)
	}
	if latestA.ProcessTree[0].Children[0].Process.Name != "worker.exe" {
		t.Fatalf("stored child name=%q want worker.exe", latestA.ProcessTree[0].Children[0].Process.Name)
	}
	if latestA.PortBindings[0].Process != "agent.exe" {
		t.Fatalf("stored binding process=%q want agent.exe", latestA.PortBindings[0].Process)
	}

	latestA.Processes[0].Name = "client-side-mutation.exe"
	latestA.CPU.PerCore[0] = 999
	latestA.ProcessTree[0].Process.Name = "changed-tree.exe"

	latestB := store.Latest()
	if latestB.Processes[0].Name != "agent.exe" {
		t.Fatalf("latest process name=%q want agent.exe", latestB.Processes[0].Name)
	}
	if latestB.CPU.PerCore[0] != 10 {
		t.Fatalf("latest per-core[0]=%v want 10", latestB.CPU.PerCore[0])
	}
	if latestB.ProcessTree[0].Process.Name != "agent.exe" {
		t.Fatalf("latest tree root=%q want agent.exe", latestB.ProcessTree[0].Process.Name)
	}
}
