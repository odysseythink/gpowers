// Package util provides common error handling helpers.
package util

import (
	"os"
	"syscall"
)

// SafeUnlink removes a file, ignoring "not exists" errors.
func SafeUnlink(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// SafeUnlinkQuiet removes a file, ignoring all errors.
func SafeUnlinkQuiet(path string) {
	_ = os.Remove(path)
}

// IsProcessAlive checks whether a process with the given PID is running.
func IsProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
