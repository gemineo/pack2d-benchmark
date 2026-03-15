package dataset

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed testdata
var embeddedFS embed.FS

type datasetMeta struct {
	Type        string
	Description string
}

var metadataRegistry = map[string]datasetMeta{
	"json/tiny-json.json":          {Type: "json", Description: "Minimal JSON object (3 fields)"},
	"json/small-json.json":         {Type: "json", Description: "Synthetic user profile with nested objects"},
	"json/medium-json.json":        {Type: "json", Description: "Product catalog with 5 products, nested arrays/objects"},
	"json/large-json.json":         {Type: "json", Description: "Array of 100 user records with addresses"},
	"json/repetitive-json.json":    {Type: "json", Description: "100 identical sensor measurements (compression best-case)"},
	"xml/tiny-xml.xml":             {Type: "xml", Description: "Minimal XML element (3 attributes)"},
	"xml/small-xml.xml":            {Type: "xml", Description: "User profile in XML with nested elements"},
	"xml/medium-xml.xml":           {Type: "xml", Description: "Product catalog in XML (5 products)"},
	"xml/large-xml.xml":            {Type: "xml", Description: "100 user records in XML"},
	"xml/repetitive-xml.xml":       {Type: "xml", Description: "100 identical sensor measurements in XML (compression best-case)"},
	"adversarial/high-entropy.bin": {Type: "binary", Description: "Seeded PRNG output (seed=42, compression worst-case)"},
}

// LoadEmbedded loads all embedded test datasets.
func LoadEmbedded() ([]Dataset, error) {
	var datasets []Dataset

	err := fs.WalkDir(embeddedFS, "testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk embedded: %w", err)
		}
		if d.IsDir() {
			return nil
		}

		data, readErr := embeddedFS.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read embedded %s: %w", path, readErr)
		}

		// Strip "testdata/" prefix for registry lookup.
		relPath := strings.TrimPrefix(path, "testdata/")
		meta, ok := metadataRegistry[relPath]
		if !ok {
			meta = datasetMeta{Type: "unknown", Description: "No description"}
		}

		name := filepath.Base(path)
		name = name[:len(name)-len(filepath.Ext(name))] // strip extension

		datasets = append(datasets, Dataset{
			Name:        name,
			Type:        meta.Type,
			Source:      "embedded",
			Data:        data,
			Size:        len(data),
			Description: meta.Description,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(datasets, func(i, j int) bool {
		return datasets[i].Size < datasets[j].Size
	})

	return datasets, nil
}
