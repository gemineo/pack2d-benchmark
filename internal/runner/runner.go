package runner

import (
	"context"
	"fmt"

	"github.com/gemineo/pack2d-benchmark/internal/config"
	"github.com/gemineo/pack2d-benchmark/internal/dataset"
)

// Runner orchestrates benchmark scenarios and collects results.
type Runner struct {
	cfg        *config.Config
	datasets   []dataset.Dataset
	progressFn ProgressFunc
}

// New creates a new Runner.
func New(cfg *config.Config, datasets []dataset.Dataset) *Runner {
	return &Runner{
		cfg:      cfg,
		datasets: datasets,
	}
}

// SetProgressFunc sets the progress callback.
func (r *Runner) SetProgressFunc(fn ProgressFunc) {
	r.progressFn = fn
}

// Run executes all configured scenarios and returns all results.
func (r *Runner) Run(ctx context.Context) ([]Result, error) {
	var allResults []Result

	for _, name := range r.cfg.Scenarios {
		scenario, ok := GetScenario(name)
		if !ok {
			return nil, fmt.Errorf("run benchmark: unknown scenario %q", name)
		}

		results, err := scenario.Run(ctx, r.datasets, r.cfg, r.progressFn)
		if err != nil {
			return nil, fmt.Errorf("run scenario %s: %w", name, err)
		}

		allResults = append(allResults, results...)
	}

	return allResults, nil
}
