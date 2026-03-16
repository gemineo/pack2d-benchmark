package report

import (
	"runtime"
	"time"

	"github.com/gemineo/pack2d-benchmark/internal/config"
	"github.com/gemineo/pack2d-benchmark/internal/dataset"
	"github.com/gemineo/pack2d-benchmark/internal/runner"
)

// Report is the top-level benchmark report.
type Report struct {
	Metadata Metadata        `json:"metadata"`
	Datasets []DatasetInfo   `json:"datasets"`
	Results  []runner.Result `json:"results"`
	Summary  *Summary        `json:"summary"`
}

// Metadata contains information about the benchmark run.
type Metadata struct {
	ToolVersion   string    `json:"toolVersion"`
	Pack2dVersion string    `json:"pack2dVersion,omitempty"`
	GoVersion     string    `json:"goVersion"`
	OS            string    `json:"os"`
	Arch          string    `json:"arch"`
	Timestamp     time.Time `json:"timestamp"`
	Iterations    int       `json:"iterations"`
	WarmUp        int       `json:"warmUp"`
	ModuleSizeMM  float64   `json:"moduleSizeMM,omitempty"`
}

// DatasetInfo describes a dataset used in the benchmark.
type DatasetInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Size        int    `json:"size"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

// Summary contains computed recommendations from the results.
type Summary struct {
	SweetSpot       []SweetSpotEntry     `json:"sweetSpot,omitempty"`
	BestRatio       []BestEntry          `json:"bestRatio,omitempty"`
	BestSpeed       []BestEntry          `json:"bestSpeed,omitempty"`
	QRFitCounts     map[string]int       `json:"qrFitCounts,omitempty"`
	Recommendations []string             `json:"recommendations,omitempty"`
}

// SweetSpotEntry identifies the recommended config for a dataset.
// Found is false when no config exceeded the marginal improvement threshold;
// in that case the entry falls back to the fastest config.
type SweetSpotEntry struct {
	Dataset   string  `json:"dataset"`
	Algorithm string  `json:"algorithm"`
	Level     int     `json:"level"`
	InputType string  `json:"inputType"`
	UseDict   bool    `json:"useDict,omitempty"`
	Ratio     float64 `json:"ratio"`
	EncodeUs  int64   `json:"encodeUs"`
	Found     bool    `json:"found"`
}

// BestEntry records a best-in-category config.
type BestEntry struct {
	Dataset   string  `json:"dataset"`
	Algorithm string  `json:"algorithm"`
	Level     int     `json:"level"`
	InputType string  `json:"inputType"`
	UseDict   bool    `json:"useDict,omitempty"`
	Ratio     float64 `json:"ratio"`
	EncodeUs  int64   `json:"encodeUs"`
}

// BuildReport constructs a Report from runner results, datasets, and config.
func BuildReport(results []runner.Result, datasets []dataset.Dataset, toolVersion string, cfg *config.Config) *Report {
	rpt := &Report{
		Metadata: Metadata{
			ToolVersion:  toolVersion,
			GoVersion:    runtime.Version(),
			OS:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			Timestamp:    time.Now().UTC(),
			Iterations:   cfg.Iterations,
			WarmUp:       cfg.WarmUp,
			ModuleSizeMM: cfg.ModuleSizeMM,
		},
		Datasets: make([]DatasetInfo, len(datasets)),
		Results:  results,
	}

	for i, ds := range datasets {
		rpt.Datasets[i] = DatasetInfo{
			Name:        ds.Name,
			Type:        ds.Type,
			Size:        ds.Size,
			Source:      ds.Source,
			Description: ds.Description,
		}
	}

	rpt.Summary = ComputeSummary(results)

	return rpt
}
