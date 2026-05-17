package terminal

import (
	"fmt"
	"os"
	"path/filepath"
)

// WritePortFile atomically writes the agent's listen port to disk.
func WritePortFile(dir string, port int) error {
	path := filepath.Join(dir, "terminal-port")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(fmt.Sprintf("%d", port)), 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ReadPortFile reads the agent's listen port from disk.
func ReadPortFile(dir string) (int, error) {
	path := filepath.Join(dir, "terminal-port")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var port int
	_, err = fmt.Sscanf(string(data), "%d", &port)
	return port, err
}

// RemovePortFile cleans up the port file.
func RemovePortFile(dir string) {
	_ = os.Remove(filepath.Join(dir, "terminal-port"))
}

// WriteInternalTokenFile writes the internal token so parent can authenticate.
func WriteInternalTokenFile(dir, token string) error {
	path := filepath.Join(dir, "terminal-internal-token")
	return os.WriteFile(path, []byte(token), 0600)
}

// ReadInternalTokenFile reads the internal token from disk.
func ReadInternalTokenFile(dir string) (string, error) {
	path := filepath.Join(dir, "terminal-internal-token")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RemoveInternalTokenFile cleans up.
func RemoveInternalTokenFile(dir string) {
	_ = os.Remove(filepath.Join(dir, "terminal-internal-token"))
}
