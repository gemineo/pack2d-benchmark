package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-json-experiment/json"
	"github.com/spf13/cobra"

	"github.com/gemineo/pack2d-benchmark/internal/htmlreport"
	"github.com/gemineo/pack2d-benchmark/internal/report"
)

func newReportCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "report <results.json>",
		Short: "Generate an HTML report from benchmark results",
		Long:  "Read a JSON export file and produce a self-contained HTML report with interactive charts.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) (retErr error) {
			inputPath := args[0]

			// Derive default output path from input.
			if output == "" {
				output = strings.TrimSuffix(inputPath, ".json") + ".html"
			}

			// Read JSON results.
			data, err := os.ReadFile(inputPath)
			if err != nil {
				return fmt.Errorf("read results: %w", err)
			}

			var rpt report.Report
			if err := json.Unmarshal(data, &rpt, json.DefaultOptionsV2()); err != nil {
				return fmt.Errorf("parse results: %w", err)
			}

			// Create output file.
			f, err := os.Create(output)
			if err != nil {
				return fmt.Errorf("create output file: %w", err)
			}
			defer func() {
				if closeErr := f.Close(); closeErr != nil && retErr == nil {
					retErr = fmt.Errorf("close output file: %w", closeErr)
				}
			}()

			if err := htmlreport.Generate(&rpt, f); err != nil {
				return fmt.Errorf("generate report: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Report written to %s\n", output)
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output HTML file path (default: input with .html extension)")

	return cmd
}
