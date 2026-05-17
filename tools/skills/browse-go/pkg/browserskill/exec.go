package browserskill

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindExecutable locates the executable entrypoint for a skill.
// It searches the skill directory for files in this order:
//   1. "run" — if present and executable (or any file on Windows)
//   2. "script" — if present and executable
//   3. "script.sh" — shell script
//   4. "script.go" — Go source (caller should use "go run")
//   5. "script.py" — Python script
//
// Returns the absolute path and a suggested interpreter (empty = direct exec).
func FindExecutable(dir string) (path, interpreter string, err error) {
	// Order of preference
	candidates := []struct {
		name        string
		interpreter string
		requireExec bool
	}{
		{"run", "", true},
		{"script", "", true},
		{"script.sh", "/bin/sh", false},
		{"script.go", "go run", false},
		{"script.py", "python3", false},
	}

	for _, c := range candidates {
		p := filepath.Join(dir, c.name)
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		if c.requireExec && !isExecutable(info) {
			continue
		}
		return p, c.interpreter, nil
	}

	return "", "", fmt.Errorf("no executable found in skill directory %s", dir)
}

// FindTestExecutable locates the test entrypoint for a skill.
// Searches for: test, run_test, script_test.go (in that order).
func FindTestExecutable(dir string) (path, interpreter string, err error) {
	candidates := []struct {
		name        string
		interpreter string
		requireExec bool
	}{
		{"test", "", true},
		{"run_test", "", true},
		{"script_test.go", "go test", false},
	}

	for _, c := range candidates {
		p := filepath.Join(dir, c.name)
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		if c.requireExec && !isExecutable(info) {
			continue
		}
		return p, c.interpreter, nil
	}

	return "", "", fmt.Errorf("no test executable found in skill directory %s", dir)
}

// isExecutable reports whether the file info indicates an executable file.
// On Unix this checks the owner-executable bit; on Windows every file is
// considered potentially executable.
func isExecutable(info os.FileInfo) bool {
	if info == nil {
		return false
	}
	// Unix: check mode bits
	mode := info.Mode()
	return mode&0111 != 0
}
