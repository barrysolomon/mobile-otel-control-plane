package fleet

import (
	"fmt"
	"sync"
	"testing"
)

func TestBudget_AtomicReservation_TwoConcurrentChains(t *testing.T) {
	bm := NewBudgetManager(25.0, 10000, 1000)

	ok1, _ := bm.Reserve("chain-1", 200)
	ok2, _ := bm.Reserve("chain-2", 100)

	if !ok1 {
		t.Error("First reservation (20%) should succeed")
	}
	if ok2 {
		t.Error("Second reservation (30% total) should fail at 25% limit")
	}
}

func TestBudget_SequentialReservation_ExactLimit(t *testing.T) {
	bm := NewBudgetManager(25.0, 10000, 1000)

	ok1, _ := bm.Reserve("chain-1", 150)
	ok2, _ := bm.Reserve("chain-2", 100)

	if !ok1 || !ok2 {
		t.Error("Both reservations should succeed (exactly 25%)")
	}
}

func TestBudget_AbsoluteLimit(t *testing.T) {
	bm := NewBudgetManager(100.0, 500, 10000)

	ok1, _ := bm.Reserve("chain-1", 500)
	if !ok1 {
		t.Error("500 devices at 500 limit should succeed")
	}

	ok2, _ := bm.Reserve("chain-2", 1)
	if ok2 {
		t.Error("501 total should fail at 500 absolute limit")
	}
}

func TestBudget_Release_FreesCapacity(t *testing.T) {
	bm := NewBudgetManager(25.0, 10000, 1000)

	bm.Reserve("chain-1", 200)
	bm.Release("chain-1")

	ok, _ := bm.Reserve("chain-2", 200)
	if !ok {
		t.Error("After release, capacity should be available")
	}
}

func TestBudget_ConcurrentGoroutines(t *testing.T) {
	bm := NewBudgetManager(50.0, 100000, 10000)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			chainID := fmt.Sprintf("chain-%d", i)
			bm.Reserve(chainID, 10)
		}(i)
	}
	wg.Wait()
}
