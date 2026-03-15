package report

import (
	"fmt"
	"io"

	"github.com/go-json-experiment/json"
)

// ExportJSON writes the report as JSON to the writer.
func ExportJSON(w io.Writer, rpt *Report) error {
	data, err := json.Marshal(rpt, json.DefaultOptionsV2())
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	_, err = w.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("write newline: %w", err)
	}
	return nil
}
