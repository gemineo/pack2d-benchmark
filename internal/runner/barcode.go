package runner

import (
	"context"
	"fmt"
	"math"

	"github.com/gemineo/pack2d"

	"github.com/gemineo/pack2d-benchmark/internal/config"
	"github.com/gemineo/pack2d-benchmark/internal/dataset"
)

// BarcodeScenario finds the best compression config per dataset and reports barcode feasibility.
type BarcodeScenario struct{}

func (s *BarcodeScenario) Name() string        { return "barcode" }
func (s *BarcodeScenario) Description() string  { return "Find best compression for barcode feasibility per dataset" }

func (s *BarcodeScenario) Run(ctx context.Context, datasets []dataset.Dataset, cfg *config.Config, progressFn ProgressFunc) ([]Result, error) {
	var results []Result

	for _, ds := range datasets {
		if err := ctx.Err(); err != nil {
			return results, fmt.Errorf("barcode scenario: %w", err)
		}

		if progressFn != nil {
			progressFn("barcode", ds.Name, "finding best config")
		}

		// Find the config that produces the smallest encoded output.
		bestLen := math.MaxInt
		var bestAlgo pack2d.CompressionType
		var bestLevel int
		var bestInputType pack2d.InputType
		var bestStats pack2d.Stats
		var bestEncTiming TimingStats
		var bestDecTiming TimingStats

		for _, algo := range cfg.Algorithms {
			levels := cfg.Levels[algo]
			if len(levels) == 0 {
				levels = config.DefaultLevels[algo]
			}

			for _, level := range levels {
				for _, inputType := range cfg.InputTypes {
					opts := []pack2d.Option{
						pack2d.WithCompression(algo),
						pack2d.WithCompressionLevel(level),
						pack2d.WithInputType(inputType),
					}

					encTiming, encoded, stats, err := MeasureEncode(ds.Data, opts, cfg.WarmUp, cfg.Iterations)
					if err != nil {
						if isSkippable(err) {
							continue
						}
						return nil, fmt.Errorf("barcode encode %s/%s/L%d/%s: %w", ds.Name, algo, level, inputType, err)
					}

					if len(encoded) < bestLen {
						bestLen = len(encoded)
						bestAlgo = algo
						bestLevel = level
						bestInputType = inputType
						bestStats = stats
						bestEncTiming = encTiming

						decTiming, decErr := MeasureDecode(encoded, opts, cfg.WarmUp, cfg.Iterations)
						if decErr == nil {
							bestDecTiming = decTiming
						}
					}
				}
			}
		}

		if bestLen == math.MaxInt {
			continue // no valid config found
		}

		checks := makeBarcodeChecks(bestLen)

		results = append(results, Result{
			Scenario:    "barcode",
			Dataset:     ds.Name,
			DatasetSize: ds.Size,
			Algorithm:   bestAlgo,
			Level:       bestLevel,
			InputType:   bestInputType,
			InputBytes:  bestStats.InputBytes,
			Compressed:  bestStats.CompressedBytes,
			Encoded:     bestStats.EncodedBytes,
			Ratio:       bestStats.CompressionRatio,
			Encode:      bestEncTiming,
			Decode:      bestDecTiming,
			Barcode:     &BarcodeResult{Checks: checks},
		})
	}

	return results, nil
}
