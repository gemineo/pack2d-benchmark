package report

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gemineo/pack2d-benchmark/internal/runner"
)

func TestComputeSummary_Empty(t *testing.T) {
	s := ComputeSummary(nil)
	require.NotNil(t, s)
	assert.Empty(t, s.BestRatio)
	assert.Empty(t, s.BestSpeed)
	assert.Empty(t, s.SweetSpot)
	assert.Empty(t, s.QRFitCounts)
	assert.Empty(t, s.Recommendations)
}

func TestComputeSummary_SingleResult(t *testing.T) {
	results := []runner.Result{
		{
			Scenario:  "compression",
			Dataset:   "test-ds",
			Algorithm: "zstd",
			Level:     3,
			InputType: "raw",
			Ratio:     2.5,
			Encode:    runner.TimingStats{Mean: 100 * time.Microsecond},
		},
	}

	s := ComputeSummary(results)
	require.NotNil(t, s)

	require.Len(t, s.BestRatio, 1)
	assert.Equal(t, "test-ds", s.BestRatio[0].Dataset)
	assert.Equal(t, 2.5, s.BestRatio[0].Ratio)

	require.Len(t, s.BestSpeed, 1)
	assert.Equal(t, "test-ds", s.BestSpeed[0].Dataset)

	// Single result: no sweet spot can be found (loop never runs).
	require.Len(t, s.SweetSpot, 1)
	assert.False(t, s.SweetSpot[0].Found)
}

func TestComputeSummary_SweetSpotFound(t *testing.T) {
	results := []runner.Result{
		{
			Scenario:  "compression",
			Dataset:   "ds",
			Algorithm: "zstd",
			Level:     1,
			InputType: "raw",
			Ratio:     0.80, // fast but weak compression
			Encode:    runner.TimingStats{Mean: 10 * time.Microsecond},
		},
		{
			Scenario:  "compression",
			Dataset:   "ds",
			Algorithm: "zstd",
			Level:     3,
			InputType: "raw",
			Ratio:     0.40, // big improvement (ratio dropped 50%)
			Encode:    runner.TimingStats{Mean: 20 * time.Microsecond},
		},
	}

	s := ComputeSummary(results)
	require.Len(t, s.SweetSpot, 1)
	assert.True(t, s.SweetSpot[0].Found)
	assert.Equal(t, 0.40, s.SweetSpot[0].Ratio)
}

func TestComputeSummary_NoSweetSpotThresholdNotMet(t *testing.T) {
	// Two configs with nearly identical ratios but vastly different times.
	// Marginal improvement is below 0.05% per µs threshold.
	results := []runner.Result{
		{
			Scenario:  "compression",
			Dataset:   "ds",
			Algorithm: "zstd",
			Level:     1,
			InputType: "raw",
			Ratio:     0.50,
			Encode:    runner.TimingStats{Mean: 10 * time.Microsecond},
		},
		{
			Scenario:  "compression",
			Dataset:   "ds",
			Algorithm: "zstd",
			Level:     9,
			InputType: "raw",
			Ratio:     0.499, // negligible improvement
			Encode:    runner.TimingStats{Mean: 10000 * time.Microsecond},
		},
	}

	s := ComputeSummary(results)
	require.Len(t, s.SweetSpot, 1)
	assert.False(t, s.SweetSpot[0].Found)
	// Falls back to fastest config.
	assert.Equal(t, int64(10), s.SweetSpot[0].EncodeUs)
}

func TestComputeSummary_ZeroRatioSkipped(t *testing.T) {
	// A config with ratio=0 must not cause a division by zero.
	results := []runner.Result{
		{
			Scenario:  "compression",
			Dataset:   "ds",
			Algorithm: "zstd",
			Level:     1,
			InputType: "raw",
			Ratio:     0.0,
			Encode:    runner.TimingStats{Mean: 10 * time.Microsecond},
		},
		{
			Scenario:  "compression",
			Dataset:   "ds",
			Algorithm: "zstd",
			Level:     3,
			InputType: "raw",
			Ratio:     2.0,
			Encode:    runner.TimingStats{Mean: 20 * time.Microsecond},
		},
	}

	// Must not panic.
	s := ComputeSummary(results)
	require.Len(t, s.SweetSpot, 1)
}

func TestComputeSummary_BestRatioAndSpeed(t *testing.T) {
	results := []runner.Result{
		{
			Scenario:  "compression",
			Dataset:   "ds",
			Algorithm: "zlib",
			Level:     1,
			InputType: "raw",
			Ratio:     0.70, // weak compression but fast
			Encode:    runner.TimingStats{Mean: 5 * time.Microsecond},
		},
		{
			Scenario:  "compression",
			Dataset:   "ds",
			Algorithm: "brotli",
			Level:     11,
			InputType: "raw",
			Ratio:     0.25, // strong compression but slow
			Encode:    runner.TimingStats{Mean: 500 * time.Microsecond},
		},
	}

	s := ComputeSummary(results)

	require.Len(t, s.BestRatio, 1)
	assert.Equal(t, "brotli", s.BestRatio[0].Algorithm) // lowest ratio = best compression
	assert.Equal(t, 0.25, s.BestRatio[0].Ratio)

	require.Len(t, s.BestSpeed, 1)
	assert.Equal(t, "zlib", s.BestSpeed[0].Algorithm)
	assert.Equal(t, int64(5), s.BestSpeed[0].EncodeUs)
}

func TestComputeSummary_QRFitCounts(t *testing.T) {
	results := []runner.Result{
		{
			Scenario: "compression",
			Dataset:  "ds",
			Barcode: &runner.BarcodeResult{
				Checks: []runner.BarcodeCheck{
					{BarcodeType: "qrcode", ECLevel: "M", Fits: true},
					{BarcodeType: "qrcode", ECLevel: "L", Fits: true},
					{BarcodeType: "qrcode", ECLevel: "H", Fits: false},
					{BarcodeType: "datamatrix", ECLevel: "ECC200", Fits: true},
				},
			},
			Ratio:  2.0,
			Encode: runner.TimingStats{Mean: 10 * time.Microsecond},
		},
	}

	s := ComputeSummary(results)
	assert.Equal(t, 1, s.QRFitCounts["QR-M"])
	assert.Equal(t, 1, s.QRFitCounts["QR-L"])
	assert.Equal(t, 0, s.QRFitCounts["QR-H"])
}

func TestComputeSummary_IgnoresNonCompressionResults(t *testing.T) {
	results := []runner.Result{
		{
			Scenario: "barcode",
			Dataset:  "ds",
			Ratio:    3.0,
			Encode:   runner.TimingStats{Mean: 10 * time.Microsecond},
		},
	}

	s := ComputeSummary(results)
	assert.Empty(t, s.BestRatio)
	assert.Empty(t, s.BestSpeed)
	assert.Empty(t, s.SweetSpot)
}

func TestComputeSummary_MultipleDatasets(t *testing.T) {
	results := []runner.Result{
		{
			Scenario:  "compression",
			Dataset:   "alpha",
			Algorithm: "zstd",
			Level:     3,
			InputType: "raw",
			Ratio:     2.0,
			Encode:    runner.TimingStats{Mean: 10 * time.Microsecond},
		},
		{
			Scenario:  "compression",
			Dataset:   "beta",
			Algorithm: "zlib",
			Level:     6,
			InputType: "json",
			Ratio:     3.0,
			Encode:    runner.TimingStats{Mean: 20 * time.Microsecond},
		},
	}

	s := ComputeSummary(results)
	require.Len(t, s.BestRatio, 2)
	require.Len(t, s.BestSpeed, 2)
	require.Len(t, s.SweetSpot, 2)

	// Sorted alphabetically by dataset.
	assert.Equal(t, "alpha", s.BestRatio[0].Dataset)
	assert.Equal(t, "beta", s.BestRatio[1].Dataset)
}

func TestGenerateRecommendations(t *testing.T) {
	s := &Summary{
		SweetSpot: []SweetSpotEntry{
			{
				Dataset:   "ds",
				Algorithm: "zstd",
				Level:     3,
				InputType: "raw",
				Ratio:     2.5,
				EncodeUs:  100,
				Found:     true,
			},
		},
		QRFitCounts: map[string]int{
			"QR-M": 5,
			"QR-L": 3,
		},
	}

	recs := generateRecommendations(s)
	require.Len(t, recs, 2)
	assert.Contains(t, recs[0], "Sweet spot")
	assert.Contains(t, recs[1], "8 configurations fit in QR codes")
}

func TestGenerateRecommendations_NoSweetSpotFound(t *testing.T) {
	s := &Summary{
		SweetSpot: []SweetSpotEntry{
			{
				Dataset:   "ds",
				Algorithm: "zstd",
				Level:     1,
				InputType: "raw",
				Ratio:     1.5,
				EncodeUs:  10,
				Found:     false,
			},
		},
	}

	recs := generateRecommendations(s)
	require.Len(t, recs, 1)
	assert.Contains(t, recs[0], "Fastest (no sweet spot found)")
}
