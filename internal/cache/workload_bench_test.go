package cache

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
)

// WorkloadConfig holds parameters for a reproducible benchmark workload.
type WorkloadConfig struct {
	Name           string
	Capacity       int
	WarmupOps      int
	MeasureOps     int
	KeySpace       int
	WorkingSetSize int
	Seed           int64
	ReadPct        int // For mixed workloads (e.g., 80 or 95)
}

// WorkloadStep represents an operation in a synthetic workload.
type WorkloadStep struct {
	IsWrite bool // true = Set, false = Get (with Set-on-miss)
	Key     string
	Value   string
}

// generateUniformSteps generates operations uniformly distributed across keySpace.
func generateUniformSteps(count int, keySpace int, seed int64) []WorkloadStep {
	rng := rand.New(rand.NewSource(seed))
	steps := make([]WorkloadStep, count)
	for i := 0; i < count; i++ {
		keyID := rng.Intn(keySpace)
		steps[i] = WorkloadStep{
			IsWrite: false,
			Key:     fmt.Sprintf("k-%d", keyID),
			Value:   "v",
		}
	}
	return steps
}

// generateZipfSteps generates operations following a Zipfian distribution (hot-key skew).
func generateZipfSteps(count int, keySpace int, s, v float64, seed int64) []WorkloadStep {
	rng := rand.New(rand.NewSource(seed))
	zipf := rand.NewZipf(rng, s, v, uint64(keySpace-1))
	steps := make([]WorkloadStep, count)
	for i := 0; i < count; i++ {
		keyID := zipf.Uint64()
		steps[i] = WorkloadStep{
			IsWrite: false,
			Key:     fmt.Sprintf("k-%d", keyID),
			Value:   "v",
		}
	}
	return steps
}

// generateSequentialSteps generates a sequential scan through keys.
func generateSequentialSteps(count int) []WorkloadStep {
	steps := make([]WorkloadStep, count)
	for i := 0; i < count; i++ {
		steps[i] = WorkloadStep{
			IsWrite: false,
			Key:     fmt.Sprintf("seq-%d", i),
			Value:   "v",
		}
	}
	return steps
}

// generateScanPollutionSteps simulates hot keys accessed, then a massive scan, then hot keys accessed again.
func generateScanPollutionSteps(hotCount int, hotRepeats int, scanCount int) []WorkloadStep {
	total := (hotCount * hotRepeats) + scanCount + (hotCount * hotRepeats)
	steps := make([]WorkloadStep, 0, total)

	// Phase 1: Establish hot items
	for r := 0; r < hotRepeats; r++ {
		for i := 0; i < hotCount; i++ {
			steps = append(steps, WorkloadStep{
				IsWrite: false,
				Key:     fmt.Sprintf("hot-%d", i),
				Value:   "hot-val",
			})
		}
	}

	// Phase 2: Massive one-time sequential scan
	for i := 0; i < scanCount; i++ {
		steps = append(steps, WorkloadStep{
			IsWrite: false,
			Key:     fmt.Sprintf("scan-%d", i),
			Value:   "scan-val",
		})
	}

	// Phase 3: Re-access hot items to measure pollution survival
	for r := 0; r < hotRepeats; r++ {
		for i := 0; i < hotCount; i++ {
			steps = append(steps, WorkloadStep{
				IsWrite: false,
				Key:     fmt.Sprintf("hot-%d", i),
				Value:   "hot-val",
			})
		}
	}

	return steps
}

// generateRepeatedHotSetSteps generates accesses restricted to a fixed small working set.
func generateRepeatedHotSetSteps(count int, workingSetSize int, seed int64) []WorkloadStep {
	rng := rand.New(rand.NewSource(seed))
	steps := make([]WorkloadStep, count)
	for i := 0; i < count; i++ {
		keyID := rng.Intn(workingSetSize)
		steps[i] = WorkloadStep{
			IsWrite: false,
			Key:     fmt.Sprintf("hotset-%d", keyID),
			Value:   "val",
		}
	}
	return steps
}

// generateMixedSteps generates mixed GET/SET operations according to readPct.
func generateMixedSteps(count int, keySpace int, readPct int, seed int64) []WorkloadStep {
	rng := rand.New(rand.NewSource(seed))
	steps := make([]WorkloadStep, count)
	for i := 0; i < count; i++ {
		keyID := rng.Intn(keySpace)
		isWrite := rng.Intn(100) >= readPct
		steps[i] = WorkloadStep{
			IsWrite: isWrite,
			Key:     fmt.Sprintf("mix-%d", keyID),
			Value:   "val",
		}
	}
	return steps
}

// runWorkload executes a sequence of workload steps on a cache using cache-aside semantics.
func runWorkload(c *Cache, steps []WorkloadStep) {
	for _, step := range steps {
		if step.IsWrite {
			c.Set(step.Key, step.Value)
		} else {
			_, found := c.Get(step.Key)
			if !found {
				// Cache miss: populate from backing store
				c.Set(step.Key, step.Value)
			}
		}
	}
}

// WorkloadResult stores measured metrics for reporting.
type WorkloadResult struct {
	Policy     string
	Workload   string
	Capacity   int
	Operations int
	Hits       uint64
	Misses     uint64
	HitRate    float64
	Evictions  uint64
	DurationNs int64
	NsPerOp    float64
	OpsPerSec  float64
}

// ExecuteWorkloadExperiment executes a workload and records comprehensive performance metrics.
func ExecuteWorkloadExperiment(policy string, cap int, warmupSteps []WorkloadStep, measureSteps []WorkloadStep, measureTiming bool) WorkloadResult {
	var c *Cache
	switch policy {
	case "LRU":
		c, _ = NewLRU(cap)
	case "LFU":
		c, _ = NewLFU(cap)
	case "2Q":
		c, _ = New2Q(cap)
	}

	// 1. Warm-up phase
	if len(warmupSteps) > 0 {
		runWorkload(c, warmupSteps)
	}

	// 2. Reset metrics before measurement
	c.Metrics().Reset()

	// 3. Exact measurement phase for metrics snapshot
	runWorkload(c, measureSteps)
	snap := c.Metrics().Snapshot()

	nsPerOp := 0.0
	opsPerSec := 0.0
	var durationNs int64

	// 4. High-resolution timing benchmark (only when explicitly requested)
	if measureTiming {
		bm := testing.Benchmark(func(b *testing.B) {
			stepLen := len(measureSteps)
			for i := 0; i < b.N; i++ {
				step := measureSteps[i%stepLen]
				if step.IsWrite {
					c.Set(step.Key, step.Value)
				} else {
					_, found := c.Get(step.Key)
					if !found {
						c.Set(step.Key, step.Value)
					}
				}
			}
		})
		durationNs = bm.T.Nanoseconds()
		nsPerOp = float64(bm.NsPerOp())
		if nsPerOp > 0 {
			opsPerSec = 1e9 / nsPerOp
		}
	}

	ops := len(measureSteps)
	return WorkloadResult{
		Policy:     policy,
		Capacity:   cap,
		Operations: ops,
		Hits:       snap.Hits,
		Misses:     snap.Misses,
		HitRate:    snap.HitRate,
		Evictions:  snap.Evictions,
		DurationNs: durationNs,
		NsPerOp:    nsPerOp,
		OpsPerSec:  opsPerSec,
	}
}

// ---------------------------------------------------------------------------
// Standard Go Benchmark Workloads
// ---------------------------------------------------------------------------

// BenchmarkWorkloadUniform evaluates cache behavior under uniform random key access.
func BenchmarkWorkloadUniform(b *testing.B) {
	const (
		keySpace  = 10000
		warmupOps = 5000
		seed      = 42
	)
	warmup := generateUniformSteps(warmupOps, keySpace, seed)

	for _, p := range policies {
		for _, cap := range benchmarkCapacities {
			name := fmt.Sprintf("%s/cap=%d", p.name, cap)
			b.Run(name, func(b *testing.B) {
				c, _ := p.fn(cap)
				runWorkload(c, warmup)
				c.Metrics().Reset()

				steps := generateUniformSteps(b.N, keySpace, seed+1)
				b.ResetTimer()
				runWorkload(c, steps)
				b.StopTimer()

				snap := c.Metrics().Snapshot()
				b.ReportMetric(snap.HitRate*100, "hit%")
				b.ReportMetric(float64(snap.Evictions), "evicts")
			})
		}
	}
}

// BenchmarkWorkloadZipfian evaluates cache behavior under skewed Zipfian/hot-key access.
func BenchmarkWorkloadZipfian(b *testing.B) {
	const (
		keySpace  = 10000
		warmupOps = 5000
		seed      = 42
		s         = 1.25
		v         = 1.0
	)
	warmup := generateZipfSteps(warmupOps, keySpace, s, v, seed)

	for _, p := range policies {
		for _, cap := range benchmarkCapacities {
			name := fmt.Sprintf("%s/cap=%d", p.name, cap)
			b.Run(name, func(b *testing.B) {
				c, _ := p.fn(cap)
				runWorkload(c, warmup)
				c.Metrics().Reset()

				steps := generateZipfSteps(b.N, keySpace, s, v, seed+1)
				b.ResetTimer()
				runWorkload(c, steps)
				b.StopTimer()

				snap := c.Metrics().Snapshot()
				b.ReportMetric(snap.HitRate*100, "hit%")
				b.ReportMetric(float64(snap.Evictions), "evicts")
			})
		}
	}
}

// BenchmarkWorkloadSequential evaluates cache behavior under sequential streaming access.
func BenchmarkWorkloadSequential(b *testing.B) {
	for _, p := range policies {
		for _, cap := range benchmarkCapacities {
			name := fmt.Sprintf("%s/cap=%d", p.name, cap)
			b.Run(name, func(b *testing.B) {
				c, _ := p.fn(cap)
				c.Metrics().Reset()

				steps := generateSequentialSteps(b.N)
				b.ResetTimer()
				runWorkload(c, steps)
				b.StopTimer()

				snap := c.Metrics().Snapshot()
				b.ReportMetric(snap.HitRate*100, "hit%")
				b.ReportMetric(float64(snap.Evictions), "evicts")
			})
		}
	}
}

// BenchmarkWorkloadScanPollution evaluates cache resistance against a large one-time scan.
func BenchmarkWorkloadScanPollution(b *testing.B) {
	const (
		hotCount   = 50
		hotRepeats = 10
		scanCount  = 5000
	)

	for _, p := range policies {
		for _, cap := range []int{100, 1000} {
			name := fmt.Sprintf("%s/cap=%d", p.name, cap)
			b.Run(name, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					c, _ := p.fn(cap)
					steps := generateScanPollutionSteps(hotCount, hotRepeats, scanCount)
					c.Metrics().Reset()
					runWorkload(c, steps)

					snap := c.Metrics().Snapshot()
					b.ReportMetric(snap.HitRate*100, "hit%")
					b.ReportMetric(float64(snap.Evictions), "evicts")
				}
			})
		}
	}
}

// BenchmarkWorkloadRepeatedHotSet evaluates working set behavior when working set fits vs does not fit.
func BenchmarkWorkloadRepeatedHotSet(b *testing.B) {
	const (
		workingSetSize = 50
		seed           = 42
	)
	// Compare cap = 20 (does not fit) vs cap = 100 (fits)
	capacities := []int{20, 100}

	for _, p := range policies {
		for _, cap := range capacities {
			name := fmt.Sprintf("%s/cap=%d", p.name, cap)
			b.Run(name, func(b *testing.B) {
				c, _ := p.fn(cap)
				c.Metrics().Reset()

				steps := generateRepeatedHotSetSteps(b.N, workingSetSize, seed)
				b.ResetTimer()
				runWorkload(c, steps)
				b.StopTimer()

				snap := c.Metrics().Snapshot()
				b.ReportMetric(snap.HitRate*100, "hit%")
				b.ReportMetric(float64(snap.Evictions), "evicts")
			})
		}
	}
}

// BenchmarkWorkloadMixed evaluates mixed read/write ratios (80% GET / 20% SET, and 95% GET / 5% SET).
func BenchmarkWorkloadMixed(b *testing.B) {
	const (
		keySpace  = 10000
		warmupOps = 5000
		seed      = 42
	)
	warmup := generateMixedSteps(warmupOps, keySpace, 80, seed)

	ratios := []struct {
		name    string
		readPct int
	}{
		{name: "80Get_20Set", readPct: 80},
		{name: "95Get_5Set", readPct: 95},
	}

	for _, p := range policies {
		for _, r := range ratios {
			name := fmt.Sprintf("%s/%s/cap=1000", p.name, r.name)
			b.Run(name, func(b *testing.B) {
				c, _ := p.fn(1000)
				runWorkload(c, warmup)
				c.Metrics().Reset()

				steps := generateMixedSteps(b.N, keySpace, r.readPct, seed+1)
				b.ResetTimer()
				runWorkload(c, steps)
				b.StopTimer()

				snap := c.Metrics().Snapshot()
				b.ReportMetric(snap.HitRate*100, "hit%")
				b.ReportMetric(float64(snap.Evictions), "evicts")
			})
		}
	}
}

// TestWorkloadEvaluationReport runs the full workload matrix and prints the formatted markdown table.
// Under standard `go test`, it runs a quick smoke run.
// Set RUN_WORKLOAD_REPORT=1 to run the full 50,000-operation empirical evaluation.
func TestWorkloadEvaluationReport(t *testing.T) {
	isFullReport := os.Getenv("RUN_WORKLOAD_REPORT") == "1"
	opCount := 200
	warmupCount := 50
	if isFullReport {
		opCount = 50000
		warmupCount = 5000
	}

	workloads := []struct {
		name       string
		capacities []int
		gen        func(cap int) ([]WorkloadStep, []WorkloadStep)
	}{
		{
			name:       "Uniform Random",
			capacities: []int{100, 1000, 10000},
			gen: func(cap int) ([]WorkloadStep, []WorkloadStep) {
				return generateUniformSteps(warmupCount, 10000, 42),
					generateUniformSteps(opCount, 10000, 43)
			},
		},
		{
			name:       "Zipfian / Hot-Key",
			capacities: []int{100, 1000, 10000},
			gen: func(cap int) ([]WorkloadStep, []WorkloadStep) {
				return generateZipfSteps(warmupCount, 10000, 1.25, 1.0, 42),
					generateZipfSteps(opCount, 10000, 1.25, 1.0, 43)
			},
		},
		{
			name:       "Sequential Access",
			capacities: []int{100, 1000, 10000},
			gen: func(cap int) ([]WorkloadStep, []WorkloadStep) {
				return nil, generateSequentialSteps(opCount)
			},
		},
		{
			name:       "Scan / Pollution",
			capacities: []int{100, 1000},
			gen: func(cap int) ([]WorkloadStep, []WorkloadStep) {
				scanSize := 5000
				if !isFullReport {
					scanSize = 200
				}
				return nil, generateScanPollutionSteps(50, 10, scanSize)
			},
		},
		{
			name:       "Repeated Hot Set (cap=20 vs 100)",
			capacities: []int{20, 100},
			gen: func(cap int) ([]WorkloadStep, []WorkloadStep) {
				return nil, generateRepeatedHotSetSteps(opCount, 50, 42)
			},
		},
		{
			name:       "Mixed (80% GET / 20% SET)",
			capacities: []int{1000},
			gen: func(cap int) ([]WorkloadStep, []WorkloadStep) {
				return generateMixedSteps(warmupCount, 10000, 80, 42),
					generateMixedSteps(opCount, 10000, 80, 43)
			},
		},
		{
			name:       "Mixed (95% GET / 5% SET)",
			capacities: []int{1000},
			gen: func(cap int) ([]WorkloadStep, []WorkloadStep) {
				return generateMixedSteps(warmupCount, 10000, 95, 42),
					generateMixedSteps(opCount, 10000, 95, 43)
			},
		},
	}

	var results []WorkloadResult
	for _, wl := range workloads {
		for _, cap := range wl.capacities {
			warmup, measure := wl.gen(cap)
			for _, policy := range []string{"LRU", "LFU", "2Q"} {
				res := ExecuteWorkloadExperiment(policy, cap, warmup, measure, isFullReport)
				res.Workload = wl.name
				results = append(results, res)
			}
		}
	}

	if isFullReport {
		t.Log("\n| Policy | Workload | Capacity | Operations | Hit Rate | Evictions | Latency (ns/op) | Throughput (ops/sec) |")
		t.Log("|--------|----------|----------|------------|----------|-----------|-----------------|----------------------|")
		for _, r := range results {
			t.Logf("| %-6s | %-30s | %8d | %10d | %7.2f%% | %9d | %15.2f | %20.0f |",
				r.Policy, r.Workload, r.Capacity, r.Operations, r.HitRate*100, r.Evictions, r.NsPerOp, r.OpsPerSec)
		}
	}
}
