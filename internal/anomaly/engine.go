package anomaly

import (
	"context"
	"sync"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/event"
	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
	"github.com/ersinkoc/WindowsTaskManager/internal/storage"
)

// Event names emitted by the engine.
const (
	EventAlertRaised  = "anomaly.raised"
	EventAlertCleared = "anomaly.cleared"
)

// Detector is the interface implemented by every detector.
type Detector interface {
	Name() string
	Analyze(ctx *AnalysisContext)
}

// ProcessActuator is the subset of the controller we call from rules. We keep
// the interface minimal so the anomaly package doesn't pull in the full
// controller dependency tree (it imports winapi which is Windows-only).
type ProcessActuator interface {
	Kill(pid uint32, confirm bool) error
	Suspend(pid uint32) error
}

// AnalysisContext bundles the inputs each detector needs.
type AnalysisContext struct {
	Now      time.Time
	Snapshot *metrics.SystemSnapshot
	History  []metrics.TimestampedSystem
	Store    *storage.Store
	Cfg      *config.Config
	Alerts   *AlertStore
	Emitter  *event.Emitter
	Actuator ProcessActuator // may be nil in tests
}

// Engine runs all registered detectors on the configured cadence.
type Engine struct {
	cfg       *config.Config
	store     *storage.Store
	emitter   *event.Emitter
	alerts    *AlertStore
	actuator  ProcessActuator
	detectors []Detector

	mu sync.RWMutex
}

func NewEngine(cfg *config.Config, store *storage.Store, emitter *event.Emitter, alerts *AlertStore) *Engine {
	e := &Engine{
		cfg:     cfg,
		store:   store,
		emitter: emitter,
		alerts:  alerts,
	}
	e.registerDefaults()
	return e
}

// SetActuator wires the process controller so rule-based actions can kill /
// suspend processes. Without it, rules still evaluate and raise alerts but
// never execute the action.
func (e *Engine) SetActuator(a ProcessActuator) {
	e.mu.Lock()
	e.actuator = a
	e.mu.Unlock()
}

// Detectors returns the registered detector list.
func (e *Engine) Detectors() []Detector {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Detector, len(e.detectors))
	copy(out, e.detectors)
	return out
}

// Register adds a custom detector.
func (e *Engine) Register(d Detector) {
	e.mu.Lock()
	e.detectors = append(e.detectors, d)
	e.mu.Unlock()
}

// SetConfig hot-swaps the active config.
func (e *Engine) SetConfig(cfg *config.Config) {
	e.mu.Lock()
	e.cfg = cfg
	e.mu.Unlock()
}

// Start runs the analysis loop until ctx is cancelled.
func (e *Engine) Start(ctx context.Context) {
	go e.loop(ctx)
}

func (e *Engine) loop(ctx context.Context) {
	interval := e.cfg.Anomaly.AnalysisInterval
	if interval < 500*time.Millisecond {
		interval = 2 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			e.analyzeOnce()
		}
	}
}

func (e *Engine) analyzeOnce() {
	snap := e.store.Latest()
	if snap == nil {
		return
	}
	e.mu.RLock()
	cfg := e.cfg
	dets := append([]Detector{}, e.detectors...)
	act := e.actuator
	e.mu.RUnlock()

	actx := &AnalysisContext{
		Now:      time.Now(),
		Snapshot: snap,
		History:  e.store.SystemHistory(),
		Store:    e.store,
		Cfg:      cfg,
		Alerts:   e.alerts,
		Emitter:  e.emitter,
		Actuator: act,
	}

	for _, d := range dets {
		safeAnalyze(d, actx)
	}
}

func safeAnalyze(d Detector, actx *AnalysisContext) {
	defer func() { _ = recover() }()
	d.Analyze(actx)
}

// raise is a helper used by detectors via AnalysisContext.RaiseAlert below.
func (a *AnalysisContext) RaiseAlert(alert Alert) {
	stored, isNew := a.Alerts.Raise(alert)
	if isNew && a.Emitter != nil {
		a.Emitter.Emit(EventAlertRaised, stored)
	}
}

// ClearAlert removes an active alert and emits a cleared event.
func (a *AnalysisContext) ClearAlert(alertType string, pid uint32) {
	a.Alerts.Clear(alertType, pid)
	if a.Emitter != nil {
		a.Emitter.Emit(EventAlertCleared, map[string]any{"type": alertType, "pid": pid})
	}
}

// registerDefaults wires every built-in detector. Implementations live in
// sibling files within this package.
func (e *Engine) registerDefaults() {
	e.detectors = []Detector{
		NewSpawnStormDetector(),
		NewMemoryLeakDetector(),
		NewHungProcessDetector(),
		NewOrphanDetector(),
		NewRunawayCPUDetector(),
		NewPortConflictDetector(),
		NewNetworkAnomalyDetector(),
		NewNewProcessDetector(),
		NewRulesDetector(),
	}
}
