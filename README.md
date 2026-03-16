# pack2d-benchmark

Benchmarking CLI for evaluating [pack2d](https://github.com/gemineo/pack2d) compression configurations, barcode feasibility, and performance trade-offs.

## Installation

```bash
task build
```

Requires Go 1.26+ and the `pack2d` repository as a sibling directory.

## Quick Start

```bash
# Run full benchmark suite with embedded datasets
./bin/pack2d-benchmark run

# Quick check with fewer iterations
./bin/pack2d-benchmark run --iterations 5 --warm-up 1

# Focus on a specific algorithm
./bin/pack2d-benchmark run --scenarios compression --algorithms zstd

# Run with zstd dictionary auto-trained from datasets
./bin/pack2d-benchmark run --dict auto

# Set barcode module size for physical dimension calculations
./bin/pack2d-benchmark run --module-size 0.5

# Specify custom compression levels
./bin/pack2d-benchmark run --levels 1,5,9

# Export results as JSON
./bin/pack2d-benchmark run --export output/results.json

# Generate interactive HTML report from exported JSON
./bin/pack2d-benchmark report output/results.json -o output/report.html
```

## Commands

### `run`

Execute benchmark scenarios against datasets.

| Flag | Default | Description |
|------|---------|-------------|
| `--scenarios` | `compression,barcode` | Comma-separated scenarios to run |
| `--algorithms` | `zlib,zstd,brotli` | Compression algorithms to benchmark |
| `--levels` | _(all)_ | Comma-separated compression levels (applied to all algorithms) |
| `--iterations` | `20` | Number of measured iterations |
| `--warm-up` | `3` | Warm-up iterations (discarded) |
| `--input-types` | `raw,json,xml,cbor` | Serialization input types |
| `--dict` | | Path to zstd dictionary file, or `auto` to train from datasets |
| `--module-size` | `0.33` | Barcode module size in mm for physical dimension calculations |
| `--data` | _(embedded)_ | Custom dataset directory |
| `--export` | | Export JSON report to file |
| `--output` | _(stdout)_ | Write ASCII output to file |
| `--quiet` | `false` | Suppress progress spinner |
| `--no-color` | `false` | Disable colored output |

### `report`

Generate a self-contained HTML report with interactive charts from a JSON export.

```bash
# Generate HTML report from benchmark results
./bin/pack2d-benchmark report output/results.json

# Specify output path
./bin/pack2d-benchmark report output/results.json -o output/report.html
```

| Flag | Default | Description |
|------|---------|-------------|
| `--output / -o` | _(input with .html ext)_ | Output HTML file path |
| `--module-size` | `0` | Override barcode module size in mm (recomputes physical dimensions) |

The report includes compression ratio comparison, smallest encoded size bar chart, serialization impact bar chart, encode speed vs ratio scatter, level sweep line charts per dataset, dictionary impact (when dict results exist), QR Code Physical Size bar chart, QR code feasibility heatmap, and DataMatrix feasibility heatmap. Heatmaps show interactive barcode previews (rendered via bwip-js) on hover.

### `datasets`

List all embedded test datasets with sizes and descriptions.

### `version`

Print tool version, Go version, and OS/architecture.

## Embedded Datasets

| Name | Type | Size | Description |
|------|------|------|-------------|
| tiny-json | json | 36 B | Minimal JSON object |
| small-json | json | ~540 B | User profile with nested objects |
| medium-json | json | ~4.6 KB | Product catalog (5 products) |
| large-json | json | ~42 KB | Array of 100 user records |
| repetitive-json | json | ~21 KB | 100 identical objects (compression best-case) |
| tiny-xml | xml | 80 B | Minimal XML element (3 attributes) |
| small-xml | xml | ~643 B | User profile in XML with nested elements |
| medium-xml | xml | ~6 KB | Product catalog in XML (5 products) |
| large-xml | xml | ~48 KB | 100 user records in XML |
| repetitive-xml | xml | ~18 KB | 100 identical sensor measurements in XML (compression best-case) |
| high-entropy | binary | 2 KB | PRNG output seed=42 (compression worst-case) |

## Scenarios

**compression** — Benchmarks all algorithm x level x input-type (raw, json, xml, cbor) combinations per dataset. By default covers the full level range for each algorithm (zlib 1–9, zstd 1–19, brotli 0–11). When `--dict` is provided, zstd configurations are additionally benchmarked with dictionary compression. Reports encode/decode timing, compression ratio, QR code feasibility, and barcode physical dimensions (modules per side, size in mm based on `--module-size`). Incompatible input types (e.g., binary data with JSON serialization) are silently skipped.

**barcode** — Finds the best compression config per dataset for barcode use, including dictionary variants when `--dict` is provided. Shows QR code (L/M/Q/H) and DataMatrix feasibility with PASS/FAIL, including QR version and physical size. Incompatible input types are skipped; real errors are propagated.

### Sweet-Spot Recommendations

The summary identifies a "sweet spot" config per dataset — the last configuration where marginal ratio improvement exceeds 0.05% per microsecond of additional encode time. When no configuration meets this threshold, the recommendation falls back to the fastest config and the label indicates that no sweet spot was found.

## Development

```bash
task test       # Run tests with race detector
task lint       # Run golangci-lint
task fmt        # Format source files
task ci         # Full CI pipeline
```

## License

See [LICENSE](LICENSE).
