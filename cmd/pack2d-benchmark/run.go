package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/gemineo/pack2d"
	"github.com/gemineo/pack2d/dict"
	"github.com/spf13/cobra"

	"github.com/gemineo/pack2d-benchmark/internal/config"
	"github.com/gemineo/pack2d-benchmark/internal/dataset"
	"github.com/gemineo/pack2d-benchmark/internal/report"
	"github.com/gemineo/pack2d-benchmark/internal/runner"
)

func newRunCmd(quiet, noColor *bool) *cobra.Command {
	var (
		dataDir    string
		format     string
		output     string
		export     string
		scenarios  string
		algorithms string
		levels     string
		iterations int
		inputTypes string
		warmUp     int
		dictPath   string
		moduleSize float64
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run benchmarks",
		Long:  "Execute benchmark scenarios against embedded or custom datasets.",
		RunE: func(cmd *cobra.Command, _ []string) (retErr error) {
			cfg := config.DefaultConfig()
			cfg.Quiet = *quiet
			cfg.NoColor = *noColor
			cfg.DataDir = dataDir
			cfg.Format = format
			cfg.Output = output
			cfg.Export = export
			cfg.DictPath = dictPath
			cfg.Iterations = iterations
			cfg.WarmUp = warmUp
			if moduleSize > 0 {
				cfg.ModuleSizeMM = moduleSize
			}

			if scenarios != "" {
				parsed, err := config.ParseScenarios(scenarios)
				if err != nil {
					return err
				}
				cfg.Scenarios = parsed
			}

			if algorithms != "" {
				parsed, err := config.ParseAlgorithms(algorithms)
				if err != nil {
					return err
				}
				cfg.Algorithms = parsed

				// Rebuild levels for selected algorithms only.
				newLevels := make(map[pack2d.CompressionType][]int)
				for _, a := range cfg.Algorithms {
					if lvls, ok := config.DefaultLevels[a]; ok {
						newLevels[a] = lvls
					}
				}
				cfg.Levels = newLevels
			}

			if inputTypes != "" {
				parsed, err := config.ParseInputTypes(inputTypes)
				if err != nil {
					return err
				}
				cfg.InputTypes = parsed
			}

			if levels != "" {
				parsed, err := config.ParseLevels(levels, cfg.Algorithms)
				if err != nil {
					return err
				}
				cfg.Levels = parsed
			}

			if err := cfg.Validate(); err != nil {
				return err
			}

			// Load datasets.
			var datasets []dataset.Dataset
			if cfg.DataDir != "" {
				custom, err := dataset.LoadCustom(cfg.DataDir)
				if err != nil {
					return fmt.Errorf("load custom datasets: %w", err)
				}
				datasets = custom
			} else {
				embedded, err := dataset.LoadEmbedded()
				if err != nil {
					return fmt.Errorf("load embedded datasets: %w", err)
				}
				datasets = embedded
			}

			if len(datasets) == 0 {
				return fmt.Errorf("no datasets found")
			}

			// Load or auto-train dictionary.
			if cfg.DictPath != "" {
				d, err := loadOrTrainDict(cfg.DictPath, datasets)
				if err != nil {
					return err
				}
				cfg.Dict = d
			}

			// Setup runner.
			r := runner.New(cfg, datasets)

			// Setup spinner for progress.
			var sp *spinner.Spinner
			if !cfg.Quiet {
				sp = spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
				sp.Prefix = " "
				defer func() {
					if sp.Active() {
						sp.Stop()
					}
				}()
				r.SetProgressFunc(func(scenario, ds, detail string) {
					sp.Suffix = fmt.Sprintf(" [%s] %s — %s", scenario, ds, detail)
					if !sp.Active() {
						sp.Start()
					}
				})
			}

			// Run.
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			results, err := r.Run(ctx)
			if err != nil {
				return err
			}

			// Build report.
			rpt := report.BuildReport(results, datasets, version, cfg)

			// Output.
			out := os.Stdout
			if cfg.Output != "" {
				f, openErr := os.Create(cfg.Output)
				if openErr != nil {
					return fmt.Errorf("create output file: %w", openErr)
				}
				defer func() {
					if closeErr := f.Close(); closeErr != nil && retErr == nil {
						retErr = fmt.Errorf("close output file: %w", closeErr)
					}
				}()
				out = f
			}

			if err := report.RenderASCII(out, rpt, cfg.NoColor); err != nil {
				return fmt.Errorf("render report: %w", err)
			}

			// JSON export.
			if cfg.Export != "" {
				f, openErr := os.Create(cfg.Export)
				if openErr != nil {
					return fmt.Errorf("create export file: %w", openErr)
				}
				defer func() {
					if closeErr := f.Close(); closeErr != nil && retErr == nil {
						retErr = fmt.Errorf("close export file: %w", closeErr)
					}
				}()
				if err := report.ExportJSON(f, rpt); err != nil {
					return fmt.Errorf("export JSON: %w", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&dataDir, "data", "", "Path to custom dataset directory")
	cmd.Flags().StringVar(&format, "format", "ascii", "Output format (ascii)")
	cmd.Flags().StringVar(&output, "output", "", "Write output to file instead of stdout")
	cmd.Flags().StringVar(&export, "export", "", "Export results as JSON to file")
	cmd.Flags().StringVar(&scenarios, "scenarios", "", "Comma-separated scenarios (compression,barcode)")
	cmd.Flags().StringVar(&algorithms, "algorithms", "", "Comma-separated algorithms (zlib,zstd,brotli)")
	cmd.Flags().StringVar(&levels, "levels", "", "Comma-separated compression levels")
	cmd.Flags().IntVar(&iterations, "iterations", 20, "Number of benchmark iterations")
	cmd.Flags().StringVar(&inputTypes, "input-types", "", "Comma-separated input types (raw,json)")
	cmd.Flags().IntVar(&warmUp, "warm-up", 3, "Number of warm-up iterations")
	cmd.Flags().StringVar(&dictPath, "dict", "", "Path to zstd dictionary file, or \"auto\" to train from datasets")
	cmd.Flags().Float64Var(&moduleSize, "module-size", 0.33, "Barcode module size in mm for physical dimension calculations")

	return cmd
}

// loadOrTrainDict loads a dictionary from file or auto-trains one from datasets.
func loadOrTrainDict(path string, datasets []dataset.Dataset) (*dict.Dictionary, error) {
	if path == "auto" {
		return trainDictFromDatasets(datasets)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load dictionary: %w", err)
	}

	return &dict.Dictionary{
		ID:   1,
		Name: "user-provided",
		Data: data,
	}, nil
}

// trainDictFromDatasets trains a zstd dictionary from all available datasets.
func trainDictFromDatasets(datasets []dataset.Dataset) (*dict.Dictionary, error) {
	var samples [][]byte
	for _, ds := range datasets {
		if len(ds.Data) > 0 {
			samples = append(samples, ds.Data)
		}
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("train dictionary: no samples available")
	}

	dictData, err := dict.Train(samples, "zstd")
	if err != nil {
		return nil, fmt.Errorf("train dictionary: %w", err)
	}

	return &dict.Dictionary{
		ID:          1,
		Name:        "auto-trained",
		Data:        dictData,
		SampleCount: len(samples),
	}, nil
}
