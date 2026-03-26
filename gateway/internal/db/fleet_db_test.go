package db

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestFleetDB_InsertAndDelete(t *testing.T) {
	path := t.TempDir() + "/test_fleet.db"
	defer os.Remove(path)

	fdb, err := NewFleetDB(path)
	if err != nil {
		t.Fatalf("NewFleetDB: %v", err)
	}
	defer fdb.Close()

	now := time.Now()
	err = fdb.InsertFleetEvent("evt-1", "dev-1", "cohort-1", "crash_marker", "{}", now)
	if err != nil {
		t.Fatalf("InsertFleetEvent: %v", err)
	}

	// Duplicate insert should not error (INSERT OR IGNORE)
	err = fdb.InsertFleetEvent("evt-1", "dev-1", "cohort-1", "crash_marker", "{}", now)
	if err != nil {
		t.Fatalf("Duplicate InsertFleetEvent: %v", err)
	}

	// Delete older than future should delete everything
	deleted, err := fdb.DeleteFleetEventsOlderThan(now.Add(1 * time.Hour))
	if err != nil {
		t.Fatalf("DeleteFleetEventsOlderThan: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
}

func TestFleetDB_BatchInsert(t *testing.T) {
	path := t.TempDir() + "/test_fleet_batch.db"
	defer os.Remove(path)

	fdb, err := NewFleetDB(path)
	if err != nil {
		t.Fatalf("NewFleetDB: %v", err)
	}
	defer fdb.Close()

	now := time.Now()
	events := make([]FleetEventRow, 100)
	for i := range events {
		events[i] = FleetEventRow{
			ID:          fmt.Sprintf("evt-%d", i),
			DeviceID:    fmt.Sprintf("dev-%d", i%10),
			CohortID:    "cohort-1",
			TriggerType: "crash_marker",
			Timestamp:   now,
		}
	}

	err = fdb.InsertFleetEventBatch(events)
	if err != nil {
		t.Fatalf("InsertFleetEventBatch: %v", err)
	}

	deleted, _ := fdb.DeleteFleetEventsOlderThan(now.Add(1 * time.Hour))
	if deleted != 100 {
		t.Errorf("expected 100 deleted, got %d", deleted)
	}
}
