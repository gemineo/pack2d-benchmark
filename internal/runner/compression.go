package runner

import (
	"context"
	"fmt"

	"github.com/gemineo/pack2d"

	"github.com/gemineo/pack2d-benchmark/internal/config"
	"github.com/gemineo/pack2d-benchmark/internal/dataset"
)

// CompressionScenario benchmarks all algorithm×level×inputType combinations.
type CompressionScenario struct{}

func (s *CompressionScenario) Name() string        { return "compression" }
func (s *CompressionScenario) Description() string  { return "Benchmark compression algorithms across datasets" }

func (s *CompressionScenario) Run(ctx context.Context, datasets []dataset.Dataset, cfg *config.Config, progressFn ProgressFunc) ([]Result, error) {
	var results []Result

	for _, ds := range datasets {
		for _, algo := range cfg.Algorithms {
			levels := cfg.Levels[algo]
			if len(levels) == 0 {
				levels = config.DefaultLevels[algo]
			}

			for _, level := range levels {
				for _, inputType := range cfg.InputTypes {
					if err := ctx.Err(); err != nil {
						return results, fmt.Errorf("compression scenario: %w", err)
					}

					if progressFn != nil {
						progressFn("compression", ds.Name, fmt.Sprintf("%s/L%d/%s", algo, level, inputType))
					}

					opts := []pack2d.Option{
						pack2d.WithCompression(algo),
						pack2d.WithCompressionLevel(level),
						pack2d.WithInputType(inputType),
					}

					encStats, encoded, p2dStats, err := MeasureEncode(ds.Data, opts, cfg.WarmUp, cfg.Iterations)
					if err != nil {
						continue // skip incompatible combinations (e.g., binary data with JSON input type)
					}

					decStats, err := MeasureDecode(encoded, opts, cfg.WarmUp, cfg.Iterations)
					if err != nil {
						continue
					}

					// Barcode feasibility checks.
					encodedLen := len(encoded)
					checks := makeBarcodeChecks(encodedLen)

					results = append(results, Result{
						Scenario:    "compression",
						Dataset:     ds.Name,
						DatasetSize: ds.Size,
						Algorithm:   algo,
						Level:       level,
						InputType:   inputType,
						InputBytes:  p2dStats.InputBytes,
						Compressed:  p2dStats.CompressedBytes,
						Encoded:     p2dStats.EncodedBytes,
						Ratio:       p2dStats.CompressionRatio,
						Encode:      encStats,
						Decode:      decStats,
						Barcode:     &BarcodeResult{Checks: checks},
					})
				}
			}
		}
	}

	return results, nil
}

func makeBarcodeChecks(encodedLen int) []BarcodeCheck {
	ecLevels := []string{"L", "M", "Q", "H"}
	var checks []BarcodeCheck

	for _, ec := range ecLevels {
		maxCap := QRMaxCapacity(ec)
		version, fits := QRVersionForSize(encodedLen, ec)
		usage := 0.0
		if maxCap > 0 {
			usage = float64(encodedLen) / float64(maxCap) * 100
		}
		if fits {
			// Compute actual capacity of the matched version.
			versionCap := qrAlphanumericCapacity[version-1][ECLevelIndex(ec)]
			usage = float64(encodedLen) / float64(versionCap) * 100
		}

		checks = append(checks, BarcodeCheck{
			BarcodeType: "qrcode",
			ECLevel:     ec,
			MaxCapacity: maxCap,
			EncodedLen:  encodedLen,
			Fits:        fits,
			QRVersion:   version,
			Usage:       usage,
		})
	}

	// DataMatrix check.
	dmFits := DataMatrixFits(encodedLen)
	dmUsage := float64(encodedLen) / float64(dataMatrixMaxCapacity) * 100
	checks = append(checks, BarcodeCheck{
		BarcodeType: "datamatrix",
		ECLevel:     "ECC200",
		MaxCapacity: dataMatrixMaxCapacity,
		EncodedLen:  encodedLen,
		Fits:        dmFits,
		Usage:       dmUsage,
	})

	return checks
}
