package dataset

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// LoadCustom loads datasets from a directory on disk.
func LoadCustom(dirPath string) ([]Dataset, error) {
	var datasets []Dataset

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk custom dir: %w", err)
		}
		if d.IsDir() {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read custom file %s: %w", path, readErr)
		}

		name := filepath.Base(path)
		ext := filepath.Ext(name)
		nameNoExt := name[:len(name)-len(ext)]

		datasets = append(datasets, Dataset{
			Name:        nameNoExt,
			Type:        inferType(ext),
			Source:      path,
			Data:        data,
			Size:        len(data),
			Description: fmt.Sprintf("Custom file: %s", name),
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return datasets, nil
}

func inferType(ext string) string {
	switch ext {
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".txt", ".csv", ".tsv":
		return "text"
	default:
		return "binary"
	}
}
