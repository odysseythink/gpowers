package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"browse-go/pkg/util"
)

// BrowserState captures cookies, open pages, storage, and ownership for
// save/restore across context recreations or handoffs.
type BrowserState struct {
	Cookies []*network.Cookie
	Pages   []PageState
}

// PageState holds per-tab data that survives context recreation.
type PageState struct {
	URL                 string
	IsActive            bool
	LocalStorage        map[string]string
	SessionStorage      map[string]string
	LoadedHtml          string
	LoadedHtmlWaitUntil string
	Owner               string
}

// SaveState captures the current browser state.
func (m *BrowserManager) SaveState() (*BrowserState, error) {
	m.mu.RLock()
	browserCtx := m.browserCtx
	m.mu.RUnlock()
	if browserCtx == nil {
		return nil, fmt.Errorf("browser not launched")
	}

	// Get all cookies
	var cookies []*network.Cookie
	if err := chromedp.Run(browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(ctx)
		return err
	})); err != nil {
		// Non-fatal: proceed without cookies
		cookies = nil
	}

	m.mu.RLock()
	tabs := make(map[int]*TabSession, len(m.tabs))
	for id, ts := range m.tabs {
		tabs[id] = ts
	}
	m.mu.RUnlock()

	pages := make([]PageState, 0, len(tabs))
	for id, ts := range tabs {
		var url string
		_ = chromedp.Run(ts.Context(), chromedp.Location(&url))
		if url == "about:blank" {
			url = ""
		}

		// Read storage via evaluate
		var storageResult string
		_ = chromedp.Run(ts.Context(), chromedp.Evaluate(`
			JSON.stringify({
				localStorage: {...localStorage},
				sessionStorage: {...sessionStorage}
			})
		`, &storageResult))

		var localStorage, sessionStorage map[string]string
		if storageResult != "" {
			var parsed struct {
				LocalStorage   map[string]string `json:"localStorage"`
				SessionStorage map[string]string `json:"sessionStorage"`
			}
			if err := jsonUnmarshalString(storageResult, &parsed); err == nil {
				localStorage = parsed.LocalStorage
				sessionStorage = parsed.SessionStorage
			}
		}

		m.mu.RLock()
		owner := m.tabOwnership[id]
		m.mu.RUnlock()

		var loadedHtml, loadedWait string
		if h, w, ok := ts.GetLoadedHtml(); ok {
			loadedHtml = h
			loadedWait = w
		}

		pages = append(pages, PageState{
			URL:                 url,
			IsActive:            id == m.activeTabId,
			LocalStorage:        localStorage,
			SessionStorage:      sessionStorage,
			LoadedHtml:          loadedHtml,
			LoadedHtmlWaitUntil: loadedWait,
			Owner:               owner,
		})
	}

	return &BrowserState{Cookies: cookies, Pages: pages}, nil
}

// RestoreState recreates tabs, navigates to URLs, restores cookies and storage.
func (m *BrowserManager) RestoreState(state *BrowserState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.browserCtx == nil {
		return fmt.Errorf("browser not launched")
	}

	// Restore cookies
	if len(state.Cookies) > 0 {
		params := make([]*network.CookieParam, 0, len(state.Cookies))
		for _, c := range state.Cookies {
			params = append(params, cookieToParam(c))
		}
		_ = chromedp.Run(m.browserCtx, network.SetCookies(params))
	}

	// Clear old ownership — old tab IDs are gone
	m.tabOwnership = make(map[int]string)

	var activeId int
	for _, saved := range state.Pages {
		id, err := m.newTabLocked("")
		if err != nil {
			continue
		}
		ts := m.tabs[id]

		if saved.Owner != "" {
			m.tabOwnership[id] = saved.Owner
		}

		if saved.LoadedHtml != "" {
			if err := setTabContent(ts.Context(), saved.LoadedHtml, saved.LoadedHtmlWaitUntil); err == nil {
				ts.SetTabContent(saved.LoadedHtml, saved.LoadedHtmlWaitUntil)
			}
		} else if saved.URL != "" {
			normalized, err := util.ValidateNavigationURL(saved.URL)
			if err == nil {
				_ = chromedp.Run(ts.Context(), chromedp.Navigate(normalized))
			}
		}

		if saved.LocalStorage != nil || saved.SessionStorage != nil {
			_ = chromedp.Run(ts.Context(), chromedp.Evaluate(fmt.Sprintf(`
				(function(ls, ss) {
					if (ls) { for (var k in ls) localStorage.setItem(k, ls[k]); }
					if (ss) { for (var k in ss) sessionStorage.setItem(k, ss[k]); }
				})(%s, %s)
			`, jsonStringify(saved.LocalStorage), jsonStringify(saved.SessionStorage)), nil))
		}

		if saved.IsActive {
			activeId = id
		}
	}

	if len(m.tabs) == 0 {
		_, _ = m.newTabLocked("")
	} else if activeId > 0 {
		m.activeTabId = activeId
	} else {
		// Pick the first available tab
		for id := range m.tabs {
			m.activeTabId = id
			break
		}
	}

	m.ClearRefs()
	return nil
}

// RecreateContext closes the current context and creates a new one with
// updated settings (user agent, viewport). State is preserved.
func (m *BrowserManager) RecreateContext() (string, error) {
	m.mu.Lock()
	if m.connectionMode == "headed" {
		m.mu.Unlock()
		return "", fmt.Errorf("cannot recreate context in headed mode")
	}
	if m.browserCtx == nil {
		m.mu.Unlock()
		return "", fmt.Errorf("browser not launched")
	}
	m.mu.Unlock()

	state, err := m.SaveState()
	if err != nil {
		return err.Error(), err
	}

	m.mu.Lock()
	// Close all tabs
	for _, ts := range m.tabs {
		ts.Close()
	}
	m.tabs = make(map[int]*TabSession)
	if m.browserCancel != nil {
		m.browserCancel()
	}
	m.browserCtx, m.browserCancel = chromedp.NewContext(m.allocCtx)
	m.mu.Unlock()

	// Re-enable network and stealth on new context
	if err := chromedp.Run(m.browserCtx, network.Enable()); err != nil {
		// Fallback: start fresh
		_, _ = m.newTabLocked("")
		return fmt.Sprintf("context recreation failed (network enable): %v", err), err
	}
	if err := ApplyStealth(m.browserCtx); err != nil {
		_, _ = m.newTabLocked("")
		return fmt.Sprintf("context recreation failed (stealth): %v", err), err
	}

	// Restore extra headers
	m.mu.RLock()
	headers := make(map[string]interface{}, len(m.extraHeaders))
	for k, v := range m.extraHeaders {
		headers[k] = v
	}
	m.mu.RUnlock()
	if len(headers) > 0 {
		_ = chromedp.Run(m.browserCtx, network.SetExtraHTTPHeaders(headers))
	}

	// Restore viewport
	m.mu.RLock()
	vw, vh := m.viewportWidth, m.viewportHeight
	m.mu.RUnlock()
	_ = chromedp.Run(m.browserCtx, chromedp.EmulateViewport(int64(vw), int64(vh)))

	// Restore state
	if err := m.RestoreState(state); err != nil {
		// Fallback: blank tab
		m.mu.Lock()
		for _, ts := range m.tabs {
			ts.Close()
		}
		m.tabs = make(map[int]*TabSession)
		m.mu.Unlock()
		_, _ = m.newTabLocked("")
		m.ClearRefs()
		return fmt.Sprintf("context recreation failed during restore: %v", err), err
	}

	return "", nil
}

// SetDeviceScaleFactor updates the device pixel ratio. This requires a full
// context recreation in chromedp (same as Playwright).
func (m *BrowserManager) SetDeviceScaleFactor(scale int) (string, error) {
	if scale < 1 || scale > 3 {
		return "", fmt.Errorf("device scale factor must be between 1 and 3, got %d", scale)
	}
	m.mu.Lock()
	m.deviceScaleFactor = scale
	m.mu.Unlock()
	return m.RecreateContext()
}

// GetDeviceScaleFactor returns the current device scale factor.
func (m *BrowserManager) GetDeviceScaleFactor() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.deviceScaleFactor
}

// GetCurrentViewport returns the tracked viewport size.
func (m *BrowserManager) GetCurrentViewport() (width, height int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viewportWidth, m.viewportHeight
}

// CloseAllPages closes every open tab.
func (m *BrowserManager) CloseAllPages() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ts := range m.tabs {
		ts.Close()
	}
	m.tabs = make(map[int]*TabSession)
	m.tabOwnership = make(map[int]string)
	return nil
}

// SyncActiveTabByUrl switches the active tab to one matching the given URL.
func (m *BrowserManager) SyncActiveTabByUrl(activeUrl string) {
	if activeUrl == "" {
		return
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.tabs) <= 1 {
		return
	}

	for id, ts := range m.tabs {
		var url string
		if err := chromedp.Run(ts.Context(), chromedp.Location(&url)); err != nil {
			continue
		}
		if url == activeUrl && id != m.activeTabId {
			m.activeTabId = id
			return
		}
	}
}

// ─── Helpers ──────────────────────────────────────────────

func cookieToParam(c *network.Cookie) *network.CookieParam {
	cp := &network.CookieParam{
		Name:         c.Name,
		Value:        c.Value,
		Domain:       c.Domain,
		Path:         c.Path,
		Secure:       c.Secure,
		HTTPOnly:     c.HTTPOnly,
		SameSite:     c.SameSite,
		Priority:     c.Priority,
		SourceScheme: c.SourceScheme,
		SourcePort:   c.SourcePort,
		PartitionKey: c.PartitionKey,
	}
	if c.Expires >= 0 {
		t := cdp.TimeSinceEpoch(time.Unix(int64(c.Expires), 0))
		cp.Expires = &t
	}
	return cp
}

func setTabContent(ctx context.Context, html string, waitUntil string) error {
	// Navigate to about:blank first, then inject HTML via document.write
	if err := chromedp.Run(ctx, chromedp.Navigate("about:blank")); err != nil {
		return err
	}
	expr := fmt.Sprintf(`document.open(); document.write(%s); document.close();`, strconv.Quote(html))
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
		_, _, err := runtime.Evaluate(expr).Do(c)
		return err
	})); err != nil {
		return err
	}
	// Wait for body to be ready
	return chromedp.Run(ctx, chromedp.WaitReady("body"))
}

func jsonStringify(v map[string]string) string {
	if v == nil {
		return "null"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// jsonUnmarshalString unmarshals a JSON string value.
func jsonUnmarshalString(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}
