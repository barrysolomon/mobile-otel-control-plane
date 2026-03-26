package fleet

import (
	"sync/atomic"
)

// EventHandler is the callback invoked for each fleet event.
type EventHandler func(event FleetEvent) []FleetRule

// FleetEventPipeline provides buffered, backpressure-aware event processing.
type FleetEventPipeline struct {
	ingestCh    chan FleetEvent
	workerCount int
	handler     EventHandler
	shedCount   atomic.Int64
	stopCh      chan struct{}
}

// NewFleetEventPipeline creates a pipeline with the given buffer size and worker count.
func NewFleetEventPipeline(bufferSize, workerCount int, handler EventHandler) *FleetEventPipeline {
	return &FleetEventPipeline{
		ingestCh:    make(chan FleetEvent, bufferSize),
		workerCount: workerCount,
		handler:     handler,
		stopCh:      make(chan struct{}),
	}
}

// Ingest submits an event to the pipeline. Returns false if the channel is full (backpressure).
func (p *FleetEventPipeline) Ingest(event FleetEvent) bool {
	select {
	case p.ingestCh <- event:
		return true
	default:
		p.shedCount.Add(1)
		return false
	}
}

// Start launches worker goroutines.
func (p *FleetEventPipeline) Start() {
	for i := 0; i < p.workerCount; i++ {
		go p.worker()
	}
}

// Stop shuts down the pipeline.
func (p *FleetEventPipeline) Stop() {
	close(p.stopCh)
}

func (p *FleetEventPipeline) worker() {
	for {
		select {
		case event := <-p.ingestCh:
			p.handler(event)
		case <-p.stopCh:
			return
		}
	}
}

// ShedCount returns the number of events shed due to backpressure.
func (p *FleetEventPipeline) ShedCount() int64 {
	return p.shedCount.Load()
}

// QueueDepth returns the current number of events waiting to be processed.
func (p *FleetEventPipeline) QueueDepth() int {
	return len(p.ingestCh)
}
