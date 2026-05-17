package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"browse-go/pkg/buffers"
	"browse-go/pkg/config"
	"browse-go/pkg/util"
)

const highWaterMark = 50_000

// ProxyConfig holds SOCKS5/HTTP proxy settings for chromium.launch().
type ProxyConfig struct {
	Server   string
	Username string
	Password string
}

// BrowserManager manages the Chromium lifecycle, tabs, and event capture.
type BrowserManager struct {
	mu sync.RWMutex

	// chromedp contexts
	allocCtx    context.Context
	allocCancel context.CancelFunc
	browserCtx  context.Context
	browserCancel context.CancelFunc

	// Tabs: id -> session
	tabs        map[int]*TabSession
	activeTabId int
	nextTabId   int

	// Viewport / context options
	deviceScaleFactor int
	viewportWidth     int
	viewportHeight    int

	// HTTP settings
	extraHeaders    map[string]string
	customUserAgent string
	proxyConfig     *ProxyConfig

	// Dialog handling
	dialogAutoAccept bool
	dialogPromptText string

	// Tab ownership (multi-agent isolation)
	tabOwnership map[int]string

	// Connection state
	connectionMode        string // "launched" | "headed"
	intentionalDisconnect bool
	onDisconnect          func()

	// Server metadata
	serverPort int
	isHeaded   bool

	// Headed mode (handoff) — separate Chrome instance
	headedAllocCtx    context.Context
	headedAllocCancel context.CancelFunc
	headedBrowserCtx  context.Context
	headedBrowserCancel context.CancelFunc
	handoffURL        string
	headedExecPath    string // resolved executable path for focus support

	// Buffers
	consoleBuffer *buffers.Circular[ConsoleEntry]
	networkBuffer *buffers.Circular[NetworkEntry]
	dialogBuffer  *buffers.Circular[DialogEntry]

	// Network response-body capture (separate from request/response log)
	captureBuffer *SizeCappedBuffer
	captureActive bool
	captureFilter *regexp.Regexp

	// Watch mode
	watching       bool
	watchStartTime time.Time
	watchSnapshots []string

	// URL cache (updated via CDP nav events to avoid round-trips)
	cachedURL   string
	cachedURLMu sync.RWMutex

	// Cookie import tracking
	cookieImportDomains   []string
	cookieImportCounts    map[string]int
	cookieImportDomainsMu sync.RWMutex
}

// NewBrowserManager creates an unlaunched manager.
func NewBrowserManager() *BrowserManager {
	return &BrowserManager{
		tabs:              make(map[int]*TabSession),
		nextTabId:         1,
		deviceScaleFactor: 1,
		viewportWidth:     1280,
		viewportHeight:    720,
		extraHeaders:      make(map[string]string),
		tabOwnership:      make(map[int]string),
		connectionMode:    "launched",
		consoleBuffer:     buffers.NewCircular[ConsoleEntry](highWaterMark),
		networkBuffer:     buffers.NewCircular[NetworkEntry](highWaterMark),
		dialogBuffer:      buffers.NewCircular[DialogEntry](highWaterMark),
		captureBuffer:     NewSizeCappedBuffer(0),
	}
}

// ─── Launch / Close ────────────────────────────────────────

// Launch starts a headless Chromium instance and creates the first tab.
func (m *BrowserManager) Launch() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.browserCtx != nil {
		return fmt.Errorf("browser already launched")
	}

	launchArgs := append([]string(nil), StealthLaunchArgs...)

	// Docker/CI/root: disable sandbox
	if os.Getenv("CI") != "" || os.Getenv("CONTAINER") != "" || isRoot() {
		launchArgs = append(launchArgs, "--no-sandbox")
	}

	// Extension support (off-screen headed window)
	extensionsDir := os.Getenv("BROWSE_EXTENSIONS_DIR")
	useHeadless := true
	if extensionsDir != "" {
		launchArgs = append(launchArgs,
			fmt.Sprintf("--disable-extensions-except=%s", extensionsDir),
			fmt.Sprintf("--load-extension=%s", extensionsDir),
			"--window-position=-9999,-9999",
			"--window-size=1,1",
		)
		useHeadless = false
	}

	// Build allocator options
	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", useHeadless),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)
	for _, arg := range launchArgs {
		allocOpts = append(allocOpts, chromedp.Flag(arg, true))
	}
	if m.customUserAgent != "" {
		allocOpts = append(allocOpts, chromedp.UserAgent(m.customUserAgent))
	}

	// Proxy
	if m.proxyConfig != nil {
		allocOpts = append(allocOpts, chromedp.ProxyServer(m.proxyConfig.Server))
	}

	// Auto-detect Chrome on macOS if standard lookup fails
	if chromePath := FindChromiumExecutable(); chromePath != "" {
		allocOpts = append(allocOpts, chromedp.ExecPath(chromePath))
	}

	m.allocCtx, m.allocCancel = chromedp.NewExecAllocator(context.Background(), allocOpts...)
	m.browserCtx, m.browserCancel = chromedp.NewContext(m.allocCtx)

	// browserCtx itself is the first tab in chromedp. Initialize it.
	if err := chromedp.Run(m.browserCtx, chromedp.Navigate("about:blank")); err != nil {
		m.cleanup()
		return fmt.Errorf("browser init failed: %w", err)
	}

	// Register browserCtx as the first tab (id=1)
	firstTab := &TabSession{ctx: m.browserCtx, cancel: func() {}}
	m.tabs[1] = firstTab
	m.activeTabId = 1
	m.nextTabId = 2
	m.wireTabEvents(firstTab.Context(), 1)

	// Enable network domain
	if err := chromedp.Run(firstTab.Context(), network.Enable()); err != nil {
		m.cleanup()
		return fmt.Errorf("network enable failed: %w", err)
	}

	// Apply stealth
	if err := ApplyStealth(firstTab.Context()); err != nil {
		m.cleanup()
		return fmt.Errorf("stealth setup failed: %w", err)
	}

	// Set extra headers
	if len(m.extraHeaders) > 0 {
		headers := make(map[string]interface{}, len(m.extraHeaders))
		for k, v := range m.extraHeaders {
			headers[k] = v
		}
		if err := chromedp.Run(firstTab.Context(), network.SetExtraHTTPHeaders(headers)); err != nil {
			m.cleanup()
			return fmt.Errorf("extra headers failed: %w", err)
		}
	}

	// Set viewport
	if err := chromedp.Run(firstTab.Context(), chromedp.EmulateViewport(int64(m.viewportWidth), int64(m.viewportHeight))); err != nil {
		m.cleanup()
		return fmt.Errorf("viewport setup failed: %w", err)
	}

	return nil
}

// Close shuts down the browser.
func (m *BrowserManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.intentionalDisconnect = true

	// Close all tabs
	for _, ts := range m.tabs {
		ts.Close()
	}
	m.tabs = make(map[int]*TabSession)

	if m.browserCancel != nil {
		m.browserCancel()
		m.browserCancel = nil
		m.browserCtx = nil
	}
	if m.allocCancel != nil {
		m.allocCancel()
		m.allocCancel = nil
		m.allocCtx = nil
	}
	// Also close headed instance
	if m.headedBrowserCancel != nil {
		m.headedBrowserCancel()
		m.headedBrowserCancel = nil
		m.headedBrowserCtx = nil
	}
	if m.headedAllocCancel != nil {
		m.headedAllocCancel()
		m.headedAllocCancel = nil
		m.headedAllocCtx = nil
	}
	return nil
}

// LaunchHeaded starts a visible Chrome instance and navigates to url.
// The headless instance (if any) is left running.
func (m *BrowserManager) LaunchHeaded(url string) error {
	return m.LaunchHeadedWithOptions(url, nil)
}

// LaunchHeadedOptions holds optional parameters for headed launch.
type LaunchHeadedOptions struct {
	AuthToken  string // token written to ~/.gstack/.auth.json for extension bootstrap
	ServerPort int    // port the HTTP server listens on
}

// LaunchHeadedWithOptions starts a visible Chrome instance with extension support.
func (m *BrowserManager) LaunchHeadedWithOptions(url string, opts *LaunchHeadedOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close any existing headed instance
	if m.headedBrowserCancel != nil {
		m.headedBrowserCancel()
		m.headedBrowserCancel = nil
		m.headedBrowserCtx = nil
	}
	if m.headedAllocCancel != nil {
		m.headedAllocCancel()
		m.headedAllocCancel = nil
		m.headedAllocCtx = nil
	}

	launchArgs := append([]string(nil), StealthLaunchArgs...)
	if os.Getenv("CI") != "" || os.Getenv("CONTAINER") != "" || isRoot() {
		launchArgs = append(launchArgs, "--no-sandbox")
	}

	// Clean stale Chromium locks before launch
	CleanSingletonLocks()

	// Find and load the gstack Chrome extension
	extensionPath := FindExtensionPath()
	if extensionPath != "" && !IsCustomChromium() {
		launchArgs = append(launchArgs,
			fmt.Sprintf("--disable-extensions-except=%s", extensionPath),
			fmt.Sprintf("--load-extension=%s", extensionPath),
		)
	}

	// Write auth token for extension bootstrap
	if opts != nil && opts.AuthToken != "" {
		writeAuthJSON(opts.AuthToken, opts.ServerPort)
	}

	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.WindowSize(m.viewportWidth, m.viewportHeight),
		chromedp.UserDataDir(config.ChromiumProfile("")),
	)
	for _, arg := range launchArgs {
		allocOpts = append(allocOpts, chromedp.Flag(arg, true))
	}
	if m.customUserAgent != "" {
		allocOpts = append(allocOpts, chromedp.UserAgent(m.customUserAgent))
	}
	if m.proxyConfig != nil {
		allocOpts = append(allocOpts, chromedp.ProxyServer(m.proxyConfig.Server))
	}
	m.headedExecPath = ""
	if chromePath := FindChromiumExecutable(); chromePath != "" {
		allocOpts = append(allocOpts, chromedp.ExecPath(chromePath))
		m.headedExecPath = chromePath
	}

	m.headedAllocCtx, m.headedAllocCancel = chromedp.NewExecAllocator(context.Background(), allocOpts...)
	m.headedBrowserCtx, m.headedBrowserCancel = chromedp.NewContext(m.headedAllocCtx)
	m.handoffURL = url
	m.connectionMode = "headed"
	m.intentionalDisconnect = false

	if err := chromedp.Run(m.headedBrowserCtx, chromedp.Navigate(url)); err != nil {
		m.closeHeadedLocked()
		return fmt.Errorf("headed navigate failed: %w", err)
	}
	return nil
}

// CloseHeaded shuts down the visible Chrome instance.
func (m *BrowserManager) CloseHeaded() error {
	m.mu.Lock()
	m.intentionalDisconnect = true
	m.mu.Unlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeHeadedLocked()
	m.connectionMode = "launched"
	return nil
}

func (m *BrowserManager) closeHeadedLocked() {
	if m.headedBrowserCancel != nil {
		m.headedBrowserCancel()
		m.headedBrowserCancel = nil
		m.headedBrowserCtx = nil
	}
	if m.headedAllocCancel != nil {
		m.headedAllocCancel()
		m.headedAllocCancel = nil
		m.headedAllocCtx = nil
	}
	m.headedExecPath = ""
}

// IsHeaded returns true if a headed instance is currently running.
func (m *BrowserManager) IsHeaded() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.headedBrowserCtx != nil
}

// HeadedExecPath returns the resolved executable path of the headed browser,
// or empty string if not in headed mode.
func (m *BrowserManager) HeadedExecPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.headedExecPath
}

// GetConnectionMode returns the current connection mode ("launched" or "headed").
func (m *BrowserManager) GetConnectionMode() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connectionMode
}

// SetOnDisconnect sets the callback invoked when the headed browser disconnects unexpectedly.
func (m *BrowserManager) SetOnDisconnect(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onDisconnect = fn
}

// OnDisconnect returns the current disconnect callback.
func (m *BrowserManager) OnDisconnect() func() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.onDisconnect
}

// SetServerPort records the HTTP server port for extension bootstrap.
func (m *BrowserManager) SetServerPort(port int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serverPort = port
}

// ServerPort returns the recorded HTTP server port.
func (m *BrowserManager) ServerPort() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.serverPort
}

// HandoffURL returns the URL saved during the last handoff.
func (m *BrowserManager) HandoffURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.handoffURL
}

// cleanup is the internal teardown helper used on launch failure.
func (m *BrowserManager) cleanup() {
	for _, ts := range m.tabs {
		ts.Close()
	}
	m.tabs = make(map[int]*TabSession)
	if m.browserCancel != nil {
		m.browserCancel()
		m.browserCancel = nil
		m.browserCtx = nil
	}
	if m.allocCancel != nil {
		m.allocCancel()
		m.allocCancel = nil
		m.allocCtx = nil
	}
}

// isRoot returns true if the current process is running as root on Linux.
func isRoot() bool {
	if os.Geteuid() == 0 {
		return true
	}
	return false
}

// ─── Tab Management ───────────────────────────────────────

// NewTab creates a new tab and optionally navigates to url.
func (m *BrowserManager) NewTab(url string) (int, error) {
	m.mu.Lock()
	if m.browserCtx == nil {
		m.mu.Unlock()
		return 0, fmt.Errorf("browser not launched")
	}

	var normalizedURL string
	if url != "" {
		var err error
		normalizedURL, err = util.ValidateNavigationURL(url)
		if err != nil {
			m.mu.Unlock()
			return 0, err
		}
	}

	ts := NewTabSession(m.browserCtx)
	id := m.nextTabId
	m.nextTabId++
	m.tabs[id] = ts
	m.activeTabId = id
	m.mu.Unlock()

	// Wire up CDP event listeners for this tab (without lock to avoid deadlock
	// with async event delivery during Navigate)
	m.wireTabEvents(ts.Context(), id)

	if normalizedURL != "" {
		if err := chromedp.Run(ts.Context(), chromedp.Navigate(normalizedURL)); err != nil {
			// Don't fail — the tab still exists, just nav failed
			_ = err
		}
	}

	return id, nil
}

func (m *BrowserManager) newTabLocked(url string) (int, error) {
	if m.browserCtx == nil {
		return 0, fmt.Errorf("browser not launched")
	}

	var normalizedURL string
	if url != "" {
		var err error
		normalizedURL, err = util.ValidateNavigationURL(url)
		if err != nil {
			return 0, err
		}
	}

	ts := NewTabSession(m.browserCtx)
	id := m.nextTabId
	m.nextTabId++
	m.tabs[id] = ts
	m.activeTabId = id

	// Wire up CDP event listeners for this tab
	m.wireTabEvents(ts.Context(), id)

	if normalizedURL != "" {
		if err := chromedp.Run(ts.Context(), chromedp.Navigate(normalizedURL)); err != nil {
			_ = err
		}
	}

	return id, nil
}

// CloseTab closes a tab by id (defaults to active).
func (m *BrowserManager) CloseTab(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if id == 0 {
		id = m.activeTabId
	}
	ts, ok := m.tabs[id]
	if !ok {
		return fmt.Errorf("tab %d not found", id)
	}

	ts.Close()
	delete(m.tabs, id)
	delete(m.tabOwnership, id)

	if id == m.activeTabId {
		// Switch to the most recently created remaining tab
		var maxId int
		for tid := range m.tabs {
			if tid > maxId {
				maxId = tid
			}
		}
		if maxId > 0 {
			m.activeTabId = maxId
		} else {
			// No tabs left — create a blank one
			_, _ = m.newTabLocked("")
		}
	}
	return nil
}

// CloseAllTabs closes every tab except one blank tab.
func (m *BrowserManager) CloseAllTabs() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, ts := range m.tabs {
		ts.Close()
		delete(m.tabs, id)
		delete(m.tabOwnership, id)
	}
	// Create a fresh blank tab
	_, err := m.newTabLocked("")
	return err
}

// SwitchTab makes id the active tab.
func (m *BrowserManager) SwitchTab(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tabs[id]; !ok {
		return fmt.Errorf("tab %d not found", id)
	}
	m.activeTabId = id
	return nil
}

// ActiveTabId returns the currently active tab id.
func (m *BrowserManager) ActiveTabId() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeTabId
}

// TabCount returns the number of open tabs.
func (m *BrowserManager) TabCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tabs)
}

// ─── Session Access ───────────────────────────────────────

// GetActiveSession returns the TabSession for the active tab.
func (m *BrowserManager) GetActiveSession() (*TabSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.tabs[m.activeTabId]
	if !ok {
		return nil, fmt.Errorf("no active page. Use \"browse goto <url>\" first.")
	}
	return session, nil
}

// GetSession returns the TabSession for a specific tab id.
func (m *BrowserManager) GetSession(tabId int) (*TabSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.tabs[tabId]
	if !ok {
		return nil, fmt.Errorf("tab %d not found", tabId)
	}
	return session, nil
}

// ─── Ref Map Delegation ───────────────────────────────────

// SetRefMap delegates to the active session.
func (m *BrowserManager) SetRefMap(refs map[string]RefEntry) {
	session, _ := m.GetActiveSession()
	if session != nil {
		session.SetRefMap(refs)
	}
}

// ClearRefs delegates to the active session.
func (m *BrowserManager) ClearRefs() {
	session, _ := m.GetActiveSession()
	if session != nil {
		session.ClearRefs()
	}
}

// ResolveRef delegates to the active session.
func (m *BrowserManager) ResolveRef(selector string) (string, error) {
	session, err := m.GetActiveSession()
	if err != nil {
		return "", err
	}
	return session.ResolveRef(selector)
}

// GetRefRole delegates to the active session.
func (m *BrowserManager) GetRefRole(selector string) string {
	session, _ := m.GetActiveSession()
	if session == nil {
		return ""
	}
	return session.GetRefRole(selector)
}

// RefCount delegates to the active session.
func (m *BrowserManager) RefCount() int {
	session, _ := m.GetActiveSession()
	if session == nil {
		return 0
	}
	return session.RefCount()
}

// RefEntries delegates to the active session.
func (m *BrowserManager) RefEntries() []struct{ Ref, Role, Name string } {
	session, _ := m.GetActiveSession()
	if session == nil {
		return nil
	}
	return session.RefEntries()
}

// ─── Snapshot Delegation ──────────────────────────────────

// SetLastSnapshot delegates to the active session.
func (m *BrowserManager) SetLastSnapshot(text string) {
	session, _ := m.GetActiveSession()
	if session != nil {
		session.SetLastSnapshot(text)
	}
}

// GetLastSnapshot delegates to the active session.
func (m *BrowserManager) GetLastSnapshot() string {
	session, _ := m.GetActiveSession()
	if session == nil {
		return ""
	}
	return session.GetLastSnapshot()
}

// ─── Page-Level Accessors ─────────────────────────────────

// CurrentURL returns the active tab's URL, or about:blank.
func (m *BrowserManager) CurrentURL() string {
	m.cachedURLMu.RLock()
	cached := m.cachedURL
	m.cachedURLMu.RUnlock()
	if cached != "" {
		return cached
	}
	// Fallback: query directly
	m.mu.RLock()
	ts, ok := m.tabs[m.activeTabId]
	m.mu.RUnlock()
	if !ok {
		return "about:blank"
	}
	var url string
	if err := chromedp.Run(ts.Context(), chromedp.Location(&url)); err != nil {
		return "about:blank"
	}
	m.setCachedURL(url)
	return url
}

func (m *BrowserManager) setCachedURL(url string) {
	m.cachedURLMu.Lock()
	m.cachedURL = url
	m.cachedURLMu.Unlock()
}

// TabTitle returns the title of the given tab.
func (m *BrowserManager) TabTitle(tabId int) string {
	m.mu.RLock()
	ts, ok := m.tabs[tabId]
	m.mu.RUnlock()
	if !ok {
		return ""
	}
	var title string
	if err := chromedp.Run(ts.Context(), chromedp.Title(&title)); err != nil {
		return ""
	}
	return title
}

// TabURL returns the URL of the given tab.
func (m *BrowserManager) TabURL(tabId int) string {
	m.mu.RLock()
	ts, ok := m.tabs[tabId]
	m.mu.RUnlock()
	if !ok {
		return ""
	}
	var url string
	if err := chromedp.Run(ts.Context(), chromedp.Location(&url)); err != nil {
		return ""
	}
	return url
}

// TabList returns metadata for all tabs.
func (m *BrowserManager) TabList() []struct {
	ID     int    `json:"id"`
	URL    string `json:"url"`
	Title  string `json:"title"`
	Active bool   `json:"active"`
} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]struct {
		ID     int    `json:"id"`
		URL    string `json:"url"`
		Title  string `json:"title"`
		Active bool   `json:"active"`
	}, 0, len(m.tabs))

	for id, ts := range m.tabs {
		var url, title string
		_ = chromedp.Run(ts.Context(), chromedp.Location(&url), chromedp.Title(&title))
		out = append(out, struct {
			ID     int    `json:"id"`
			URL    string `json:"url"`
			Title  string `json:"title"`
			Active bool   `json:"active"`
		}{
			ID: id, URL: url, Title: title, Active: id == m.activeTabId,
		})
	}
	return out
}

// ─── Health Check ─────────────────────────────────────────

// IsHealthy checks whether Chromium is responsive.
func (m *BrowserManager) IsHealthy() bool {
	m.mu.RLock()
	ts, ok := m.tabs[m.activeTabId]
	m.mu.RUnlock()
	if !ok {
		return m.browserCtx != nil
	}

	ctx, cancel := context.WithTimeout(ts.Context(), 2*time.Second)
	defer cancel()

	var one int
	if err := chromedp.Run(ctx, chromedp.Evaluate("1", &one)); err != nil {
		return false
	}
	return true
}

// ─── Dialog Control ───────────────────────────────────────

// SetDialogAutoAccept controls whether dialogs are auto-accepted.
func (m *BrowserManager) SetDialogAutoAccept(accept bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dialogAutoAccept = accept
}

// DialogAutoAccept returns the current dialog auto-accept setting.
func (m *BrowserManager) DialogAutoAccept() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dialogAutoAccept
}

// SetDialogPromptText sets the text used for prompt auto-accept.
func (m *BrowserManager) SetDialogPromptText(text string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dialogPromptText = text
}

// DialogPromptText returns the current prompt text.
func (m *BrowserManager) DialogPromptText() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dialogPromptText
}

// ─── Viewport ─────────────────────────────────────────────

// SetViewport updates the viewport size on the active tab.
func (m *BrowserManager) SetViewport(width, height int) error {
	m.mu.Lock()
	m.viewportWidth = width
	m.viewportHeight = height
	m.mu.Unlock()

	ts, err := m.GetActiveSession()
	if err != nil {
		return err
	}
	return chromedp.Run(ts.Context(), chromedp.EmulateViewport(int64(width), int64(height)))
}

// ─── Extra Headers ────────────────────────────────────────

// SetExtraHeader sets an extra HTTP header for all tabs.
func (m *BrowserManager) SetExtraHeader(name, value string) error {
	m.mu.Lock()
	m.extraHeaders[name] = value
	headers := make(map[string]interface{}, len(m.extraHeaders))
	for k, v := range m.extraHeaders {
		headers[k] = v
	}
	m.mu.Unlock()

	m.mu.RLock()
	browserCtx := m.browserCtx
	m.mu.RUnlock()
	if browserCtx == nil {
		return nil
	}
	return chromedp.Run(browserCtx, network.SetExtraHTTPHeaders(headers))
}

// ─── User Agent ───────────────────────────────────────────

// SetUserAgent sets a custom user agent. Requires a relaunch to take effect.
func (m *BrowserManager) SetUserAgent(ua string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.customUserAgent = ua
}

// UserAgent returns the custom user agent, or empty if not set.
func (m *BrowserManager) UserAgent() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.customUserAgent
}

// ─── Proxy ────────────────────────────────────────────────

// SetProxyConfig sets the proxy used at launch time.
func (m *BrowserManager) SetProxyConfig(cfg *ProxyConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.proxyConfig = cfg
}

// ─── Tab Ownership ────────────────────────────────────────

// GetTabOwner returns the owner clientId for a tab, or empty if unowned.
func (m *BrowserManager) GetTabOwner(tabId int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tabOwnership[tabId]
}

// CheckTabAccess checks whether a client can access a tab.
func (m *BrowserManager) CheckTabAccess(tabId int, clientId string, ownOnly bool) bool {
	if clientId == "root" {
		return true
	}
	if ownOnly {
		m.mu.RLock()
		owner := m.tabOwnership[tabId]
		m.mu.RUnlock()
		return owner == clientId
	}
	return true
}

// TransferTab transfers ownership of a tab.
func (m *BrowserManager) TransferTab(tabId int, toClientId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tabs[tabId]; !ok {
		return fmt.Errorf("tab %d not found", tabId)
	}
	m.tabOwnership[tabId] = toClientId
	return nil
}

// ─── Network Capture ──────────────────────────────────────

// StartCapture begins capturing response bodies matching the optional filter.
func (m *BrowserManager) StartCapture(filterPattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.captureActive = true
	if filterPattern != "" {
		m.captureFilter = regexp.MustCompile(filterPattern)
	} else {
		m.captureFilter = nil
	}
	return nil
}

// StopCapture stops capturing and returns stats.
func (m *BrowserManager) StopCapture() (count int, sizeKB int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.captureActive = false
	m.captureFilter = nil
	return m.captureBuffer.Len(), m.captureBuffer.ByteSize() / 1024
}

// ClearCapture clears the capture buffer.
func (m *BrowserManager) ClearCapture() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.captureBuffer.Clear()
}

// ExportCapture writes captured responses to a JSONL file.
func (m *BrowserManager) ExportCapture(filePath string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.captureBuffer.ExportToFile(filePath)
}

// CaptureBuffer returns the underlying buffer (for tests).
func (m *BrowserManager) CaptureBuffer() *SizeCappedBuffer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.captureBuffer
}

// ─── Buffer Access ────────────────────────────────────────

// ConsoleBuffer returns the console log buffer.
func (m *BrowserManager) ConsoleBuffer() *buffers.Circular[ConsoleEntry] {
	return m.consoleBuffer
}

// NetworkBuffer returns the network request/response buffer.
func (m *BrowserManager) NetworkBuffer() *buffers.Circular[NetworkEntry] {
	return m.networkBuffer
}

// DialogBuffer returns the dialog event buffer.
func (m *BrowserManager) DialogBuffer() *buffers.Circular[DialogEntry] {
	return m.dialogBuffer
}

// ─── Watch Mode ───────────────────────────────────────────

// WatchResult holds the outcome of a watch session.
type WatchResult struct {
	Duration   time.Duration
	Snapshots  []string
}

// IsWatching reports whether watch mode is active.
func (m *BrowserManager) IsWatching() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.watching
}

// StartWatch begins collecting periodic snapshots.
func (m *BrowserManager) StartWatch() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.watching = true
	m.watchStartTime = time.Now()
	m.watchSnapshots = []string{}
}

// StopWatch ends watch mode and returns the collected data.
func (m *BrowserManager) StopWatch() WatchResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.watching = false
	return WatchResult{
		Duration:  time.Since(m.watchStartTime),
		Snapshots: append([]string(nil), m.watchSnapshots...),
	}
}

// AddWatchSnapshot stores a snapshot taken during watch mode.
func (m *BrowserManager) AddWatchSnapshot(snap string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.watching {
		return
	}
	m.watchSnapshots = append(m.watchSnapshots, snap)
}

// ─── CDP Event Wiring ─────────────────────────────────────

// wireTabEvents attaches console/network/dialog listeners to a tab context.
func (m *BrowserManager) wireTabEvents(tabCtx context.Context, tabId int) {
	// In-flight request tracking for response correlation
	pending := make(map[network.RequestID]*NetworkEntry)
	var pendingMu sync.Mutex

	chromedp.ListenTarget(tabCtx, func(ev interface{}) {
		switch e := ev.(type) {
		// ─── Navigation (URL cache) ──────────────────────────
		case *page.EventFrameNavigated:
			if e.Frame != nil && e.Frame.ParentID == "" {
				m.setCachedURL(e.Frame.URL)
			}

		// ─── Console ─────────────────────────────────────────
		case *runtime.EventConsoleAPICalled:
			var text string
			for _, arg := range e.Args {
				if arg.Value != nil {
					text += string(arg.Value) + " "
				}
			}
			text = strings.TrimSpace(text)
			m.consoleBuffer.Add(ConsoleEntry{
				Timestamp: time.Now().UnixMilli(),
				Level:     string(e.Type),
				Text:      text,
			})

		// ─── Dialog ──────────────────────────────────────────
		case *page.EventJavascriptDialogOpening:
			m.mu.RLock()
			autoAccept := m.dialogAutoAccept
			promptText := m.dialogPromptText
			m.mu.RUnlock()

			action := "dismissed"
			if autoAccept {
				action = "accepted"
			}
			var response string
			if autoAccept && e.Type == page.DialogTypePrompt {
				response = promptText
			}

			m.dialogBuffer.Add(DialogEntry{
				Timestamp:  time.Now().UnixMilli(),
				Type:       string(e.Type),
				Message:    e.Message,
				DefaultVal: e.DefaultPrompt,
				Action:     action,
				Response:   response,
			})

			// Auto-handle the dialog
			go func() {
				ctx, cancel := context.WithTimeout(tabCtx, 5*time.Second)
				defer cancel()
				action := page.HandleJavaScriptDialog(autoAccept)
				if autoAccept && e.Type == page.DialogTypePrompt {
					action = action.WithPromptText(promptText)
				}
				_ = action.Do(ctx)
			}()

		// ─── Network Request ─────────────────────────────────
		case *network.EventRequestWillBeSent:
			entry := &NetworkEntry{
				Timestamp: time.Now().UnixMilli(),
				Method:    e.Request.Method,
				URL:       e.Request.URL,
			}
			m.networkBuffer.Add(*entry)
			pendingMu.Lock()
			pending[e.RequestID] = entry
			pendingMu.Unlock()

		// ─── Network Response ────────────────────────────────
		case *network.EventResponseReceived:
			pendingMu.Lock()
			if entry, ok := pending[e.RequestID]; ok {
				entry.Status = int(e.Response.Status)
				entry.Duration = time.Now().UnixMilli() - entry.Timestamp
				m.networkBuffer.Add(*entry)
			}
			pendingMu.Unlock()

			// Response-body capture
			m.mu.RLock()
			captureActive := m.captureActive
			captureFilter := m.captureFilter
			m.mu.RUnlock()
			if captureActive {
				if captureFilter == nil || captureFilter.MatchString(e.Response.URL) {
					go m.captureResponseBody(tabCtx, e.RequestID, e.Response)
				}
			}

		// ─── Network Loading Finished ────────────────────────
		case *network.EventLoadingFinished:
			pendingMu.Lock()
			delete(pending, e.RequestID)
			pendingMu.Unlock()
		}
	})
}

// captureResponseBody fetches the body of a response and stores it.
func (m *BrowserManager) captureResponseBody(ctx context.Context, reqID network.RequestID, resp *network.Response) {
	// Skip non-content responses
	if resp.Status == 204 || resp.Status == 301 || resp.Status == 302 || resp.Status == 304 {
		return
	}

	contentType := ""
	for k, v := range resp.Headers {
		if strings.EqualFold(k, "content-type") {
			contentType = fmt.Sprint(v)
			break
		}
	}

	body, size, truncated := "", 0, false

	// Try to get the response body via Network.getResponseBody
	var rawBody []byte
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
		var err error
		rawBody, err = network.GetResponseBody(reqID).Do(c)
		return err
	})); err == nil {
		size = len(rawBody)
		if size > defaultMaxEntrySize {
			truncated = true
			body = ""
		} else if isTextContent(contentType) {
			body = string(rawBody)
		} else {
			// Binary: store empty with truncated flag
			body = ""
			truncated = true
		}
	}

	m.captureBuffer.Push(CapturedResponse{
		URL:           resp.URL,
		Status:        int(resp.Status),
		Headers:       headersToMap(resp.Headers),
		Body:          body,
		ContentType:   contentType,
		Timestamp:     time.Now().UnixMilli(),
		Size:          size,
		BodyTruncated: truncated,
	})
}

func isTextContent(ct string) bool {
	ct = strings.ToLower(ct)
	return strings.Contains(ct, "json") || strings.Contains(ct, "text") ||
		strings.Contains(ct, "xml") || strings.Contains(ct, "html")
}

func headersToMap(headers network.Headers) map[string]string {
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		out[k] = fmt.Sprint(v)
	}
	return out
}

// ─── Chromium Path Helpers ────────────────────────────────

// FindChromiumExecutable resolves the Chromium binary path.
func FindChromiumExecutable() string {
	if p := os.Getenv("GSTACK_CHROMIUM_PATH"); p != "" {
		return p
	}
	// macOS: check well-known app bundle paths
	macOSPaths := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
	}
	for _, p := range macOSPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// PATH lookup
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

// IsCustomChromium returns true when running against a custom Chromium build
// that bakes the gstack extension in as a component extension.
func IsCustomChromium() bool {
	if os.Getenv("GSTACK_CHROMIUM_KIND") == "custom-extension-baked" {
		return true
	}
	p := os.Getenv("GSTACK_CHROMIUM_PATH")
	return strings.Contains(p, "GBrowser") || strings.Contains(p, "gbrowser")
}

// FindExtensionPath searches for the gstack Chrome extension directory.
func FindExtensionPath() string {
	candidates := []string{
		os.Getenv("BROWSE_EXTENSIONS_DIR"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			config.Home()+"/skills/gstack/extension",
			home+"/.claude/skills/gstack/extension",
		)
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if _, err := os.Stat(c + "/manifest.json"); err == nil {
			return c
		}
	}
	return ""
}

// CleanSingletonLocks removes stale Chromium lockfiles from the profile dir.
func CleanSingletonLocks() {
	config.CleanSingletonLocks(config.ChromiumProfile(""))
}

// writeAuthJSON writes the auth token and port to ~/.gstack/.auth.json for the extension.
func writeAuthJSON(token string, port int) {
	gstackDir := config.Home()
	_ = os.MkdirAll(gstackDir, 0700)
	authFile := filepath.Join(gstackDir, ".auth.json")
	data, _ := json.Marshal(map[string]interface{}{
		"token": token,
		"port":  port,
	})
	_ = os.WriteFile(authFile, data, 0600)
}

// TrackCookieImportDomains records domains whose cookies were imported from a browser.
func (m *BrowserManager) TrackCookieImportDomains(domains []string) {
	m.cookieImportDomainsMu.Lock()
	defer m.cookieImportDomainsMu.Unlock()
	for _, d := range domains {
		found := false
		for _, existing := range m.cookieImportDomains {
			if existing == d {
				found = true
				break
			}
		}
		if !found {
			m.cookieImportDomains = append(m.cookieImportDomains, d)
		}
	}
}

// GetCookieImportDomains returns domains whose cookies were imported from a browser.
func (m *BrowserManager) GetCookieImportDomains() []string {
	m.cookieImportDomainsMu.RLock()
	defer m.cookieImportDomainsMu.RUnlock()
	out := make([]string, len(m.cookieImportDomains))
	copy(out, m.cookieImportDomains)
	return out
}

// TrackCookieImportWithCounts records domains and their cookie counts.
func (m *BrowserManager) TrackCookieImportWithCounts(counts map[string]int) {
	m.cookieImportDomainsMu.Lock()
	defer m.cookieImportDomainsMu.Unlock()
	if m.cookieImportCounts == nil {
		m.cookieImportCounts = make(map[string]int)
	}
	for domain, count := range counts {
		m.cookieImportCounts[domain] = count
		found := false
		for _, existing := range m.cookieImportDomains {
			if existing == domain {
				found = true
				break
			}
		}
		if !found {
			m.cookieImportDomains = append(m.cookieImportDomains, domain)
		}
	}
}

// UntrackCookieImportDomain removes a domain from import tracking.
func (m *BrowserManager) UntrackCookieImportDomain(domain string) {
	m.cookieImportDomainsMu.Lock()
	defer m.cookieImportDomainsMu.Unlock()
	delete(m.cookieImportCounts, domain)
	var filtered []string
	for _, d := range m.cookieImportDomains {
		if d != domain {
			filtered = append(filtered, d)
		}
	}
	m.cookieImportDomains = filtered
}

// GetCookieImportCounts returns a copy of the domain -> count map.
func (m *BrowserManager) GetCookieImportCounts() map[string]int {
	m.cookieImportDomainsMu.RLock()
	defer m.cookieImportDomainsMu.RUnlock()
	out := make(map[string]int, len(m.cookieImportCounts))
	for k, v := range m.cookieImportCounts {
		out[k] = v
	}
	return out
}

// SetCDPCookies sets cookies in the browser context via CDP.
func (m *BrowserManager) SetCDPCookies(params []*network.CookieParam) error {
	m.mu.RLock()
	browserCtx := m.browserCtx
	m.mu.RUnlock()
	if browserCtx == nil {
		return fmt.Errorf("browser not launched")
	}
	return chromedp.Run(browserCtx, network.SetCookies(params))
}

// ClearCookiesForDomain removes all cookies matching the given domain from the browser.
func (m *BrowserManager) ClearCookiesForDomain(domain string) error {
	m.mu.RLock()
	browserCtx := m.browserCtx
	m.mu.RUnlock()
	if browserCtx == nil {
		return fmt.Errorf("browser not launched")
	}

	var allCookies []*network.Cookie
	if err := chromedp.Run(browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var result struct {
			Cookies []*network.Cookie `json:"cookies"`
		}
		if err := cdp.Execute(ctx, "Network.getAllCookies", nil, &result); err != nil {
			return err
		}
		allCookies = result.Cookies
		return nil
	})); err != nil {
		return fmt.Errorf("failed to list cookies: %w", err)
	}

	var deleted int
	for _, c := range allCookies {
		match := c.Domain == domain
		if !match && strings.HasPrefix(c.Domain, ".") {
			match = c.Domain == "."+domain
		}
		if !match && strings.HasPrefix(domain, ".") {
			match = c.Domain == domain || c.Domain == domain[1:]
		}
		if !match {
			continue
		}
		if err := chromedp.Run(browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			return network.DeleteCookies(c.Name).
				WithDomain(c.Domain).
				WithPath(c.Path).
				Do(ctx)
		})); err != nil {
			// Non-fatal: some cookies may fail to delete
			continue
		}
		deleted++
	}
	if deleted == 0 {
		return nil // no matching cookies is not an error
	}
	return nil
}

// ─── Tab Awareness ──────────────────────────────────────────

// WriteTabState writes tabs.json and active-tab.json to stateDir for
// sidebar tab-awareness. Safe to call frequently (best-effort, no errors).
func (m *BrowserManager) WriteTabState(stateDir string) {
	if stateDir == "" {
		return
	}
	_ = os.MkdirAll(stateDir, 0755)

	// tabs.json — full list
	tabs := m.TabList()
	type tabEntry struct {
		ID       int    `json:"id"`
		URL      string `json:"url"`
		Title    string `json:"title"`
		Active   bool   `json:"active"`
		Pinned   bool   `json:"pinned"`
		Audible  bool   `json:"audible"`
	}
	entries := make([]tabEntry, len(tabs))
	for i, t := range tabs {
		entries[i] = tabEntry{ID: t.ID, URL: t.URL, Title: t.Title, Active: t.Active}
	}
	tabsPayload := struct {
		UpdatedAt string     `json:"updatedAt"`
		Tabs      []tabEntry `json:"tabs"`
	}{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Tabs:      entries,
	}
	if data, err := json.MarshalIndent(tabsPayload, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(stateDir, "tabs.json"), data, 0644)
	}

	// active-tab.json — current active tab
	activeID := m.ActiveTabId()
	if activeID > 0 {
		for _, t := range tabs {
			if t.ID == activeID && t.URL != "" &&
				!strings.HasPrefix(t.URL, "chrome://") &&
				!strings.HasPrefix(t.URL, "chrome-extension://") {
				activePayload := struct {
					TabID int    `json:"tabId"`
					URL   string `json:"url"`
					Title string `json:"title"`
				}{TabID: activeID, URL: t.URL, Title: t.Title}
				if data, err := json.MarshalIndent(activePayload, "", "  "); err == nil {
					_ = os.WriteFile(filepath.Join(stateDir, "active-tab.json"), data, 0644)
				}
				break
			}
		}
	}
}
