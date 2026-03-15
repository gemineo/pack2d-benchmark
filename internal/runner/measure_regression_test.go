package runner

import (
	"testing"

	"github.com/gemineo/pack2d"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeasureEncode_DoesNotRequireDefensiveCopy(t *testing.T) {
	// Regression: pack2d.Encode previously mutated the input slice via
	// in-place JSON compaction. This was fixed upstream in pack2d v0.2.1
	// (commit f3f68db). Verify that running MeasureEncode multiple times
	// on the same data produces consistent results without data corruption.
	data := []byte(`{
		"id": "usr_001",
		"firstName": "Alice",
		"lastName": "Martin",
		"email": "alice@example.com",
		"age": 34,
		"active": true,
		"tags": ["beta", "premium"]
	}`)

	original := make([]byte, len(data))
	copy(original, data)

	opts := []pack2d.Option{
		pack2d.WithCompression(pack2d.Zstd),
		pack2d.WithCompressionLevel(1),
		pack2d.WithInputType(pack2d.JSON),
	}

	// Run multiple iterations — would fail on iteration 1+ before the fix.
	stats, encoded, p2dStats, err := MeasureEncode(data, opts, 2, 5)
	require.NoError(t, err)

	assert.Greater(t, stats.Mean.Nanoseconds(), int64(0))
	assert.NotEmpty(t, encoded)
	assert.Greater(t, p2dStats.InputBytes, 0)

	// Verify the source data was not corrupted.
	assert.Equal(t, original, data, "MeasureEncode must not mutate the input data")
}

func TestMeasureEncode_SkipsIncompatibleInputTypes(t *testing.T) {
	// Binary data with JSON input type should fail with a clear error,
	// not silently produce garbage.
	binaryData := make([]byte, 256)
	for i := range binaryData {
		binaryData[i] = byte(i)
	}

	opts := []pack2d.Option{
		pack2d.WithCompression(pack2d.Zstd),
		pack2d.WithCompressionLevel(1),
		pack2d.WithInputType(pack2d.JSON),
	}

	_, _, _, err := MeasureEncode(binaryData, opts, 0, 1)
	assert.Error(t, err, "encoding binary data as JSON should fail")
}
