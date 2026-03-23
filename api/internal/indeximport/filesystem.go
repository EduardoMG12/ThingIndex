package indeximport

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func DiscoverExportFiles(rootPath string) ([]string, error) {
	trimmedPath := strings.TrimSpace(rootPath)
	if trimmedPath == "" {
		return nil, errors.New("path is required")
	}

	info, err := os.Stat(trimmedPath)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		if !isJSONExportFile(trimmedPath) {
			return nil, errors.New("path must point to a .json export file or a directory containing .json exports")
		}

		return []string{trimmedPath}, nil
	}

	files := make([]string, 0)
	walkErr := filepath.WalkDir(trimmedPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if isJSONExportFile(path) {
			files = append(files, path)
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	sort.Strings(files)
	if len(files) == 0 {
		return nil, errors.New("no .json export files were found in the provided path")
	}

	return files, nil
}

func isJSONExportFile(path string) bool {
	return strings.EqualFold(filepath.Ext(strings.TrimSpace(path)), ".json")
}
