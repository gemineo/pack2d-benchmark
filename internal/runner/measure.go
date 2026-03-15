package runner

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/gemineo/pack2d"
)

// MeasureEncode runs warmUp+iterations encode cycles and returns timing stats and the encoded output.
// A fresh copy of data is used for each call because pack2d.Encode may mutate the input slice.
func MeasureEncode(data []byte, opts []pack2d.Option, warmUp, iterations int) (TimingStats, string, pack2d.Stats, error) {
	copyData := func() []byte {
		buf := make([]byte, len(data))
		copy(buf, data)
		return buf
	}

	// Warm-up runs (discarded).
	for range warmUp {
		_, _, err := pack2d.Encode(copyData(), opts...)
		if err != nil {
			return TimingStats{}, "", pack2d.Stats{}, fmt.Errorf("encode warm-up: %w", err)
		}
	}

	durations := make([]time.Duration, iterations)
	var encoded string
	var stats pack2d.Stats
	var err error

	for i := range iterations {
		buf := copyData()
		start := time.Now()
		encoded, stats, err = pack2d.Encode(buf, opts...)
		durations[i] = time.Since(start)
		if err != nil {
			return TimingStats{}, "", pack2d.Stats{}, fmt.Errorf("encode iteration %d: %w", i, err)
		}
	}

	return ComputeTimingStats(durations), encoded, stats, nil
}

// MeasureDecode runs warmUp+iterations decode cycles and returns timing stats.
func MeasureDecode(encoded string, opts []pack2d.Option, warmUp, iterations int) (TimingStats, error) {
	for range warmUp {
		_, _, err := pack2d.Decode(encoded, opts...)
		if err != nil {
			return TimingStats{}, fmt.Errorf("decode warm-up: %w", err)
		}
	}

	durations := make([]time.Duration, iterations)
	for i := range iterations {
		start := time.Now()
		_, _, err := pack2d.Decode(encoded, opts...)
		durations[i] = time.Since(start)
		if err != nil {
			return TimingStats{}, fmt.Errorf("decode iteration %d: %w", i, err)
		}
	}

	return ComputeTimingStats(durations), nil
}

// ComputeTimingStats computes descriptive statistics from a slice of durations.
func ComputeTimingStats(durations []time.Duration) TimingStats {
	n := len(durations)
	if n == 0 {
		return TimingStats{}
	}

	sorted := make([]time.Duration, n)
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	var sum float64
	for _, d := range sorted {
		sum += float64(d)
	}
	mean := sum / float64(n)

	var median float64
	if n%2 == 0 {
		median = float64(sorted[n/2-1]+sorted[n/2]) / 2
	} else {
		median = float64(sorted[n/2])
	}

	var variance float64
	for _, d := range sorted {
		diff := float64(d) - mean
		variance += diff * diff
	}
	variance /= float64(n)
	stddev := math.Sqrt(variance)

	return TimingStats{
		Mean:   time.Duration(mean),
		Median: time.Duration(median),
		P95:    percentile(sorted, 95),
		P99:    percentile(sorted, 99),
		Min:    sorted[0],
		Max:    sorted[n-1],
		StdDev: time.Duration(stddev),
	}
}

func percentile(sorted []time.Duration, p int) time.Duration {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	idx := float64(p) / 100.0 * float64(n-1)
	lower := int(idx)
	upper := lower + 1
	if upper >= n {
		return sorted[n-1]
	}
	frac := idx - float64(lower)
	return time.Duration(float64(sorted[lower])*(1-frac) + float64(sorted[upper])*frac)
}
