//go:build windows

package collector

import "testing"

func TestParseGPUEngineInstance(t *testing.T) {
	adapter, engine, ok := parseGPUEngineInstance("pid_15304_luid_0x00000000_0x0001688d_phys_0_eng_13_engtype_copy")
	if !ok {
		t.Fatal("expected GPU engine instance to parse")
	}
	if adapter != "luid_0x00000000_0x0001688d_phys_0" {
		t.Fatalf("adapter=%q", adapter)
	}
	if engine != "13" {
		t.Fatalf("engine=%q want 13", engine)
	}
}

func TestAggregateGPUSamplesCapsEngineAndKeepsMemory(t *testing.T) {
	samples := aggregateGPUSamples(
		map[string]float64{
			"pid_1_luid_0x0_0x1_phys_0_eng_0_engtype_3d":   80,
			"pid_2_luid_0x0_0x1_phys_0_eng_0_engtype_3d":   35,
			"pid_3_luid_0x0_0x2_phys_1_eng_1_engtype_copy": 20,
		},
		map[string]float64{
			"luid_0x0_0x1_phys_0": 1024,
			"luid_0x0_0x2_phys_1": 4096,
		},
		map[string]float64{
			"luid_0x0_0x1_phys_0": 512,
		},
	)

	first := samples["luid_0x0_0x1_phys_0"]
	if first.utilization != 100 {
		t.Fatalf("utilization=%v want 100", first.utilization)
	}
	if first.dedicated != 1024 || first.shared != 512 {
		t.Fatalf("memory sample=%+v", first)
	}

	key, sample, ok := pickPrimaryGPU(samples)
	if !ok {
		t.Fatal("expected primary GPU to be selected")
	}
	if key != "luid_0x0_0x1_phys_0" {
		t.Fatalf("primary key=%q", key)
	}
	if sample.utilization != 100 {
		t.Fatalf("primary utilization=%v want 100", sample.utilization)
	}
}

func TestParseGPUAdapterIndex(t *testing.T) {
	got, err := parseGPUAdapterIndex("luid_0x00000000_0x0001688d_phys_3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 3 {
		t.Fatalf("index=%d want 3", got)
	}
}
