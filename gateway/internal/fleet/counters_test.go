package fleet

import (
	"testing"
	"time"
)

func TestCounter_Add_IncreasesCount(t *testing.T) {
	c := NewSlidingWindow(5 * time.Minute)
	now := time.Now()

	c.Add("dev-1", now)
	c.Add("dev-2", now.Add(1*time.Second))

	if c.CountDistinctDevices(now.Add(2*time.Second)) != 2 {
		t.Error("Expected 2 distinct devices")
	}
}

func TestCounter_SameDevice_CountedOnce(t *testing.T) {
	c := NewSlidingWindow(5 * time.Minute)
	now := time.Now()

	c.Add("dev-1", now)
	c.Add("dev-1", now.Add(1*time.Second))
	c.Add("dev-1", now.Add(2*time.Second))

	if c.CountDistinctDevices(now.Add(3*time.Second)) != 1 {
		t.Errorf("Same device should count as 1, got %d", c.CountDistinctDevices(now.Add(3*time.Second)))
	}
}

func TestCounter_WindowExpiry(t *testing.T) {
	c := NewSlidingWindow(2 * time.Minute)
	base := time.Now()

	c.Add("dev-1", base)
	c.Add("dev-2", base.Add(1*time.Minute))

	// At base+3min, dev-1's event is outside 2min window
	at := base.Add(3 * time.Minute)
	if count := c.CountDistinctDevices(at); count != 1 {
		t.Errorf("Expected 1 device in window, got %d", count)
	}
}

func TestCounter_WindowBoundary(t *testing.T) {
	c := NewSlidingWindow(2 * time.Minute)
	base := time.Now()

	c.Add("dev-1", base)

	// At exactly base+2min, event at base is at boundary — include it
	at := base.Add(2 * time.Minute)
	count := c.CountDistinctDevices(at)
	if count != 1 {
		t.Errorf("Event at exact boundary should be included, got %d", count)
	}
}

func TestCounter_TotalEventCount(t *testing.T) {
	c := NewSlidingWindow(5 * time.Minute)
	now := time.Now()

	c.Add("dev-1", now)
	c.Add("dev-1", now.Add(1*time.Second))
	c.Add("dev-2", now.Add(2*time.Second))

	if c.TotalEvents(now.Add(3*time.Second)) != 3 {
		t.Error("Total events should be 3")
	}
}

func TestCounter_Trim_RemovesOldEvents(t *testing.T) {
	c := NewSlidingWindow(1 * time.Minute)
	base := time.Now()

	for i := 0; i < 100; i++ {
		c.Add("dev-1", base.Add(time.Duration(i)*time.Second))
	}

	c.Trim(base.Add(2 * time.Minute))
	if c.TotalEvents(base.Add(2*time.Minute)) > 60 {
		t.Error("Trim should remove events older than window")
	}
}
