package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gemineo/pack2d"
	"github.com/gemineo/pack2d/dict"

	"github.com/gemineo/pack2d-benchmark/internal/config"
	"github.com/gemineo/pack2d-benchmark/internal/dataset"
)

// Errors that indicate an incompatible dataset/input-type combination
// rather than a real failure. These are silently skipped.
var skippableErrors = []error{
	pack2d.ErrUnknownSerializer,
	pack2d.ErrInvalidEncoding,
}

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

			// Determine dict modes: always run without dict; also run with dict for zstd when available.
			type dictMode struct {
				useDict bool
				d       *dict.Dictionary
			}
			modes := []dictMode{{useDict: false}}
			if cfg.Dict != nil && algo == pack2d.Zstd {
				modes = append(modes, dictMode{useDict: true, d: cfg.Dict})
			}

			for _, level := range levels {
				for _, inputType := range cfg.InputTypes {
					for _, dm := range modes {
						if err := ctx.Err(); err != nil {
							return results, fmt.Errorf("compression scenario: %w", err)
						}

						label := fmt.Sprintf("%s/L%d/%s", algo, level, inputType)
						if dm.useDict {
							label += "+dict"
						}
						if progressFn != nil {
							progressFn("compression", ds.Name, label)
						}

						opts := []pack2d.Option{
							pack2d.WithCompression(algo),
							pack2d.WithCompressionLevel(level),
							pack2d.WithInputType(inputType),
						}
						if dm.useDict {
							opts = append(opts, pack2d.WithDictionary(dm.d))
						}

						encStats, encoded, p2dStats, err := MeasureEncode(ds.Data, opts, cfg.WarmUp, cfg.Iterations)
						if err != nil {
							if isSkippable(err) {
								continue
							}
							return nil, fmt.Errorf("compression encode %s/%s: %w", ds.Name, label, err)
						}

						// For decode with dictionary, provide a dict store.
						decOpts := opts
						if dm.useDict {
							store := dict.NewMemoryStore()
							if saveErr := store.Save(dm.d); saveErr != nil {
								return nil, fmt.Errorf("compression decode dict store %s/%s: %w", ds.Name, label, saveErr)
							}
							decOpts = []pack2d.Option{
								pack2d.WithCompression(algo),
								pack2d.WithCompressionLevel(level),
								pack2d.WithInputType(inputType),
								pack2d.WithDictStore(store),
							}
						}

						decStats, err := MeasureDecode(encoded, decOpts, cfg.WarmUp, cfg.Iterations)
						if err != nil {
							return nil, fmt.Errorf("compression decode %s/%s: %w", ds.Name, label, err)
						}

						// Barcode feasibility checks.
						encodedLen := len(encoded)
						checks := makeBarcodeChecks(encodedLen, cfg.ModuleSizeMM)

						results = append(results, Result{
							Scenario:    "compression",
							Dataset:     ds.Name,
							DatasetSize: ds.Size,
							Algorithm:   algo,
							Level:       level,
							InputType:   inputType,
							UseDict:     dm.useDict,
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
	}

	return results, nil
}

func makeBarcodeChecks(encodedLen int, moduleMM float64) []BarcodeCheck {
	ecLevels := []string{"L", "M", "Q", "H"}
	var checks []BarcodeCheck

	for _, ec := range ecLevels {
		maxCap := QRMaxCapacity(ec)
		version, fits := QRVersionForSize(encodedLen, ec)
		usage := 0.0
		if maxCap > 0 {
			usage = float64(encodedLen) / float64(maxCap) * 100
		}

		var modules int
		var sizeMM float64
		if fits {
			// Compute actual capacity of the matched version.
			versionCap := qrAlphanumericCapacity[version-1][ECLevelIndex(ec)]
			usage = float64(encodedLen) / float64(versionCap) * 100
			modules = QRModules(version)
			sizeMM = QRSizeMM(version, moduleMM)
		}

		checks = append(checks, BarcodeCheck{
			BarcodeType: "qrcode",
			ECLevel:     ec,
			MaxCapacity: maxCap,
			EncodedLen:  encodedLen,
			Fits:        fits,
			QRVersion:   version,
			Usage:       usage,
			Modules:     modules,
			SizeMM:      sizeMM,
		})
	}

	// DataMatrix check.
	dmFits := DataMatrixFits(encodedLen)
	dmUsage := float64(encodedLen) / float64(dataMatrixMaxCapacity) * 100
	var dmModules int
	var dmSizeMM float64
	if dmFits {
		dmModules = DataMatrixModules(encodedLen)
		dmSizeMM = DataMatrixSizeMM(encodedLen, moduleMM)
	}
	checks = append(checks, BarcodeCheck{
		BarcodeType: "datamatrix",
		ECLevel:     "ECC200",
		MaxCapacity: dataMatrixMaxCapacity,
		EncodedLen:  encodedLen,
		Fits:        dmFits,
		Usage:       dmUsage,
		Modules:     dmModules,
		SizeMM:      dmSizeMM,
	})

	return checks
}

func isSkippable(err error) bool {
	for _, target := range skippableErrors {
		if errors.Is(err, target) {
			return true
		}
	}
	// Serialization errors (e.g. XML data through JSON serializer or vice versa)
	// indicate an incompatible dataset/input-type combination.
	if strings.Contains(err.Error(), "pack2d encode: serialize:") {
		return true
	}
	return false
}
