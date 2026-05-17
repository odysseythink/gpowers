// browse — CLI entry for the gstack browse daemon (Go/chromedp rewrite).
//
// Usage:
//
//	browse server [--proxy <url>] [--headed]
//	browse start [--proxy <url>] [--headed]
//	browse stop
//	browse status
//	browse <command> [args...]
//
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"browse-go/pkg/browser"
	"browse-go/pkg/config"
	"browse-go/pkg/server"
	"browse-go/pkg/socks"
	"browse-go/pkg/terminal"
)

// StateFile is the JSON written to .gstack/browse.json.
type StateFile struct {
	PID           int    `json:"pid"`
	Port          int    `json:"port"`
	Token         string `json:"token"`
	StartedAt     string `json:"startedAt"`
	ServerPath    string `json:"serverPath"`
	BinaryVersion string `json:"binaryVersion,omitempty"`
	ConfigHash    string `json:"configHash,omitempty"`
	Mode          string `json:"mode,omitempty"` // "headed" or empty
}

// GlobalFlags holds parsed --proxy and --headed flags.
type GlobalFlags struct {
	Args           []string
	ProxyURL       string
	Headed         bool
	ConfigHash     string
	RedactedProxy  string
}

// module-level reference to global flags for crash-retry path.
var _globalFlags *GlobalFlags

func main() {
	log.SetFlags(0)

	// Parse global flags
	flags, err := extractGlobalFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "[browse] error: %v\n", err)
		os.Exit(1)
	}
	_globalFlags = flags
	args := flags.Args

	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp()
		os.Exit(0)
	}

	// One-time legacy cleanup
	cleanupLegacyState()

	sub := args[0]
	cmdArgs := args[1:]

	switch sub {
	case "server":
		os.Exit(runServer(flags))
	case "start":
		os.Exit(startDaemon(flags))
	case "stop":
		os.Exit(stopDaemon())
	case "status":
		os.Exit(showStatus())
	case "connect":
		os.Exit(handleConnect(flags))
	case "disconnect":
		os.Exit(handleDisconnect())
	case "pair-agent":
		os.Exit(handlePairAgent(flags, cmdArgs))
	default:
		os.Exit(sendCommand(sub, cmdArgs))
	}
}

func printHelp() {
	fmt.Println(`gstack browse — Fast headless browser for AI coding agents

Usage: browse <command> [args...]
       browse server [--proxy <url>] [--headed]
       browse start [--proxy <url>] [--headed]
       browse stop | status | connect | disconnect

Global flags:
  --proxy <url>   SOCKS5/HTTP proxy (credentials via BROWSE_PROXY_USER/PASS)
  --headed        Launch visible Chrome with sidebar extension

Navigation:     goto <url> | back | forward | reload | url | viewport <WxH>
Content:        text | html [sel] | links | forms | accessibility
Interaction:    click <sel> | fill <sel> <val> | select | hover | type | press
                scroll [sel] | wait <sel|--networkidle|--load>
Inspection:     js <expr> | eval <file> | css <sel> <prop> | attrs <sel>
                console | network | dialog | storage | perf
Visual:         screenshot [path] | pdf [path] | responsive [prefix]
Snapshot:       snapshot [-i] [-c] [-d N] [-s sel] [--diff]
Tabs:           tabs | tab <id> | newtab [url] | closetab [id]
Server:         status | stop | restart | cookie <n>=<v> | header <n>:<v>
Pairing:        pair-agent [--headless]  # Share browser with remote agent`)
}

// ─── Global Flag Parsing ──────────────────────────────────

func extractGlobalFlags(rawArgs []string) (*GlobalFlags, error) {
	out := make([]string, 0, len(rawArgs))
	var proxyURL string
	var headed bool

	for i := 0; i < len(rawArgs); i++ {
		arg := rawArgs[i]
		if arg == "--proxy" {
			if i+1 >= len(rawArgs) {
				return nil, fmt.Errorf("--proxy requires a URL value")
			}
			proxyURL = rawArgs[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--proxy=") {
			proxyURL = strings.TrimPrefix(arg, "--proxy=")
			continue
		}
		if arg == "--headed" {
			headed = true
			continue
		}
		out = append(out, arg)
	}

	// Resolve proxy credentials from env
	canonicalProxy := resolveProxyURL(proxyURL)

	return &GlobalFlags{
		Args:          out,
		ProxyURL:      canonicalProxy,
		Headed:        headed,
		ConfigHash:    computeConfigHash(canonicalProxy, headed),
		RedactedProxy: redactProxyURL(canonicalProxy),
	}, nil
}

func resolveProxyURL(proxyURL string) string {
	if proxyURL == "" {
		return ""
	}
	// Embed credentials from env if present
	user := os.Getenv("BROWSE_PROXY_USER")
	pass := os.Getenv("BROWSE_PROXY_PASS")
	if user == "" && pass == "" {
		return proxyURL
	}
	// Parse and rebuild with creds
	if !strings.Contains(proxyURL, "://") {
		proxyURL = "http://" + proxyURL
	}
	u, err := parseURL(proxyURL)
	if err != nil {
		return proxyURL
	}
	if u.User == nil || u.User.Username() == "" {
		if pass != "" {
			u.User = urlUserPassword(user, pass)
		} else {
			u.User = urlUser(user)
		}
	}
	return u.String()
}

func parseURL(s string) (*parsedURL, error) {
	// Simple URL parser for proxy URLs
	schemeEnd := strings.Index(s, "://")
	if schemeEnd < 0 {
		return nil, fmt.Errorf("invalid URL")
	}
	scheme := s[:schemeEnd]
	rest := s[schemeEnd+3:]

	var user, password, host string
	if at := strings.LastIndex(rest, "@"); at >= 0 {
		cred := rest[:at]
		rest = rest[at+1:]
		if colon := strings.Index(cred, ":"); colon >= 0 {
			user = cred[:colon]
			password = cred[colon+1:]
		} else {
			user = cred
		}
	}
	host = rest

	return &parsedURL{Scheme: scheme, Host: host, User: &urlUserInfo{username: user, password: password}}, nil
}

type parsedURL struct {
	Scheme string
	Host   string
	User   *urlUserInfo
}

func (u *parsedURL) String() string {
	s := u.Scheme + "://"
	if u.User != nil && u.User.Username() != "" {
		if u.User.Password() != "" {
			s += u.User.Username() + ":" + u.User.Password() + "@"
		} else {
			s += u.User.Username() + "@"
		}
	}
	s += u.Host
	return s
}

type urlUserInfo struct {
	username string
	password string
}

func (u *urlUserInfo) Username() string { return u.username }
func (u *urlUserInfo) Password() string { return u.password }

func urlUser(user string) *urlUserInfo {
	return &urlUserInfo{username: user}
}
func urlUserPassword(user, pass string) *urlUserInfo {
	return &urlUserInfo{username: user, password: pass}
}

func computeConfigHash(proxyURL string, headed bool) string {
	h := sha256.New()
	fmt.Fprintf(h, "proxy=%s|headed=%v", proxyURL, headed)
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func redactProxyURL(proxyURL string) string {
	if proxyURL == "" {
		return ""
	}
	u, err := parseURL(proxyURL)
	if err != nil {
		return "<invalid>"
	}
	if u.User != nil && u.User.Password() != "" {
		u.User = urlUserPassword(u.User.Username(), "***")
	}
	return u.String()
}

// ─── Proxy Setup ──────────────────────────────────────────

// setupProxy reads BROWSE_PROXY_URL from the environment and configures the
// BrowserManager accordingly. For authenticated SOCKS5 proxies, it starts a
// local SOCKS5 bridge (Chromium cannot do SOCKS5 auth directly) and points
// Chromium at the bridge. For HTTP/HTTPS or unauthenticated SOCKS5, the URL
// is passed straight through to chromedp's --proxy-server flag.
//
// Returns the running bridge (if any) so the caller can close it on shutdown.
func setupProxy(bm *browser.BrowserManager) (*socks.Bridge, error) {
	proxyURL := os.Getenv("BROWSE_PROXY_URL")
	if proxyURL == "" {
		return nil, nil
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid BROWSE_PROXY_URL: %w", err)
	}

	scheme := u.Scheme
	host := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		switch scheme {
		case "http":
			portStr = "80"
		case "https":
			portStr = "443"
		case "socks5":
			portStr = "1080"
		}
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy port %q: %w", portStr, err)
	}

	user := u.User.Username()
	pass, _ := u.User.Password()

	if scheme == "socks5" && (user != "" || pass != "") {
		// Chromium cannot do SOCKS5 auth natively — use the local bridge.
		up := socks.UpstreamConfig{
			Host:     host,
			Port:     port,
			UserID:   user,
			Password: pass,
		}
		log.Printf("[browse] [proxy] testing SOCKS5 upstream %s:%d ...", host, port)
		if _, err := socks.TestUpstream(up, "", 0); err != nil {
			return nil, fmt.Errorf("SOCKS5 upstream unreachable: %w", err)
		}
		bridge, err := socks.StartBridge(up, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to start SOCKS5 bridge: %w", err)
		}
		localProxy := fmt.Sprintf("socks5://127.0.0.1:%d", bridge.Port)
		bm.SetProxyConfig(&browser.ProxyConfig{Server: localProxy})
		log.Printf("[browse] [proxy] bridge listening on %s → %s:%d", localProxy, host, port)
		return bridge, nil
	}

	// HTTP/HTTPS or unauthenticated SOCKS5 — pass through directly.
	serverURL := fmt.Sprintf("%s://%s:%d", scheme, host, port)
	bm.SetProxyConfig(&browser.ProxyConfig{Server: serverURL})
	log.Printf("[browse] [proxy] using %s", redactProxyURL(proxyURL))
	return nil, nil
}

// ─── Server Mode ──────────────────────────────────────────

func runServer(flags *GlobalFlags) int {
	cfg := config.Resolve(nil)
	if err := config.EnsureStateDir(cfg); err != nil {
		log.Printf("Error: cannot ensure state dir: %v", err)
		return 1
	}

	// Generate auth token
	token := os.Getenv("AUTH_TOKEN")
	if token == "" {
		token = generateToken()
	}

	bm := browser.NewBrowserManager()

	// Configure proxy before launching Chromium (proxy is set at alloc time).
	var proxyBridge *socks.Bridge
	if bridge, err := setupProxy(bm); err != nil {
		log.Printf("Error: %v", err)
		return 1
	} else {
		proxyBridge = bridge
	}

	isHeaded := flags.Headed || os.Getenv("BROWSE_HEADED") == "1"

	if isHeaded {
		// Headed mode: launch visible Chrome with extension
		log.Println("[browse] Launching headed Chromium...")
		port := 0
		if p := os.Getenv("BROWSE_PORT"); p != "" {
			port, _ = strconv.Atoi(p)
		}
		if port == 0 {
			port = 34567
		}
		welcomeURL := fmt.Sprintf("http://127.0.0.1:%d/welcome", port)
		if err := bm.LaunchHeadedWithOptions(welcomeURL, &browser.LaunchHeadedOptions{
			AuthToken:  token,
			ServerPort: port,
		}); err != nil {
			log.Printf("Error: failed to launch headed Chromium: %v", err)
			return 1
		}

	} else {
		// Headless mode
		log.Println("[browse] Launching Chromium...")
		if err := bm.Launch(); err != nil {
			log.Printf("Error: failed to launch Chromium: %v", err)
			return 1
		}
	}
	defer bm.Close()

	// Start HTTP server
	srv := server.New(bm, token)
	if d := os.Getenv("BROWSE_IDLE_TIMEOUT"); d != "" {
		if ms, err := strconv.Atoi(d); err == nil && ms > 0 {
			srv.SetIdleTimeout(time.Duration(ms) * time.Millisecond)
		}
	}
	port := 0
	if p := os.Getenv("BROWSE_PORT"); p != "" {
		port, _ = strconv.Atoi(p)
	}
	if err := srv.Start(port); err != nil {
		log.Printf("Error: failed to start server: %v", err)
		return 1
	}

	// In headed mode, disable idle timeout unless explicitly set
	if isHeaded && os.Getenv("BROWSE_IDLE_TIMEOUT") == "" {
		srv.SetIdleTimeout(0)
	}

	// Compute version hash (binary modification time)
	binaryVersion := readVersionHash()

	mode := ""
	if flags.Headed || os.Getenv("BROWSE_HEADED") == "1" {
		mode = "headed"
	}

	// Write state file
	state := StateFile{
		PID:           os.Getpid(),
		Port:          srv.Port(),
		Token:         token,
		StartedAt:     time.Now().UTC().Format(time.RFC3339),
		ServerPath:    os.Args[0],
		BinaryVersion: binaryVersion,
		ConfigHash:    flags.ConfigHash,
		Mode:          mode,
	}
	if err := writeState(cfg.StateFile, state); err != nil {
		log.Printf("Warning: cannot write state file: %v", err)
	}
	defer os.Remove(cfg.StateFile)

	log.Printf("[browse] Server ready on port %d", srv.Port())

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, sigTerm)
	<-sigCh

	log.Println("[browse] Shutting down...")

	// Cleanup SOCKS bridge if active
	if proxyBridge != nil {
		_ = proxyBridge.Close()
	}

	// Cleanup terminal agent in headed mode
	if isHeaded {
		_ = killTerminalAgent()
		terminal.RemovePortFile(cfg.StateDir)
		terminal.RemoveInternalTokenFile(cfg.StateDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	return 0
}

// ─── Daemon Lifecycle ─────────────────────────────────────

func startDaemon(flags *GlobalFlags) int {
	release := acquireServerLock()
	if release == nil {
		log.Println("Error: another browse start is in progress")
		return 1
	}
	defer release()

	state, ok := readState()
	if ok && isServerHealthy(state.Port) {
		// Daemon-mismatch check
		if flags.ConfigHash != "" && state.ConfigHash != "" && state.ConfigHash != flags.ConfigHash {
			log.Println("[browse] existing daemon has different config (proxy/headed mismatch).")
			log.Println("[browse] run 'browse disconnect' first to apply --proxy/--headed.")
			return 1
		}
		if flags.ConfigHash != "" && state.ConfigHash == "" && (flags.ProxyURL != "" || flags.Headed) {
			log.Println("[browse] existing daemon was started without --proxy/--headed.")
			log.Println("[browse] run 'browse disconnect' first to apply new flags.")
			return 1
		}
		log.Printf("Daemon already running on port %d (pid %d)", state.Port, state.PID)
		return 0
	}

	// Kill stale PID if present
	if ok && state.PID > 0 {
		_ = safeKill(state.PID, sigTerm)
		time.Sleep(500 * time.Millisecond)
	}

	// Spawn background server process
	cmd := exec.Command(os.Args[0], append([]string{"server"}, flags.Args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	setProcessGroup(cmd)

	// Pass proxy/headed through env
	if flags.ProxyURL != "" {
		cmd.Env = append(os.Environ(), "BROWSE_PROXY_URL="+flags.ProxyURL)
	}
	if flags.Headed {
		cmd.Env = append(os.Environ(), "BROWSE_HEADED=1")
	}
	if flags.ConfigHash != "" {
		cmd.Env = append(os.Environ(), "BROWSE_CONFIG_HASH="+flags.ConfigHash)
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Error: failed to start daemon: %v", err)
		return 1
	}

	// Wait for health check
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		newState, ok := readState()
		if ok && isServerHealthy(newState.Port) {
			log.Printf("Daemon started on port %d (pid %d)", newState.Port, newState.PID)
			return 0
		}
	}

	log.Println("Error: daemon failed to start within timeout")
	return 1
}

func stopDaemon() int {
	state, ok := readState()
	if !ok {
		log.Println("No daemon running")
		return 0
	}

	// Send /stop request
	if isServerHealthy(state.Port) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/stop", state.Port), nil)
		req.Header.Set("Authorization", "Bearer "+state.Token)
		client := &http.Client{Timeout: 5 * time.Second}
		_, _ = client.Do(req)
	}

	// Kill process if still alive
	if state.PID > 0 {
		_ = safeKill(state.PID, sigTerm)
		time.Sleep(300 * time.Millisecond)
		_ = safeKill(state.PID, sigKill)
	}

	cfg := config.Resolve(nil)
	_ = os.Remove(cfg.StateFile)
	log.Println("Daemon stopped")
	return 0
}

func showStatus() int {
	state, ok := readState()
	if !ok {
		log.Println("No daemon running")
		return 1
	}

	healthy := isServerHealthy(state.Port)
	alive := false
	if state.PID > 0 {
		alive = isProcessAlive(state.PID)
	}

	if !healthy || !alive {
		log.Println("Daemon is not responding")
		return 1
	}

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", state.Port))
	if err != nil {
		log.Printf("Health check failed: %v", err)
		return 1
	}
	defer resp.Body.Close()
	var body map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&body)

	fmt.Printf("PID:    %d\n", state.PID)
	fmt.Printf("Port:   %d\n", state.Port)
	fmt.Printf("URL:    %v\n", body["url"])
	fmt.Printf("Tabs:   %v\n", body["tabs"])
	fmt.Printf("Healthy: %v\n", body["healthy"])
	if state.ConfigHash != "" {
		fmt.Printf("Config: %s\n", state.ConfigHash[:8])
	}
	return 0
}

// ─── Command Client ───────────────────────────────────────

func sendCommand(command string, args []string) int {
	state, ok := readState()
	if !ok {
		// BROWSE_NO_AUTOSTART: sidebar agent sets this
		if os.Getenv("BROWSE_NO_AUTOSTART") == "1" {
			log.Println("[browse] Server not available and BROWSE_NO_AUTOSTART is set.")
			log.Println("[browse] Run 'browse start' to launch the daemon.")
			return 1
		}
		// Auto-start daemon
		log.Println("[browse] Daemon not running, starting...")
		if rc := startDaemon(_globalFlags); rc != 0 {
			return rc
		}
		state, ok = readState()
		if !ok {
			log.Println("Error: daemon start failed")
			return 1
		}
	}

	return doSendCommand(state, command, args, 0)
}

func doSendCommand(state StateFile, command string, args []string, retries int) int {
	if !isServerHealthy(state.Port) {
		if retries >= 1 {
			log.Println("[browse] Server crashed twice in a row — aborting")
			return 1
		}
		log.Println("[browse] Server connection lost. Restarting...")
		// Kill old server
		if state.PID > 0 {
			_ = safeKill(state.PID, sigTerm)
			time.Sleep(500 * time.Millisecond)
			_ = safeKill(state.PID, sigKill)
		}
		// Restart with same flags
		if rc := startDaemon(_globalFlags); rc != 0 {
			return rc
		}
		newState, ok := readState()
		if !ok {
			log.Println("Error: restart failed")
			return 1
		}
		return doSendCommand(newState, command, args, retries+1)
	}

	// Extract --tab-id
	tabID, strippedArgs := extractTabID(args)
	args = strippedArgs

	bodyMap := map[string]interface{}{
		"command": command,
		"args":    args,
	}
	if tabID > 0 {
		bodyMap["tabId"] = tabID
	}
	payload, _ := json.Marshal(bodyMap)

	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/command", state.Port),
		bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+state.Token)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Connection error — try crash recovery
		if isConnError(err) {
			if retries >= 1 {
				log.Println("[browse] Server crashed twice in a row — aborting")
				return 1
			}
			log.Println("[browse] Server connection lost. Restarting...")
			if state.PID > 0 {
				_ = safeKill(state.PID, sigTerm)
				time.Sleep(500 * time.Millisecond)
				_ = safeKill(state.PID, sigKill)
			}
			if rc := startDaemon(_globalFlags); rc != 0 {
				return rc
			}
			newState, ok := readState()
			if !ok {
				log.Println("Error: restart failed")
				return 1
			}
			return doSendCommand(newState, command, args, retries+1)
		}
		log.Printf("Error: %v", err)
		return 1
	}
	defer resp.Body.Close()

	// 401 retry: token may have rotated
	if resp.StatusCode == http.StatusUnauthorized {
		log.Println("[browse] Auth failed — server may have restarted. Retrying...")
		newState, ok := readState()
		if ok && newState.Token != state.Token {
			return doSendCommand(newState, command, args, retries)
		}
		log.Println("[browse] Authentication failed")
		return 1
	}

	var result struct {
		OK     bool   `json:"ok"`
		Result string `json:"result"`
		Error  string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
		return 0
	}

	if !result.OK {
		fmt.Fprintln(os.Stderr, result.Error)
		return 1
	}
	fmt.Println(result.Result)
	return 0
}

func isConnError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "connection refused") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "EOF")
}

func extractTabID(args []string) (int, []string) {
	stripped := make([]string, 0, len(args))
	var tabID int
	for i := 0; i < len(args); i++ {
		if args[i] == "--tab-id" {
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
					tabID = n
				}
				i++
			}
			continue
		}
		stripped = append(stripped, args[i])
	}
	return tabID, stripped
}

// ─── Disconnect ───────────────────────────────────────────

func handleDisconnect() int {
	state, ok := readState()
	if !ok {
		log.Println("Not in headed/custom-config mode — nothing to disconnect.")
		return 0
	}
	hasCustomConfig := state.Mode == "headed" || state.ConfigHash != ""
	if !hasCustomConfig {
		log.Println("Not in headed/custom-config mode — nothing to disconnect.")
		return 0
	}

	// Try graceful shutdown first for headed mode
	if state.Mode == "headed" && isServerHealthy(state.Port) {
		payload, _ := json.Marshal(map[string]interface{}{"command": "disconnect", "args": []string{}})
		req, _ := http.NewRequest(http.MethodPost,
			fmt.Sprintf("http://127.0.0.1:%d/command", state.Port),
			bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+state.Token)
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			log.Println("Disconnected from real browser.")
			return 0
		}
	}

	// Force kill
	if state.PID > 0 && isProcessAlive(state.PID) {
		_ = safeKill(state.PID, sigTerm)
		time.Sleep(2 * time.Second)
		if isProcessAlive(state.PID) {
			_ = safeKill(state.PID, sigKill)
		}
	}

	// Clean profile locks
	profileDir := config.ChromiumProfile("")
	for _, f := range []string{"SingletonLock", "SingletonSocket", "SingletonCookie"} {
		_ = os.Remove(filepath.Join(profileDir, f))
	}

	// Kill terminal agent
	_ = killTerminalAgent()
	cfg := config.Resolve(nil)
	terminal.RemovePortFile(cfg.StateDir)
	terminal.RemoveInternalTokenFile(cfg.StateDir)

	_ = os.Remove(cfg.StateFile)
	log.Println("Disconnected (server was unresponsive — force cleaned).")
	return 0
}

// ─── Pair Agent ───────────────────────────────────────────

func handlePairAgent(flags *GlobalFlags, args []string) int {
	state, ok := readState()
	if !ok || !isServerHealthy(state.Port) {
		log.Println("[browse] Server not running. Start with 'browse start' first.")
		return 1
	}

	// POST /pair to get setup key
	var pairReqBody map[string]interface{}
	if hasFlag(args, "--control") {
		pairReqBody = map[string]interface{}{"control": true}
	}
	if hasFlag(args, "--admin") {
		pairReqBody = map[string]interface{}{"admin": true}
	}
	if clientID := flagValue(args, "--client-id"); clientID != "" {
		pairReqBody["clientId"] = clientID
	}

	payload, _ := json.Marshal(pairReqBody)
	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/pair", state.Port),
		bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+state.Token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error: %v", err)
		return 1
	}
	defer resp.Body.Close()

	var pairResp struct {
		SetupKey  string   `json:"setup_key"`
		ExpiresAt string   `json:"expires_at"`
		Scopes    []string `json:"scopes"`
		ServerURL string   `json:"server_url"`
		Error     string   `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pairResp); err != nil {
		log.Printf("Error parsing response: %v", err)
		return 1
	}
	if pairResp.Error != "" {
		log.Printf("Error: %s", pairResp.Error)
		return 1
	}

	fmt.Printf("Setup key: %s\n", pairResp.SetupKey)
	fmt.Printf("Expires:   %s\n", pairResp.ExpiresAt)
	fmt.Printf("Scopes:    %s\n", strings.Join(pairResp.Scopes, ", "))
	fmt.Printf("Server:    %s\n", pairResp.ServerURL)
	fmt.Println("\nShare this setup key with the remote agent.")
	fmt.Println("They should POST to /connect with {\"setup_key\": \"...\"}")
	return 0
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func flagValue(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// ─── Lockfile ─────────────────────────────────────────────

func acquireServerLock() func() {
	cfg := config.Resolve(nil)
	lockPath := cfg.StateFile + ".lock"

	// Try to create exclusively
	fd, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		// Lock held — check if holder is alive
		data, err := os.ReadFile(lockPath)
		if err == nil {
			pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
			if pid > 0 && isProcessAlive(pid) {
				return nil
			}
		}
		// Stale lock — remove and retry
		_ = os.Remove(lockPath)
		return acquireServerLock()
	}
	_, _ = fmt.Fprintf(fd, "%d\n", os.Getpid())
	fd.Close()
	return func() { _ = os.Remove(lockPath) }
}

// ─── State File I/O ───────────────────────────────────────

func readState() (StateFile, bool) {
	cfg := config.Resolve(nil)
	data, err := os.ReadFile(cfg.StateFile)
	if err != nil {
		return StateFile{}, false
	}
	var s StateFile
	if err := json.Unmarshal(data, &s); err != nil {
		return StateFile{}, false
	}
	return s, true
}

func writeState(path string, s StateFile) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ─── Legacy Cleanup ───────────────────────────────────────

func cleanupLegacyState() {
	// Skip on Windows
	if runtime.GOOS == "windows" {
		return
	}
	files, err := os.ReadDir("/tmp")
	if err != nil {
		return
	}
	for _, f := range files {
		name := f.Name()
		if strings.HasPrefix(name, "browse-server") && strings.HasSuffix(name, ".json") {
			fullPath := filepath.Join("/tmp", name)
			data, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}
			var legacy struct {
				PID int `json:"pid"`
			}
			if json.Unmarshal(data, &legacy) == nil && legacy.PID > 0 {
				// Verify it's a browse process before killing
				out, _ := exec.Command("ps", "-p", strconv.Itoa(legacy.PID), "-o", "command=").Output()
				cmd := strings.TrimSpace(string(out))
				if strings.Contains(cmd, "browse") || strings.Contains(cmd, "server") {
					_ = safeKill(legacy.PID, sigTerm)
				}
			}
			_ = os.Remove(fullPath)
		}
		// Legacy log files
		if strings.HasPrefix(name, "browse-console") ||
			strings.HasPrefix(name, "browse-network") ||
			strings.HasPrefix(name, "browse-dialog") {
			_ = os.Remove(filepath.Join("/tmp", name))
		}
	}
}

// ─── Headed Connect ───────────────────────────────────────

func handleConnect(flags *GlobalFlags) int {
	cfg := config.Resolve(nil)

	// Check if already in headed mode and healthy
	existingState, ok := readState()
	if ok && existingState.Mode == "headed" && isProcessAlive(existingState.PID) {
		if isServerHealthy(existingState.Port) {
			fmt.Println("Already connected in headed mode.")
			return 0
		}
		// Headed server alive but not responding — kill and restart
	}

	// Kill ANY existing server (SIGTERM → wait 2s → SIGKILL)
	if ok && existingState.PID > 0 && isProcessAlive(existingState.PID) {
		_ = safeKill(existingState.PID, sigTerm)
		time.Sleep(2 * time.Second)
		if isProcessAlive(existingState.PID) {
			_ = safeKill(existingState.PID, sigKill)
			time.Sleep(1 * time.Second)
		}
	}

	// Kill orphaned Chromium processes that may still hold the profile lock
	profileDir := config.ChromiumProfile("")
	killOrphanedChromium(profileDir)

	// Clean up Chromium profile locks (can persist after crashes)
	for _, f := range []string{"SingletonLock", "SingletonSocket", "SingletonCookie"} {
		_ = os.Remove(filepath.Join(profileDir, f))
	}

	// Delete stale state file
	_ = os.Remove(cfg.StateFile)

	fmt.Println("Launching headed Chromium with extension + terminal agent...")

	// Build extra environment for headed mode
	extraEnv := []string{
		"BROWSE_HEADED=1",
		"BROWSE_PORT=34567",
		"BROWSE_SIDEBAR_CHAT=1",
		"BROWSE_PARENT_PID=0",
	}
	if flags.ProxyURL != "" {
		extraEnv = append(extraEnv, "BROWSE_PROXY_URL="+flags.ProxyURL)
	}
	if flags.ConfigHash != "" {
		extraEnv = append(extraEnv, "BROWSE_CONFIG_HASH="+flags.ConfigHash)
	}

	// Start server in headed mode
	if rc := startDaemonWithEnv(flags, extraEnv); rc != 0 {
		return rc
	}

	newState, ok := readState()
	if !ok || !isServerHealthy(newState.Port) {
		log.Println("Error: server failed to start")
		return 1
	}

	// Confirm connection with a status command
	resp, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/command", newState.Port),
		"application/json",
		strings.NewReader(fmt.Sprintf(`{"command":"status","args":[],"token":"%s"}`, newState.Token)))
	if err == nil && resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("Connected to real Chrome\n%s\n", string(body))
	} else {
		fmt.Println("Connected to real Chrome")
	}

	// Spawn terminal agent
	termCmd, err := spawnTerminalAgent(cfg.StateDir, newState.Port)
	if err != nil {
		log.Printf("[browse] Terminal agent failed to start: %v", err)
		// Non-fatal: browser still works without terminal agent
	} else {
		log.Printf("[browse] Terminal agent started (PID: %d)", termCmd.Process.Pid)
		termCmd.Process.Release()
	}

	return 0
}

// startDaemonWithEnv is like startDaemon but passes extra environment variables.
func startDaemonWithEnv(flags *GlobalFlags, extraEnv []string) int {
	release := acquireServerLock()
	if release == nil {
		log.Println("Error: another browse start is in progress")
		return 1
	}
	defer release()

	state, ok := readState()
	if ok && isServerHealthy(state.Port) {
		// Daemon-mismatch check
		if flags.ConfigHash != "" && state.ConfigHash != "" && state.ConfigHash != flags.ConfigHash {
			log.Println("[browse] existing daemon has different config (proxy/headed mismatch).")
			log.Println("[browse] run 'browse disconnect' first to apply --proxy/--headed.")
			return 1
		}
		if flags.ConfigHash != "" && state.ConfigHash == "" && (flags.ProxyURL != "" || flags.Headed) {
			log.Println("[browse] existing daemon was started without --proxy/--headed.")
			log.Println("[browse] run 'browse disconnect' first to apply new flags.")
			return 1
		}
		log.Printf("Daemon already running on port %d (pid %d)", state.Port, state.PID)
		return 0
	}

	// Kill stale PID if present
	if ok && state.PID > 0 {
		_ = safeKill(state.PID, sigTerm)
		time.Sleep(500 * time.Millisecond)
	}

	// Spawn background server process
	cmd := exec.Command(os.Args[0], append([]string{"server"}, flags.Args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	setProcessGroup(cmd)

	// Build environment: current env + extra env + proxy/headed
	cmd.Env = append(os.Environ(), extraEnv...)
	if flags.ProxyURL != "" {
		cmd.Env = append(cmd.Env, "BROWSE_PROXY_URL="+flags.ProxyURL)
	}
	if flags.Headed {
		cmd.Env = append(cmd.Env, "BROWSE_HEADED=1")
	}
	if flags.ConfigHash != "" {
		cmd.Env = append(cmd.Env, "BROWSE_CONFIG_HASH="+flags.ConfigHash)
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Error: failed to start daemon: %v", err)
		return 1
	}

	// Wait for health check
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		newState, ok := readState()
		if ok && isServerHealthy(newState.Port) {
			log.Printf("Daemon started on port %d (pid %d)", newState.Port, newState.PID)
			return 0
		}
	}

	log.Println("Error: daemon failed to start within timeout")
	return 1
}

// spawnTerminalAgent starts the terminal-agent binary as a subprocess.
func spawnTerminalAgent(stateDir string, serverPort int) (*exec.Cmd, error) {
	agentBin := findTerminalAgentBinary()
	if agentBin == "" {
		return nil, fmt.Errorf("terminal-agent binary not found")
	}

	// Kill old terminal-agents so a stale port file can't trick the server
	_ = killTerminalAgent()

	cmd := exec.Command(agentBin)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("BROWSE_STATE_FILE=%s", filepath.Join(stateDir, "browse.json")),
		fmt.Sprintf("BROWSE_SERVER_PORT=%d", serverPort),
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	setProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// findTerminalAgentBinary searches for the terminal-agent binary.
func findTerminalAgentBinary() string {
	// Check if terminal-agent is in the same directory as the browse binary
	browseDir := filepath.Dir(os.Args[0])
	candidates := []string{
		filepath.Join(browseDir, "terminal-agent"),
		filepath.Join(browseDir, "terminal-agent.exe"),
	}
	// Development: go run from repo root
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "tools", "skills", "browse-go", "cmd", "terminal-agent", "terminal-agent"),
			filepath.Join(cwd, "cmd", "terminal-agent", "terminal-agent"),
		)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// ─── Helpers ──────────────────────────────────────────────

func isServerHealthy(port int) bool {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

func readVersionHash() string {
	// Use binary modification time as a simple version hash
	info, err := os.Stat(os.Args[0])
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d-%d", info.Size(), info.ModTime().Unix())
}
