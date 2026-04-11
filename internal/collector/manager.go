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
	interval := m.cfg.Monitoring.Interval
	if interval < 200*time.Millisecond {
		interval = 1000 * time.Millisecond
	}
	t := time.NewTicker(interval)
	defer t.Stop()

	pruneT := time.NewTicker(30 * time.Second)
	defer pruneT.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.fastSample()
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
	interval := m.cfg.Monitoring.ProcessTreeInterval
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
			snap := m.store.Latest()
			if snap == nil {
				continue
			}
			tree := BuildProcessTree(snap.Processes)
			snap.ProcessTree = tree
			m.store.SetLatest(snap)
			if m.emitter != nil {
				m.emitter.Emit(EventProcessTree, tree)
			}
		}
	}
}

func (m *Manager) portsLoop(ctx context.Context) {
	interval := m.cfg.Monitoring.PortScanInterval
	if interval < 500*time.Millisecond {
		interval = 3 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			bindings := m.ports.Collect(m.lookupPID)
			snap := m.store.Latest()
			if snap == nil {
				continue
			}
			snap.PortBindings = bindings
			m.store.SetLatest(snap)
			if m.emitter != nil {
				m.emitter.Emit(EventPortBindings, bindings)
			}
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
	m.cfg = cfg
	m.ports.SetWellKnown(cfg.WellKnownPorts)
}

// CPUInfoFromRegistry reads the processor name and base MHz from registry.
func CPUInfoFromRegistry() (string, uint32) {
	const path = `HARDWARE\DESCRIPTION\System\CentralProcessor\0`
	name, _ := winapi.RegReadString(winapi.HKEY_LOCAL_MACHINE, path, "ProcessorNameString")
	mhz, _ := winapi.RegReadDWORD(winapi.HKEY_LOCAL_MACHINE, path, "~MHz")
	return name, mhz
}
