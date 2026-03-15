package dataset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCustom(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.json"), []byte(`{"a":1}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0x00, 0x01}, 0o644))

	datasets, err := LoadCustom(dir)
	require.NoError(t, err)
	require.Len(t, datasets, 2)

	// Results are not sorted by size (unlike embedded), order depends on WalkDir.
	byName := make(map[string]Dataset)
	for _, ds := range datasets {
		byName[ds.Name] = ds
	}

	jsonDS := byName["test"]
	assert.Equal(t, "json", jsonDS.Type)
	assert.Equal(t, 7, jsonDS.Size)
	assert.Equal(t, []byte(`{"a":1}`), jsonDS.Data)
	assert.Contains(t, jsonDS.Source, "test.json")
	assert.Contains(t, jsonDS.Description, "Custom file")

	binDS := byName["data"]
	assert.Equal(t, "binary", binDS.Type)
	assert.Equal(t, 2, binDS.Size)
}

func TestLoadCustom_NestedDirectories(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "nested.xml"), []byte("<a/>"), 0o644))

	datasets, err := LoadCustom(dir)
	require.NoError(t, err)
	require.Len(t, datasets, 1)
	assert.Equal(t, "nested", datasets[0].Name)
	assert.Equal(t, "xml", datasets[0].Type)
}

func TestLoadCustom_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	datasets, err := LoadCustom(dir)
	require.NoError(t, err)
	assert.Empty(t, datasets)
}

func TestLoadCustom_NonExistentDirectory(t *testing.T) {
	_, err := LoadCustom("/nonexistent/path/that/does/not/exist")
	require.Error(t, err)
}

func TestLoadCustom_TextFileTypes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.csv"), []byte("a,b\n1,2"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tab.tsv"), []byte("x\ty"), 0o644))

	datasets, err := LoadCustom(dir)
	require.NoError(t, err)
	require.Len(t, datasets, 3)

	for _, ds := range datasets {
		assert.Equal(t, "text", ds.Type, "expected text type for %s", ds.Name)
	}
}

func TestInferType(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".json", "json"},
		{".xml", "xml"},
		{".txt", "text"},
		{".csv", "text"},
		{".tsv", "text"},
		{".bin", "binary"},
		{".dat", "binary"},
		{"", "binary"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			assert.Equal(t, tt.want, inferType(tt.ext))
		})
	}
}
