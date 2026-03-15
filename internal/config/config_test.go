package config

import (
	"testing"

	"github.com/gemineo/pack2d"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, "ascii", cfg.Format)
	assert.Equal(t, 20, cfg.Iterations)
	assert.Equal(t, 3, cfg.WarmUp)
	assert.Len(t, cfg.Algorithms, 3)
	assert.Len(t, cfg.InputTypes, 3)
	assert.Len(t, cfg.Scenarios, 2)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{"default is valid", func(_ *Config) {}, false},
		{"zero iterations", func(c *Config) { c.Iterations = 0 }, true},
		{"negative warm-up", func(c *Config) { c.WarmUp = -1 }, true},
		{"unknown scenario", func(c *Config) { c.Scenarios = []string{"unknown"} }, true},
		{"unknown algorithm", func(c *Config) { c.Algorithms = []pack2d.CompressionType{"lz4"} }, true},
		{"unknown format", func(c *Config) { c.Format = "html" }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseAlgorithms(t *testing.T) {
	algos, err := ParseAlgorithms("zlib,zstd")
	require.NoError(t, err)
	assert.Len(t, algos, 2)

	_, err = ParseAlgorithms("unknown")
	assert.Error(t, err)
}

func TestParseInputTypes(t *testing.T) {
	types, err := ParseInputTypes("raw,json")
	require.NoError(t, err)
	assert.Len(t, types, 2)

	_, err = ParseInputTypes("foobar")
	assert.Error(t, err)
}

func TestParseScenarios(t *testing.T) {
	scenarios, err := ParseScenarios("compression,barcode")
	require.NoError(t, err)
	assert.Len(t, scenarios, 2)

	_, err = ParseScenarios("unknown")
	assert.Error(t, err)
}
