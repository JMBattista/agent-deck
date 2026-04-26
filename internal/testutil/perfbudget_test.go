package testutil

import (
	"os"
	"testing"
	"time"
)

func TestBudget_DefaultPassesThrough(t *testing.T) {
	old, had := os.LookupEnv(PerfBudgetMultiplierEnv)
	os.Unsetenv(PerfBudgetMultiplierEnv)
	defer func() {
		if had {
			os.Setenv(PerfBudgetMultiplierEnv, old)
		}
	}()

	got := Budget(t, 100*time.Millisecond)
	if got != 100*time.Millisecond {
		t.Fatalf("Budget(100ms) without env = %v, want 100ms", got)
	}
}

func TestBudget_HonorsMultiplier(t *testing.T) {
	old, had := os.LookupEnv(PerfBudgetMultiplierEnv)
	os.Setenv(PerfBudgetMultiplierEnv, "2.5")
	defer func() {
		if had {
			os.Setenv(PerfBudgetMultiplierEnv, old)
		} else {
			os.Unsetenv(PerfBudgetMultiplierEnv)
		}
	}()

	got := Budget(t, 100*time.Millisecond)
	if got != 250*time.Millisecond {
		t.Fatalf("Budget(100ms) with multiplier=2.5 = %v, want 250ms", got)
	}
}

func TestBudget_InvalidMultiplierFallsBack(t *testing.T) {
	old, had := os.LookupEnv(PerfBudgetMultiplierEnv)
	os.Setenv(PerfBudgetMultiplierEnv, "garbage")
	defer func() {
		if had {
			os.Setenv(PerfBudgetMultiplierEnv, old)
		} else {
			os.Unsetenv(PerfBudgetMultiplierEnv)
		}
	}()

	got := Budget(t, 100*time.Millisecond)
	if got != 100*time.Millisecond {
		t.Fatalf("Budget with invalid multiplier = %v, want 100ms (fallback)", got)
	}
}

func TestMedianOf_DiscardsWarmup(t *testing.T) {
	calls := 0
	_ = MedianOf(3, func() { calls++ })
	if calls != 4 {
		t.Fatalf("MedianOf(3) ran fn %d times, want 4 (1 warm-up + 3 samples)", calls)
	}
}

func TestMedianTimedOp_ExcludesSetup(t *testing.T) {
	setupCalls, opCalls := 0, 0
	got := MedianTimedOp(3,
		func() {
			setupCalls++
			time.Sleep(20 * time.Millisecond)
		},
		func() {
			opCalls++
			time.Sleep(2 * time.Millisecond)
		},
	)
	if setupCalls != 4 || opCalls != 4 {
		t.Fatalf("MedianTimedOp(3) ran setup=%d op=%d, want both=4 (1 warm-up + 3 samples)", setupCalls, opCalls)
	}
	// The 20ms sleep is in setup and must NOT be in the timed median.
	// Op sleeps 2ms; allow generous upper bound for scheduler jitter.
	if got > 15*time.Millisecond {
		t.Fatalf("MedianTimedOp included setup time: got %v, want <15ms", got)
	}
}

func TestMedianOf_ReturnsMiddleSample(t *testing.T) {
	// Run a workload that takes slightly longer each call. With n=5, the
	// 3rd recorded sample (index 2 == n/2) should be the median.
	var i int
	got := MedianOf(5, func() {
		i++
		time.Sleep(time.Duration(i) * time.Millisecond)
	})
	// Lower bound: 3rd post-warmup call slept ~3ms. Upper bound is generous
	// to absorb scheduler jitter without being flaky.
	if got < 2*time.Millisecond || got > 50*time.Millisecond {
		t.Fatalf("MedianOf median = %v, want roughly 3ms", got)
	}
}
