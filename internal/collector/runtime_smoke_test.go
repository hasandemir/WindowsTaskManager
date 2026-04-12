//go:build windows

package collector

import "testing"

func TestDiskCollectorCollectSmoke(t *testing.T) {
	collector := NewDiskCollector()
	snap := collector.Collect()
	if snap.Drives == nil {
		t.Fatal("expected drives slice to be initialized")
	}
}

func TestGPUCollectorCollectSmoke(t *testing.T) {
	collector := NewGPUCollector()
	snap := collector.Collect()
	if snap.Name == "" {
		t.Fatal("expected GPU name fallback to be populated")
	}
	if snap.Available {
		if snap.Utilization < 0 {
			t.Fatalf("unexpected negative utilization: %v", snap.Utilization)
		}
		if snap.VRAMUsed > snap.VRAMTotal && snap.VRAMTotal != 0 {
			t.Fatalf("unexpected VRAM shape: used=%d total=%d", snap.VRAMUsed, snap.VRAMTotal)
		}
	}
}
