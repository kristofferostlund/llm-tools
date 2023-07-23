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

func FileContent(ctx context.Context, relPath string) (string, error) {
	b, err := os.ReadFile(filepath(relPath, 2))
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	return string(b), nil
}
