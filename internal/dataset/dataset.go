package dataset

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gemineo/pack2d"
)

// Dataset represents a test dataset for benchmarking.
type Dataset struct {
	Name        string
	Type        string // "json", "xml", "text", "binary"
	Source      string // "embedded" or file path
	Data        []byte
	Size        int
	Description string
}

// InferInputType maps a filename extension to a pack2d InputType.
func InferInputType(filename string) pack2d.InputType {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return pack2d.JSON
	case ".xml":
		return pack2d.XML
	default:
		return pack2d.Raw
	}
}

// FormatSize returns a human-readable file size.
func FormatSize(bytes int) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
}
