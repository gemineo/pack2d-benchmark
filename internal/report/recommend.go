package report

import (
	"fmt"
	"sort"

	"github.com/gemineo/pack2d-benchmark/internal/runner"
)

// ComputeSummary analyzes results and produces recommendations.
func ComputeSummary(results []runner.Result) *Summary {
	s := &Summary{
		QRFitCounts: make(map[string]int),
	}

	// Group compression results by dataset.
	byDataset := make(map[string][]runner.Result)
	for _, r := range results {
		if r.Scenario == "compression" {
			byDataset[r.Dataset] = append(byDataset[r.Dataset], r)
		}
	}

	for dataset, dResults := range byDataset {
		if len(dResults) == 0 {
			continue
		}

		// Best ratio (lowest = most compression).
		bestRatio := dResults[0]
		for _, r := range dResults[1:] {
			if r.Ratio < bestRatio.Ratio {
				bestRatio = r
			}
		}
		s.BestRatio = append(s.BestRatio, BestEntry{
			Dataset:   dataset,
			Algorithm: string(bestRatio.Algorithm),
			Level:     bestRatio.Level,
			InputType: string(bestRatio.InputType),
			UseDict:   bestRatio.UseDict,
			Ratio:     bestRatio.Ratio,
			EncodeUs:  bestRatio.Encode.Mean.Microseconds(),
		})

		// Best speed (lowest encode time).
		bestSpeed := dResults[0]
		for _, r := range dResults[1:] {
			if r.Encode.Mean < bestSpeed.Encode.Mean {
				bestSpeed = r
			}
		}
		s.BestSpeed = append(s.BestSpeed, BestEntry{
			Dataset:   dataset,
			Algorithm: string(bestSpeed.Algorithm),
			Level:     bestSpeed.Level,
			InputType: string(bestSpeed.InputType),
			UseDict:   bestSpeed.UseDict,
			Ratio:     bestSpeed.Ratio,
			EncodeUs:  bestSpeed.Encode.Mean.Microseconds(),
		})

		// Sweet spot: sort by encode time, find last config where marginal ratio
		// improvement exceeds 0.05% per µs.
		sorted := make([]runner.Result, len(dResults))
		copy(sorted, dResults)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Encode.Mean < sorted[j].Encode.Mean
		})

		sweetIdx := 0
		sweetFound := false
		for i := 1; i < len(sorted); i++ {
			timeDelta := sorted[i].Encode.Mean.Microseconds() - sorted[i-1].Encode.Mean.Microseconds()
			if timeDelta <= 0 {
				continue
			}
			if sorted[i-1].Ratio == 0 {
				continue
			}
			// Ratio = compressed/original, so improvement = ratio going down.
			ratioDrop := sorted[i-1].Ratio - sorted[i].Ratio
			marginal := (ratioDrop / sorted[i-1].Ratio * 100) / float64(timeDelta)
			if marginal > 0.05 {
				sweetIdx = i
				sweetFound = true
			}
		}

		sweet := sorted[sweetIdx]
		s.SweetSpot = append(s.SweetSpot, SweetSpotEntry{
			Dataset:   dataset,
			Algorithm: string(sweet.Algorithm),
			Level:     sweet.Level,
			InputType: string(sweet.InputType),
			UseDict:   sweet.UseDict,
			Ratio:     sweet.Ratio,
			EncodeUs:  sweet.Encode.Mean.Microseconds(),
			Found:     sweetFound,
		})

		// Count QR fits.
		for _, r := range dResults {
			if r.Barcode == nil {
				continue
			}
			for _, check := range r.Barcode.Checks {
				if check.Fits && check.BarcodeType == "qrcode" {
					key := fmt.Sprintf("QR-%s", check.ECLevel)
					s.QRFitCounts[key]++
				}
			}
		}
	}

	// Sort for deterministic output.
	sort.Slice(s.SweetSpot, func(i, j int) bool { return s.SweetSpot[i].Dataset < s.SweetSpot[j].Dataset })
	sort.Slice(s.BestRatio, func(i, j int) bool { return s.BestRatio[i].Dataset < s.BestRatio[j].Dataset })
	sort.Slice(s.BestSpeed, func(i, j int) bool { return s.BestSpeed[i].Dataset < s.BestSpeed[j].Dataset })

	// Generate textual recommendations.
	s.Recommendations = generateRecommendations(s)

	return s
}

func generateRecommendations(s *Summary) []string {
	var recs []string

	for _, ss := range s.SweetSpot {
		label := "Sweet spot"
		if !ss.Found {
			label = "Fastest (no sweet spot found)"
		}
		dictSuffix := ""
		if ss.UseDict {
			dictSuffix = "+dict"
		}
		recs = append(recs, fmt.Sprintf("[%s] %s: %s/L%d/%s%s (ratio: %.2fx, encode: %dµs)",
			ss.Dataset, label, ss.Algorithm, ss.Level, ss.InputType, dictSuffix, ss.Ratio, ss.EncodeUs))
	}

	if len(s.QRFitCounts) > 0 {
		total := 0
		for _, c := range s.QRFitCounts {
			total += c
		}
		recs = append(recs, fmt.Sprintf("%d configurations fit in QR codes across all EC levels", total))
	}

	return recs
}
