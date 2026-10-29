package disk

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type AnalyzeResult struct {
	Path string
	Size int64
}

func Analyze(path string, top int) ([]AnalyzeResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("access path %s: %w", path, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	if top <= 0 {
		top = 10
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", path, err)
	}

	results := make([]AnalyzeResult, 0, len(entries))

	for _, entry := range entries {
		itemPath := filepath.Join(path, entry.Name())

		// 分析根目录时，跳过虚拟文件系统
		if filepath.Clean(path) == "/" {
			switch itemPath {
			case "/proc", "/sys", "/dev", "/run":
				continue
			}
		}

		var totalSize int64

		err := filepath.WalkDir(itemPath, func(currentPath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}

			if entry.IsDir() {
				return nil
			}

			info, err := entry.Info()
			if err != nil {
				return err
			}

			if info.Mode().IsRegular() {
				totalSize += info.Size()
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("analyze %s: %w", itemPath, err)
		}

		results = append(results, AnalyzeResult{
			Path: itemPath,
			Size: totalSize,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Size > results[j].Size
	})

	if len(results) > top {
		results = results[:top]
	}

	return results, nil
}

func PrintAnalyze(results []AnalyzeResult) {
	const gib = 1024 * 1024 * 1024
	const mib = 1024 * 1024

	fmt.Printf("%-60s %12s\n", "Path", "Size")

	for _, item := range results {
		size := float64(item.Size) / float64(mib)
		unit := "MiB"

		if item.Size >= gib {
			size = float64(item.Size) / float64(gib)
			unit = "GiB"
		}

		fmt.Printf("%-60s %10.2f %s\n", item.Path, size, unit)
	}
}