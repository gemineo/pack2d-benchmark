package report

import (
	"bytes"
	"testing"
	"time"

	"github.com/gemineo/pack2d"
	"github.com/go-json-experiment/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gemineo/pack2d-benchmark/internal/runner"
)

func TestExportJSON_RoundTrip(t *testing.T) {
	rpt := &Report{
		Metadata: Metadata{
			ToolVersion: "test-v1",
			GoVersion:   "go1.26.1",
			OS:          "darwin",
			Arch:        "arm64",
			Timestamp:   time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
			Iterations:  10,
			WarmUp:      3,
		},
		Datasets: []DatasetInfo{
			{Name: "test", Type: "json", Size: 100, Source: "embedded", Description: "test dataset"},
		},
		Results: []runner.Result{
			{
				Scenario:    "compression",
				Dataset:     "test",
				DatasetSize: 100,
				Algorithm:   pack2d.Zstd,
				Level:       1,
				InputType:   pack2d.Raw,
				InputBytes:  100,
				Compressed:  80,
				Encoded:     120,
				Ratio:       0.8,
				Encode:      runner.TimingStats{Mean: 100 * time.Microsecond, Min: 80 * time.Microsecond, Max: 120 * time.Microsecond},
				Decode:      runner.TimingStats{Mean: 50 * time.Microsecond, Min: 40 * time.Microsecond, Max: 60 * time.Microsecond},
			},
		},
		Summary: &Summary{
			Recommendations: []string{"test recommendation"},
		},
	}

	var buf bytes.Buffer
	err := ExportJSON(&buf, rpt)
	require.NoError(t, err)

	// Unmarshal back.
	var decoded Report
	err = json.Unmarshal(buf.Bytes(), &decoded, json.DefaultOptionsV2())
	require.NoError(t, err)

	assert.Equal(t, rpt.Metadata.ToolVersion, decoded.Metadata.ToolVersion)
	assert.Equal(t, rpt.Metadata.Iterations, decoded.Metadata.Iterations)
	assert.Len(t, decoded.Results, 1)
	assert.Equal(t, "compression", decoded.Results[0].Scenario)
	assert.Equal(t, 100*time.Microsecond, decoded.Results[0].Encode.Mean)
}
