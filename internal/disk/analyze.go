package disk

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
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

	rootStat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("cannot get filesystem device: %s", path)
	}
	rootDev := rootStat.Dev

	// 保存每个路径自身占用的磁盘块大小。
	sizes := make(map[string]int64)

	// 保存路径是否为目录。
	dirs := make(map[string]bool)

	err = filepath.WalkDir(path, func(
		currentPath string,
		entry fs.DirEntry,
		walkErr error,
	) error {
		// 忽略权限错误、文件消失等遍历错误。
		if walkErr != nil {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return nil
		}

		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return nil
		}

		// 类似 du -x：不进入其他文件系统。
		if stat.Dev != rootDev {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 使用实际分配的磁盘块大小，单位转换为字节。
		sizes[currentPath] = stat.Blocks * 512
		dirs[currentPath] = entry.IsDir()

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("analyze %s: %w", path, err)
	}

	// 按目录层级从深到浅排列。
	paths := make([]string, 0, len(sizes))
	for p := range sizes {
		paths = append(paths, p)
	}

	sort.Slice(paths, func(i, j int) bool {
		depthI := strings.Count(filepath.Clean(paths[i]), string(os.PathSeparator))
		depthJ := strings.Count(filepath.Clean(paths[j]), string(os.PathSeparator))

		if depthI != depthJ {
			return depthI > depthJ
		}
		return len(paths[i]) > len(paths[j])
	})

	// 将子项大小逐级累加到父目录。
	for _, p := range paths {
		if !dirs[p] {
			continue
		}

		parent := filepath.Dir(p)
		if parent == p {
			continue
		}

		if _, exists := sizes[parent]; exists {
			sizes[parent] += sizes[p]
		}
	}

	// 转换成结果列表。
	results := make([]AnalyzeResult, 0, len(sizes))
	for p, size := range sizes {
		results = append(results, AnalyzeResult{
			Path: p,
			Size: size,
		})
	}

	// 从大到小排序。
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

	fmt.Printf("%-75s %12s\n", "Path", "Size")

	for _, item := range results {
		size := float64(item.Size) / float64(mib)
		unit := "MiB"

		if item.Size >= gib {
			size = float64(item.Size) / float64(gib)
			unit = "GiB"
		}

		fmt.Printf("%-75s %10.2f %s\n", item.Path, size, unit)
	}
}