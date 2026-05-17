package cookieimport

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/gorilla/websocket"
)

var chromePathsWin = []string{
	`C:\Program Files\Google\Chrome\Application\chrome.exe`,
	`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
}

var edgePathsWin = []string{
	`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
	`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
}

func findBrowserExe(browserName string) string {
	candidates := chromePathsWin
	if strings.Contains(strings.ToLower(browserName), "edge") {
		candidates = edgePathsWin
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func isBrowserRunning(browserName string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	exe := "chrome.exe"
	if strings.Contains(strings.ToLower(browserName), "edge") {
		exe = "msedge.exe"
	}
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", exe), "/NH")
	out, _ := cmd.Output()
	return strings.Contains(strings.ToLower(string(out)), strings.ToLower(exe))
}

// ImportCookiesViaCDP extracts cookies via Chrome DevTools Protocol.
// This is the fallback for Windows v20 App-Bound Encryption.
func ImportCookiesViaCDP(browserName string, domains []string, profile string) (*ImportResult, error) {
	if len(domains) == 0 {
		return &ImportResult{Cookies: nil, Count: 0, Failed: 0, DomainCounts: map[string]int{}}, nil
	}
	if runtime.GOOS != "windows" {
		return nil, NewError("not_supported", "CDP extraction is only needed on Windows")
	}

	browser, err := resolveBrowser(browserName)
	if err != nil {
		return nil, err
	}
	exePath := findBrowserExe(browser.Name)
	if exePath == "" {
		return nil, NewError("not_installed",
			fmt.Sprintf("Cannot find %s executable. Install it or use /connect-chrome.", browser.Name))
	}

	if isBrowserRunning(browser.Name) {
		return nil, NewRetryError("browser_running",
			fmt.Sprintf("%s is running. Close it first so we can launch headless with your profile, or use /connect-chrome to control your real browser directly.", browser.Name))
	}

	dataDir := dataDirForPlatform(browser, platformWin32)
	if dataDir == "" {
		return nil, NewError("not_installed", fmt.Sprintf("No Windows data dir for %s", browser.Name))
	}
	userDataDir := filepath.Join(baseDir(platformWin32), dataDir)

	// Randomize debug port to avoid collisions.
	n, _ := rand.Int(rand.Reader, big.NewInt(100))
	debugPort := 9222 + int(n.Int64())

	chromeCmd := exec.Command(exePath,
		fmt.Sprintf("--remote-debugging-port=%d", debugPort),
		fmt.Sprintf("--user-data-dir=%s", userDataDir),
		fmt.Sprintf("--profile-directory=%s", profile),
		"--headless=new",
		"--no-first-run",
		"--disable-background-networking",
		"--disable-default-apps",
		"--disable-extensions",
		"--disable-sync",
		"--no-default-browser-check",
	)
	chromeCmd.Stdout = os.NewFile(0, os.DevNull)
	chromeCmd.Stderr = os.NewFile(0, os.DevNull)

	if err := chromeCmd.Start(); err != nil {
		return nil, NewError("cdp_error", fmt.Sprintf("Failed to start Chrome: %v", err))
	}
	defer func() {
		_ = chromeCmd.Process.Kill()
		_ = chromeCmd.Wait()
	}()

	// Wait for Chrome to start and find a page target's WebSocket URL.
	var wsURL string
	startTime := time.Now()
	for time.Since(startTime) < 15*time.Second {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/json/list", debugPort))
		if err == nil && resp.StatusCode == http.StatusOK {
			var targets []struct {
				Type                   string `json:"type"`
				WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&targets); err == nil {
				resp.Body.Close()
				for _, t := range targets {
					if t.Type == "page" && t.WebSocketDebuggerURL != "" {
						wsURL = t.WebSocketDebuggerURL
						break
					}
				}
				if wsURL != "" {
					break
				}
			} else {
				resp.Body.Close()
			}
		} else if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(300 * time.Millisecond)
	}

	if wsURL == "" {
		return nil, NewRetryError("cdp_timeout", fmt.Sprintf("%s headless did not start within 15s", browser.Name))
	}

	cookies, err := extractCookiesViaCDP(wsURL, domains)
	if err != nil {
		return nil, err
	}

	domainCounts := make(map[string]int)
	for _, c := range cookies {
		domainCounts[c.Domain]++
	}

	return &ImportResult{
		Cookies:      cookies,
		Count:        len(cookies),
		Failed:       0,
		DomainCounts: domainCounts,
	}, nil
}

func extractCookiesViaCDP(wsURL string, domains []string) ([]*CDPCookie, error) {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return nil, NewError("cdp_error", fmt.Sprintf("WebSocket dial failed: %v", err))
	}
	defer conn.Close()

	domainSet := make(map[string]struct{})
	for _, d := range domains {
		domainSet[d] = struct{}{}
		if strings.HasPrefix(d, ".") {
			domainSet[d[1:]] = struct{}{}
		} else {
			domainSet["."+d] = struct{}{}
		}
	}

	// Send Network.enable
	if err := conn.WriteJSON(map[string]interface{}{"id": 1, "method": "Network.enable"}); err != nil {
		return nil, NewError("cdp_error", fmt.Sprintf("Failed to send Network.enable: %v", err))
	}

	// Wait for Network.enable response
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			return nil, NewError("cdp_error", fmt.Sprintf("WebSocket read error: %v", err))
		}
		id, _ := msg["id"].(float64)
		if int(id) == 1 {
			if msg["error"] != nil {
				return nil, NewError("cdp_error", fmt.Sprintf("Network.enable failed: %v", msg["error"]))
			}
			break
		}
	}

	// Send Network.getAllCookies
	if err := conn.WriteJSON(map[string]interface{}{"id": 2, "method": "Network.getAllCookies"}); err != nil {
		return nil, NewError("cdp_error", fmt.Sprintf("Failed to send Network.getAllCookies: %v", err))
	}

	// Wait for response with timeout
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			return nil, NewError("cdp_error", fmt.Sprintf("WebSocket read error: %v", err))
		}
		id, _ := msg["id"].(float64)
		if int(id) == 2 {
			if msg["error"] != nil {
				return nil, NewError("cdp_error", fmt.Sprintf("CDP error: %v", msg["error"]))
			}
			result, ok := msg["result"].(map[string]interface{})
			if !ok {
				return nil, NewError("cdp_error", "Unexpected response format")
			}
			cookiesRaw, ok := result["cookies"].([]interface{})
			if !ok {
				return nil, NewError("cdp_error", "Unexpected cookies format")
			}

			var matched []*CDPCookie
			for _, c := range cookiesRaw {
				cMap, ok := c.(map[string]interface{})
				if !ok {
					continue
				}
				domain, _ := cMap["domain"].(string)
				if _, ok := domainSet[domain]; !ok {
					continue
				}
				name, _ := cMap["name"].(string)
				value, _ := cMap["value"].(string)
				path, _ := cMap["path"].(string)
				secure, _ := cMap["secure"].(bool)
				httpOnly, _ := cMap["httpOnly"].(bool)
				expires := -1.0
				if e, ok := cMap["expires"].(float64); ok {
					expires = e
				}
				sameSite := network.CookieSameSiteLax
				if ss, ok := cMap["sameSite"].(string); ok {
					sameSite = cdpSameSite(ss)
				}

				matched = append(matched, &CDPCookie{
					Name:     name,
					Value:    value,
					Domain:   domain,
					Path:     path,
					Expires:  expires,
					Secure:   secure,
					HTTPOnly: httpOnly,
					SameSite: sameSite,
				})
			}
			return matched, nil
		}
	}
}

func cdpSameSite(value string) network.CookieSameSite {
	switch value {
	case "Strict":
		return network.CookieSameSiteStrict
	case "Lax":
		return network.CookieSameSiteLax
	case "None":
		return network.CookieSameSiteNone
	default:
		return network.CookieSameSiteLax
	}
}

// HasV20Cookies checks if a browser's cookie DB contains v20 encrypted cookies.
func HasV20Cookies(browserName string, profile string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	browser, err := resolveBrowser(browserName)
	if err != nil {
		return false
	}
	match, err := getBrowserMatch(browser, profile)
	if err != nil {
		return false
	}
	db, tmpPath, err := openDB(match.dbPath, browser.Name)
	if err != nil {
		return false
	}
	defer closeDB(db, tmpPath)

	rows, err := db.Query("SELECT encrypted_value FROM cookies LIMIT 10")
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var ev []byte
		if err := rows.Scan(&ev); err != nil {
			continue
		}
		if len(ev) >= 3 && string(ev[:3]) == "v20" {
			return true
		}
	}
	return false
}
