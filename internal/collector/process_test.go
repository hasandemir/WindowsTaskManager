//go:build windows

package collector

import (
	"testing"
	"time"
)

func TestProcessIODeltasUsePreviousCounters(t *testing.T) {
	prev := procPrev{
		ioReadBytes:  100,
		ioWriteBytes: 200,
		ioReadOps:    3,
		ioWriteOps:   5,
		sampleTime:   time.Now().Add(-time.Second),
	}
	if got := saturatingSub(180, prev.ioReadBytes); got != 80 {
		t.Fatalf("read delta=%d want 80", got)
	}
	if got := saturatingSub(260, prev.ioWriteBytes); got != 60 {
		t.Fatalf("write delta=%d want 60", got)
	}
	if got := saturatingSub(9, prev.ioReadOps); got != 6 {
		t.Fatalf("read ops delta=%d want 6", got)
	}
	if got := saturatingSub(11, prev.ioWriteOps); got != 6 {
		t.Fatalf("write ops delta=%d want 6", got)
	}
}
