package htmlreport

import (
	"testing"
	"time"

	"github.com/gemineo/pack2d"
	"github.com/gemineo/pack2d-benchmark/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeResult(dataset, algo string, level int, inputType string, ratio float64, encodeUs int64, useDict bool) runner.Result {
	return runner.Result{
		Scenario:    "compression",
		Dataset:     dataset,
		DatasetSize: 1000,
		Algorithm:   pack2d.CompressionType(algo),
		Level:       level,
		InputType:   pack2d.InputType(inputType),
		UseDict:     useDict,
		InputBytes:  1000,
		Compressed:  int(float64(1000) / ratio),
		Encoded:     int(float64(1000) / ratio),
		Ratio:       ratio,
		Encode:      runner.TimingStats{Mean: time.Duration(encodeUs) * time.Microsecond},
	}
}

func makeBarcodeResult(dataset, algo string, level int, encoded int, checks []runner.BarcodeCheck) runner.Result {
	return runner.Result{
		Scenario:  "barcode",
		Dataset:   dataset,
		Algorithm: pack2d.CompressionType(algo),
		Level:     level,
		InputType: "raw",
		Encoded:   encoded,
		Barcode:   &runner.BarcodeResult{Checks: checks},
	}
}

func TestCompressionRatioByDataset(t *testing.T) {
	results := []runner.Result{
		makeResult("small", "zstd", 1, "raw", 2.0, 10, false),
		makeResult("small", "zstd", 9, "raw", 3.0, 50, false),
		makeResult("small", "zlib", 6, "raw", 2.5, 30, false),
		makeResult("large", "zstd", 1, "raw", 4.0, 20, false),
		makeResult("large", "zlib", 9, "raw", 3.5, 80, false),
	}

	data := CompressionRatioByDataset(results)

	assert.Equal(t, []string{"large", "small"}, data.Datasets)
	assert.InDelta(t, 3.0, data.ByAlgo["zstd"][1], 0.001) // small: best is 3.0
	assert.InDelta(t, 4.0, data.ByAlgo["zstd"][0], 0.001) // large: 4.0
	assert.InDelta(t, 2.5, data.ByAlgo["zlib"][1], 0.001) // small: 2.5
	assert.InDelta(t, 3.5, data.ByAlgo["zlib"][0], 0.001) // large: 3.5
}

func TestCompressionRatioByDataset_IgnoresBarcode(t *testing.T) {
	results := []runner.Result{
		makeResult("ds1", "zstd", 1, "raw", 2.0, 10, false),
		{Scenario: "barcode", Dataset: "ds1", Algorithm: "zstd", Ratio: 99.0},
	}

	data := CompressionRatioByDataset(results)

	assert.Equal(t, []string{"ds1"}, data.Datasets)
	assert.InDelta(t, 2.0, data.ByAlgo["zstd"][0], 0.001)
}

func TestCompressionRatioByDataset_Empty(t *testing.T) {
	data := CompressionRatioByDataset(nil)
	assert.Empty(t, data.Datasets)
	assert.Empty(t, data.ByAlgo)
}

func TestSpeedVsRatio(t *testing.T) {
	results := []runner.Result{
		makeResult("ds1", "zstd", 3, "raw", 2.5, 100, false),
		makeResult("ds1", "zlib", 6, "json", 3.0, 200, false),
		makeResult("ds1", "zstd", 3, "raw", 2.8, 120, true),
	}

	points := SpeedVsRatio(results)

	require.Len(t, points, 3)

	assert.Equal(t, "zstd/L3/raw", points[0].Label)
	assert.InDelta(t, 100.0, points[0].EncodeUs, 0.001)
	assert.InDelta(t, 2.5, points[0].Ratio, 0.001)

	assert.Equal(t, "zlib/L6/json", points[1].Label)

	assert.Equal(t, "zstd/L3/raw/dict", points[2].Label)
}

func TestSpeedVsRatio_Empty(t *testing.T) {
	points := SpeedVsRatio(nil)
	assert.Empty(t, points)
}

func TestLevelSweep(t *testing.T) {
	results := []runner.Result{
		makeResult("ds1", "zstd", 1, "raw", 1.5, 10, false),
		makeResult("ds1", "zstd", 3, "raw", 2.0, 20, false),
		makeResult("ds1", "zstd", 9, "raw", 2.5, 50, false),
		makeResult("ds1", "zlib", 1, "raw", 1.2, 15, false),
		makeResult("ds1", "zlib", 6, "raw", 2.0, 30, false),
		// Different dataset — should be excluded.
		makeResult("ds2", "zstd", 1, "raw", 3.0, 10, false),
		// Dictionary result — should be excluded.
		makeResult("ds1", "zstd", 1, "raw", 1.8, 8, true),
	}

	series := LevelSweep(results, "ds1")

	require.Len(t, series, 2)

	// Find zstd series.
	var zstdSeries LevelSweepSeries
	for _, s := range series {
		if s.Algorithm == "zstd" {
			zstdSeries = s
		}
	}

	assert.Equal(t, []int{1, 3, 9}, zstdSeries.Levels)
	assert.InDelta(t, 1.5, zstdSeries.Ratios[0], 0.001)
	assert.InDelta(t, 2.0, zstdSeries.Ratios[1], 0.001)
	assert.InDelta(t, 2.5, zstdSeries.Ratios[2], 0.001)
}

func TestLevelSweep_PrefersRaw(t *testing.T) {
	results := []runner.Result{
		makeResult("ds1", "zstd", 3, "json", 2.0, 20, false),
		makeResult("ds1", "zstd", 3, "raw", 1.8, 15, false),
	}

	series := LevelSweep(results, "ds1")

	require.Len(t, series, 1)
	assert.InDelta(t, 1.8, series[0].Ratios[0], 0.001) // raw preferred
}

func TestLevelSweep_Empty(t *testing.T) {
	series := LevelSweep(nil, "ds1")
	assert.Empty(t, series)
}

func TestDictImpact(t *testing.T) {
	results := []runner.Result{
		makeResult("ds1", "zstd", 3, "raw", 2.0, 20, false),
		makeResult("ds1", "zstd", 3, "raw", 3.0, 25, true),
		makeResult("ds1", "zstd", 9, "raw", 2.5, 50, false),
		makeResult("ds1", "zstd", 9, "raw", 3.5, 55, true),
		// zlib results should be ignored.
		makeResult("ds1", "zlib", 6, "raw", 2.0, 30, false),
	}

	pairs := DictImpact(results)

	require.Len(t, pairs, 1)
	assert.Equal(t, "ds1", pairs[0].Dataset)
	assert.Equal(t, 9, pairs[0].Level)                          // best dict ratio
	assert.InDelta(t, 2.5, pairs[0].RatioNoDict, 0.001)
	assert.InDelta(t, 3.5, pairs[0].RatioDict, 0.001)
}

func TestDictImpact_NoDict(t *testing.T) {
	results := []runner.Result{
		makeResult("ds1", "zstd", 3, "raw", 2.0, 20, false),
	}

	pairs := DictImpact(results)
	assert.Empty(t, pairs)
}

func TestDictImpact_Empty(t *testing.T) {
	pairs := DictImpact(nil)
	assert.Empty(t, pairs)
}

func TestBarcodeHeatmap(t *testing.T) {
	results := []runner.Result{
		makeBarcodeResult("small", "zstd", 3, 100, []runner.BarcodeCheck{
			{BarcodeType: "qr", ECLevel: "L", Fits: true, QRVersion: 5},
			{BarcodeType: "qr", ECLevel: "M", Fits: true, QRVersion: 7},
			{BarcodeType: "qr", ECLevel: "H", Fits: false, QRVersion: 0},
		}),
		makeBarcodeResult("large", "zstd", 3, 500, []runner.BarcodeCheck{
			{BarcodeType: "qr", ECLevel: "L", Fits: true, QRVersion: 15},
			{BarcodeType: "qr", ECLevel: "M", Fits: false, QRVersion: 0},
			{BarcodeType: "qr", ECLevel: "H", Fits: false, QRVersion: 0},
		}),
		// Better config for large (smaller encoded) — should replace above.
		makeBarcodeResult("large", "brotli", 6, 300, []runner.BarcodeCheck{
			{BarcodeType: "qr", ECLevel: "L", Fits: true, QRVersion: 10},
			{BarcodeType: "qr", ECLevel: "M", Fits: true, QRVersion: 12},
			{BarcodeType: "qr", ECLevel: "H", Fits: false, QRVersion: 0},
		}),
	}

	datasets, ecLevels, cells := BarcodeHeatmap(results)

	assert.Equal(t, []string{"large", "small"}, datasets)
	assert.Equal(t, []string{"L", "M", "H"}, ecLevels)

	// Build lookup.
	cellMap := map[string]HeatmapCell{}
	for _, c := range cells {
		cellMap[c.Dataset+"/"+c.ECLevel] = c
	}

	// large/L uses brotli result (encoded=300 < 500).
	assert.True(t, cellMap["large/L"].Fits)
	assert.Equal(t, 10, cellMap["large/L"].Version)

	// large/M: brotli result fits.
	assert.True(t, cellMap["large/M"].Fits)

	// small/H doesn't fit.
	assert.False(t, cellMap["small/H"].Fits)
}

func TestBarcodeHeatmap_Empty(t *testing.T) {
	datasets, ecLevels, cells := BarcodeHeatmap(nil)
	assert.Empty(t, datasets)
	assert.Empty(t, ecLevels)
	assert.Empty(t, cells)
}

func TestBarcodeHeatmap_IgnoresNonQR(t *testing.T) {
	results := []runner.Result{
		makeBarcodeResult("ds1", "zstd", 3, 100, []runner.BarcodeCheck{
			{BarcodeType: "datamatrix", ECLevel: "", Fits: true},
		}),
	}

	datasets, _, cells := BarcodeHeatmap(results)
	assert.Empty(t, datasets)
	assert.Empty(t, cells)
}

func TestDataMatrixHeatmap(t *testing.T) {
	results := []runner.Result{
		makeBarcodeResult("small", "zstd", 3, 100, []runner.BarcodeCheck{
			{BarcodeType: "datamatrix", ECLevel: "ECC200", Fits: true},
		}),
		makeBarcodeResult("large", "zstd", 3, 500, []runner.BarcodeCheck{
			{BarcodeType: "datamatrix", ECLevel: "ECC200", Fits: false},
		}),
		// Better config for large (smaller encoded) — should replace above.
		makeBarcodeResult("large", "brotli", 6, 300, []runner.BarcodeCheck{
			{BarcodeType: "datamatrix", ECLevel: "ECC200", Fits: true},
		}),
	}

	datasets, cells := DataMatrixHeatmap(results)

	assert.Equal(t, []string{"large", "small"}, datasets)
	require.Len(t, cells, 2)

	cellMap := map[string]HeatmapCell{}
	for _, c := range cells {
		cellMap[c.Dataset] = c
	}

	// large uses brotli result (encoded=300 < 500).
	assert.True(t, cellMap["large"].Fits)
	assert.Equal(t, "ECC200", cellMap["large"].ECLevel)

	assert.True(t, cellMap["small"].Fits)
}

func TestDataMatrixHeatmap_Empty(t *testing.T) {
	datasets, cells := DataMatrixHeatmap(nil)
	assert.Empty(t, datasets)
	assert.Empty(t, cells)
}

func TestDataMatrixHeatmap_IgnoresQR(t *testing.T) {
	results := []runner.Result{
		makeBarcodeResult("ds1", "zstd", 3, 100, []runner.BarcodeCheck{
			{BarcodeType: "qr", ECLevel: "L", Fits: true, QRVersion: 5},
		}),
	}

	datasets, cells := DataMatrixHeatmap(results)
	assert.Empty(t, datasets)
	assert.Empty(t, cells)
}

func TestBarcodeHeatmap_AcceptsQRCodeType(t *testing.T) {
	results := []runner.Result{
		makeBarcodeResult("ds1", "zstd", 3, 100, []runner.BarcodeCheck{
			{BarcodeType: "qrcode", ECLevel: "L", Fits: true, QRVersion: 5},
		}),
	}

	datasets, ecLevels, cells := BarcodeHeatmap(results)
	assert.Equal(t, []string{"ds1"}, datasets)
	assert.Equal(t, []string{"L"}, ecLevels)
	require.Len(t, cells, 1)
	assert.True(t, cells[0].Fits)
}

func TestDatasets(t *testing.T) {
	results := []runner.Result{
		makeResult("beta", "zstd", 1, "raw", 1.0, 10, false),
		makeResult("alpha", "zstd", 1, "raw", 1.0, 10, false),
		makeResult("beta", "zlib", 1, "raw", 1.0, 10, false),
	}

	ds := Datasets(results)
	assert.Equal(t, []string{"alpha", "beta"}, ds)
}

func TestSortedECLevels(t *testing.T) {
	m := map[string]struct{}{
		"H": {}, "L": {}, "Q": {}, "M": {},
	}
	assert.Equal(t, []string{"L", "M", "Q", "H"}, sortedECLevels(m))
}
