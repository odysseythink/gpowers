// Package platform provides cross-platform path utilities for browse.
package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

// IsWindows reports whether the current OS is Windows.
var IsWindows = runtime.GOOS == "windows"

// TempDir returns the platform temporary directory.
func TempDir() string {
	if IsWindows {
		return os.TempDir()
	}
	return "/tmp"
}

// IsPathWithin reports whether resolvedPath is within dir.
func IsPathWithin(resolvedPath, dir string) bool {
	resolvedPath = filepath.Clean(resolvedPath)
	dir = filepath.Clean(dir)
	if resolvedPath == dir {
		return true
	}
	prefix := dir + string(filepath.Separator)
	return len(resolvedPath) > len(prefix) && resolvedPath[:len(prefix)] == prefix
}
