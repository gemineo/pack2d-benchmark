package runner

import (
	"context"
	"time"

	"github.com/gemineo/pack2d"

	"github.com/gemineo/pack2d-benchmark/internal/config"
	"github.com/gemineo/pack2d-benchmark/internal/dataset"
)

// Scenario defines a benchmark scenario that can be run.
type Scenario interface {
	Name() string
	Description() string
	Run(ctx context.Context, datasets []dataset.Dataset, cfg *config.Config, progressFn ProgressFunc) ([]Result, error)
}

// ProgressFunc reports progress during scenario execution.
type ProgressFunc func(scenario, dataset, detail string)

// Result holds the outcome of a single benchmark measurement.
type Result struct {
	Scenario    string              `json:"scenario"`
	Dataset     string              `json:"dataset"`
	DatasetSize int                 `json:"datasetSize"`
	Algorithm   pack2d.CompressionType `json:"algorithm"`
	Level       int                 `json:"level"`
	InputType   pack2d.InputType    `json:"inputType"`
	InputBytes  int                 `json:"inputBytes"`
	Compressed  int                 `json:"compressedBytes"`
	Encoded     int                 `json:"encodedBytes"`
	Ratio       float64             `json:"compressionRatio"`
	Encode      TimingStats         `json:"encodeTiming"`
	Decode      TimingStats         `json:"decodeTiming"`
	Barcode     *BarcodeResult      `json:"barcode,omitempty"`
}

// TimingStats holds timing statistics from repeated measurements.
type TimingStats struct {
	Mean   time.Duration `json:"mean,format:nano"`
	Median time.Duration `json:"median,format:nano"`
	P95    time.Duration `json:"p95,format:nano"`
	P99    time.Duration `json:"p99,format:nano"`
	Min    time.Duration `json:"min,format:nano"`
	Max    time.Duration `json:"max,format:nano"`
	StdDev time.Duration `json:"stdDev,format:nano"`
}

// BarcodeResult holds barcode feasibility information.
type BarcodeResult struct {
	Checks []BarcodeCheck `json:"checks"`
}

// BarcodeCheck represents a feasibility check for a specific barcode/EC config.
type BarcodeCheck struct {
	BarcodeType string `json:"barcodeType"`
	ECLevel     string `json:"ecLevel"`
	MaxCapacity int    `json:"maxCapacity"`
	EncodedLen  int    `json:"encodedLen"`
	Fits        bool   `json:"fits"`
	QRVersion   int    `json:"qrVersion,omitempty"`
	Usage       float64 `json:"usagePercent"`
}

// scenarioRegistry stores registered scenarios.
var scenarioRegistry = map[string]Scenario{}

// RegisterScenario adds a scenario to the global registry.
func RegisterScenario(s Scenario) {
	scenarioRegistry[s.Name()] = s
}

// GetScenario returns a registered scenario by name.
func GetScenario(name string) (Scenario, bool) {
	s, ok := scenarioRegistry[name]
	return s, ok
}

func init() {
	RegisterScenario(&CompressionScenario{})
	RegisterScenario(&BarcodeScenario{})
}
