//go:build windows

package config

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherReloadsOnSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wtm.yaml")
	cfg := DefaultConfig()
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	changed := make(chan *Config, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := NewWatcher(path, func(next *Config) {
		select {
		case changed <- next:
		default:
		}
	})
	go w.Start(ctx)
	time.Sleep(150 * time.Millisecond)

	cfg.Server.Port = 23456
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-changed:
		if got.Server.Port != 23456 {
			t.Fatalf("port=%d want %d", got.Server.Port, 23456)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("watcher did not reload config after save")
	}
}
