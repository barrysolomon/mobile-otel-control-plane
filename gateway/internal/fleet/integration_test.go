package fleet_test

import (
	"testing"
	"time"

	"github.com/mobile-observability/gateway/internal/db"
	"github.com/mobile-observability/gateway/internal/fleet"
	"github.com/mobile-observability/gateway/internal/push"
	"github.com/mobile-observability/gateway/internal/security"
)

func TestIntegration_CrashTriggersFleetAlert(t *testing.T) {
	// Setup split DBs
	tmpDir := t.TempDir()
	fleetDB, err := db.NewFleetDB(tmpDir + "/fleet.db")
	if err != nil {
		t.Fatalf("NewFleetDB: %v", err)
	}
	defer fleetDB.Close()
	auditDB, err := db.NewAuditDB(tmpDir + "/audit.db")
	if err != nil {
		t.Fatalf("NewAuditDB: %v", err)
	}
	defer auditDB.Close()

	// Setup fleet components
	engine := fleet.NewFleetRuleEngine(5 * time.Minute)
	engine.AddRule(fleet.FleetRule{
		ID: "rule-1", RuleType: "fleet_threshold", CohortID: "pixel7",
		ConfigJSON: `{"trigger_type":"crash_marker","threshold":3,"window_minutes":5}`,
		Enabled: true, Priority: fleet.PriorityCritical,
	})

	breakerState := fleet.NewBreakerState()
	budget := fleet.NewBudgetManager(25.0, 10000, 100)
	dedup := fleet.NewEventDeduplicator(6*time.Minute, 1000)
	wsRegistry := push.NewConnectionRegistry()
	broker := push.NewBroker(wsRegistry)
	secret := []byte("test-secret")

	now := time.Now()

	// Simulate 3 crashes from different devices
	events := []fleet.FleetEvent{
		{ID: fleet.GenerateEventID("dev-1", "crash_marker", now, "s1"), DeviceID: "dev-1", CohortID: "pixel7", TriggerType: "crash_marker", Timestamp: now},
		{ID: fleet.GenerateEventID("dev-2", "crash_marker", now, "s2"), DeviceID: "dev-2", CohortID: "pixel7", TriggerType: "crash_marker", Timestamp: now},
		{ID: fleet.GenerateEventID("dev-3", "crash_marker", now, "s3"), DeviceID: "dev-3", CohortID: "pixel7", TriggerType: "crash_marker", Timestamp: now},
	}

	var firedRules []fleet.FleetRule
	for _, event := range events {
		if dedup.IsDuplicate(event.ID) {
			t.Error("Events should not be duplicates")
		}
		fired := engine.OnFleetEvent(event)
		firedRules = append(firedRules, fired...)
	}

	if len(firedRules) == 0 {
		t.Fatal("Expected rule to fire after 3 crashes")
	}
	if firedRules[0].ID != "rule-1" {
		t.Errorf("Expected rule-1, got %s", firedRules[0].ID)
	}

	// Check breakers
	cfg := fleet.CircuitBreakerConfig{MaxCascadeDepth: 3, CooldownMinutes: 15}
	result := breakerState.CheckAll(cfg, budget, "chain-1", 10, 0, "crash_marker", "pixel7")
	if !result.Passed {
		t.Errorf("Breakers should pass: %s", result.Reason)
	}

	// Build and sign alert
	alert := fleet.FleetAlert{
		Type:           "fleet_alert",
		AlertID:        "fa-001",
		CascadeChainID: "chain-1",
		Priority:       fleet.PriorityCritical,
		SourceTrigger:  "crash_marker",
		SourceCohort:   "pixel7",
		Actions:        []fleet.AlertAction{{Type: "flush_buffer", Config: map[string]interface{}{"minutes": 5}}},
		ExpiresAt:      now.Add(10 * time.Minute),
		IssuedAt:       now,
	}
	alert.Signature = security.Sign(alert.AlertID, secret)

	// Verify signature
	if !security.Verify(alert.AlertID, alert.Signature, secret) {
		t.Error("Signature should verify")
	}

	// Deliver (no actual WS connections in test — all will be poll)
	targetDevices := []string{"dev-4", "dev-5", "dev-6"}
	results := broker.Deliver(alert, targetDevices)
	if len(results) != 3 {
		t.Errorf("Expected 3 delivery results, got %d", len(results))
	}
	for _, r := range results {
		if r.Channel != "poll" {
			t.Errorf("Expected poll channel (no WS in test), got %s", r.Channel)
		}
	}

	// Record in audit DB
	auditErr := auditDB.InsertCascadeChain(db.CascadeChainRow{
		ChainID: "chain-1", RootTriggerType: "crash_marker", RootCohortID: "pixel7",
		StartedAt: now, Status: "active", ChainTimeoutMin: 60, HopTimeoutMin: 10,
		LastHopAt: now, WorkflowID: "wf-1",
	})
	if auditErr != nil {
		t.Fatalf("InsertCascadeChain: %v", auditErr)
	}

	chains, _ := auditDB.GetActiveChains()
	if len(chains) != 1 {
		t.Errorf("Expected 1 active chain, got %d", len(chains))
	}
}
