package db

import (
	"os"
	"testing"
	"time"
)

func TestAuditDB_CascadeChainLifecycle(t *testing.T) {
	path := t.TempDir() + "/test_audit.db"
	defer os.Remove(path)

	adb, err := NewAuditDB(path)
	if err != nil {
		t.Fatalf("NewAuditDB: %v", err)
	}
	defer adb.Close()

	now := time.Now()

	// Insert chain
	err = adb.InsertCascadeChain(CascadeChainRow{
		ChainID:         "chain-1",
		RootTriggerType: "crash_marker",
		RootCohortID:    "cohort-1",
		StartedAt:       now,
		Status:          "active",
		ChainTimeoutMin: 60,
		HopTimeoutMin:   10,
		LastHopAt:       now,
		WorkflowID:      "wf-1",
	})
	if err != nil {
		t.Fatalf("InsertCascadeChain: %v", err)
	}

	// Get active chains
	chains, err := adb.GetActiveChains()
	if err != nil {
		t.Fatalf("GetActiveChains: %v", err)
	}
	if len(chains) != 1 {
		t.Fatalf("expected 1 active chain, got %d", len(chains))
	}
	if chains[0].ChainID != "chain-1" {
		t.Errorf("expected chain-1, got %s", chains[0].ChainID)
	}

	// Complete chain
	endedAt := now.Add(5 * time.Minute)
	err = adb.UpdateCascadeChainStatus("chain-1", "completed", &endedAt)
	if err != nil {
		t.Fatalf("UpdateCascadeChainStatus: %v", err)
	}

	// No more active chains
	chains, _ = adb.GetActiveChains()
	if len(chains) != 0 {
		t.Errorf("expected 0 active chains, got %d", len(chains))
	}
}

func TestAuditDB_WorkflowAuditTrail(t *testing.T) {
	path := t.TempDir() + "/test_audit2.db"
	defer os.Remove(path)

	adb, err := NewAuditDB(path)
	if err != nil {
		t.Fatalf("NewAuditDB: %v", err)
	}
	defer adb.Close()

	err = adb.InsertWorkflowAudit(WorkflowAuditRow{
		WorkflowID:  "wf-1",
		Action:      "published",
		Actor:       "barry@dash0.com",
		NewJSON:     `{"type":"fleet"}`,
		DiffSummary: "initial publish",
	})
	if err != nil {
		t.Fatalf("InsertWorkflowAudit: %v", err)
	}

	entries, err := adb.ListWorkflowAudit("wf-1")
	if err != nil {
		t.Fatalf("ListWorkflowAudit: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Actor != "barry@dash0.com" {
		t.Errorf("expected barry@dash0.com, got %s", entries[0].Actor)
	}
}

func TestAuditDB_DeviceTargetingDedup(t *testing.T) {
	path := t.TempDir() + "/test_audit3.db"
	defer os.Remove(path)

	adb, err := NewAuditDB(path)
	if err != nil {
		t.Fatalf("NewAuditDB: %v", err)
	}
	defer adb.Close()

	now := time.Now()

	adb.InsertCascadeChain(CascadeChainRow{
		ChainID: "chain-1", RootTriggerType: "crash", RootCohortID: "c-1",
		StartedAt: now, Status: "active", ChainTimeoutMin: 60, HopTimeoutMin: 10, LastHopAt: now, WorkflowID: "wf-1",
	})

	deliveredAt := now
	adb.InsertCascadeDeviceLog(CascadeDeviceRow{
		ChainID: "chain-1", HopID: "hop-1", DeviceID: "dev-1",
		DeliveryChannel: "websocket", DeliveredAt: &deliveredAt,
	})
	adb.InsertCascadeDeviceLog(CascadeDeviceRow{
		ChainID: "chain-1", HopID: "hop-1", DeviceID: "dev-2",
		DeliveryChannel: "poll", DeliveredAt: &deliveredAt,
	})

	targeted, err := adb.GetTargetedDevices("chain-1")
	if err != nil {
		t.Fatalf("GetTargetedDevices: %v", err)
	}
	if len(targeted) != 2 {
		t.Errorf("expected 2 targeted devices, got %d", len(targeted))
	}
}
