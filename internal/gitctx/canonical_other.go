//go:build !windows

package gitctx

import "path/filepath"

func canonicalPath(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
