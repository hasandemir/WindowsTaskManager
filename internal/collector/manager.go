//go:build windows

package collector

import (
	"context"
	"sync"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/event"
	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
	"github.com/ersinkoc/WindowsTaskManager/internal/storage"
	"github.com/ersinkoc/WindowsTaskManager/internal/winapi"
)

// Event names emitted by the Manager via its Emitter.
const (
	EventSnapshot     = "metrics.snapshot"
	EventProcessTree  = "metrics.process_tree"
	EventPortBindings = "metrics.port_bindings"
)

// Manager owns every collector, the latest snapshot, and the storage.
// It runs three independent ticker loops:
//   - Fast: CPU/memory/network/disk + processes (config.Monitoring.Interval)
//   - ProcessTree: builds a tree from the latest processes
//   - Ports: enumerates TCP/UDP endpoints
type Manager struct {
	cfg     *config.Config
	store   *storage.Store
	emitter *event.Emitter

	cpu   *CPUCollector
	mem   *MemoryCollector
	proc  *ProcessCollector
	net   *NetworkCollector
	disk  *DiskCollector
	gpu   *GPUCollector
	ports *PortCollector

	mu        sync.RWMutex
	latestPID map[uint32]string
}

// NewManager wires every collector together. cpuName/freqMHz are sourced
// from the registry by the caller (cmd/wtm) before calling here.
func NewManager(cfg *config.Config, store *storage.Store, emitter *event.Emitter, cpuName string, cpuMHz uint32) *Manager {
	return &Manager{
		cfg:       cfg,
		store:     store,
		emitter:   emitter,
		cpu:       NewCPUCollector(cpuName, cpuMHz),
		mem:       NewMemoryCollector(),
		proc:      NewProcessCollector(cfg.Monitoring.MaxProcesses),
		net:       NewNetworkCollector(),
		disk:      NewDiskCollector(),
		gpu:       NewGPUCollector(),
		ports:     NewPortCollector(cfg.WellKnownPorts),
		latestPID: make(map[uint32]string),
	}
}

// Start launches all background sample loops. It returns immediately;
// loops exit when ctx is cancelled.
func (m *Manager) Start(ctx context.Context) {
	go m.fastLoop(ctx)
	go m.treeLoop(ctx)
	go m.portsLoop(ctx)
}

// CollectOnce runs the fast collectors synchronously and returns the snapshot.
// Used at startup so the first HTTP request sees data.
func (m *Manager) CollectOnce() *metrics.SystemSnapshot {
	return m.fastSample()
}

func (m *Manager) fastLoop(ctx context.Context) {
	pruneT := time.NewTicker(30 * time.Second)
	defer pruneT.Stop()
	timer := time.NewTimer(m.fastInterval())
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			m.fastSample()
			timer.Reset(m.fastInterval())
		case <-pruneT.C:
			m.store.PruneStaleProcesses(time.Now().Add(-2 * time.Minute))
		}
	}
}

func (m *Manager) fastSample() *metrics.SystemSnapshot {
	snap := &metrics.SystemSnapshot{
		Timestamp: time.Now(),
		CPU:       m.cpu.Collect(),
		Memory:    m.mem.Collect(),
		GPU:       m.gpu.Collect(),
		Network:   m.net.Collect(),
		Disk:      m.disk.Collect(),
		Processes: m.proc.Collect(),
	}

	pidMap := make(map[uint32]string, len(snap.Processes))
	for i := range snap.Processes {
		pidMap[snap.Processes[i].PID] = snap.Processes[i].Name
	}
	m.mu.Lock()
	m.latestPID = pidMap
	m.mu.Unlock()

	// Carry forward the previously computed tree/ports if available.
	if prev := m.store.Latest(); prev != nil {
		snap.ProcessTree = prev.ProcessTree
		snap.PortBindings = prev.PortBindings
	}

	m.store.SetLatest(snap)
	if m.emitter != nil {
		m.emitter.Emit(EventSnapshot, snap)
	}
	return snap
}

func (m *Manager) treeLoop(ctx context.Context) {
	timer := time.NewTimer(m.treeInterval())
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			snap := m.store.Latest()
			if snap == nil {
				timer.Reset(m.treeInterval())
				continue
			}
			tree := BuildProcessTree(snap.Processes)
			m.store.UpdateLatest(func(latest *metrics.SystemSnapshot) {
				latest.ProcessTree = tree
			})
			if m.emitter != nil {
				m.emitter.Emit(EventProcessTree, tree)
			}
			timer.Reset(m.treeInterval())
		}
	}
}

func (m *Manager) portsLoop(ctx context.Context) {
	timer := time.NewTimer(m.portsInterval())
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			bindings := m.ports.Collect(m.lookupPID)
			snap := m.store.Latest()
			if snap == nil {
				timer.Reset(m.portsInterval())
				continue
			}
			m.store.UpdateLatest(func(latest *metrics.SystemSnapshot) {
				latest.PortBindings = bindings
			})
			if m.emitter != nil {
				m.emitter.Emit(EventPortBindings, bindings)
			}
			timer.Reset(m.portsInterval())
		}
	}
}

func (m *Manager) lookupPID(pid uint32) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.latestPID[pid]
}

// ApplyConfig updates collectors that have config-derived state.
func (m *Manager) ApplyConfig(cfg *config.Config) {
	m.mu.Lock()
	m.cfg = cfg
	m.mu.Unlock()
	m.ports.SetWellKnown(cfg.WellKnownPorts)
}

func (m *Manager) currentConfig() *config.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func (m *Manager) fastInterval() time.Duration {
	cfg := m.currentConfig()
	interval := time.Second
	if cfg != nil {
		interval = cfg.Monitoring.Interval
	}
	if interval < 200*time.Millisecond {
		return time.Second
	}
	return interval
}

func (m *Manager) treeInterval() time.Duration {
	cfg := m.currentConfig()
	interval := 2 * time.Second
	if cfg != nil {
		interval = cfg.Monitoring.ProcessTreeInterval
	}
	if interval < 500*time.Millisecond {
		return 2 * time.Second
	}
	return interval
}

func (m *Manager) portsInterval() time.Duration {
	cfg := m.currentConfig()
	interval := 3 * time.Second
	if cfg != nil {
		interval = cfg.Monitoring.PortScanInterval
	}
	if interval < 500*time.Millisecond {
		return 3 * time.Second
	}
	return interval
}

// CPUInfoFromRegistry reads the processor name and base MHz from registry.
func CPUInfoFromRegistry() (string, uint32) {
	const path = `HARDWARE\DESCRIPTION\System\CentralProcessor\0`
	name, _ := winapi.RegReadString(winapi.HKEY_LOCAL_MACHINE, path, "ProcessorNameString")
	mhz, _ := winapi.RegReadDWORD(winapi.HKEY_LOCAL_MACHINE, path, "~MHz")
	return name, mhz
}
