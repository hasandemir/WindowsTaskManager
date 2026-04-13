//go:build windows

package controller

import (
	"slices"
	"testing"
	"time"

	"golang.org/x/sys/windows"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/event"
	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
	"github.com/ersinkoc/WindowsTaskManager/internal/storage"
	"github.com/ersinkoc/WindowsTaskManager/internal/winapi"
)

func TestFindProcessReturnsMatchAndNotFound(t *testing.T) {
	store := storage.NewStore(60, 10)
	store.SetLatest(&metrics.SystemSnapshot{
		Timestamp: time.Now(),
		Processes: []metrics.ProcessInfo{
			{PID: 101, Name: "alpha.exe"},
			{PID: 202, Name: "beta.exe"},
		},
	})

	ctrl := NewController(config.DefaultConfig(), store, nil)

	info, err := ctrl.findProcess(202)
	if err != nil {
		t.Fatalf("findProcess: %v", err)
	}
	if info.Name != "beta.exe" {
		t.Fatalf("name=%q want beta.exe", info.Name)
	}

	if _, err := ctrl.findProcess(999); err != ErrNotFound {
		t.Fatalf("err=%v want %v", err, ErrNotFound)
	}
}

func TestCollectDescendantsIncludesNestedChildren(t *testing.T) {
	procs := []metrics.ProcessInfo{
		{PID: 10, ParentPID: 1},
		{PID: 11, ParentPID: 10},
		{PID: 12, ParentPID: 10},
		{PID: 13, ParentPID: 11},
		{PID: 20, ParentPID: 2},
	}

	got := collectDescendants(procs, 10)
	slices.Sort(got)
	want := []uint32{11, 12, 13}
	if !slices.Equal(got, want) {
		t.Fatalf("descendants=%v want %v", got, want)
	}
}

func TestPriorityClassFromStringSupportsAliases(t *testing.T) {
	cases := map[string]uint32{
		"idle":         winapi.IDLE_PRIORITY_CLASS,
		"below_normal": winapi.BELOW_NORMAL_PRIORITY_CLASS,
		"below-normal": winapi.BELOW_NORMAL_PRIORITY_CLASS,
		"normal":       winapi.NORMAL_PRIORITY_CLASS,
		"above_normal": winapi.ABOVE_NORMAL_PRIORITY_CLASS,
		"above-normal": winapi.ABOVE_NORMAL_PRIORITY_CLASS,
		"high":         winapi.HIGH_PRIORITY_CLASS,
		"realtime":     winapi.REALTIME_PRIORITY_CLASS,
	}
	for input, want := range cases {
		got, ok := priorityClassFromString(input)
		if !ok {
			t.Fatalf("priorityClassFromString(%q) returned ok=false", input)
		}
		if got != want {
			t.Fatalf("priorityClassFromString(%q)=%d want %d", input, got, want)
		}
	}

	if _, ok := priorityClassFromString("turbo"); ok {
		t.Fatal("unexpected success for unknown priority class")
	}
}

func TestEmitPublishesUniformPayload(t *testing.T) {
	em := event.NewEmitter()
	got := make(chan map[string]any, 1)
	errCh := make(chan string, 1)
	em.Subscribe(func(eventType string, data any) {
		if eventType != EventAffinity {
			return
		}
		payload, ok := data.(map[string]any)
		if !ok {
			errCh <- "payload type mismatch"
			return
		}
		got <- payload
	})

	ctrl := NewController(config.DefaultConfig(), storage.NewStore(60, 10), em)
	ctrl.emit(EventAffinity, 55, map[string]any{"mask": uint64(3)})

	select {
	case payload := <-got:
		if payload["pid"] != uint32(55) {
			t.Fatalf("pid=%v want 55", payload["pid"])
		}
		if payload["mask"] != uint64(3) {
			t.Fatalf("mask=%v want 3", payload["mask"])
		}
	case msg := <-errCh:
		t.Fatal(msg)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("controller event was not emitted")
	}
}

func TestActiveLimitsAndClearLimit(t *testing.T) {
	em := event.NewEmitter()
	cleared := make(chan map[string]any, 1)
	errCh := make(chan string, 1)
	em.Subscribe(func(eventType string, data any) {
		if eventType != EventLimitCleared {
			return
		}
		payload, ok := data.(map[string]any)
		if !ok {
			errCh <- "payload type mismatch"
			return
		}
		cleared <- payload
	})

	ctrl := NewController(config.DefaultConfig(), storage.NewStore(60, 10), em)
	ctrl.jobs[101] = &jobEntry{job: windows.Handle(0), pid: 101, cpuPct: 25, memBytes: 1024}
	ctrl.jobs[202] = &jobEntry{job: windows.Handle(0), pid: 202, cpuPct: 50, memBytes: 2048}

	limits := ctrl.ActiveLimits()
	slices.SortFunc(limits, func(a, b LimitInfo) int {
		switch {
		case a.PID < b.PID:
			return -1
		case a.PID > b.PID:
			return 1
		default:
			return 0
		}
	})
	if len(limits) != 2 {
		t.Fatalf("limits=%d want 2", len(limits))
	}
	if limits[0].PID != 101 || limits[1].PID != 202 {
		t.Fatalf("limits=%+v", limits)
	}

	if err := ctrl.ClearLimit(101); err != nil {
		t.Fatalf("ClearLimit: %v", err)
	}
	if _, ok := ctrl.jobs[101]; ok {
		t.Fatal("pid 101 limit should be removed")
	}

	select {
	case payload := <-cleared:
		if payload["pid"] != uint32(101) {
			t.Fatalf("pid=%v want 101", payload["pid"])
		}
	case msg := <-errCh:
		t.Fatal(msg)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("limit cleared event was not emitted")
	}

	if err := ctrl.ClearLimit(404); err == nil {
		t.Fatal("expected error for missing limit")
	}
}
