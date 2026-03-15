package htmlreport

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"regexp"
	"strings"

	"github.com/go-echarts/go-echarts/v2/components"

	"github.com/gemineo/pack2d-benchmark/internal/report"
)

const pageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>pack2d Benchmark Report</title>
{{.HeadScripts}}
<style>
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 0; background: #f5f5f5; }
.header { background: #1a1a2e; color: #eee; padding: 24px 40px; }
.header h1 { margin: 0 0 8px; font-size: 24px; }
.header .meta { font-size: 13px; color: #aaa; line-height: 1.7; }
.header .meta span { margin-right: 24px; }
.summary { background: #16213e; color: #ddd; padding: 16px 40px; }
.summary h2 { margin: 0 0 12px; font-size: 18px; color: #eee; }
.sweet-spots { display: flex; flex-wrap: wrap; gap: 12px; margin-bottom: 12px; }
.spot { background: #0f3460; border-radius: 6px; padding: 10px 16px; font-size: 13px; line-height: 1.5; }
.spot .ds { font-weight: 600; color: #e0e0e0; }
.spot .cfg { color: #91cc75; }
.recommendations { font-size: 13px; line-height: 1.8; }
.recommendations li { margin-bottom: 2px; }
.ratio-info { background: #fff; margin: 20px 40px 0; padding: 14px 20px; border-left: 4px solid #e74c3c; border-radius: 4px; font-size: 14px; line-height: 1.6; color: #333; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
.ratio-info strong { color: #1a1a2e; }
.charts { padding: 20px 40px; }
.chart-section { background: #fff; border-radius: 8px; margin-bottom: 24px; padding: 16px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
.chart-section h3 { margin: 0 0 12px; font-size: 16px; color: #333; }
</style>
</head>
<body>
<div class="header">
  <h1>pack2d Benchmark Report</h1>
  <div class="meta">
    <span>Tool: {{.Metadata.ToolVersion}}</span>
    <span>Go: {{.Metadata.GoVersion}}</span>
    <span>Platform: {{.Metadata.OS}}/{{.Metadata.Arch}}</span>
    <span>Timestamp: {{.Metadata.Timestamp.Format "2006-01-02 15:04:05 UTC"}}</span>
    <span>Iterations: {{.Metadata.Iterations}}</span>
    <span>Warm-up: {{.Metadata.WarmUp}}</span>
  </div>
</div>
{{if .Summary}}
<div class="summary">
  <h2>Summary</h2>
  {{if .Summary.BestRatio}}
  <h3 style="margin:12px 0 8px;font-size:15px;color:#91cc75;">Best Compression Ratio</h3>
  <div class="sweet-spots">
    {{range .Summary.BestRatio}}
    <div class="spot" style="background:#1a3a5c;">
      <div class="ds">{{.Dataset}}</div>
      <div class="cfg">{{.Algorithm}} L{{.Level}} ({{.InputType}}){{if .UseDict}} +dict{{end}} — ratio {{printf "%.2f" .Ratio}}x, {{.EncodeUs}}µs</div>
    </div>
    {{end}}
  </div>
  {{end}}
  {{if .Summary.SweetSpot}}
  <h3 style="margin:12px 0 8px;font-size:15px;color:#fac858;">Sweet Spot <span style="font-weight:400;font-size:12px;color:#aaa;">(best ratio improvement per µs of encode time)</span></h3>
  <div class="sweet-spots">
    {{range .Summary.SweetSpot}}
    <div class="spot">
      <div class="ds">{{.Dataset}}</div>
      <div class="cfg">{{.Algorithm}} L{{.Level}} ({{.InputType}}){{if .UseDict}} +dict{{end}} — ratio {{printf "%.2f" .Ratio}}x, {{.EncodeUs}}µs</div>
      {{if not .Found}}<div style="color:#fac858;font-size:11px;">No clear sweet spot; fastest config shown</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
</div>
{{end}}
<div class="ratio-info">
  <strong>Compression ratio</strong> = compressed size / original size.
  A ratio of 0.40 means the output is 40% of the original size — the data was compressed to less than half its size.
  A ratio above 1.0 means the output is <em>larger</em> than the input (compression overhead). The red dashed line marks this break-even point.
</div>
<div class="charts">
{{.ChartsHTML}}
</div>
</body>
</html>`

// Generate creates a self-contained HTML report from benchmark results.
func Generate(rpt *report.Report, w io.Writer) error {
	page := components.NewPage()
	page.PageTitle = "pack2d Benchmark Report"

	// 1. Compression ratio bar chart.
	ratioData := CompressionRatioByDataset(rpt.Results)
	if len(ratioData.Datasets) > 0 {
		page.AddCharts(compressionRatioChart(ratioData))
	}

	// 2. Smallest encoded size bar chart.
	sizeData := SmallestEncodedSize(rpt.Results)
	if len(sizeData.Datasets) > 0 {
		page.AddCharts(encodedSizeChart(sizeData))
	}

	// 3. Serialization impact bar chart.
	serData := SerializationImpact(rpt.Results)
	if len(serData.Datasets) > 0 && len(serData.ByInputType) > 1 {
		page.AddCharts(serializationImpactChart(serData))
	}

	// 4. Speed vs ratio scatter.
	scatter := SpeedVsRatio(rpt.Results)
	if len(scatter) > 0 {
		page.AddCharts(speedVsRatioChart(scatter))
	}

	// 5. Level sweep per dataset.
	for _, ds := range Datasets(rpt.Results) {
		series := LevelSweep(rpt.Results, ds)
		if len(series) > 0 {
			page.AddCharts(levelSweepChart(ds, series))
		}
	}

	// 6. Dictionary impact (only if dict results exist).
	dictPairs := DictImpact(rpt.Results)
	if len(dictPairs) > 0 {
		page.AddCharts(dictImpactChart(dictPairs))
	}

	// 7. QR barcode heatmap.
	datasets, ecLevels, cells := BarcodeHeatmap(rpt.Results)
	if len(cells) > 0 {
		page.AddCharts(barcodeHeatmapChart(datasets, ecLevels, cells))
	}

	// 8. DataMatrix heatmap.
	dmDatasets, dmCells := DataMatrixHeatmap(rpt.Results)
	if len(dmCells) > 0 {
		page.AddCharts(datamatrixHeatmapChart(dmDatasets, dmCells))
	}

	// Render go-echarts page to buffer.
	var chartsBuf bytes.Buffer
	if err := page.Render(&chartsBuf); err != nil {
		return fmt.Errorf("render charts: %w", err)
	}

	// Extract parts from go-echarts rendered output.
	rendered := chartsBuf.String()
	headScripts := extractHeadScripts(rendered)
	chartsHTML := extractBody(rendered)

	// Render full page with metadata header wrapping charts.
	tmpl, err := template.New("report").Parse(pageTemplate)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	data := struct {
		Metadata    report.Metadata
		Summary     *report.Summary
		HeadScripts template.HTML
		ChartsHTML  template.HTML
	}{
		Metadata:    rpt.Metadata,
		Summary:     rpt.Summary,
		HeadScripts: template.HTML(headScripts),
		ChartsHTML:  template.HTML(chartsHTML),
	}

	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}

var scriptSrcRe = regexp.MustCompile(`<script[^>]+src=[^>]+></script>`)

// extractHeadScripts returns all <script src="..."> tags from the <head> section
// of the go-echarts rendered page (the ECharts library CDN link).
func extractHeadScripts(html string) string {
	headEnd := strings.Index(html, "</head>")
	if headEnd < 0 {
		headEnd = len(html)
	}
	head := html[:headEnd]
	matches := scriptSrcRe.FindAllString(head, -1)
	return strings.Join(matches, "\n")
}

// extractBody returns the <style> blocks and <body> content from the go-echarts
// rendered page (chart divs and inline initialization scripts).
func extractBody(html string) string {
	var result strings.Builder

	// Extract style blocks.
	for _, pair := range findAllBetween(html, "<style>", "</style>") {
		result.WriteString(pair)
	}

	// Extract body content.
	if si := strings.Index(html, "<body>"); si >= 0 {
		content := html[si+len("<body>"):]
		if ei := strings.Index(content, "</body>"); ei >= 0 {
			result.WriteString(content[:ei])
		} else {
			result.WriteString(content)
		}
	}

	if result.Len() == 0 {
		return html
	}
	return result.String()
}

// findAllBetween returns all substrings between start and end markers (inclusive).
func findAllBetween(s, start, end string) []string {
	var results []string
	for {
		si := strings.Index(s, start)
		if si < 0 {
			break
		}
		ei := strings.Index(s[si:], end)
		if ei < 0 {
			break
		}
		results = append(results, s[si:si+ei+len(end)])
		s = s[si+ei+len(end):]
	}
	return results
}
