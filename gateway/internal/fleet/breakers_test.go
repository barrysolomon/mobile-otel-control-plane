package fleet

import (
	"testing"
	"time"
)

func TestBreaker_AllPass(t *testing.T) {
	bs := NewBreakerState()
	bm := NewBudgetManager(25.0, 10000, 1000)
	cfg := CircuitBreakerConfig{MaxCascadeDepth: 3, CooldownMinutes: 15}

	result := bs.CheckAll(cfg, bm, "chain-1", 100, 0, "crash", "cohort-1")
	if !result.Passed {
		t.Errorf("All breakers should pass, got: %s", result.Reason)
	}
}

func TestBreaker_KillSwitch(t *testing.T) {
	bs := NewBreakerState()
	bm := NewBudgetManager(100.0, 100000, 1000)
	cfg := CircuitBreakerConfig{MaxCascadeDepth: 3}

	bs.EngageKillSwitch()
	result := bs.CheckAll(cfg, bm, "chain-1", 10, 0, "crash", "cohort-1")
	if result.Passed {
		t.Error("Kill switch should block")
	}

	bs.DisengageKillSwitch()
	result = bs.CheckAll(cfg, bm, "chain-2", 10, 0, "crash", "cohort-1")
	if !result.Passed {
		t.Errorf("After disengage, should pass: %s", result.Reason)
	}
}

func TestBreaker_DepthLimit(t *testing.T) {
	bs := NewBreakerState()
	bm := NewBudgetManager(100.0, 100000, 1000)
	cfg := CircuitBreakerConfig{MaxCascadeDepth: 3}

	result := bs.CheckAll(cfg, bm, "chain-1", 10, 3, "crash", "cohort-1")
	if result.Passed {
		t.Error("Depth 3 at max_depth=3 should block")
	}
}

func TestBreaker_CooldownPreventsRefire(t *testing.T) {
	bs := NewBreakerState()
	bm := NewBudgetManager(100.0, 100000, 1000)
	cfg := CircuitBreakerConfig{MaxCascadeDepth: 3}

	bs.SetCooldown("crash", "cohort-1", 15*time.Minute)

	result := bs.CheckAll(cfg, bm, "chain-1", 10, 0, "crash", "cohort-1")
	if result.Passed {
		t.Error("Cooldown should block")
	}
}
