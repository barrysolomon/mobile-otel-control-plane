package fleet

import (
	"fmt"
	"testing"
	"time"
)

func TestDedup_SameEventTwice_CountedOnce(t *testing.T) {
	d := NewEventDeduplicator(6*time.Minute, 1000)

	if d.IsDuplicate("evt-1") {
		t.Error("First call should not be duplicate")
	}
	if !d.IsDuplicate("evt-1") {
		t.Error("Second call should be duplicate")
	}
}

func TestDedup_DifferentEvents_BothCounted(t *testing.T) {
	d := NewEventDeduplicator(6*time.Minute, 1000)

	if d.IsDuplicate("evt-1") {
		t.Error("evt-1 should not be duplicate")
	}
	if d.IsDuplicate("evt-2") {
		t.Error("evt-2 should not be duplicate")
	}
}

func TestDedup_EvictionAfterMaxSize(t *testing.T) {
	d := NewEventDeduplicator(1*time.Hour, 5)

	for i := 0; i < 10; i++ {
		d.IsDuplicate(fmt.Sprintf("evt-%d", i))
	}

	// After eviction, early events should no longer be tracked
	d.mu.Lock()
	size := len(d.seen)
	d.mu.Unlock()
	if size > 6 { // some slack for eviction timing
		t.Errorf("Expected <=6 entries after eviction, got %d", size)
	}
}

func TestDedup_DeterministicID(t *testing.T) {
	id1 := GenerateEventID("dev-1", "crash_marker", time.Unix(1711454400, 0), "span-abc")
	id2 := GenerateEventID("dev-1", "crash_marker", time.Unix(1711454400, 0), "span-abc")
	id3 := GenerateEventID("dev-1", "crash_marker", time.Unix(1711454400, 0), "span-def")

	if id1 != id2 {
		t.Error("Same inputs should produce same ID")
	}
	if id1 == id3 {
		t.Error("Different span IDs should produce different IDs")
	}
}
