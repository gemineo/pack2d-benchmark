package htmlreport

import (
	"sort"
	"strings"

	"github.com/gemineo/pack2d-benchmark/internal/runner"
)

// RatioBarData holds best compression ratio per algorithm per dataset.
type RatioBarData struct {
	Datasets []string
	ByAlgo   map[string][]float64 // algo → ratio per dataset (same order as Datasets)
}

// ScatterPoint represents one data point for the speed-vs-ratio scatter.
type ScatterPoint struct {
	EncodeUs  float64
	Ratio     float64
	Algorithm string
	Label     string
	Dataset   string
}

// LevelSweepSeries holds compression ratios at each level for one algorithm.
type LevelSweepSeries struct {
	Algorithm string
	Levels    []int
	Ratios    []float64
}

// DictPair holds compression ratios with and without dictionary for the same config.
type DictPair struct {
	Dataset     string
	Level       int
	InputType   string
	RatioNoDict float64
	RatioDict   float64
}

// HeatmapCell represents one cell in the barcode feasibility heatmap.
type HeatmapCell struct {
	Dataset string
	ECLevel string
	Fits    bool
	Version int
	Modules int
	SizeMM  float64
}

// EncodedSizeBarData holds smallest encoded size per algorithm per dataset.
type EncodedSizeBarData struct {
	Datasets []string
	ByAlgo   map[string][]int // algo → smallest encoded bytes per dataset (same order as Datasets)
}

// SmallestEncodedSize returns the smallest absolute encoded size per algorithm per dataset.
// This complements CompressionRatioByDataset: ratio rewards verbose inputs (like XML)
// with a larger denominator, while encoded size shows the actual barcode payload.
func SmallestEncodedSize(results []runner.Result) EncodedSizeBarData {
	type key struct {
		dataset string
		algo    string
	}

	best := map[key]int{}
	datasetSet := map[string]struct{}{}
	algoSet := map[string]struct{}{}

	for _, r := range results {
		if r.Scenario != "compression" {
			continue
		}
		k := key{r.Dataset, string(r.Algorithm)}
		datasetSet[r.Dataset] = struct{}{}
		algoSet[string(r.Algorithm)] = struct{}{}
		if prev, ok := best[k]; !ok || r.Encoded < prev {
			best[k] = r.Encoded
		}
	}

	datasets := sortedKeys(datasetSet)
	algos := sortedKeys(algoSet)

	byAlgo := make(map[string][]int, len(algos))
	for _, algo := range algos {
		sizes := make([]int, len(datasets))
		for i, ds := range datasets {
			sizes[i] = best[key{ds, algo}]
		}
		byAlgo[algo] = sizes
	}

	return EncodedSizeBarData{Datasets: datasets, ByAlgo: byAlgo}
}

// CompressionRatioByDataset returns the best (lowest) compression ratio per algorithm per dataset.
func CompressionRatioByDataset(results []runner.Result) RatioBarData {
	type key struct {
		dataset string
		algo    string
	}

	best := map[key]float64{}
	datasetSet := map[string]struct{}{}
	algoSet := map[string]struct{}{}

	for _, r := range results {
		if r.Scenario != "compression" {
			continue
		}
		k := key{r.Dataset, string(r.Algorithm)}
		datasetSet[r.Dataset] = struct{}{}
		algoSet[string(r.Algorithm)] = struct{}{}
		if prev, ok := best[k]; !ok || r.Ratio < prev {
			best[k] = r.Ratio
		}
	}

	datasets := sortedKeys(datasetSet)
	algos := sortedKeys(algoSet)

	byAlgo := make(map[string][]float64, len(algos))
	for _, algo := range algos {
		ratios := make([]float64, len(datasets))
		for i, ds := range datasets {
			ratios[i] = best[key{ds, algo}]
		}
		byAlgo[algo] = ratios
	}

	return RatioBarData{Datasets: datasets, ByAlgo: byAlgo}
}

// SpeedVsRatio returns a scatter point for each compression result.
func SpeedVsRatio(results []runner.Result) []ScatterPoint {
	var points []ScatterPoint
	for _, r := range results {
		if r.Scenario != "compression" {
			continue
		}
		label := string(r.Algorithm) + "/L" + itoa(r.Level) + "/" + string(r.InputType)
		if r.UseDict {
			label += "/dict"
		}
		points = append(points, ScatterPoint{
			EncodeUs:  float64(r.Encode.Mean.Microseconds()),
			Ratio:     r.Ratio,
			Algorithm: string(r.Algorithm),
			Label:     label,
			Dataset:   r.Dataset,
		})
	}
	return points
}

// LevelSweep returns compression ratio by level for each algorithm, filtered to a single dataset.
// It uses the "raw" input type when available to show the native compression behaviour.
func LevelSweep(results []runner.Result, dataset string) []LevelSweepSeries {
	type key struct {
		algo  string
		level int
	}

	best := map[key]float64{}
	algoSet := map[string]struct{}{}

	for _, r := range results {
		if r.Scenario != "compression" || r.Dataset != dataset || r.UseDict {
			continue
		}
		// Prefer raw input type for a cleaner curve; fall back to whatever is available.
		k := key{string(r.Algorithm), r.Level}
		algoSet[string(r.Algorithm)] = struct{}{}
		if r.InputType == "raw" || best[k] == 0 {
			best[k] = r.Ratio
		}
	}

	algos := sortedKeys(algoSet)
	var series []LevelSweepSeries

	for _, algo := range algos {
		// Collect levels for this algorithm.
		levelSet := map[int]struct{}{}
		for k := range best {
			if k.algo == algo {
				levelSet[k.level] = struct{}{}
			}
		}
		levels := sortedInts(levelSet)

		ratios := make([]float64, len(levels))
		for i, lvl := range levels {
			ratios[i] = best[key{algo, lvl}]
		}

		series = append(series, LevelSweepSeries{
			Algorithm: algo,
			Levels:    levels,
			Ratios:    ratios,
		})
	}

	return series
}

// DictImpact returns side-by-side ratio pairs for zstd with and without dictionary.
// It picks the single best (lowest ratio) config per dataset across all levels and input types.
func DictImpact(results []runner.Result) []DictPair {
	type key struct {
		dataset   string
		level     int
		inputType string
	}

	noDict := map[key]float64{}
	withDict := map[key]float64{}

	for _, r := range results {
		if r.Scenario != "compression" || string(r.Algorithm) != "zstd" {
			continue
		}
		k := key{r.Dataset, r.Level, string(r.InputType)}
		if r.UseDict {
			withDict[k] = r.Ratio
		} else {
			noDict[k] = r.Ratio
		}
	}

	// Only include entries where both exist — pick the best level per dataset
	// across all input types.
	bestPair := map[string]DictPair{} // dataset → best pair

	for k, nd := range noDict {
		wd, ok := withDict[k]
		if !ok {
			continue
		}
		if existing, exists := bestPair[k.dataset]; !exists || wd < existing.RatioDict {
			bestPair[k.dataset] = DictPair{
				Dataset:     k.dataset,
				Level:       k.level,
				InputType:   k.inputType,
				RatioNoDict: nd,
				RatioDict:   wd,
			}
		}
	}

	var pairs []DictPair
	for _, p := range bestPair {
		pairs = append(pairs, p)
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Dataset < pairs[j].Dataset
	})

	return pairs
}

// BarcodeHeatmap returns datasets, EC levels, and cells for the barcode feasibility heatmap.
// It uses the best compression config per dataset (smallest encoded size).
func BarcodeHeatmap(results []runner.Result) (datasets []string, ecLevels []string, cells []HeatmapCell) {
	// Collect barcode results — use the best (smallest encoded) config per dataset.
	type bestKey struct {
		dataset string
		ecLevel string
	}

	type bestVal struct {
		fits    bool
		version int
		modules int
		sizeMM  float64
		encoded int
	}

	best := map[bestKey]bestVal{}
	datasetSet := map[string]struct{}{}
	ecSet := map[string]struct{}{}

	for _, r := range results {
		if r.Barcode == nil {
			continue
		}
		for _, chk := range r.Barcode.Checks {
			if chk.BarcodeType != "qr" && chk.BarcodeType != "qrcode" {
				continue
			}
			datasetSet[r.Dataset] = struct{}{}
			ecSet[chk.ECLevel] = struct{}{}
			bk := bestKey{r.Dataset, chk.ECLevel}

			existing, ok := best[bk]
			if !ok || r.Encoded < existing.encoded {
				best[bk] = bestVal{
					fits:    chk.Fits,
					version: chk.QRVersion,
					modules: chk.Modules,
					sizeMM:  chk.SizeMM,
					encoded: r.Encoded,
				}
			}
		}
	}

	datasets = sortedKeys(datasetSet)
	ecLevels = sortedECLevels(ecSet)

	for _, ds := range datasets {
		for _, ec := range ecLevels {
			bk := bestKey{ds, ec}
			if v, ok := best[bk]; ok {
				cells = append(cells, HeatmapCell{
					Dataset: ds,
					ECLevel: ec,
					Fits:    v.fits,
					Version: v.version,
					Modules: v.modules,
					SizeMM:  v.sizeMM,
				})
			}
		}
	}

	return datasets, ecLevels, cells
}

// DataMatrixHeatmap returns datasets and cells for the DataMatrix feasibility heatmap.
// DataMatrix uses a single EC level (ECC200), so the result is a single-column heatmap.
func DataMatrixHeatmap(results []runner.Result) (datasets []string, cells []HeatmapCell) {
	type bestVal struct {
		fits    bool
		modules int
		sizeMM  float64
		encoded int
	}

	best := map[string]bestVal{} // dataset → best
	datasetSet := map[string]struct{}{}

	for _, r := range results {
		if r.Barcode == nil {
			continue
		}
		for _, chk := range r.Barcode.Checks {
			if chk.BarcodeType != "datamatrix" {
				continue
			}
			datasetSet[r.Dataset] = struct{}{}

			existing, ok := best[r.Dataset]
			if !ok || r.Encoded < existing.encoded {
				best[r.Dataset] = bestVal{
					fits:    chk.Fits,
					modules: chk.Modules,
					sizeMM:  chk.SizeMM,
					encoded: r.Encoded,
				}
			}
		}
	}

	datasets = sortedKeys(datasetSet)

	for _, ds := range datasets {
		if v, ok := best[ds]; ok {
			cells = append(cells, HeatmapCell{
				Dataset: ds,
				ECLevel: "ECC200",
				Fits:    v.fits,
				Modules: v.modules,
				SizeMM:  v.sizeMM,
			})
		}
	}

	return datasets, cells
}

// SerializationBarData holds best compression ratio per input type per dataset.
type SerializationBarData struct {
	Datasets    []string
	ByInputType map[string][]float64 // inputType → best ratio per dataset (same order as Datasets)
}

// SerializationImpact returns the best (lowest) compression ratio per input type per dataset,
// showing how serialization format affects compression. Dictionary results are excluded
// for clarity.
func SerializationImpact(results []runner.Result) SerializationBarData {
	type key struct {
		dataset   string
		inputType string
	}

	best := map[key]float64{}
	datasetSet := map[string]struct{}{}
	inputTypeSet := map[string]struct{}{}

	for _, r := range results {
		if r.Scenario != "compression" || r.UseDict {
			continue
		}
		k := key{r.Dataset, string(r.InputType)}
		datasetSet[r.Dataset] = struct{}{}
		inputTypeSet[string(r.InputType)] = struct{}{}
		if prev, ok := best[k]; !ok || r.Ratio < prev {
			best[k] = r.Ratio
		}
	}

	datasets := sortedKeys(datasetSet)
	inputTypes := sortedKeys(inputTypeSet)

	byInputType := make(map[string][]float64, len(inputTypes))
	for _, it := range inputTypes {
		ratios := make([]float64, len(datasets))
		for i, ds := range datasets {
			ratios[i] = best[key{ds, it}]
		}
		byInputType[it] = ratios
	}

	return SerializationBarData{Datasets: datasets, ByInputType: byInputType}
}

// BarcodeSizeBarData holds barcode physical sizes per EC level per dataset.
type BarcodeSizeBarData struct {
	Datasets  []string
	ByECLevel map[string][]float64 // ecLevel → sizeMM per dataset (same order as Datasets)
}

// BarcodeSizeData returns the smallest physical QR code size (in mm) per EC level per dataset.
// Only includes entries where the data fits.
func BarcodeSizeData(results []runner.Result) BarcodeSizeBarData {
	type key struct {
		dataset string
		ecLevel string
	}

	best := map[key]float64{}
	datasetSet := map[string]struct{}{}
	ecSet := map[string]struct{}{}

	for _, r := range results {
		if r.Barcode == nil {
			continue
		}
		for _, chk := range r.Barcode.Checks {
			if chk.BarcodeType != "qrcode" || !chk.Fits || chk.SizeMM == 0 {
				continue
			}
			k := key{r.Dataset, chk.ECLevel}
			datasetSet[r.Dataset] = struct{}{}
			ecSet[chk.ECLevel] = struct{}{}
			if prev, ok := best[k]; !ok || chk.SizeMM < prev {
				best[k] = chk.SizeMM
			}
		}
	}

	datasets := sortedKeys(datasetSet)
	ecLevels := sortedECLevels(ecSet)

	byEC := make(map[string][]float64, len(ecLevels))
	for _, ec := range ecLevels {
		sizes := make([]float64, len(datasets))
		for i, ds := range datasets {
			sizes[i] = best[key{ds, ec}] // 0 if not present (doesn't fit)
		}
		byEC[ec] = sizes
	}

	return BarcodeSizeBarData{Datasets: datasets, ByECLevel: byEC}
}

// Datasets returns the unique sorted dataset names from results.
func Datasets(results []runner.Result) []string {
	set := map[string]struct{}{}
	for _, r := range results {
		set[r.Dataset] = struct{}{}
	}
	return sortedKeys(set)
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedInts(m map[int]struct{}) []int {
	vals := make([]int, 0, len(m))
	for v := range m {
		vals = append(vals, v)
	}
	sort.Ints(vals)
	return vals
}

// sortedECLevels sorts QR EC levels in canonical order: L, M, Q, H.
func sortedECLevels(m map[string]struct{}) []string {
	order := map[string]int{"L": 0, "M": 1, "Q": 2, "H": 3}
	keys := sortedKeys(m)
	sort.Slice(keys, func(i, j int) bool {
		oi, ok1 := order[strings.ToUpper(keys[i])]
		oj, ok2 := order[strings.ToUpper(keys[j])]
		if ok1 && ok2 {
			return oi < oj
		}
		return keys[i] < keys[j]
	})
	return keys
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
