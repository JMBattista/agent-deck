package testutil

import (
	"os"
	"sort"
	"strconv"
	"testing"
	"time"
)

// PerfBudgetMultiplierEnv is the env var read by Budget. CI sets this to 2.0
// to absorb shared-runner variance; developers on slow laptops can bump it
// locally. Default is 1.0.
const PerfBudgetMultiplierEnv = "PERF_BUDGET_MULTIPLIER"

// Budget scales a base walltime by PERF_BUDGET_MULTIPLIER. Use it to wrap a
// hard-coded budget constant so the same TestPerf_* test can run on a fast
// dev box (multiplier=1.0) and a noisy CI runner (multiplier=2.0) without
// flake. Invalid or unset env returns base unchanged.
//
//	budget := testutil.Budget(t, 50*time.Millisecond)
//	if got := testutil.MedianOf(5, doWork); got > budget {
//	    t.Fatalf("regression: %v > budget %v", got, budget)
//	}
func Budget(t *testing.T, base time.Duration) time.Duration {
	t.Helper()
	raw := os.Getenv(PerfBudgetMultiplierEnv)
	if raw == "" {
		return base
	}
	mult, err := strconv.ParseFloat(raw, 64)
	if err != nil || mult <= 0 {
		t.Logf("ignoring invalid %s=%q (using 1.0)", PerfBudgetMultiplierEnv, raw)
		return base
	}
	return time.Duration(float64(base) * mult)
}

// MedianOf runs fn n+1 times, discards the first as warm-up, and returns the
// median of the remaining n samples. n must be >= 1; n=5 is the project
// default for TestPerf_* tests. A single outlier (GC pause, scheduler hiccup)
// will not fail the test because the median ignores it.
func MedianOf(n int, fn func()) time.Duration {
	if n < 1 {
		n = 1
	}
	// Warm-up: caches, package-level init in fn's transitive callees, JIT-ish
	// effects in Go's runtime. Discarded.
	fn()
	samples := make([]time.Duration, n)
	for i := 0; i < n; i++ {
		start := time.Now()
		fn()
		samples[i] = time.Since(start)
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	return samples[n/2]
}

// MedianTimedOp runs each iteration as setup() then op(), but only op() is
// included in the timing. Use when the timed primitive requires a fresh
// fixture per iteration (e.g. timing DeleteGroup needs 100 groups already
// present, which we don't want counted against the budget).
//
// Setup and op typically share state via closure capture in the caller.
// Pattern:
//
//	var tree *GroupTree
//	got := testutil.MedianTimedOp(5,
//	    func() { tree = buildPopulatedTree() },
//	    func() { tree.DeleteAll() },
//	)
func MedianTimedOp(n int, setup, op func()) time.Duration {
	if n < 1 {
		n = 1
	}
	// Warm-up: full setup + op cycle.
	setup()
	op()
	samples := make([]time.Duration, n)
	for i := 0; i < n; i++ {
		setup()
		start := time.Now()
		op()
		samples[i] = time.Since(start)
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	return samples[n/2]
}

// SkipIfShort skips the test when `go test -short` is in effect. TestPerf_*
// tests are expensive enough that contributors running quick unit-test loops
// shouldn't pay for them. CI always runs in long mode.
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping perf test in -short mode")
	}
}
