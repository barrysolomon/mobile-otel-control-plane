package fleet

import (
	"fmt"
	"sync"
	"time"
)

// BudgetManager tracks fleet-wide cascade budget with atomic reservations.
type BudgetManager struct {
	mu              sync.Mutex
	devicesAffected map[string]int
	windowStart     time.Time
	maxPercent      float64
	maxAbsolute     int
	fleetSize       int
}

// NewBudgetManager creates a budget manager with the given limits.
func NewBudgetManager(maxPercent float64, maxAbsolute, fleetSize int) *BudgetManager {
	return &BudgetManager{
		devicesAffected: make(map[string]int),
		windowStart:     time.Now(),
		maxPercent:      maxPercent,
		maxAbsolute:     maxAbsolute,
		fleetSize:       fleetSize,
	}
}

// Reserve atomically reserves budget for a cascade chain. Returns false if budget exceeded.
func (b *BudgetManager) Reserve(chainID string, requestedDevices int) (bool, string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	totalAffected := 0
	for _, count := range b.devicesAffected {
		totalAffected += count
	}

	newTotal := totalAffected + requestedDevices

	if b.fleetSize > 0 {
		percentUsed := float64(newTotal) / float64(b.fleetSize) * 100
		if percentUsed > b.maxPercent {
			return false, fmt.Sprintf("budget_percent: %.1f%% > %.1f%% limit", percentUsed, b.maxPercent)
		}
	}

	if newTotal > b.maxAbsolute {
		return false, fmt.Sprintf("budget_absolute: %d > %d limit", newTotal, b.maxAbsolute)
	}

	b.devicesAffected[chainID] += requestedDevices
	return true, ""
}

// Release frees the budget reserved by a chain.
func (b *BudgetManager) Release(chainID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.devicesAffected, chainID)
}

// CurrentPercent returns the current budget utilization percentage.
func (b *BudgetManager) CurrentPercent() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	total := 0
	for _, count := range b.devicesAffected {
		total += count
	}
	if b.fleetSize == 0 {
		return 0
	}
	return float64(total) / float64(b.fleetSize) * 100
}

// UpdateFleetSize updates the known fleet size (called periodically).
func (b *BudgetManager) UpdateFleetSize(size int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.fleetSize = size
}
