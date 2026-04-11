package config

import (
	"context"
	"os"
	"time"
)

// Watcher polls a config file for modifications and invokes a callback when it changes.
// Polling is portable and avoids extra dependencies.
type Watcher struct {
	path     string
	onChange func(*Config)
	interval time.Duration
}

func NewWatcher(path string, onChange func(*Config)) *Watcher {
	return &Watcher{
		path:     path,
		onChange: onChange,
		interval: 2 * time.Second,
	}
}

// Start begins polling. The watcher exits when the context is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	var lastMod time.Time
	if info, err := os.Stat(w.path); err == nil {
		lastMod = info.ModTime()
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := os.Stat(w.path)
			if err != nil {
				continue
			}
			if info.ModTime().After(lastMod) {
				lastMod = info.ModTime()
				time.Sleep(300 * time.Millisecond) // settle
				cfg, err := Load(w.path)
				if err == nil && w.onChange != nil {
					w.onChange(cfg)
				}
			}
		}
	}
}
