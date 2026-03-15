package htmlreport

import (
	"fmt"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

var algoColors = map[string]string{
	"zlib":   "#5470c6",
	"zstd":   "#91cc75",
	"brotli": "#fac858",
}

// gridWithPadding returns a grid option with enough top margin for title+legend.
func gridWithPadding() charts.GlobalOpts {
	return charts.WithGridOpts(opts.Grid{
		Top:    "80px",
		Left:   "80px",
		Right:  "120px",
		Bottom: "60px",
	})
}

func compressionRatioChart(data RatioBarData) *charts.Bar {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Best Compression Ratio by Dataset",
			Subtitle: "Higher ratio = better compression",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true)}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true), Top: "30px"}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Dataset"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Ratio"}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1100px",
			Height: "450px",
		}),
		gridWithPadding(),
	)
	bar.SetXAxis(data.Datasets)

	// Deterministic order.
	for _, algo := range sortedKeys(toSet(keys(data.ByAlgo))) {
		ratios := data.ByAlgo[algo]
		items := make([]opts.BarData, len(ratios))
		for i, r := range ratios {
			items[i] = opts.BarData{Value: fmt.Sprintf("%.2f", r)}
		}
		barOpts := []charts.SeriesOpts{}
		if c, ok := algoColors[algo]; ok {
			barOpts = append(barOpts, charts.WithItemStyleOpts(opts.ItemStyle{Color: c}))
		}
		bar.AddSeries(algo, items, barOpts...)
	}

	return bar
}

func speedVsRatioChart(points []ScatterPoint) *charts.Scatter {
	scatter := charts.NewScatter()
	scatter.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Encode Speed vs Compression Ratio",
			Subtitle: "Bottom-right is ideal: fast encode + high ratio",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "item",
		}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true), Top: "30px"}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Encode Time (µs)", Type: "log"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Ratio", Type: "value"}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1100px",
			Height: "500px",
		}),
		gridWithPadding(),
	)

	// Group by algorithm.
	byAlgo := map[string][]opts.ScatterData{}
	for _, p := range points {
		byAlgo[p.Algorithm] = append(byAlgo[p.Algorithm], opts.ScatterData{
			Value:      []interface{}{p.EncodeUs, p.Ratio},
			Symbol:     "circle",
			SymbolSize: 8,
		})
	}

	for _, algo := range sortedKeys(toSet(keys(byAlgo))) {
		scatterOpts := []charts.SeriesOpts{}
		if c, ok := algoColors[algo]; ok {
			scatterOpts = append(scatterOpts, charts.WithItemStyleOpts(opts.ItemStyle{Color: c}))
		}
		scatter.AddSeries(algo, byAlgo[algo], scatterOpts...)
	}

	return scatter
}

func levelSweepChart(dataset string, series []LevelSweepSeries) *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    fmt.Sprintf("Level Sweep — %s", dataset),
			Subtitle: "Higher ratio = better compression",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true), Top: "30px"}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Level", Type: "category"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Ratio"}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1100px",
			Height: "400px",
		}),
		gridWithPadding(),
	)

	// Build union of all levels for x-axis.
	levelSet := map[int]struct{}{}
	for _, s := range series {
		for _, l := range s.Levels {
			levelSet[l] = struct{}{}
		}
	}
	allLevels := sortedInts(levelSet)
	xLabels := make([]string, len(allLevels))
	for i, l := range allLevels {
		xLabels[i] = itoa(l)
	}
	line.SetXAxis(xLabels)

	for _, s := range series {
		// Build a lookup for this series' levels.
		ratioByLevel := map[int]float64{}
		for i, l := range s.Levels {
			ratioByLevel[l] = s.Ratios[i]
		}

		items := make([]opts.LineData, len(allLevels))
		for i, l := range allLevels {
			if v, ok := ratioByLevel[l]; ok {
				items[i] = opts.LineData{Value: v}
			} else {
				items[i] = opts.LineData{Value: "-"}
			}
		}

		lineOpts := []charts.SeriesOpts{
			charts.WithLineChartOpts(opts.LineChart{ConnectNulls: opts.Bool(false)}),
		}
		if c, ok := algoColors[s.Algorithm]; ok {
			lineOpts = append(lineOpts, charts.WithItemStyleOpts(opts.ItemStyle{Color: c}))
		}
		line.AddSeries(s.Algorithm, items, lineOpts...)
	}

	return line
}

func dictImpactChart(pairs []DictPair) *charts.Bar {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Dictionary Impact (zstd)",
			Subtitle: "Higher ratio = better compression",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true)}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true), Top: "30px"}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Dataset"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Ratio"}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1100px",
			Height: "400px",
		}),
		gridWithPadding(),
	)

	labels := make([]string, len(pairs))
	noDictItems := make([]opts.BarData, len(pairs))
	dictItems := make([]opts.BarData, len(pairs))

	for i, p := range pairs {
		labels[i] = fmt.Sprintf("%s (%s)", p.Dataset, p.InputType)
		noDictItems[i] = opts.BarData{Value: fmt.Sprintf("%.2f", p.RatioNoDict)}
		dictItems[i] = opts.BarData{Value: fmt.Sprintf("%.2f", p.RatioDict)}
	}

	bar.SetXAxis(labels)
	bar.AddSeries("Without Dict", noDictItems, charts.WithItemStyleOpts(opts.ItemStyle{Color: "#5470c6"}))
	bar.AddSeries("With Dict", dictItems, charts.WithItemStyleOpts(opts.ItemStyle{Color: "#91cc75"}))

	return bar
}

func barcodeHeatmapChart(datasets, ecLevels []string, cells []HeatmapCell) *charts.HeatMap {
	hm := charts.NewHeatMap()
	hm.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "QR Code Feasibility",
			Subtitle: "Green = fits, Red = does not fit\nL = Low (~7% recovery) · M = Medium (~15%) · Q = Quartile (~25%) · H = High (~30%)",
			Left:     "left",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true)}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(false)}),
		charts.WithXAxisOpts(opts.XAxis{
			Type:      "category",
			Data:      toInterfaceSlice(ecLevels),
			SplitArea: &opts.SplitArea{Show: opts.Bool(true)},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type:      "category",
			Data:      toInterfaceSlice(datasets),
			SplitArea: &opts.SplitArea{Show: opts.Bool(true)},
		}),
		charts.WithVisualMapOpts(opts.VisualMap{
			Calculable: opts.Bool(false),
			Min:        0,
			Max:        1,
			InRange: &opts.VisualMapInRange{
				Color: []string{"#e74c3c", "#2ecc71"},
			},
			Show: opts.Bool(false),
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1100px",
			Height: "400px",
		}),
		charts.WithGridOpts(opts.Grid{
			Top:    "80px",
			Left:   "120px",
			Right:  "500px",
			Bottom: "60px",
		}),
	)

	// Build index lookups.
	ecIdx := map[string]int{}
	for i, ec := range ecLevels {
		ecIdx[ec] = i
	}
	dsIdx := map[string]int{}
	for i, ds := range datasets {
		dsIdx[ds] = i
	}

	items := make([]opts.HeatMapData, len(cells))
	for i, c := range cells {
		val := 0
		if c.Fits {
			val = 1
		}
		items[i] = opts.HeatMapData{
			Value: [3]interface{}{ecIdx[c.ECLevel], dsIdx[c.Dataset], val},
		}
	}

	hm.SetXAxis(ecLevels)
	hm.AddSeries("Feasibility", items)

	return hm
}

func datamatrixHeatmapChart(datasets []string, cells []HeatmapCell) *charts.HeatMap {
	hm := charts.NewHeatMap()
	hm.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "DataMatrix Feasibility",
			Subtitle: "Green = fits, Red = does not fit",
			Left:     "left",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true)}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(false)}),
		charts.WithXAxisOpts(opts.XAxis{
			Type:      "category",
			Data:      []interface{}{"ECC200"},
			SplitArea: &opts.SplitArea{Show: opts.Bool(true)},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type:      "category",
			Data:      toInterfaceSlice(datasets),
			SplitArea: &opts.SplitArea{Show: opts.Bool(true)},
		}),
		charts.WithVisualMapOpts(opts.VisualMap{
			Calculable: opts.Bool(false),
			Min:        0,
			Max:        1,
			InRange: &opts.VisualMapInRange{
				Color: []string{"#e74c3c", "#2ecc71"},
			},
			Show: opts.Bool(false),
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1100px",
			Height: "400px",
		}),
		charts.WithGridOpts(opts.Grid{
			Top:    "70px",
			Left:   "120px",
			Right:  "700px",
			Bottom: "60px",
		}),
	)

	dsIdx := map[string]int{}
	for i, ds := range datasets {
		dsIdx[ds] = i
	}

	items := make([]opts.HeatMapData, len(cells))
	for i, c := range cells {
		val := 0
		if c.Fits {
			val = 1
		}
		items[i] = opts.HeatMapData{
			Value: [3]interface{}{0, dsIdx[c.Dataset], val},
		}
	}

	hm.SetXAxis([]string{"ECC200"})
	hm.AddSeries("Feasibility", items)

	return hm
}

// Helper to convert map keys to a set.
func toSet[K comparable, V any](m map[K]V) map[K]struct{} {
	s := make(map[K]struct{}, len(m))
	for k := range m {
		s[k] = struct{}{}
	}
	return s
}

func keys[K comparable, V any](m map[K]V) map[K]V {
	return m
}

func toInterfaceSlice(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
