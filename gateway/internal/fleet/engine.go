package fleet

import (
	"encoding/json"
	"sync"
	"time"
)

// FleetRuleEngine evaluates fleet-level rules against incoming events.
type FleetRuleEngine struct {
	mu       sync.RWMutex
	rules    []FleetRule
	counters *CounterRegistry
}

// NewFleetRuleEngine creates a fleet rule engine with the given default window.
func NewFleetRuleEngine(defaultWindow time.Duration) *FleetRuleEngine {
	return &FleetRuleEngine{
		rules:    make([]FleetRule, 0),
		counters: NewCounterRegistry(defaultWindow),
	}
}

// AddRule registers a fleet rule for evaluation.
func (e *FleetRuleEngine) AddRule(rule FleetRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append(e.rules, rule)
}

// ClearRules removes all rules (used on config republish).
func (e *FleetRuleEngine) ClearRules() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = e.rules[:0]
}

// OnFleetEvent processes a single event and returns any rules that fired.
func (e *FleetRuleEngine) OnFleetEvent(event FleetEvent) []FleetRule {
	window := e.counters.GetOrCreate(event.CohortID, event.TriggerType)
	window.Add(event.DeviceID, event.Timestamp)

	e.mu.RLock()
	defer e.mu.RUnlock()

	var fired []FleetRule
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		if e.evaluateRule(rule, event) {
			fired = append(fired, rule)
		}
	}
	return fired
}

func (e *FleetRuleEngine) evaluateRule(rule FleetRule, event FleetEvent) bool {
	switch rule.RuleType {
	case "fleet_threshold":
		return e.evaluateThreshold(rule, event)
	case "fleet_rate":
		return e.evaluateRate(rule, event)
	default:
		return false
	}
}

func (e *FleetRuleEngine) evaluateThreshold(rule FleetRule, event FleetEvent) bool {
	var cfg ThresholdConfig
	if err := json.Unmarshal([]byte(rule.ConfigJSON), &cfg); err != nil {
		return false
	}

	if rule.CohortID != event.CohortID {
		return false
	}
	if cfg.TriggerType != event.TriggerType {
		return false
	}

	window := e.counters.GetOrCreate(event.CohortID, event.TriggerType)
	distinctDevices := window.CountDistinctDevices(event.Timestamp)

	return distinctDevices >= cfg.Threshold
}

func (e *FleetRuleEngine) evaluateRate(rule FleetRule, event FleetEvent) bool {
	var cfg RateConfig
	if err := json.Unmarshal([]byte(rule.ConfigJSON), &cfg); err != nil {
		return false
	}

	if rule.CohortID != event.CohortID || cfg.TriggerType != event.TriggerType {
		return false
	}

	window := e.counters.GetOrCreate(event.CohortID, event.TriggerType)
	currentCount := window.TotalEvents(event.Timestamp)

	if currentCount == 0 {
		return false
	}

	return float64(currentCount) >= cfg.Factor
}

// GetRules returns a copy of all registered rules.
func (e *FleetRuleEngine) GetRules() []FleetRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	rules := make([]FleetRule, len(e.rules))
	copy(rules, e.rules)
	return rules
}
