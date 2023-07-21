package relative

import (
	"path"
	"runtime"
)

func Filepath(relPath string) string {
	_, callerFile, _, _ := runtime.Caller(1)
	dir := path.Dir(callerFile)
	return path.Join(dir, relPath)
}
