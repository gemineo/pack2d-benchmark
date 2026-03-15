package main

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/gemineo/pack2d-benchmark/internal/dataset"
)

func newDatasetsCmd(noColor *bool) *cobra.Command {
	_ = noColor

	return &cobra.Command{
		Use:   "datasets",
		Short: "List embedded datasets",
		Long:  "Display all embedded test datasets with their sizes and descriptions.",
		RunE: func(_ *cobra.Command, _ []string) error {
			datasets, err := dataset.LoadEmbedded()
			if err != nil {
				return fmt.Errorf("load embedded datasets: %w", err)
			}

			t := table.NewWriter()
			t.SetOutputMirror(os.Stdout)
			t.AppendHeader(table.Row{"Name", "Type", "Size", "Description"})
			t.SetStyle(table.StyleLight)

			for _, ds := range datasets {
				t.AppendRow(table.Row{
					ds.Name,
					ds.Type,
					dataset.FormatSize(ds.Size),
					ds.Description,
				})
			}

			t.Render()
			return nil
		},
	}
}
