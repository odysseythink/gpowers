package cookieimport

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// BROWSER_REGISTRY is the hardcoded list of supported Chromium browsers.
// NEVER interpolate user input into shell commands derived from this.
var BROWSER_REGISTRY = []*BrowserInfo{
	{
		Name:            "Comet",
		DataDir:         "Comet/",
		KeychainService: "Comet Safe Storage",
		Aliases:         []string{"comet", "perplexity"},
	},
	{
		Name:            "Chrome",
		DataDir:         "Google/Chrome/",
		KeychainService: "Chrome Safe Storage",
		Aliases:         []string{"chrome", "google-chrome", "google-chrome-stable"},
		LinuxDataDir:    "google-chrome/",
		LinuxApp:        "chrome",
		WindowsDataDir:  "Google/Chrome/User Data/",
	},
	{
		Name:            "Chromium",
		DataDir:         "chromium/",
		KeychainService: "Chromium Safe Storage",
		Aliases:         []string{"chromium"},
		LinuxDataDir:    "chromium/",
		LinuxApp:        "chromium",
		WindowsDataDir:  "Chromium/User Data/",
	},
	{
		Name:            "Arc",
		DataDir:         "Arc/User Data/",
		KeychainService: "Arc Safe Storage",
		Aliases:         []string{"arc"},
	},
	{
		Name:            "Brave",
		DataDir:         "BraveSoftware/Brave-Browser/",
		KeychainService: "Brave Safe Storage",
		Aliases:         []string{"brave"},
		LinuxDataDir:    "BraveSoftware/Brave-Browser/",
		LinuxApp:        "brave",
		WindowsDataDir:  "BraveSoftware/Brave-Browser/User Data/",
	},
	{
		Name:            "Edge",
		DataDir:         "Microsoft Edge/",
		KeychainService: "Microsoft Edge Safe Storage",
		Aliases:         []string{"edge"},
		LinuxDataDir:    "microsoft-edge/",
		LinuxApp:        "microsoft-edge",
		WindowsDataDir:  "Microsoft/Edge/User Data/",
	},
}

// resolveBrowser finds a browser by name or alias.
func resolveBrowser(nameOrAlias string) (*BrowserInfo, error) {
	needle := strings.ToLower(strings.TrimSpace(nameOrAlias))
	for _, b := range BROWSER_REGISTRY {
		if strings.ToLower(b.Name) == needle {
			return b, nil
		}
		for _, a := range b.Aliases {
			if a == needle {
				return b, nil
			}
		}
	}
	var supported []string
	for _, b := range BROWSER_REGISTRY {
		supported = append(supported, b.Aliases...)
	}
	return nil, NewError("unknown_browser",
		fmt.Sprintf("Unknown browser '%s'. Supported: %s", nameOrAlias, strings.Join(supported, ", ")))
}

func validateProfile(profile string) error {
	if strings.ContainsAny(profile, `/\`) || strings.Contains(profile, "..") {
		return NewError("bad_request", fmt.Sprintf("Invalid profile name: '%s'", profile))
	}
	for i := 0; i < len(profile); i++ {
		if profile[i] < 0x20 {
			return NewError("bad_request", fmt.Sprintf("Invalid profile name: '%s'", profile))
		}
	}
	return nil
}

func hostPlatform() browserPlatform {
	switch runtime.GOOS {
	case "darwin":
		return platformDarwin
	case "linux":
		return platformLinux
	case "windows":
		return platformWin32
	}
	return ""
}

func searchPlatforms() []browserPlatform {
	current := hostPlatform()
	order := []browserPlatform{}
	if current != "" {
		order = append(order, current)
	}
	for _, p := range []browserPlatform{platformDarwin, platformLinux, platformWin32} {
		found := false
		for _, o := range order {
			if o == p {
				found = true
				break
			}
		}
		if !found {
			order = append(order, p)
		}
	}
	return order
}

func dataDirForPlatform(browser *BrowserInfo, platform browserPlatform) string {
	switch platform {
	case platformDarwin:
		return browser.DataDir
	case platformLinux:
		return browser.LinuxDataDir
	case platformWin32:
		return browser.WindowsDataDir
	}
	return ""
}

func baseDir(platform browserPlatform) string {
	home, _ := os.UserHomeDir()
	switch platform {
	case platformDarwin:
		return filepath.Join(home, "Library", "Application Support")
	case platformWin32:
		return filepath.Join(home, "AppData", "Local")
	default:
		return filepath.Join(home, ".config")
	}
}

func cookieDBPath(platform browserPlatform, baseProfile string) []string {
	if platform == platformWin32 {
		return []string{
			filepath.Join(baseProfile, "Network", "Cookies"),
			filepath.Join(baseProfile, "Cookies"),
		}
	}
	return []string{filepath.Join(baseProfile, "Cookies")}
}

func findBrowserMatch(browser *BrowserInfo, profile string) *browserMatch {
	_ = validateProfile(profile)
	for _, platform := range searchPlatforms() {
		dataDir := dataDirForPlatform(browser, platform)
		if dataDir == "" {
			continue
		}
		baseProfile := filepath.Join(baseDir(platform), dataDir, profile)
		for _, dbPath := range cookieDBPath(platform, baseProfile) {
			if _, err := os.Stat(dbPath); err == nil {
				return &browserMatch{browser: browser, platform: platform, dbPath: dbPath}
			}
		}
	}
	return nil
}

func getBrowserMatch(browser *BrowserInfo, profile string) (*browserMatch, error) {
	match := findBrowserMatch(browser, profile)
	if match != nil {
		return match, nil
	}
	var attempted []string
	for _, platform := range searchPlatforms() {
		dataDir := dataDirForPlatform(browser, platform)
		if dataDir == "" {
			continue
		}
		attempted = append(attempted, filepath.Join(baseDir(platform), dataDir, profile, "Cookies"))
	}
	return nil, NewError("not_installed",
		fmt.Sprintf("%s is not installed (no cookie database at %s)", browser.Name, strings.Join(attempted, " or ")))
}

// FindInstalledBrowsers returns browsers that have a cookie DB on disk.
func FindInstalledBrowsers() []*BrowserInfo {
	var result []*BrowserInfo
	for _, browser := range BROWSER_REGISTRY {
		if findBrowserMatch(browser, "Default") != nil {
			result = append(result, browser)
			continue
		}
		found := false
		for _, platform := range searchPlatforms() {
			dataDir := dataDirForPlatform(browser, platform)
			if dataDir == "" {
				continue
			}
			browserDir := filepath.Join(baseDir(platform), dataDir)
			entries, err := os.ReadDir(browserDir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() || (e.Name() != "Default" && !strings.HasPrefix(e.Name(), "Profile ")) {
					continue
				}
				for _, dbPath := range cookieDBPath(platform, filepath.Join(browserDir, e.Name())) {
					if _, err := os.Stat(dbPath); err == nil {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
		if found {
			result = append(result, browser)
		}
	}
	return result
}

// ListSupportedBrowserNames returns browsers supported on the current platform.
func ListSupportedBrowserNames() []string {
	host := hostPlatform()
	var names []string
	for _, b := range BROWSER_REGISTRY {
		if host != "" && dataDirForPlatform(b, host) == "" {
			continue
		}
		names = append(names, b.Name)
	}
	return names
}

// ListProfiles returns available profiles for a browser.
func ListProfiles(browserName string) ([]ProfileEntry, error) {
	browser, err := resolveBrowser(browserName)
	if err != nil {
		return nil, err
	}
	var profiles []ProfileEntry
	for _, platform := range searchPlatforms() {
		dataDir := dataDirForPlatform(browser, platform)
		if dataDir == "" {
			continue
		}
		browserDir := filepath.Join(baseDir(platform), dataDir)
		entries, err := os.ReadDir(browserDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || (e.Name() != "Default" && !strings.HasPrefix(e.Name(), "Profile ")) {
				continue
			}
			found := false
			for _, dbPath := range cookieDBPath(platform, filepath.Join(browserDir, e.Name())) {
				if _, err := os.Stat(dbPath); err == nil {
					found = true
					break
				}
			}
			if !found {
				continue
			}
			// Avoid duplicates
			dup := false
			for _, p := range profiles {
				if p.Name == e.Name() {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			displayName := e.Name()
			// Try to read account email or profile name from Preferences
			prefsPath := filepath.Join(browserDir, e.Name(), "Preferences")
			if data, err := os.ReadFile(prefsPath); err == nil {
				// Simple JSON parsing — look for "email" and "profile.name"
				if email := jsonExtractString(data, `"account_info"`, `"email"`); email != "" {
					displayName = email
				} else if name := jsonExtractString(data, `"profile"`, `"name"`); name != "" {
					displayName = name
				}
			}
			profiles = append(profiles, ProfileEntry{Name: e.Name(), DisplayName: displayName})
		}
		if len(profiles) > 0 {
			break
		}
	}
	return profiles, nil
}

// jsonExtractString does a very simple double-quoted string extraction.
// It searches for the keys in order and returns the next string value.
func jsonExtractString(data []byte, keys ...string) string {
	pos := 0
	for _, key := range keys {
		idx := strings.Index(string(data[pos:]), key)
		if idx == -1 {
			return ""
		}
		pos += idx + len(key)
	}
	// Skip whitespace and colon
	for pos < len(data) && (data[pos] == ' ' || data[pos] == '\t' || data[pos] == '\n' || data[pos] == ':' || data[pos] == ',') {
		pos++
	}
	if pos >= len(data) || data[pos] != '"' {
		return ""
	}
	pos++ // skip opening quote
	end := pos
	for end < len(data) {
		if data[end] == '\\' && end+1 < len(data) {
			end += 2
			continue
		}
		if data[end] == '"' {
			break
		}
		end++
	}
	if end >= len(data) {
		return ""
	}
	return string(data[pos:end])
}
