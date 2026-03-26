package fleet

import (
	"sync"
	"time"
)

// TimestampedEvent records an event with its device source.
type TimestampedEvent struct {
	DeviceID  string
	Timestamp time.Time
}

// SlidingWindow is a thread-safe sliding window of events for a single (cohort, trigger) pair.
type SlidingWindow struct {
	mu       sync.Mutex
	events   []TimestampedEvent
	duration time.Duration
}

// NewSlidingWindow creates a sliding window with the given duration.
func NewSlidingWindow(duration time.Duration) *SlidingWindow {
	return &SlidingWindow{
		events:   make([]TimestampedEvent, 0, 64),
		duration: duration,
	}
}

// Add records an event in the window.
func (w *SlidingWindow) Add(deviceID string, ts time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.events = append(w.events, TimestampedEvent{DeviceID: deviceID, Timestamp: ts})
}

// CountDistinctDevices returns the number of unique devices with events in the window.
func (w *SlidingWindow) CountDistinctDevices(now time.Time) int {
	w.mu.Lock()
	defer w.mu.Unlock()

	cutoff := now.Add(-w.duration)
	seen := make(map[string]bool)
	for _, e := range w.events {
		if !e.Timestamp.Before(cutoff) && e.Timestamp.Before(now.Add(1*time.Second)) {
			seen[e.DeviceID] = true
		}
	}
	return len(seen)
}

// TotalEvents returns the total number of events (not deduplicated) in the window.
func (w *SlidingWindow) TotalEvents(now time.Time) int {
	w.mu.Lock()
	defer w.mu.Unlock()

	cutoff := now.Add(-w.duration)
	count := 0
	for _, e := range w.events {
		if !e.Timestamp.Before(cutoff) {
			count++
		}
	}
	return count
}

// Trim removes events that have fallen outside the window.
func (w *SlidingWindow) Trim(now time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	cutoff := now.Add(-w.duration)
	kept := w.events[:0]
	for _, e := range w.events {
		if !e.Timestamp.Before(cutoff) {
			kept = append(kept, e)
		}
	}
	w.events = kept
}

// CounterKey identifies a specific sliding window.
type CounterKey struct {
	CohortID    string
	TriggerType string
}

// CounterRegistry manages a set of sliding windows keyed by (cohort, trigger).
type CounterRegistry struct {
	mu       sync.RWMutex
	windows  map[CounterKey]*SlidingWindow
	duration time.Duration
}

// NewCounterRegistry creates a counter registry with the given default window duration.
func NewCounterRegistry(defaultDuration time.Duration) *CounterRegistry {
	return &CounterRegistry{
		windows:  make(map[CounterKey]*SlidingWindow),
		duration: defaultDuration,
	}
}

// GetOrCreate returns the sliding window for the given key, creating it if needed.
func (r *CounterRegistry) GetOrCreate(cohortID, triggerType string) *SlidingWindow {
	key := CounterKey{CohortID: cohortID, TriggerType: triggerType}

	r.mu.RLock()
	w, exists := r.windows[key]
	r.mu.RUnlock()
	if exists {
		return w
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if w, exists = r.windows[key]; exists {
		return w
	}
	w = NewSlidingWindow(r.duration)
	r.windows[key] = w
	return w
}

// TrimAll trims all windows.
func (r *CounterRegistry) TrimAll(now time.Time) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, w := range r.windows {
		w.Trim(now)
	}
}
