package fleet

import (
	"testing"
	"time"
)

func newTestEngine() *FleetRuleEngine {
	return NewFleetRuleEngine(5 * time.Minute)
}

func TestThresholdRule_ExactThreshold(t *testing.T) {
	engine := newTestEngine()
	engine.AddRule(FleetRule{
		ID: "rule-1", RuleType: "fleet_threshold", CohortID: "cohort-1",
		ConfigJSON: `{"trigger_type":"crash_marker","threshold":3,"window_minutes":5}`,
		Enabled: true,
	})

	now := time.Now()
	var fired []FleetRule

	engine.OnFleetEvent(FleetEvent{DeviceID: "dev-1", CohortID: "cohort-1", TriggerType: "crash_marker", Timestamp: now})
	engine.OnFleetEvent(FleetEvent{DeviceID: "dev-2", CohortID: "cohort-1", TriggerType: "crash_marker", Timestamp: now})

	fired = engine.OnFleetEvent(FleetEvent{DeviceID: "dev-3", CohortID: "cohort-1", TriggerType: "crash_marker", Timestamp: now})

	if len(fired) != 1 {
		t.Fatalf("Expected 1 rule to fire at threshold, got %d", len(fired))
	}
	if fired[0].ID != "rule-1" {
		t.Errorf("Expected rule-1, got %s", fired[0].ID)
	}
}

func TestThresholdRule_OneBelow(t *testing.T) {
	engine := newTestEngine()
	engine.AddRule(FleetRule{
		ID: "rule-1", RuleType: "fleet_threshold", CohortID: "cohort-1",
		ConfigJSON: `{"trigger_type":"crash_marker","threshold":3,"window_minutes":5}`,
		Enabled: true,
	})

	now := time.Now()
	fired := engine.OnFleetEvent(FleetEvent{DeviceID: "dev-1", CohortID: "cohort-1", TriggerType: "crash_marker", Timestamp: now})
	if len(fired) != 0 {
		t.Error("Should not fire at 1 device (threshold is 3)")
	}

	fired = engine.OnFleetEvent(FleetEvent{DeviceID: "dev-2", CohortID: "cohort-1", TriggerType: "crash_marker", Timestamp: now})
	if len(fired) != 0 {
		t.Error("Should not fire at 2 devices (threshold is 3)")
	}
}

func TestThresholdRule_SameDeviceDoesNotCountTwice(t *testing.T) {
	engine := newTestEngine()
	engine.AddRule(FleetRule{
		ID: "rule-1", RuleType: "fleet_threshold", CohortID: "cohort-1",
		ConfigJSON: `{"trigger_type":"crash_marker","threshold":3,"window_minutes":5}`,
		Enabled: true,
	})

	now := time.Now()
	for i := 0; i < 10; i++ {
		fired := engine.OnFleetEvent(FleetEvent{DeviceID: "dev-1", CohortID: "cohort-1", TriggerType: "crash_marker", Timestamp: now})
		if len(fired) != 0 {
			t.Error("Same device 10 times = 1 distinct device, threshold 3 should NOT fire")
		}
	}
}

func TestThresholdRule_MultipleCohortsIsolated(t *testing.T) {
	engine := newTestEngine()
	engine.AddRule(FleetRule{
		ID: "rule-1", RuleType: "fleet_threshold", CohortID: "cohort-A",
		ConfigJSON: `{"trigger_type":"crash_marker","threshold":2,"window_minutes":5}`,
		Enabled: true,
	})

	now := time.Now()
	engine.OnFleetEvent(FleetEvent{DeviceID: "dev-1", CohortID: "cohort-A", TriggerType: "crash_marker", Timestamp: now})
	engine.OnFleetEvent(FleetEvent{DeviceID: "dev-2", CohortID: "cohort-B", TriggerType: "crash_marker", Timestamp: now})

	fired := engine.OnFleetEvent(FleetEvent{DeviceID: "dev-3", CohortID: "cohort-B", TriggerType: "crash_marker", Timestamp: now})
	if len(fired) != 0 {
		t.Error("Cohort B events should not affect cohort A rule")
	}
}

func TestDisabledRule_DoesNotFire(t *testing.T) {
	engine := newTestEngine()
	engine.AddRule(FleetRule{
		ID: "rule-1", RuleType: "fleet_threshold", CohortID: "cohort-1",
		ConfigJSON: `{"trigger_type":"crash_marker","threshold":1,"window_minutes":5}`,
		Enabled: false,
	})

	now := time.Now()
	fired := engine.OnFleetEvent(FleetEvent{DeviceID: "dev-1", CohortID: "cohort-1", TriggerType: "crash_marker", Timestamp: now})
	if len(fired) != 0 {
		t.Error("Disabled rule should not fire")
	}
}
