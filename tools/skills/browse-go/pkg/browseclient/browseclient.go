// Package browseclient provides a thin Go SDK for browser-skill scripts to
// drive the browse daemon over loopback HTTP.
//
// Auth resolution (in order):
//  1. GSTACK_PORT + GSTACK_SKILL_TOKEN env vars (set by `$B skill run`).
//  2. BROWSE_STATE_FILE env var or <project>/.gstack/browse.json fallback.
//
// Usage:
//
//	client, err := browseclient.New()
//	if err != nil { log.Fatal(err) }
//	out, err := client.Goto("https://example.com")
//
// Or use the singleton for quick scripts:
//
//	out, err := browseclient.Default.Goto("https://example.com")
package browseclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"browse-go/pkg/config"
)

// Client is a thin HTTP wrapper over the browse daemon's /command endpoint.
type Client struct {
	Port      int
	Token     string
	TabID     int
	Timeout   time.Duration
	HTTPClient *http.Client
}

// Options configures a Client.
type Options struct {
	Port    int
	Token   string
	TabID   int
	Timeout time.Duration
}

// New creates a Client with auto-resolved auth.
func New(opts ...Options) (*Client, error) {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}

	port, token, err := resolveAuth(opt.Port, opt.Token)
	if err != nil {
		return nil, err
	}

	timeout := opt.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	tabID := opt.TabID
	if tabID == 0 {
		if v := os.Getenv("BROWSE_TAB"); v != "" {
			tabID, _ = strconv.Atoi(v)
		}
	}

	return &Client{
		Port:       port,
		Token:      token,
		TabID:      tabID,
		Timeout:    timeout,
		HTTPClient: &http.Client{Timeout: timeout},
	}, nil
}

// resolveAuth finds port + token from env or state file.
func resolveAuth(overridePort int, overrideToken string) (int, string, error) {
	if overridePort != 0 && overrideToken != "" {
		return overridePort, overrideToken, nil
	}

	// 1. Env vars
	if envPort := os.Getenv("GSTACK_PORT"); envPort != "" {
		if envToken := os.Getenv("GSTACK_SKILL_TOKEN"); envToken != "" {
			port, err := strconv.Atoi(envPort)
			if err == nil {
				if overridePort != 0 {
					port = overridePort
				}
				token := envToken
				if overrideToken != "" {
					token = overrideToken
				}
				return port, token, nil
			}
		}
	}

	// 2. State file fallback
	stateFile := os.Getenv("BROWSE_STATE_FILE")
	if stateFile == "" {
		cfg := config.Resolve(nil)
		stateFile = cfg.StateFile
	}
	if stateFile != "" {
		data, err := os.ReadFile(stateFile)
		if err == nil {
			var s struct {
				Port  int    `json:"port"`
				Token string `json:"token"`
			}
			if json.Unmarshal(data, &s) == nil && s.Port != 0 && s.Token != "" {
				port := s.Port
				if overridePort != 0 {
					port = overridePort
				}
				token := s.Token
				if overrideToken != "" {
					token = overrideToken
				}
				return port, token, nil
			}
		}
	}

	return 0, "", fmt.Errorf(
		"browseclient: cannot resolve daemon port + token. " +
			"Run via `$B skill run` (sets GSTACK_PORT + GSTACK_SKILL_TOKEN) " +
			"or ensure a daemon state file exists",
	)
}

// Command sends an arbitrary command to the daemon.
func (c *Client) Command(name string, args []string) (string, error) {
	bodyMap := map[string]interface{}{
		"command": name,
		"args":    args,
	}
	if c.TabID != 0 {
		bodyMap["tabId"] = c.TabID
	}
	body, _ := json.Marshal(bodyMap)

	req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/command", c.Port), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return "", fmt.Errorf("browseclient: daemon not running on port %d", c.Port)
		}
		return "", fmt.Errorf("browseclient: %w", err)
	}
	defer resp.Body.Close()

	textBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	text := string(textBytes)

	if resp.StatusCode >= 400 {
		msg := fmt.Sprintf("browseclient: command %q failed with status %d", name, resp.StatusCode)
		if text != "" {
			msg += ": " + text
		}
		return "", fmt.Errorf("%s", msg)
	}
	return text, nil
}

// ─── Navigation ─────────────────────────────────────────────────

func (c *Client) Goto(url string) (string, error)     { return c.Command("goto", []string{url}) }
func (c *Client) Back() (string, error)                { return c.Command("back", nil) }
func (c *Client) Forward() (string, error)             { return c.Command("forward", nil) }
func (c *Client) Reload() (string, error)              { return c.Command("reload", nil) }
func (c *Client) Wait(arg string) (string, error)      { return c.Command("wait", []string{arg}) }
func (c *Client) Viewport(w, h int) (string, error)    { return c.Command("viewport", []string{strconv.Itoa(w), strconv.Itoa(h)}) }

// ─── Reading ────────────────────────────────────────────────────

func (c *Client) Text(selector string) (string, error)          { return c.Command("text", []string{selector}) }
func (c *Client) HTML() (string, error)                         { return c.Command("html", nil) }
func (c *Client) Links() (string, error)                        { return c.Command("links", nil) }
func (c *Client) Forms() (string, error)                        { return c.Command("forms", nil) }
func (c *Client) Accessibility() (string, error)                { return c.Command("accessibility", nil) }
func (c *Client) Media(flags ...string) (string, error)         { return c.Command("media", flags) }
func (c *Client) Data(flags ...string) (string, error)          { return c.Command("data", flags) }

// ─── Interaction ────────────────────────────────────────────────

func (c *Client) Click(ref string) (string, error)              { return c.Command("click", []string{ref}) }
func (c *Client) Fill(ref, value string) (string, error)        { return c.Command("fill", []string{ref, value}) }
func (c *Client) Select(ref, value string) (string, error)      { return c.Command("select", []string{ref, value}) }
func (c *Client) Hover(ref string) (string, error)              { return c.Command("hover", []string{ref}) }
func (c *Client) Type(text string) (string, error)              { return c.Command("type", []string{text}) }
func (c *Client) Press(key string) (string, error)              { return c.Command("press", []string{key}) }
func (c *Client) Scroll(args ...string) (string, error)         { return c.Command("scroll", args) }

// ─── Snapshot + Screenshot ──────────────────────────────────────

func (c *Client) Snapshot(flags ...string) (string, error)      { return c.Command("snapshot", flags) }
func (c *Client) Screenshot(args ...string) (string, error)     { return c.Command("screenshot", args) }

// ─── Singleton ──────────────────────────────────────────────────

// defaultClient is the lazily-initialized singleton.
var defaultClient *Client

// Default returns the global singleton. Safe for concurrent use after first call.
func Default() *Client {
	if defaultClient == nil {
		c, err := New()
		if err != nil {
			// Return a client that will fail on every call with the resolution error.
			c = &Client{}
		}
		defaultClient = c
	}
	return defaultClient
}

// SetDefault overrides the global singleton (useful for testing).
func SetDefault(c *Client) {
	defaultClient = c
}
