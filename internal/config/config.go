package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gemineo/pack2d"
	"github.com/gemineo/pack2d/dict"
)

// DefaultLevels defines the default compression levels to benchmark per algorithm.
// Full ranges for comprehensive benchmarking.
var DefaultLevels = map[pack2d.CompressionType][]int{
	pack2d.Zlib:   {1, 2, 3, 4, 5, 6, 7, 8, 9},
	pack2d.Zstd:   {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
	pack2d.Brotli: {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
}

// QuickLevels defines a reduced set of levels for fast runs.
var QuickLevels = map[pack2d.CompressionType][]int{
	pack2d.Zlib:   {1, 6, 9},
	pack2d.Zstd:   {1, 9, 19},
	pack2d.Brotli: {1, 6, 11},
}

// Config holds the benchmark configuration parsed from CLI flags.
type Config struct {
	DataDir    string
	Format     string
	Output     string
	Export     string
	Scenarios  []string
	Algorithms []pack2d.CompressionType
	Levels     map[pack2d.CompressionType][]int
	InputTypes []pack2d.InputType
	Iterations int
	WarmUp     int
	DictPath string
	Dict     *dict.Dictionary
	Quiet    bool
	NoColor  bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Format:     "ascii",
		Scenarios:  []string{"compression", "barcode"},
		Algorithms: []pack2d.CompressionType{pack2d.Zlib, pack2d.Zstd, pack2d.Brotli},
		Levels:     DefaultLevels,
		InputTypes: []pack2d.InputType{pack2d.Raw, pack2d.JSON, pack2d.CBOR},
		Iterations: 20,
		WarmUp:     3,
	}
}

// Validate checks that the config values are acceptable.
func (c *Config) Validate() error {
	if c.Iterations < 1 {
		return fmt.Errorf("validate config: iterations must be >= 1, got %d", c.Iterations)
	}
	if c.WarmUp < 0 {
		return fmt.Errorf("validate config: warm-up must be >= 0, got %d", c.WarmUp)
	}
	for _, s := range c.Scenarios {
		switch s {
		case "compression", "barcode":
		default:
			return fmt.Errorf("validate config: unknown scenario %q", s)
		}
	}
	for _, a := range c.Algorithms {
		switch a {
		case pack2d.Zlib, pack2d.Zstd, pack2d.Brotli:
		default:
			return fmt.Errorf("validate config: unknown algorithm %q", a)
		}
	}
	for _, it := range c.InputTypes {
		switch it {
		case pack2d.Raw, pack2d.JSON, pack2d.XML, pack2d.CBOR:
		default:
			return fmt.Errorf("validate config: unknown input type %q", it)
		}
	}
	if c.Format != "ascii" {
		return fmt.Errorf("validate config: unsupported format %q (only \"ascii\" supported)", c.Format)
	}
	return nil
}

// ParseAlgorithms parses a comma-separated string of algorithm names.
func ParseAlgorithms(s string) ([]pack2d.CompressionType, error) {
	parts := strings.Split(s, ",")
	result := make([]pack2d.CompressionType, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		ct := pack2d.CompressionType(p)
		switch ct {
		case pack2d.Zlib, pack2d.Zstd, pack2d.Brotli:
			result = append(result, ct)
		default:
			return nil, fmt.Errorf("parse algorithms: unknown algorithm %q", p)
		}
	}
	return result, nil
}

// ParseInputTypes parses a comma-separated string of input type names.
func ParseInputTypes(s string) ([]pack2d.InputType, error) {
	parts := strings.Split(s, ",")
	result := make([]pack2d.InputType, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		it := pack2d.InputType(p)
		switch it {
		case pack2d.Raw, pack2d.JSON, pack2d.XML, pack2d.CBOR:
			result = append(result, it)
		default:
			return nil, fmt.Errorf("parse input types: unknown input type %q", p)
		}
	}
	return result, nil
}

// ParseLevels parses a comma-separated string of compression levels.
// The resulting levels are applied uniformly to all algorithms.
func ParseLevels(s string, algorithms []pack2d.CompressionType) (map[pack2d.CompressionType][]int, error) {
	parts := strings.Split(s, ",")
	var levels []int
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("parse levels: invalid level %q: %w", p, err)
		}
		if n < 0 || n > 19 {
			return nil, fmt.Errorf("parse levels: level %d out of range [0, 19]", n)
		}
		levels = append(levels, n)
	}
	if len(levels) == 0 {
		return nil, fmt.Errorf("parse levels: no valid levels provided")
	}
	result := make(map[pack2d.CompressionType][]int, len(algorithms))
	for _, a := range algorithms {
		result[a] = levels
	}
	return result, nil
}

// ParseScenarios parses a comma-separated string of scenario names.
func ParseScenarios(s string) ([]string, error) {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		switch p {
		case "compression", "barcode":
			result = append(result, p)
		default:
			return nil, fmt.Errorf("parse scenarios: unknown scenario %q", p)
		}
	}
	return result, nil
}
