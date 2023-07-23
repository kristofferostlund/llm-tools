package relative

import (
	"context"
	"fmt"
	"os"
	"path"
	"runtime"
)

func Filepath(relPath string) string {
	return filepath(relPath, 2) // 2 is the caller's depth
}

func filepath(relPath string, depth int) string {
	_, callerFile, _, _ := runtime.Caller(depth)
	dir := path.Dir(callerFile)
	return path.Join(dir, relPath)
}

func Files(ctx context.Context, relPaths ...string) (map[string]string, error) {
	files := make(map[string]string)
	for _, relPath := range relPaths {
		file, err := readFile(ctx, filepath(relPath, 2))
		if err != nil {
			return nil, fmt.Errorf("getting file: %w", err)
		}
		files[relPath] = file
	}
	return files, nil
}

func FileContent(ctx context.Context, relPath string) (string, error) {
	return readFile(ctx, filepath(relPath, 2))
}

func readFile(ctx context.Context, absPath string) (string, error) {
	b, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	return string(b), nil
}
