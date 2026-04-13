package storage

import (
	"sync"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
	"github.com/ersinkoc/WindowsTaskManager/internal/stats"
)

// ProcessSample is a single point in a process's history.
type ProcessSample struct {
	Time         time.Time `json:"time"`
	CPUPercent   float64   `json:"cpu_percent"`
	WorkingSet   uint64    `json:"working_set"`
	PrivateBytes uint64    `json:"private_bytes"`
	IOReadBytes  uint64    `json:"io_read_bytes"`
	IOWriteBytes uint64    `json:"io_write_bytes"`
	ThreadCount  uint32    `json:"thread_count"`
}

// Store holds the latest snapshot plus historical ring buffers for system
// metrics and per-process samples.
type Store struct {
	mu sync.RWMutex

	latest *metrics.SystemSnapshot

	systemHistory *stats.RingBuffer[metrics.TimestampedSystem]

	procHistory map[uint32]*stats.RingBuffer[ProcessSample]
	procCap     int
	procSeen    map[uint32]time.Time
}

// NewStore allocates a store sized for `historyCap` system snapshots and
// `procHistoryCap` samples per process.
func NewStore(historyCap, procHistoryCap int) *Store {
	if historyCap < 60 {
		historyCap = 60
	}
	if procHistoryCap < 10 {
		procHistoryCap = 10
	}
	return &Store{
		systemHistory: stats.NewRingBuffer[metrics.TimestampedSystem](historyCap),
		procHistory:   make(map[uint32]*stats.RingBuffer[ProcessSample]),
		procCap:       procHistoryCap,
		procSeen:      make(map[uint32]time.Time),
	}
}

// SetLatest replaces the latest system snapshot and pushes a timestamped
// row into the system history ring.
func (s *Store) SetLatest(snap *metrics.SystemSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setLatestLocked(cloneSnapshot(snap))
	s.recordSnapshotLocked(snap)
}

// UpdateLatest mutates the latest snapshot in place without appending a new
// history row. Use this for enriching a previously sampled snapshot with
// derived data such as process trees or port bindings.
func (s *Store) UpdateLatest(update func(*metrics.SystemSnapshot)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.latest == nil {
		return false
	}
	update(s.latest)
	return true
}

func (s *Store) setLatestLocked(snap *metrics.SystemSnapshot) {
	s.latest = snap
}

func (s *Store) recordSnapshotLocked(snap *metrics.SystemSnapshot) {
	s.systemHistory.Add(metrics.TimestampedSystem{
		Time:    snap.Timestamp,
		CPU:     snap.CPU,
		Memory:  snap.Memory,
		GPU:     snap.GPU,
		Network: snap.Network,
		Disk:    snap.Disk,
	})

	now := snap.Timestamp
	for i := range snap.Processes {
		p := &snap.Processes[i]
		buf, ok := s.procHistory[p.PID]
		if !ok {
			buf = stats.NewRingBuffer[ProcessSample](s.procCap)
			s.procHistory[p.PID] = buf
		}
		buf.Add(ProcessSample{
			Time:         now,
			CPUPercent:   p.CPUPercent,
			WorkingSet:   p.WorkingSet,
			PrivateBytes: p.PrivateBytes,
			IOReadBytes:  p.IOReadBytes,
			IOWriteBytes: p.IOWriteBytes,
			ThreadCount:  p.ThreadCount,
		})
		s.procSeen[p.PID] = now
	}
}

// Latest returns the most recent snapshot or nil if none has been recorded.
func (s *Store) Latest() *metrics.SystemSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSnapshot(s.latest)
}

// SystemHistory returns a copy of the system history ring buffer contents.
func (s *Store) SystemHistory() []metrics.TimestampedSystem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.systemHistory.Slice()
}

// SystemHistorySince returns history rows whose timestamp is after `since`.
func (s *Store) SystemHistorySince(since time.Time) []metrics.TimestampedSystem {
	all := s.SystemHistory()
	out := make([]metrics.TimestampedSystem, 0, len(all))
	for _, r := range all {
		if r.Time.After(since) {
			out = append(out, r)
		}
	}
	return out
}

// ProcessHistory returns the recorded samples for a single PID.
func (s *Store) ProcessHistory(pid uint32) []ProcessSample {
	s.mu.RLock()
	defer s.mu.RUnlock()
	buf, ok := s.procHistory[pid]
	if !ok {
		return nil
	}
	return buf.Slice()
}

// PruneStaleProcesses drops history for any PID not seen since `cutoff`.
func (s *Store) PruneStaleProcesses(cutoff time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	for pid, last := range s.procSeen {
		if last.Before(cutoff) {
			delete(s.procHistory, pid)
			delete(s.procSeen, pid)
			removed++
		}
	}
	return removed
}

// TrackedProcessCount returns how many PIDs currently have history buffers.
func (s *Store) TrackedProcessCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.procHistory)
}

func cloneSnapshot(src *metrics.SystemSnapshot) *metrics.SystemSnapshot {
	if src == nil {
		return nil
	}
	dst := *src
	dst.CPU.PerCore = append([]float64(nil), src.CPU.PerCore...)
	dst.Disk.Drives = append([]metrics.DriveInfo(nil), src.Disk.Drives...)
	dst.Network.Interfaces = append([]metrics.InterfaceInfo(nil), src.Network.Interfaces...)
	dst.Processes = append([]metrics.ProcessInfo(nil), src.Processes...)
	dst.PortBindings = append([]metrics.PortBinding(nil), src.PortBindings...)
	dst.ProcessTree = cloneProcessTree(src.ProcessTree)
	return &dst
}

func cloneProcessTree(nodes []*metrics.ProcessNode) []*metrics.ProcessNode {
	if len(nodes) == 0 {
		return nil
	}
	out := make([]*metrics.ProcessNode, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			out = append(out, nil)
			continue
		}
		cp := *node
		cp.Children = cloneProcessTree(node.Children)
		out = append(out, &cp)
	}
	return out
}
