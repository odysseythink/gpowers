package browser

import (
	"context"
	"fmt"
	"sync"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

// RefEntry maps a snapshot ref (e.g. "e3", "c1") to a CSS selector and
// descriptive metadata.  Unlike the Playwright port which stores a Locator
// object, the Go port stores the raw selector string and validates it at
// resolve-time with a lightweight DOM query.
type RefEntry struct {
	Selector string
	Role     string
	Name     string
}

// StyleMod records a single style modification for undo support.
type StyleMod struct {
	Selector string
	Property string
	OldValue string
	NewValue string
}

// TabSession holds per-tab state: element refs, snapshot baseline, and
// loaded-html replay metadata.  Each session wraps a chromedp context that
// represents one browser target (tab).
type TabSession struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu                  sync.RWMutex
	refs                map[string]RefEntry
	lastSnapshot        string
	loadedHtml          string
	loadedHtmlWaitUntil string
	inFrame             bool // true when operating inside an iframe

	// Frame switching (CDP Target.attachToTarget)
	frameCtx      context.Context
	frameCancel   context.CancelFunc
	frameTargetID target.ID

	// Style modification history (for style --undo)
	styleHistory []StyleMod
}

// NewTabSession creates a new chromedp tab context under the given browser
// context and returns the session wrapper.
func NewTabSession(parentCtx context.Context) *TabSession {
	ctx, cancel := chromedp.NewContext(parentCtx)
	return &TabSession{
		ctx:    ctx,
		cancel: cancel,
		refs:   make(map[string]RefEntry),
	}
}

// Context returns the chromedp context for this tab.
// If a frame context is active, returns the frame context instead.
func (t *TabSession) Context() context.Context {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.frameCtx != nil {
		return t.frameCtx
	}
	return t.ctx
}

// Close cancels the chromedp context, closing the underlying tab.
func (t *TabSession) Close() {
	t.mu.Lock()
	if t.frameCancel != nil {
		t.frameCancel()
	}
	t.mu.Unlock()
	t.cancel()
}

// ─── Ref Map ───────────────────────────────────────────────

// SetRefMap replaces the entire ref map.
func (t *TabSession) SetRefMap(refs map[string]RefEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.refs = refs
}

// ClearRefs empties the ref map.
func (t *TabSession) ClearRefs() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.refs = make(map[string]RefEntry)
}

// ResolveRef turns a @-prefixed ref selector into its underlying CSS
// selector.  Non-ref selectors are returned unchanged.  Stale refs (element
// no longer in the DOM) return an error.
func (t *TabSession) ResolveRef(selector string) (string, error) {
	if !isRefSelector(selector) {
		return selector, nil
	}
	ref := selector[1:] // strip leading '@'

	t.mu.RLock()
	entry, ok := t.refs[ref]
	t.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("ref %s not found. Run 'snapshot' to get fresh refs.", selector)
	}

	// Verify the element is still present in the DOM.
	var nodes []*cdp.Node
	if err := chromedp.Run(t.Context(), chromedp.Nodes(entry.Selector, &nodes, chromedp.AtLeast(0))); err != nil {
		return "", fmt.Errorf("ref %s (%s %q) is stale — element no longer exists. Run 'snapshot' for fresh refs.", selector, entry.Role, entry.Name)
	}
	if len(nodes) == 0 {
		return "", fmt.Errorf("ref %s (%s %q) is stale — element no longer exists. Run 'snapshot' for fresh refs.", selector, entry.Role, entry.Name)
	}
	return entry.Selector, nil
}

// GetRefRole returns the ARIA role for a ref selector, or "" for CSS
// selectors / unknown refs.
func (t *TabSession) GetRefRole(selector string) string {
	if !isRefSelector(selector) {
		return ""
	}
	t.mu.RLock()
	entry, ok := t.refs[selector[1:]]
	t.mu.RUnlock()
	if !ok {
		return ""
	}
	return entry.Role
}

// RefCount returns the number of stored refs.
func (t *TabSession) RefCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.refs)
}

// RefEntries returns all refs for the /refs endpoint.
func (t *TabSession) RefEntries() []struct{ Ref, Role, Name string } {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]struct{ Ref, Role, Name string }, 0, len(t.refs))
	for ref, entry := range t.refs {
		out = append(out, struct{ Ref, Role, Name string }{
			Ref: ref, Role: entry.Role, Name: entry.Name,
		})
	}
	return out
}

func isRefSelector(s string) bool {
	return len(s) >= 2 && s[0] == '@' && (s[1] == 'e' || s[1] == 'c')
}

// ─── Snapshot Diffing ─────────────────────────────────────

// SetLastSnapshot stores the text baseline for diffing.
func (t *TabSession) SetLastSnapshot(text string) { t.lastSnapshot = text }

// GetLastSnapshot returns the stored baseline.
func (t *TabSession) GetLastSnapshot() string { return t.lastSnapshot }

// ─── Loaded HTML (load-html replay) ───────────────────────

// SetTabContent stores HTML for replay after context recreation.
func (t *TabSession) SetTabContent(html string, waitUntil string) {
	t.loadedHtml = html
	t.loadedHtmlWaitUntil = waitUntil
}

// GetLoadedHtml returns stored HTML + waitUntil, or nil if none.
func (t *TabSession) GetLoadedHtml() (html string, waitUntil string, ok bool) {
	if t.loadedHtml == "" {
		return "", "", false
	}
	return t.loadedHtml, t.loadedHtmlWaitUntil, true
}

// ClearLoadedHtml clears stored HTML. Called before goto/back/forward/reload.
func (t *TabSession) ClearLoadedHtml() {
	t.loadedHtml = ""
	t.loadedHtmlWaitUntil = ""
}

// OnMainFrameNavigated clears stale refs and replay metadata on navigation.
func (t *TabSession) OnMainFrameNavigated() {
	t.ClearRefs()
	t.ClearLoadedHtml()
	t.ClearStyleHistory()
	// Reset frame state directly (avoid nested lock in SwitchToMainFrame)
	t.mu.Lock()
	if t.frameCancel != nil {
		t.frameCancel()
	}
	t.frameCtx = nil
	t.frameCancel = nil
	t.frameTargetID = ""
	t.inFrame = false
	t.mu.Unlock()
}

// SetInFrame marks whether subsequent operations target an iframe.
func (t *TabSession) SetInFrame(v bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.inFrame = v
}

// InFrame reports whether the session is currently targeting an iframe.
func (t *TabSession) InFrame() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.inFrame
}

// ─── Frame Switching ──────────────────────────────────────

// SwitchToFrame switches the session context to the iframe identified by
// the given target ID. It uses chromedp's WithTargetID to create a new
// context bound to the iframe's CDP target.
func (t *TabSession) SwitchToFrame(targetID target.ID) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Clean up any existing frame context
	if t.frameCancel != nil {
		t.frameCancel()
	}

	// Create a new chromedp context bound to the iframe target.
	// The parent context (t.ctx) carries the browser allocator info
	// required by chromedp to resolve the target.
	frameCtx, cancel := chromedp.NewContext(t.ctx, chromedp.WithTargetID(targetID))

	// Wait for the frame to be ready before accepting it.
	if err := chromedp.Run(frameCtx, chromedp.WaitReady("html", chromedp.ByQuery)); err != nil {
		cancel()
		return fmt.Errorf("frame target did not become ready: %w", err)
	}

	t.frameCtx = frameCtx
	t.frameCancel = cancel
	t.frameTargetID = targetID
	t.inFrame = true
	return nil
}

// SwitchToMainFrame returns the session context to the main page,
// cancelling any active frame context.
func (t *TabSession) SwitchToMainFrame() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.frameCancel != nil {
		t.frameCancel()
	}
	t.frameCtx = nil
	t.frameCancel = nil
	t.frameTargetID = ""
	t.inFrame = false
}

// GetFrameTargetID returns the target ID of the currently active frame,
// or empty string if operating on the main page.
func (t *TabSession) GetFrameTargetID() target.ID {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.frameTargetID
}

// GetURL returns the current page URL, or empty string on error.
func (t *TabSession) GetURL() string {
	var location string
	if err := chromedp.Run(t.ctx, chromedp.Evaluate(`window.location.href`, &location)); err != nil {
		return ""
	}
	return location
}

// ─── Style History ────────────────────────────────────────

// PushStyleMod appends a style modification to the history stack.
func (t *TabSession) PushStyleMod(mod StyleMod) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.styleHistory = append(t.styleHistory, mod)
}

// PopStyleMod removes and returns the most recent style modification.
// Returns nil if the history is empty.
func (t *TabSession) PopStyleMod() *StyleMod {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := len(t.styleHistory)
	if n == 0 {
		return nil
	}
	mod := t.styleHistory[n-1]
	t.styleHistory = t.styleHistory[:n-1]
	return &mod
}

// StyleModCount returns the number of stored style modifications.
func (t *TabSession) StyleModCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.styleHistory)
}

// ClearStyleHistory empties the style modification stack.
func (t *TabSession) ClearStyleHistory() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.styleHistory = nil
}
