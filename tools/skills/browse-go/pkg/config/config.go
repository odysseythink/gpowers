// Package config resolves browse runtime configuration and paths.
package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"browse-go/pkg/fs"
	"browse-go/pkg/util"
)

// BrowseConfig holds all resolved paths for a browse session.
type BrowseConfig struct {
	ProjectDir string
	StateDir   string
	StateFile  string
	ConsoleLog string
	NetworkLog string
	DialogLog  string
	AuditLog   string
}

// Resolve computes browse config from environment.
func Resolve(env map[string]string) *BrowseConfig {
	var stateFile, stateDir, projectDir string

	if env == nil {
		env = make(map[string]string)
		for _, e := range os.Environ() {
			if k, v, ok := strings.Cut(e, "="); ok {
				env[k] = v
			}
		}
	}

	if sf := env["BROWSE_STATE_FILE"]; sf != "" {
		stateFile = sf
		stateDir = filepath.Dir(stateFile)
		projectDir = filepath.Dir(stateDir)
	} else {
		projectDir = GitRoot()
		if projectDir == "" {
			projectDir, _ = os.Getwd()
		}
		stateDir = filepath.Join(projectDir, ".gstack")
		stateFile = filepath.Join(stateDir, "browse.json")
	}

	return &BrowseConfig{
		ProjectDir: projectDir,
		StateDir:   stateDir,
		StateFile:  stateFile,
		ConsoleLog: filepath.Join(stateDir, "browse-console.log"),
		NetworkLog: filepath.Join(stateDir, "browse-network.log"),
		DialogLog:  filepath.Join(stateDir, "browse-dialog.log"),
		AuditLog:   filepath.Join(stateDir, "browse-audit.jsonl"),
	}
}

// EnsureStateDir creates the state directory and adds .gstack/ to .gitignore.
func EnsureStateDir(cfg *BrowseConfig) error {
	if err := fs.MkdirSecure(cfg.StateDir); err != nil {
		return fmt.Errorf("cannot create state directory %s: %w", cfg.StateDir, err)
	}

	gitignore := filepath.Join(cfg.ProjectDir, ".gitignore")
	data, err := os.ReadFile(gitignore)
	if err != nil {
		if !os.IsNotExist(err) {
			logWarning(cfg, "could not read .gitignore: %v", err)
		}
		return nil
	}
	content := string(data)
	if strings.Contains(content, ".gstack") {
		return nil
	}
	sep := ""
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		sep = "\n"
	}
	if err := fs.AppendSecureFile(gitignore, []byte(sep+".gstack/\n")); err != nil {
		logWarning(cfg, "could not update .gitignore: %v", err)
	}
	return nil
}

// GitRoot returns the git repository root or empty string.
func GitRoot() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// RemoteSlug derives owner-repo from git remote origin.
func RemoteSlug() string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err == nil {
		url := strings.TrimSpace(string(out))
		if i := strings.LastIndex(url, "/"); i >= 0 {
			part := url[i+1:]
			part = strings.TrimSuffix(part, ".git")
			j := strings.LastIndex(url[:i], "/")
			if j < 0 {
				j = strings.LastIndex(url[:i], ":")
			}
			if j >= 0 {
				return url[j+1 : i] + "-" + part
			}
		}
	}
	root := GitRoot()
	if root != "" {
		return filepath.Base(root)
	}
	wd, _ := os.Getwd()
	return filepath.Base(wd)
}

// Home returns the gstack home directory.
func Home() string {
	if v := os.Getenv("GSTACK_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gstack")
}

// ChromiumProfile resolves the Chromium user data directory.
func ChromiumProfile(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if v := os.Getenv("CHROMIUM_PROFILE"); v != "" {
		return v
	}
	return filepath.Join(Home(), "chromium-profile")
}

// CleanSingletonLocks removes stale Chromium lockfiles.
func CleanSingletonLocks(userDataDir string) {
	if !filepath.IsAbs(userDataDir) {
		return
	}
	resolved := filepath.Clean(userDataDir)
	base := filepath.Base(resolved)
	explicit := os.Getenv("CHROMIUM_PROFILE")
	var explicitAbs string
	if explicit != "" && filepath.IsAbs(explicit) {
		explicitAbs = filepath.Clean(explicit)
	}
	safe := base == "chromium-profile" || (explicitAbs != "" && resolved == explicitAbs)
	if !safe {
		return
	}
	for _, f := range []string{"SingletonLock", "SingletonSocket", "SingletonCookie"} {
		util.SafeUnlinkQuiet(filepath.Join(resolved, f))
	}
}

func logWarning(cfg *BrowseConfig, format string, args ...any) {
	logPath := filepath.Join(cfg.StateDir, "browse-server.log")
	msg := fmt.Sprintf("[%s] Warning: "+format+"\n", append([]any{time.Now().Format(time.RFC3339)}, args...)...)
	_ = fs.AppendSecureFile(logPath, []byte(msg))
}
