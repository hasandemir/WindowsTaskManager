//go:build windows

package collector

import "testing"

func TestNormalizeCounterMapDropsTotalAndNormalizesKeys(t *testing.T) {
	got := normalizeCounterMap(map[string]float64{
		"C:":       42.4,
		" _Total ": 99,
		"D:":       11,
	})
	if len(got) != 2 {
		t.Fatalf("normalizeCounterMap len=%d want 2", len(got))
	}
	if got["c:"] != 42.4 {
		t.Fatalf("c:=%v want 42.4", got["c:"])
	}
	if got["d:"] != 11 {
		t.Fatalf("d:=%v want 11", got["d:"])
	}
	if _, ok := got["_total"]; ok {
		t.Fatal("_total should be filtered out")
	}
}

func TestCounterUint64RoundsPositiveValues(t *testing.T) {
	if got := counterUint64(12.6); got != 13 {
		t.Fatalf("counterUint64(12.6)=%d want 13", got)
	}
	if got := counterUint64(-5); got != 0 {
		t.Fatalf("counterUint64(-5)=%d want 0", got)
	}
}
