package report

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/gemineo/pack2d-benchmark/internal/runner"
)

// RenderASCII writes the report as formatted ASCII tables.
func RenderASCII(w io.Writer, rpt *Report, noColor bool) error {
	// Save and restore global color state to avoid side effects.
	prevNoColor := color.NoColor
	color.NoColor = noColor
	defer func() { color.NoColor = prevNoColor }()

	headerColor := color.New(color.FgCyan, color.Bold)
	passColor := color.New(color.FgGreen)
	failColor := color.New(color.FgRed)

	// Header.
	headerColor.Fprintf(w, "pack2d-benchmark %s\n", rpt.Metadata.ToolVersion)
	fmt.Fprintf(w, "Go %s | %s/%s | %s\n",
		rpt.Metadata.GoVersion, rpt.Metadata.OS, rpt.Metadata.Arch,
		rpt.Metadata.Timestamp.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(w, "Iterations: %d | Warm-up: %d\n\n", rpt.Metadata.Iterations, rpt.Metadata.WarmUp)

	// Group results by scenario, then by dataset.
	byScenario := make(map[string][]runner.Result)
	for _, r := range rpt.Results {
		byScenario[r.Scenario] = append(byScenario[r.Scenario], r)
	}

	// Compression results.
	if results, ok := byScenario["compression"]; ok {
		headerColor.Fprintln(w, "═══ Compression Benchmark ═══")
		fmt.Fprintln(w)

		// Check if any result uses a dictionary.
		hasDict := false
		for _, r := range results {
			if r.UseDict {
				hasDict = true
				break
			}
		}

		byDataset := groupByDataset(results)
		for _, dsName := range sortedKeys(byDataset) {
			dsResults := byDataset[dsName]
			fmt.Fprintf(w, "Dataset: %s (%s)\n", dsName, formatSize(dsResults[0].DatasetSize))

			t := table.NewWriter()
			t.SetOutputMirror(w)

			if hasDict {
				t.AppendHeader(table.Row{"Algorithm", "Level", "Input", "Dict", "Compressed", "Encoded", "Ratio", "Encode", "Decode", "QR-M"})
				t.SetColumnConfigs([]table.ColumnConfig{
					{Number: 5, Align: text.AlignRight},
					{Number: 6, Align: text.AlignRight},
					{Number: 7, Align: text.AlignRight},
					{Number: 8, Align: text.AlignRight},
					{Number: 9, Align: text.AlignRight},
					{Number: 10, Align: text.AlignCenter},
				})
			} else {
				t.AppendHeader(table.Row{"Algorithm", "Level", "Input", "Compressed", "Encoded", "Ratio", "Encode", "Decode", "QR-M"})
				t.SetColumnConfigs([]table.ColumnConfig{
					{Number: 4, Align: text.AlignRight},
					{Number: 5, Align: text.AlignRight},
					{Number: 6, Align: text.AlignRight},
					{Number: 7, Align: text.AlignRight},
					{Number: 8, Align: text.AlignRight},
					{Number: 9, Align: text.AlignCenter},
				})
			}

			for _, r := range dsResults {
				qrStatus := "N/A"
				if r.Barcode != nil {
					for _, check := range r.Barcode.Checks {
						if check.BarcodeType == "qrcode" && check.ECLevel == "M" {
							if check.Fits {
								qrStatus = passColor.Sprintf("PASS(V%d)", check.QRVersion)
							} else {
								qrStatus = failColor.Sprint("FAIL")
							}
							break
						}
					}
				}

				if hasDict {
					dictLabel := ""
					if r.UseDict {
						dictLabel = "yes"
					}
					t.AppendRow(table.Row{
						string(r.Algorithm),
						r.Level,
						string(r.InputType),
						dictLabel,
						formatSize(r.Compressed),
						formatSize(r.Encoded),
						fmt.Sprintf("%.2fx", r.Ratio),
						formatDuration(r.Encode.Mean),
						formatDuration(r.Decode.Mean),
						qrStatus,
					})
				} else {
					t.AppendRow(table.Row{
						string(r.Algorithm),
						r.Level,
						string(r.InputType),
						formatSize(r.Compressed),
						formatSize(r.Encoded),
						fmt.Sprintf("%.2fx", r.Ratio),
						formatDuration(r.Encode.Mean),
						formatDuration(r.Decode.Mean),
						qrStatus,
					})
				}
			}

			t.SetStyle(table.StyleLight)
			t.Render()
			fmt.Fprintln(w)
		}
	}

	// Barcode results.
	if results, ok := byScenario["barcode"]; ok {
		headerColor.Fprintln(w, "═══ Barcode Feasibility ═══")
		fmt.Fprintln(w)

		t := table.NewWriter()
		t.SetOutputMirror(w)
		t.AppendHeader(table.Row{"Dataset", "Size", "Best Config", "Encoded", "QR-L", "QR-M", "QR-Q", "QR-H", "DM"})
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 5, Align: text.AlignCenter},
			{Number: 6, Align: text.AlignCenter},
			{Number: 7, Align: text.AlignCenter},
			{Number: 8, Align: text.AlignCenter},
			{Number: 9, Align: text.AlignCenter},
		})

		for _, r := range results {
			configStr := fmt.Sprintf("%s/L%d/%s", r.Algorithm, r.Level, r.InputType)
			if r.UseDict {
				configStr += "+dict"
			}

			row := table.Row{
				r.Dataset,
				formatSize(r.DatasetSize),
				configStr,
				formatSize(r.Encoded),
			}

			if r.Barcode != nil {
				for _, ecLevel := range []string{"L", "M", "Q", "H"} {
					row = append(row, formatCheck(r.Barcode.Checks, "qrcode", ecLevel, passColor, failColor))
				}
				row = append(row, formatCheck(r.Barcode.Checks, "datamatrix", "ECC200", passColor, failColor))
			}

			t.AppendRow(row)
		}

		t.SetStyle(table.StyleLight)
		t.Render()
		fmt.Fprintln(w)
	}

	// Summary.
	if rpt.Summary != nil && len(rpt.Summary.Recommendations) > 0 {
		headerColor.Fprintln(w, "═══ Summary ═══")
		fmt.Fprintln(w)
		for _, rec := range rpt.Summary.Recommendations {
			fmt.Fprintf(w, "  • %s\n", rec)
		}
		fmt.Fprintln(w)
	}

	return nil
}

func formatCheck(checks []runner.BarcodeCheck, barcodeType, ecLevel string, pass, fail *color.Color) string {
	for _, c := range checks {
		if c.BarcodeType == barcodeType && c.ECLevel == ecLevel {
			if c.Fits {
				if c.QRVersion > 0 {
					return pass.Sprintf("V%d", c.QRVersion)
				}
				return pass.Sprint("PASS")
			}
			return fail.Sprint("FAIL")
		}
	}
	return "N/A"
}

func formatSize(bytes int) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
}

func formatDuration(d time.Duration) string {
	us := d.Microseconds()
	switch {
	case us < 1000:
		return fmt.Sprintf("%dµs", us)
	case us < 1_000_000:
		return fmt.Sprintf("%.1fms", float64(us)/1000)
	default:
		return fmt.Sprintf("%.2fs", float64(us)/1_000_000)
	}
}

func groupByDataset(results []runner.Result) map[string][]runner.Result {
	m := make(map[string][]runner.Result)
	for _, r := range results {
		m[r.Dataset] = append(m[r.Dataset], r)
	}
	return m
}

func sortedKeys(m map[string][]runner.Result) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
