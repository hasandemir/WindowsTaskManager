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
	s.latest = snap
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
	s.mu.Unlock()
}

// Latest returns the most recent snapshot or nil if none has been recorded.
func (s *Store) Latest() *metrics.SystemSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latest
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
