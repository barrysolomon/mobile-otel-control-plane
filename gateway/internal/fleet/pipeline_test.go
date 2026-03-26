package fleet

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestPipeline_EventProcessed(t *testing.T) {
	var processed atomic.Int32
	handler := func(event FleetEvent) []FleetRule {
		processed.Add(1)
		return nil
	}

	p := NewFleetEventPipeline(100, 2, handler)
	p.Start()
	defer p.Stop()

	p.Ingest(FleetEvent{DeviceID: "dev-1", TriggerType: "crash", Timestamp: time.Now()})
	time.Sleep(50 * time.Millisecond)

	if processed.Load() != 1 {
		t.Errorf("Expected 1 processed, got %d", processed.Load())
	}
}

func TestPipeline_BurstAbsorbed(t *testing.T) {
	var processed atomic.Int32
	handler := func(event FleetEvent) []FleetRule {
		processed.Add(1)
		return nil
	}

	p := NewFleetEventPipeline(5000, 4, handler)
	p.Start()
	defer p.Stop()

	for i := 0; i < 1000; i++ {
		p.Ingest(FleetEvent{DeviceID: "dev-1", TriggerType: "crash", Timestamp: time.Now()})
	}

	time.Sleep(200 * time.Millisecond)
	if processed.Load() < 900 {
		t.Errorf("Expected most events processed, got %d", processed.Load())
	}
}

func TestPipeline_ChannelFull_ReturnsFalse(t *testing.T) {
	handler := func(event FleetEvent) []FleetRule {
		time.Sleep(1 * time.Second)
		return nil
	}

	p := NewFleetEventPipeline(5, 1, handler)
	p.Start()
	defer p.Stop()

	for i := 0; i < 5; i++ {
		p.Ingest(FleetEvent{DeviceID: "dev-1", TriggerType: "crash", Timestamp: time.Now()})
	}

	// Next should be shed
	ok := p.Ingest(FleetEvent{DeviceID: "dev-1", TriggerType: "crash", Timestamp: time.Now()})
	if ok {
		t.Error("Expected false (backpressure) when channel full")
	}
}
