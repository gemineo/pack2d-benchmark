package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "pack2d-benchmark")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build failed: %s", out)
	return bin
}

func TestIntegration_Version(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "version")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "version failed: %s", out)
	assert.Contains(t, string(out), "pack2d-benchmark")
	assert.Contains(t, string(out), "Go:")
}

func TestIntegration_Datasets(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "datasets")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "datasets failed: %s", out)
	assert.Contains(t, string(out), "tiny-json")
	assert.Contains(t, string(out), "high-entropy")
}

func TestIntegration_RunCompression(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "run",
		"--scenarios", "compression",
		"--algorithms", "zstd",
		"--iterations", "2",
		"--warm-up", "1",
		"--quiet",
		"--no-color",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "run failed: %s", out)
	assert.Contains(t, string(out), "Compression Benchmark")
	assert.Contains(t, string(out), "zstd")
}

func TestIntegration_JSONExport(t *testing.T) {
	bin := buildBinary(t)
	exportFile := filepath.Join(t.TempDir(), "results.json")
	cmd := exec.Command(bin, "run",
		"--scenarios", "compression",
		"--algorithms", "zstd",
		"--iterations", "2",
		"--warm-up", "1",
		"--quiet",
		"--no-color",
		"--export", exportFile,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "run with export failed: %s", out)

	data, err := os.ReadFile(exportFile)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), `"metadata"`)
	assert.Contains(t, string(data), `"results"`)
	assert.Contains(t, string(data), `"summary"`)
}
