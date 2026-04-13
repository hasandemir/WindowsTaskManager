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

func TestEmitRecoversFromSubscriberPanic(t *testing.T) {
	e := NewEmitter()
	done := make(chan struct{}, 1)

	e.Subscribe(func(eventType string, data any) {
		panic("boom")
	})
	e.Subscribe(func(eventType string, data any) {
		done <- struct{}{}
	})

	e.Emit("test", 1)

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("healthy subscriber was not invoked after panic")
	}
}
