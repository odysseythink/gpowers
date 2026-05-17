// Package tunnel manages ngrok tunnel lifecycle for remote agent access.
package tunnel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Tunnel holds the state of an active ngrok tunnel.
type Tunnel struct {
	cmd       *exec.Cmd
	listener  *http.Server
	url       string
	localPort int
}

// Start launches ngrok pointing at the given local port.
// It starts an ngrok subprocess and polls the local API for the public URL.
func Start(localPort int, authtoken string) (*Tunnel, error) {
	if authtoken == "" {
		return nil, fmt.Errorf("no ngrok authtoken provided")
	}

	// Kill any stale ngrok processes for this port
	_ = exec.Command("pkill", "-f", fmt.Sprintf("ngrok http 127.0.0.1:%d", localPort)).Run()
	time.Sleep(100 * time.Millisecond)

	cmd := exec.Command("ngrok", "http", fmt.Sprintf("127.0.0.1:%d", localPort), "--authtoken", authtoken)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("ngrok start failed: %w", err)
	}

	// Poll ngrok local API for tunnel URL
	var tunnelURL string
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		resp, err := http.Get("http://127.0.0.1:4040/api/tunnels")
		if err != nil {
			continue
		}
		var result struct {
			Tunnels []struct {
				PublicURL string `json:"public_url"`
			} `json:"tunnels"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && len(result.Tunnels) > 0 {
			tunnelURL = result.Tunnels[0].PublicURL
			resp.Body.Close()
			break
		}
		resp.Body.Close()
	}

	if tunnelURL == "" {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("ngrok did not expose a URL within timeout")
	}

	return &Tunnel{
		cmd:       cmd,
		url:       tunnelURL,
		localPort: localPort,
	}, nil
}

// URL returns the public ngrok URL.
func (t *Tunnel) URL() string {
	return t.url
}

// Close shuts down the ngrok tunnel.
func (t *Tunnel) Close() error {
	var err error
	if t.cmd != nil && t.cmd.Process != nil {
		err = t.cmd.Process.Kill()
	}
	return err
}

// ResolveAuthtoken finds the ngrok authtoken from env or config files.
func ResolveAuthtoken() string {
	if tok := os.Getenv("NGROK_AUTHTOKEN"); tok != "" {
		return tok
	}

	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}

	// ~/.gstack/ngrok.env
	ngrokEnv := filepath.Join(home, ".gstack", "ngrok.env")
	if data, err := os.ReadFile(ngrokEnv); err == nil {
		if m := regexp.MustCompile(`^NGROK_AUTHTOKEN=(.+)$`).FindStringSubmatch(string(data)); len(m) > 1 {
			return strings.TrimSpace(m[1])
		}
	}

	// ngrok config files
	configs := []string{
		filepath.Join(home, "Library", "Application Support", "ngrok", "ngrok.yml"),
		filepath.Join(home, ".config", "ngrok", "ngrok.yml"),
		filepath.Join(home, ".ngrok2", "ngrok.yml"),
	}
	for _, path := range configs {
		if data, err := os.ReadFile(path); err == nil {
			if m := regexp.MustCompile(`authtoken:\s*(.+)`).FindStringSubmatch(string(data)); len(m) > 1 {
				return strings.TrimSpace(m[1])
			}
		}
	}
	return ""
}

// TunnelCommands is the allowlist for commands reachable via the tunnel surface.
var TunnelCommands = map[string]bool{
	"connect":     true,
	"goto":        true,
	"back":        true,
	"forward":     true,
	"reload":      true,
	"text":        true,
	"html":        true,
	"links":       true,
	"forms":       true,
	"accessibility": true,
	"snapshot":    true,
	"click":       true,
	"fill":        true,
	"scroll":      true,
	"wait":        true,
	"screenshot":  true,
	"status":      true,
	"tabs":        true,
	"tab":         true,
	"newtab":      true,
	"closetab":    true,
	"stop":        true,
}

// IsTunnelAllowed returns whether a command is allowed on the tunnel surface.
func IsTunnelAllowed(command string) bool {
	return TunnelCommands[command]
}
