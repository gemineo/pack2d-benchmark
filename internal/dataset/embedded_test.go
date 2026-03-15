package dataset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEmbedded(t *testing.T) {
	datasets, err := LoadEmbedded()
	require.NoError(t, err)
	require.NotEmpty(t, datasets)

	// Expect 11 embedded datasets (5 JSON + 5 XML + 1 adversarial).
	assert.Len(t, datasets, 11)

	// Verify sorted by size.
	for i := 1; i < len(datasets); i++ {
		assert.LessOrEqual(t, datasets[i-1].Size, datasets[i].Size,
			"datasets should be sorted by size: %s (%d) > %s (%d)",
			datasets[i-1].Name, datasets[i-1].Size, datasets[i].Name, datasets[i].Size)
	}

	// Verify all datasets have data.
	for _, ds := range datasets {
		assert.NotEmpty(t, ds.Name)
		assert.NotEmpty(t, ds.Type)
		assert.Equal(t, "embedded", ds.Source)
		assert.NotEmpty(t, ds.Data)
		assert.Equal(t, len(ds.Data), ds.Size)
		assert.NotEmpty(t, ds.Description)
	}

	// Verify known dataset.
	var tinyJSON *Dataset
	for _, ds := range datasets {
		if ds.Name == "tiny-json" {
			tinyJSON = &ds
			break
		}
	}
	require.NotNil(t, tinyJSON)
	assert.Equal(t, "json", tinyJSON.Type)
	assert.Equal(t, 36, tinyJSON.Size)
}

func TestInferInputType(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"data.json", "json"},
		{"data.xml", "xml"},
		{"data.bin", "raw"},
		{"data.txt", "raw"},
		{"data.JSON", "json"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := InferInputType(tt.filename)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int
		want  string
	}{
		{36, "36 B"},
		{1024, "1.0 KB"},
		{4742, "4.6 KB"},
		{1048576, "1.0 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatSize(tt.bytes))
		})
	}
}
