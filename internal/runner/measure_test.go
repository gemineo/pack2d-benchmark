package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComputeTimingStats(t *testing.T) {
	tests := []struct {
		name      string
		durations []time.Duration
		wantMean  time.Duration
		wantMin   time.Duration
		wantMax   time.Duration
	}{
		{
			name:      "single value",
			durations: []time.Duration{100 * time.Microsecond},
			wantMean:  100 * time.Microsecond,
			wantMin:   100 * time.Microsecond,
			wantMax:   100 * time.Microsecond,
		},
		{
			name:      "two values",
			durations: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			wantMean:  150 * time.Microsecond,
			wantMin:   100 * time.Microsecond,
			wantMax:   200 * time.Microsecond,
		},
		{
			name: "five values",
			durations: []time.Duration{
				10 * time.Microsecond,
				20 * time.Microsecond,
				30 * time.Microsecond,
				40 * time.Microsecond,
				50 * time.Microsecond,
			},
			wantMean: 30 * time.Microsecond,
			wantMin:  10 * time.Microsecond,
			wantMax:  50 * time.Microsecond,
		},
		{
			name:      "empty",
			durations: nil,
			wantMean:  0,
			wantMin:   0,
			wantMax:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := ComputeTimingStats(tt.durations)
			assert.Equal(t, tt.wantMean, stats.Mean)
			assert.Equal(t, tt.wantMin, stats.Min)
			assert.Equal(t, tt.wantMax, stats.Max)

			if len(tt.durations) > 0 {
				assert.GreaterOrEqual(t, stats.P95, stats.Median)
				assert.GreaterOrEqual(t, stats.P99, stats.P95)
				assert.GreaterOrEqual(t, stats.Max, stats.P99)
				assert.GreaterOrEqual(t, stats.Median, stats.Min)
			}
		})
	}
}

func TestComputeTimingStats_Median(t *testing.T) {
	// Odd number of elements.
	stats := ComputeTimingStats([]time.Duration{
		10 * time.Microsecond,
		30 * time.Microsecond,
		50 * time.Microsecond,
	})
	assert.Equal(t, 30*time.Microsecond, stats.Median)

	// Even number of elements.
	stats = ComputeTimingStats([]time.Duration{
		10 * time.Microsecond,
		20 * time.Microsecond,
		30 * time.Microsecond,
		40 * time.Microsecond,
	})
	assert.Equal(t, 25*time.Microsecond, stats.Median)
}

func TestComputeTimingStats_StdDev(t *testing.T) {
	// All identical values → stddev = 0.
	stats := ComputeTimingStats([]time.Duration{
		100 * time.Microsecond,
		100 * time.Microsecond,
		100 * time.Microsecond,
	})
	assert.Equal(t, time.Duration(0), stats.StdDev)
}
