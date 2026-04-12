package anomaly

import (
	"sync"
	"time"
)

// Severity describes the impact level of an alert.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Alert is a single anomaly detection event.
type Alert struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Severity    Severity       `json:"severity"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	PID         uint32         `json:"pid,omitempty"`
	ProcessName string         `json:"process_name,omitempty"`
	Detected    time.Time      `json:"detected"`
	Cleared     *time.Time     `json:"cleared,omitempty"`
	Action      string         `json:"action,omitempty"`
	Details     map[string]any `json:"details,omitempty"`
}

// AlertStore keeps active and historical alerts in a thread-safe ring.
type AlertStore struct {
	mu         sync.RWMutex
	active     map[string]*Alert
	history    []Alert
	snoozed    map[string]time.Time
	maxHistory int
	maxActive  int
	nextID     uint64
}

func NewAlertStore(maxHistory int) *AlertStore {
	if maxHistory < 32 {
		maxHistory = 32
	}
	return &AlertStore{
		active:     make(map[string]*Alert),
		history:    make([]Alert, 0, maxHistory),
		snoozed:    make(map[string]time.Time),
		maxHistory: maxHistory,
		maxActive:  200,
	}
}

func alertKey(typeName string, pid uint32) string {
	key := typeName
	if pid > 0 {
		key = typeName + "/" + uint32ToA(pid)
	}
	return key
}

// SetMaxActive updates the active-alert cap. Called when config is (re)loaded
// so the user can tune how much alert noise the UI is allowed to hold. A value
// <= 0 disables the cap.
func (s *AlertStore) SetMaxActive(n int) {
	s.mu.Lock()
	s.maxActive = n
	s.mu.Unlock()
}

func (s *AlertStore) Raise(a Alert) (Alert, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := alertKey(a.Type, a.PID)
	if until, ok := s.snoozed[key]; ok {
		if time.Now().Before(until) {
			return a, false
		}
		delete(s.snoozed, key)
	}
	if existing, ok := s.active[key]; ok {
		// Already raised; refresh details but do not return as new.
		existing.Description = a.Description
		existing.Severity = a.Severity
		existing.Details = a.Details
		return *existing, false
	}
	// Safety cap — when the active set is full, drop new raises on the floor
	// (still appended to history so the user sees them in the log). Without
	// this the UI can accumulate tens of thousands of entries from noisy
	// detectors on real Windows systems.
	if s.maxActive > 0 && len(s.active) >= s.maxActive {
		s.nextID++
		a.ID = key
		a.Detected = time.Now()
		s.appendHistory(a)
		return a, false
	}
	s.nextID++
	a.ID = key
	a.Detected = time.Now()
	cp := a
	s.active[key] = &cp
	s.appendHistory(cp)
	return cp, true
}

// ClearByType removes every active alert whose Type matches typeName,
// regardless of PID. Called when a detector gets disabled at runtime so the
// UI isn't left with stale entries that will never be re-raised or cleared.
// Returns the number of alerts removed.
func (s *AlertStore) ClearByType(typeName string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	removed := 0
	for key, a := range s.active {
		if a.Type == typeName {
			a.Cleared = &now
			delete(s.active, key)
			removed++
		}
	}
	return removed
}

// ClearAll empties the active alert set. Used by the "clear all" UI button
// to dig the user out of an alert flood without restarting the process.
// Returns the number of alerts removed.
func (s *AlertStore) ClearAll() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	removed := len(s.active)
	for key, a := range s.active {
		a.Cleared = &now
		delete(s.active, key)
	}
	return removed
}

// Clear marks an active alert as cleared.
func (s *AlertStore) Clear(typeName string, pid uint32) {
	s.ClearByKey(alertKey(typeName, pid))
}

// ClearByKey removes a specific active alert by its internal ID.
func (s *AlertStore) ClearByKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.active[key]; ok {
		now := time.Now()
		existing.Cleared = &now
		delete(s.active, key)
	}
}

// Snooze suppresses raises for one alert key until `until` and clears any
// currently active alert for the same key.
func (s *AlertStore) Snooze(typeName string, pid uint32, until time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := alertKey(typeName, pid)
	s.snoozed[key] = until
	if existing, ok := s.active[key]; ok {
		existing.Cleared = &until
		delete(s.active, key)
	}
	return true
}

// Active returns a copy of the currently raised alerts.
func (s *AlertStore) Active() []Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Alert, 0, len(s.active))
	for _, a := range s.active {
		out = append(out, *a)
	}
	return out
}

// History returns a copy of the alert history ring.
func (s *AlertStore) History() []Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Alert, len(s.history))
	copy(out, s.history)
	return out
}

func (s *AlertStore) appendHistory(a Alert) {
	if len(s.history) >= s.maxHistory {
		copy(s.history, s.history[1:])
		s.history = s.history[:len(s.history)-1]
	}
	s.history = append(s.history, a)
}

func uint32ToA(n uint32) string {
	if n == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
