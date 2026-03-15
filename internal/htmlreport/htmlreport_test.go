package htmlreport

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gemineo/pack2d"
	"github.com/gemineo/pack2d-benchmark/internal/report"
	"github.com/gemineo/pack2d-benchmark/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_ContainsExpectedElements(t *testing.T) {
	rpt := &report.Report{
		Metadata: report.Metadata{
			ToolVersion: "1.0.0-test",
			GoVersion:   "go1.26.1",
			OS:          "darwin",
			Arch:        "arm64",
			Timestamp:   time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC),
			Iterations:  20,
			WarmUp:      3,
		},
		Results: []runner.Result{
			{
				Scenario:    "compression",
				Dataset:     "test-ds",
				DatasetSize: 1000,
				Algorithm:   pack2d.Zstd,
				Level:       3,
				InputType:   pack2d.Raw,
				InputBytes:  1000,
				Compressed:  400,
				Encoded:     500,
				Ratio:       2.5,
				Encode:      runner.TimingStats{Mean: 100 * time.Microsecond},
			},
			{
				Scenario:    "compression",
				Dataset:     "test-ds",
				DatasetSize: 1000,
				Algorithm:   pack2d.Zlib,
				Level:       6,
				InputType:   pack2d.Raw,
				InputBytes:  1000,
				Compressed:  500,
				Encoded:     600,
				Ratio:       2.0,
				Encode:      runner.TimingStats{Mean: 200 * time.Microsecond},
			},
			{
				Scenario:    "compression",
				Dataset:     "test-ds",
				DatasetSize: 1000,
				Algorithm:   pack2d.Zstd,
				Level:       3,
				InputType:   pack2d.CBOR,
				InputBytes:  1000,
				Compressed:  350,
				Encoded:     450,
				Ratio:       2.86,
				Encode:      runner.TimingStats{Mean: 110 * time.Microsecond},
			},
		},
		Summary: &report.Summary{
			BestRatio: []report.BestEntry{
				{
					Dataset:   "test-ds",
					Algorithm: "zstd",
					Level:     3,
					InputType: "raw",
					Ratio:     2.5,
					EncodeUs:  100,
				},
			},
			SweetSpot: []report.SweetSpotEntry{
				{
					Dataset:   "test-ds",
					Algorithm: "zstd",
					Level:     3,
					InputType: "raw",
					Ratio:     2.5,
					EncodeUs:  100,
					Found:     true,
				},
			},
			Recommendations: []string{"Use zstd level 3 for best trade-off"},
		},
	}

	var buf bytes.Buffer
	err := Generate(rpt, &buf)
	require.NoError(t, err)

	html := buf.String()

	// Metadata header.
	assert.True(t, strings.Contains(html, "pack2d Benchmark Report"), "should contain title")
	assert.True(t, strings.Contains(html, "1.0.0-test"), "should contain tool version")
	assert.True(t, strings.Contains(html, "go1.26.1"), "should contain Go version")
	assert.True(t, strings.Contains(html, "darwin/arm64"), "should contain platform")

	// Summary section.
	assert.True(t, strings.Contains(html, "test-ds"), "should contain dataset name")
	assert.True(t, strings.Contains(html, "zstd L3"), "should contain sweet spot config")
	assert.True(t, strings.Contains(html, "Best Compression Ratio"), "should contain best ratio heading")
	assert.True(t, strings.Contains(html, "Sweet Spot"), "should contain sweet spot heading")

	// Charts (go-echarts renders these as divs with echarts init).
	assert.True(t, strings.Contains(html, "echarts"), "should contain echarts reference")
	assert.True(t, strings.Contains(html, "Serialization Impact"), "should contain serialization impact chart")

	// Valid HTML structure.
	assert.True(t, strings.Contains(html, "<!DOCTYPE html>"), "should be valid HTML")
	assert.True(t, strings.Contains(html, "</html>"), "should close HTML")
}

func TestGenerate_EmptyResults(t *testing.T) {
	rpt := &report.Report{
		Metadata: report.Metadata{
			ToolVersion: "test",
			GoVersion:   "go1.26.1",
			Timestamp:   time.Now(),
		},
	}

	var buf bytes.Buffer
	err := Generate(rpt, &buf)
	require.NoError(t, err)

	html := buf.String()
	assert.True(t, strings.Contains(html, "<!DOCTYPE html>"))
	assert.True(t, strings.Contains(html, "pack2d Benchmark Report"))
}

func TestGenerate_WithBarcodeResults(t *testing.T) {
	rpt := &report.Report{
		Metadata: report.Metadata{
			ToolVersion: "test",
			GoVersion:   "go1.26.1",
			Timestamp:   time.Now(),
		},
		Results: []runner.Result{
			{
				Scenario: "barcode",
				Dataset:  "small",
				Encoded:  200,
				Barcode: &runner.BarcodeResult{
					Checks: []runner.BarcodeCheck{
						{BarcodeType: "qr", ECLevel: "L", Fits: true, QRVersion: 5},
						{BarcodeType: "qr", ECLevel: "H", Fits: false},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := Generate(rpt, &buf)
	require.NoError(t, err)

	html := buf.String()
	assert.True(t, strings.Contains(html, "QR Code Feasibility") || strings.Contains(html, "echarts"),
		"should contain heatmap or echarts reference")
}

func TestGenerate_WithDataMatrixResults(t *testing.T) {
	rpt := &report.Report{
		Metadata: report.Metadata{
			ToolVersion: "test",
			GoVersion:   "go1.26.1",
			Timestamp:   time.Now(),
		},
		Results: []runner.Result{
			{
				Scenario: "barcode",
				Dataset:  "small",
				Encoded:  200,
				Barcode: &runner.BarcodeResult{
					Checks: []runner.BarcodeCheck{
						{BarcodeType: "datamatrix", ECLevel: "ECC200", Fits: true},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := Generate(rpt, &buf)
	require.NoError(t, err)

	html := buf.String()
	assert.True(t, strings.Contains(html, "DataMatrix Feasibility") || strings.Contains(html, "echarts"),
		"should contain DataMatrix heatmap or echarts reference")
}

func TestGenerate_WithDictResults(t *testing.T) {
	rpt := &report.Report{
		Metadata: report.Metadata{
			ToolVersion: "test",
			GoVersion:   "go1.26.1",
			Timestamp:   time.Now(),
		},
		Results: []runner.Result{
			{
				Scenario:   "compression",
				Dataset:    "ds1",
				Algorithm:  pack2d.Zstd,
				Level:      3,
				InputType:  pack2d.Raw,
				UseDict:    false,
				Ratio:      2.0,
				InputBytes: 1000,
				Compressed: 500,
				Encoded:    500,
				Encode:     runner.TimingStats{Mean: 100 * time.Microsecond},
			},
			{
				Scenario:   "compression",
				Dataset:    "ds1",
				Algorithm:  pack2d.Zstd,
				Level:      3,
				InputType:  pack2d.Raw,
				UseDict:    true,
				Ratio:      3.0,
				InputBytes: 1000,
				Compressed: 333,
				Encoded:    333,
				Encode:     runner.TimingStats{Mean: 110 * time.Microsecond},
			},
		},
	}

	var buf bytes.Buffer
	err := Generate(rpt, &buf)
	require.NoError(t, err)

	html := buf.String()
	assert.True(t, strings.Contains(html, "Dictionary Impact") || strings.Contains(html, "echarts"),
		"should contain dict chart or echarts reference")
}
