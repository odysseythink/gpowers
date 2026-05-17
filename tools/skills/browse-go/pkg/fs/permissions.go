// Package fs provides secure file creation utilities.
package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// MkdirSecure creates a directory with 0700 permissions.
func MkdirSecure(path string) error {
	return os.MkdirAll(path, 0o700)
}

// WriteSecureFile writes data to path with 0600 permissions.
// The parent directory is created if it does not exist.
func WriteSecureFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := MkdirSecure(dir); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return os.WriteFile(path, data, 0o600)
}

// AppendSecureFile appends data to path, creating it with 0600 if needed.
func AppendSecureFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}
