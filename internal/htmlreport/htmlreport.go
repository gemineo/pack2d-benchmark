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
<script src="https://cdn.jsdelivr.net/npm/bwip-js@4/dist/bwip-js-min.js"></script>
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
    {{if gt .Metadata.ModuleSizeMM 0.0}}<span>Module size: {{printf "%.2f" .Metadata.ModuleSizeMM}}mm</span>{{end}}
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
<style>
.barcode-preview {
  position: absolute; right: 20px; top: 80px;
  width: 200px; padding: 12px; background: #fff;
  border: 1px solid #e0e0e0; border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  text-align: center; font-family: inherit; font-size: 13px;
  z-index: 10;
}
.barcode-preview .bp-title { font-weight: 600; margin-bottom: 8px; color: #333; }
.barcode-preview .bp-caption { margin-top: 8px; color: #666; line-height: 1.4; }
.barcode-preview .bp-placeholder { color: #aaa; padding: 40px 0; }
.barcode-preview canvas { display: block; margin: 0 auto; image-rendering: pixelated; max-width: 180px; max-height: 180px; }
</style>
<script>
(function() {
  function clearPanel(panel) {
    while (panel.firstChild) panel.removeChild(panel.firstChild);
  }

  function addTitle(panel, text) {
    var d = document.createElement('div');
    d.className = 'bp-title';
    d.textContent = text;
    panel.appendChild(d);
  }

  function addCaption(panel, text) {
    var d = document.createElement('div');
    d.className = 'bp-caption';
    d.innerHTML = text;
    panel.appendChild(d);
  }

  function addPlaceholder(panel, text) {
    var d = document.createElement('div');
    d.className = 'bp-placeholder';
    d.textContent = text;
    panel.appendChild(d);
  }

  var LOREM_BLOCK = 'Lorem ipsum dolor sit amet, consectetuer adipiscing elit. Aenean commodo ligula eget dolor. Aenean massa. Cum sociis natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Donec quam felis, ultricies nec, pellentesque eu, pretium quis, sem. Nulla consequat massa quis enim. Donec pede justo, fringilla vel, aliquet nec, vulputate eget, arcu. In enim justo, rhoncus ut, imperdiet a, venenatis vitae, justo. Nullam dictum felis eu pede mollis pretium. Integer tincidunt. Cras dapibus. Vivamus elementum semper nisi. Aenean vulputate eleifend tellus. Aenean leo ligula, porttitor eu, consequat vitae, eleifend ac, enim. Aliquam lorem ante, dapibus in, viverra quis, feugiat a, tellus. Phasellus viverra nulla ut metus varius laoreet. Quisque rutrum. Aenean imperdiet. Etiam ultricies nisi vel augue. Curabitur ullamcorper ultricies nisi. Nam eget dui. Etiam rhoncus. Maecenas tempus, tellus eget condimentum rhoncus, sem quam semper libero, sit amet adipiscing sem neque sed ipsum. Nam quam nunc, blandit vel, luctus pulvinar, hendrerit id, lorem. Maecenas nec odio et ante tincidunt tempus. Donec vitae sapien ut libero venenatis faucibus. Nullam quis ante. Etiam sit amet orci eget eros faucibus tincidunt. Duis leo. Sed fringilla mauris sit amet nibh. Donec sodales sagittis magna. Sed consequat, leo eget bibendum sodales, augue velit cursus nunc, quis gravida magna mi a libero. Fusce vulputate eleifend sapien. Vestibulum purus quam, scelerisque ut, mollis sed, nonummy id, metus. Nullam accumsan lorem in dui. Cras ultricies mi eu turpis hendrerit fringilla. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; In ac dui quis mi consectetuer lacinia. Nam pretium turpis et arcu. Duis arcu tortor, suscipit eget, imperdiet nec, imperdiet iaculis, ipsum. Sed aliquam ultrices mauris. Integer ante arcu, accumsan a, consectetuer eget, posuere ut, mauris. Praesent adipiscing. Phasellus ullamcorper ipsum rutrum nunc. Nunc nonummy metus. Vestibulum volutpat pretium libero. Cras id dui. Aenean ut eros et nisl sagittis vestibulum. Nullam nulla eros, ultricies sit amet, nonummy id, imperdiet feugiat, pede. Sed lectus. Donec mollis hendrerit risus. Phasellus nec sem in justo pellentesque facilisis. Etiam imperdiet imperdiet orci. Nunc nec neque. Phasellus leo dolor, tempus non, auctor et, hendrerit quis, nisi. Curabitur ligula sapien, tincidunt non, euismod vitae, posuere imperdiet, leo. Maecenas malesuada. Praesent congue erat at massa. Sed cursus turpis vitae tortor. Donec posuere vulputate arcu. Phasellus accumsan cursus velit. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Sed aliquam, nisi quis porttitor congue, elit erat euismod orci, ac placerat dolor lectus quis orci. Phasellus consectetuer vestibulum elit. Aenean tellus metus, bibendum sed, posuere ac, mattis non, nunc. Vestibulum fringilla pede sit amet augue. In turpis. Pellentesque posuere. Praesent turpis. Aenean posuere, tortor sed cursus feugiat, nunc augue blandit nunc, eu sollicitudin urna dolor sagittis lacus. Donec elit libero, sodales nec, volutpat a, suscipit non, turpis. Nullam sagittis. Suspendisse pulvinar, augue ac venenatis condimentum, sem libero volutpat nibh, nec pellentesque velit pede quis nunc. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Fusce id purus. Ut varius tincidunt libero. Phasellus dolor. Maecenas vestibulum mollis.';
  function lorem(n) { var s = ''; while (s.length < n) s += LOREM_BLOCK; return s.substr(0, Math.max(1, n)); }

  // QR byte-mode capacity per version (1-40) and EC level [L,M,Q,H] — ISO 18004.
  var QR_CAP = [
    [17,14,11,7],[32,26,20,14],[53,42,32,24],[78,62,46,34],[106,84,60,44],
    [134,106,74,58],[154,122,86,64],[192,152,108,84],[230,180,130,98],[271,213,151,119],
    [321,251,177,137],[367,287,203,155],[425,331,241,177],[458,362,258,194],[520,412,292,220],
    [586,450,322,250],[644,504,364,280],[718,560,394,310],[792,624,442,338],[858,666,482,382],
    [929,711,509,403],[1003,779,565,439],[1091,857,611,461],[1171,911,661,511],[1273,997,715,535],
    [1367,1059,751,593],[1465,1125,805,625],[1528,1190,868,658],[1628,1264,908,698],[1732,1370,982,742],
    [1840,1452,1030,790],[1952,1538,1112,842],[2068,1628,1168,898],[2188,1722,1228,958],[2303,1809,1283,983],
    [2431,1911,1351,1051],[2563,1989,1423,1093],[2699,2099,1499,1139],[2809,2213,1579,1219],[2953,2331,1663,1273]
  ];
  var EC_IDX = {L:0,M:1,Q:2,H:3};

  // DataMatrix ECC200 byte-mode capacity by modules — ISO 16022.
  var DM_CAP = {10:3,12:5,14:8,16:12,18:18,20:22,22:30,24:36,26:44,32:62,36:86,40:114,44:144,48:174,52:204,64:280,72:368,80:456,88:576,96:696,104:816,120:1050,132:1304,144:1556};

  function renderQR(panel, ver, ec, modules, sizeMM) {
    var canvas = document.createElement('canvas');
    var cap = QR_CAP[ver - 1] ? QR_CAP[ver - 1][EC_IDX[ec] || 1] : 17;
    var text = lorem(Math.floor(cap * 0.9));
    try {
      bwipjs.toCanvas(canvas, { bcid: 'qrcode', text: text, version: ver, eclevel: ec, scale: 2 });
    } catch(e1) {
      try {
        bwipjs.toCanvas(canvas, { bcid: 'qrcode', text: text, eclevel: ec, scale: 2 });
      } catch(e2) {
        addPlaceholder(panel, 'Preview unavailable');
        return;
      }
    }
    panel.appendChild(canvas);
    addCaption(panel, modules + '\u00d7' + modules + ' modules<br>' + sizeMM.toFixed(1) + ' mm');
  }

  function renderDM(panel, modules, sizeMM) {
    var canvas = document.createElement('canvas');
    var ver = modules + 'x' + modules;
    var cap = DM_CAP[modules] || 3;
    var text = lorem(Math.floor(cap * 0.9));
    try {
      bwipjs.toCanvas(canvas, { bcid: 'datamatrix', text: text, version: ver, scale: 2 });
    } catch(e1) {
      try {
        bwipjs.toCanvas(canvas, { bcid: 'datamatrix', text: text, scale: 2 });
      } catch(e2) {
        addPlaceholder(panel, 'Preview unavailable');
        return;
      }
    }
    panel.appendChild(canvas);
    addCaption(panel, modules + '\u00d7' + modules + ' modules<br>' + sizeMM.toFixed(1) + ' mm');
  }

  function setupPreview(chartDom, type) {
    var container = chartDom.parentElement;
    container.style.position = 'relative';
    var panel = document.createElement('div');
    panel.className = 'barcode-preview';
    addPlaceholder(panel, 'Hover over a cell to preview');
    container.appendChild(panel);

    var chart = echarts.getInstanceByDom(chartDom);
    if (!chart) return;

    chart.on('mouseover', function(params) {
      if (!params.value) return;
      var v = params.value;
      var fits, modules, sizeMM, title;
      if (type === 'qr') {
        fits = v[2] == 1; modules = v[4]; sizeMM = v[5];
        var ver = v[3];
        var ecLabels = chart.getOption().xAxis[0].data;
        var ec = ecLabels ? ecLabels[v[0]] : 'M';
        title = 'QR V' + ver + ' \u00b7 ' + ec;
      } else {
        fits = v[2] == 1; modules = v[3]; sizeMM = v[4];
        title = 'DataMatrix ECC200';
      }
      clearPanel(panel);
      addTitle(panel, title);
      if (!fits || !modules) {
        addPlaceholder(panel, 'Does not fit');
        return;
      }
      if (type === 'qr') {
        renderQR(panel, ver, ec, modules, sizeMM);
      } else {
        renderDM(panel, modules, sizeMM);
      }
    });

    chart.on('mouseout', function() {
      clearPanel(panel);
      addPlaceholder(panel, 'Hover over a cell to preview');
    });
  }

  function init() {
    var containers = document.querySelectorAll('[class*="go-echarts"]');
    if (containers.length === 0) containers = document.querySelectorAll('div[id]');
    containers.forEach(function(el) {
      var inst = echarts.getInstanceByDom(el);
      if (!inst) return;
      var opt = inst.getOption();
      if (!opt || !opt.title || !opt.title[0]) return;
      var t = opt.title[0].text || '';
      if (t === 'QR Code Feasibility') setupPreview(el, 'qr');
      else if (t === 'DataMatrix Feasibility') setupPreview(el, 'dm');
    });
  }

  if (document.readyState === 'complete') { setTimeout(init, 500); }
  else { window.addEventListener('load', function() { setTimeout(init, 500); }); }
})();
</script>
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

	// 7. Barcode physical size chart.
	bcSizeData := BarcodeSizeData(rpt.Results)
	if len(bcSizeData.Datasets) > 0 && len(bcSizeData.ByECLevel) > 0 {
		page.AddCharts(barcodeSizeChart(bcSizeData))
	}

	// 9. QR barcode heatmap.
	datasets, ecLevels, cells := BarcodeHeatmap(rpt.Results)
	if len(cells) > 0 {
		page.AddCharts(barcodeHeatmapChart(datasets, ecLevels, cells))
	}

	// 10. DataMatrix heatmap.
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
