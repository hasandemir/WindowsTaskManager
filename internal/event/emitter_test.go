package event

import (
	"testing"
	"time"
)

func TestEmitDoesNotBlockOnSlowSubscriber(t *testing.T) {
	e := NewEmitter()
	blocked := make(chan struct{})
	release := make(chan struct{})
	e.Subscribe(func(eventType string, data any) {
		close(blocked)
		<-release
	})

	start := time.Now()
	e.Emit("test", 1)
	elapsed := time.Since(start)

	select {
	case <-blocked:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("subscriber was not invoked")
	}
	if elapsed > 50*time.Millisecond {
		t.Fatalf("Emit blocked for %s", elapsed)
	}
	close(release)
}
