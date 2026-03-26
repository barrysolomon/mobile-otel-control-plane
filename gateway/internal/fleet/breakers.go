package fleet

import (
	"sync"
	"time"
)

// BreakerState tracks circuit breaker evaluation results.
type BreakerState struct {
	mu         sync.RWMutex
	killSwitch bool
	cooldowns  map[string]time.Time
}

// NewBreakerState creates a new circuit breaker state.
func NewBreakerState() *BreakerState {
	return &BreakerState{
		cooldowns: make(map[string]time.Time),
	}
}

// BreakerResult is the outcome of a circuit breaker check.
type BreakerResult struct {
	Passed bool   `json:"passed"`
	Reason string `json:"reason,omitempty"`
}

// CheckAll evaluates all circuit breakers for a potential cascade.
func (bs *BreakerState) CheckAll(cfg CircuitBreakerConfig, budget *BudgetManager, chainID string, targetDevices int, currentDepth int, triggerType, cohortID string) BreakerResult {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	if bs.killSwitch {
		return BreakerResult{Passed: false, Reason: "kill_switch_engaged"}
	}

	if currentDepth >= cfg.MaxCascadeDepth {
		return BreakerResult{Passed: false, Reason: "max_cascade_depth_reached"}
	}

	key := triggerType + ":" + cohortID
	if expires, ok := bs.cooldowns[key]; ok {
		if time.Now().Before(expires) {
			return BreakerResult{Passed: false, Reason: "cooldown_active"}
		}
	}

	ok, reason := budget.Reserve(chainID, targetDevices)
	if !ok {
		return BreakerResult{Passed: false, Reason: reason}
	}

	return BreakerResult{Passed: true}
}

// SetCooldown activates a cooldown for the given trigger+cohort pair.
func (bs *BreakerState) SetCooldown(triggerType, cohortID string, duration time.Duration) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	key := triggerType + ":" + cohortID
	bs.cooldowns[key] = time.Now().Add(duration)
}

// EngageKillSwitch halts all cascades.
func (bs *BreakerState) EngageKillSwitch() {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.killSwitch = true
}

// DisengageKillSwitch resumes cascade processing.
func (bs *BreakerState) DisengageKillSwitch() {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.killSwitch = false
}

// IsKillSwitchEngaged returns the kill switch state.
func (bs *BreakerState) IsKillSwitchEngaged() bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.killSwitch
}
