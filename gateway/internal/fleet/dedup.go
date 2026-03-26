package fleet

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// EventDeduplicator tracks seen event IDs to prevent duplicate processing.
type EventDeduplicator struct {
	seen    map[string]time.Time
	maxAge  time.Duration
	maxSize int
	mu      sync.Mutex
}

// NewEventDeduplicator creates a deduplicator with the given max age and size.
func NewEventDeduplicator(maxAge time.Duration, maxSize int) *EventDeduplicator {
	return &EventDeduplicator{
		seen:    make(map[string]time.Time),
		maxAge:  maxAge,
		maxSize: maxSize,
	}
}

// IsDuplicate returns true if this event ID has been seen before.
func (d *EventDeduplicator) IsDuplicate(eventID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.seen[eventID]; exists {
		return true
	}
	d.seen[eventID] = time.Now()

	if len(d.seen) > d.maxSize {
		d.evict()
	}
	return false
}

func (d *EventDeduplicator) evict() {
	// First pass: remove expired entries
	cutoff := time.Now().Add(-d.maxAge)
	for id, ts := range d.seen {
		if ts.Before(cutoff) {
			delete(d.seen, id)
		}
	}

	// Second pass: if still over capacity, remove oldest entries
	for len(d.seen) > d.maxSize {
		var oldestID string
		var oldestTime time.Time
		first := true
		for id, ts := range d.seen {
			if first || ts.Before(oldestTime) {
				oldestID = id
				oldestTime = ts
				first = false
			}
		}
		delete(d.seen, oldestID)
	}
}

// GenerateEventID creates a deterministic event ID from its components.
func GenerateEventID(deviceID, triggerType string, ts time.Time, spanID string) string {
	input := fmt.Sprintf("%s|%s|%d|%s", deviceID, triggerType, ts.Unix(), spanID)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:8]) // 16 hex chars
}
