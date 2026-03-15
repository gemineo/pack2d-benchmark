package main

import (
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	var quiet bool
	var noColor bool

	cmd := &cobra.Command{
		Use:   "pack2d-benchmark",
		Short: "Benchmark tool for pack2d compression and barcode configurations",
		Long:  "Evaluate compression configurations, barcode feasibility, and performance trade-offs for the pack2d encoding pipeline.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress progress output")
	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	cmd.AddCommand(newRunCmd(&quiet, &noColor))
	cmd.AddCommand(newDatasetsCmd(&noColor))
	cmd.AddCommand(newVersionCmd())

	return cmd
}
