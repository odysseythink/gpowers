package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/sergi/go-diff/diffmatchpatch"

	"browse-go/pkg/browser"
	"browse-go/pkg/config"
	"browse-go/pkg/fs"
	"browse-go/pkg/util"
)

func (r *Registry) registerMeta() {
	r.Register("diff", CommandDesc{Category: "Meta", Description: "Text diff between pages", Usage: "diff <url1> <url2>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) < 2 {
				return "", fmt.Errorf("usage: diff <url1> <url2>")
			}
			url1, err := util.ValidateNavigationURL(ctx.Args[0])
			if err != nil {
				return "", err
			}
			url2, err := util.ValidateNavigationURL(ctx.Args[1])
			if err != nil {
				return "", err
			}

			// Fetch text from url1
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Navigate(url1)); err != nil {
				return "", fmt.Errorf("navigate url1 failed: %w", err)
			}
			var text1 string
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(getCleanTextJS, &text1)); err != nil {
				return "", fmt.Errorf("text1 extraction failed: %w", err)
			}

			// Fetch text from url2
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Navigate(url2)); err != nil {
				return "", fmt.Errorf("navigate url2 failed: %w", err)
			}
			var text2 string
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(getCleanTextJS, &text2)); err != nil {
				return "", fmt.Errorf("text2 extraction failed: %w", err)
			}

			dmp := diffmatchpatch.New()
			diffs := dmp.DiffMain(text1, text2, true)

			var out []string
			out = append(out, fmt.Sprintf("--- %s", url1), fmt.Sprintf("+++ %s", url2), "")
			for _, d := range diffs {
				prefix := " "
				switch d.Type {
				case diffmatchpatch.DiffInsert:
					prefix = "+"
				case diffmatchpatch.DiffDelete:
					prefix = "-"
				}
				for _, line := range strings.Split(d.Text, "\n") {
					if line != "" {
						out = append(out, prefix+" "+line)
					}
				}
			}
			return strings.Join(out, "\n"), nil
		})

	r.Register("chain", CommandDesc{Category: "Meta", Description: "Run a sequence of commands", Usage: `chain '["cmd", ...]' or chain 'cmd1 | cmd2'`},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf(`usage: chain '["goto","url"], ["text"]]'  or  chain 'goto url | text'`)
			}
			input := strings.Join(ctx.Args, " ")
			var rawCommands [][]string
			if err := json.Unmarshal([]byte(input), &rawCommands); err != nil {
				// Fallback: pipe-delimited format
				parts := strings.Split(input, "|")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p == "" {
						continue
					}
					fields := tokenize(p)
					if len(fields) > 0 {
						rawCommands = append(rawCommands, fields)
					}
				}
			}

			var results []string
			for _, cmd := range rawCommands {
				if len(cmd) == 0 {
					continue
				}
				name := Canonicalize(cmd[0])
				var args []string
				if len(cmd) > 1 {
					args = cmd[1:]
				}
				result, err := r.Execute(ctx.BM, name, args)
				if err != nil {
					return "", fmt.Errorf("chain failed at %q: %w", cmd[0], err)
				}
				results = append(results, result)
			}
			return strings.Join(results, "\n---\n"), nil
		})

	r.Register("pdf", CommandDesc{Category: "Meta", Description: "Save page as PDF", Usage: "pdf [path] [--format letter|a4|legal] [--width <dim>] [--height <dim>] [--margins <dim>] [--margin-top <dim>] [--margin-right <dim>] [--margin-bottom <dim>] [--margin-left <dim>] [--header-template <html>] [--footer-template <html>] [--page-numbers] [--tagged] [--outline] [--print-background] [--prefer-css-page-size] [--toc] | pdf --from-file <payload.json>"},
		handlePdf)

	r.Register("frame", CommandDesc{Category: "Meta", Description: "Switch to iframe context", Usage: "frame <sel|@ref|--name n|--url pattern|main>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: frame <sel|@ref|--name n|--url pattern|main>")
			}
			targetArg := ctx.Args[0]
			if targetArg == "main" {
				ctx.Session.SwitchToMainFrame()
				ctx.Session.ClearRefs()
				ctx.Session.SetLastSnapshot("")
				return "Switched to main frame", nil
			}

			// Enumerate all CDP targets so we can match iframe URLs.
			var targets []*target.Info
			if err := chromedp.Run(ctx.Session.Context(), chromedp.ActionFunc(func(c context.Context) error {
				var err error
				targets, err = target.GetTargets().Do(c)
				return err
			})); err != nil {
				return "", fmt.Errorf("failed to enumerate targets: %w", err)
			}

			var matchURL string
			var err error

			switch targetArg {
			case "--name":
				if len(ctx.Args) < 2 {
					return "", fmt.Errorf("usage: frame --name <name>")
				}
				matchURL, err = resolveIframeURLByName(ctx.Session.Context(), ctx.Args[1])
				if err != nil {
					return "", err
				}
			case "--url":
				if len(ctx.Args) < 2 {
					return "", fmt.Errorf("usage: frame --url <pattern>")
				}
				pattern := ctx.Args[1]
				iframeTarget := findTargetByURLPattern(targets, pattern)
				if iframeTarget == nil {
					return "", fmt.Errorf("no iframe target matches URL pattern %q", pattern)
				}
				if err := ctx.Session.SwitchToFrame(iframeTarget.TargetID); err != nil {
					return "", fmt.Errorf("failed to switch to frame: %w", err)
				}
				ctx.Session.ClearRefs()
				ctx.Session.SetLastSnapshot("")
				return fmt.Sprintf("Switched to iframe: %s (%s)", iframeTarget.Title, iframeTarget.URL), nil
			default:
				// CSS selector or @ref path
				sel, resolveErr := ctx.Session.ResolveRef(targetArg)
				if resolveErr != nil {
					return "", resolveErr
				}
				matchURL, err = resolveIframeURLBySelector(ctx.Session.Context(), sel)
				if err != nil {
					return "", err
				}
			}

			iframeTarget := findTargetByURL(targets, matchURL)
			if iframeTarget == nil {
				return "", fmt.Errorf("no matching iframe target found for URL %q", matchURL)
			}
			if err := ctx.Session.SwitchToFrame(iframeTarget.TargetID); err != nil {
				return "", fmt.Errorf("failed to switch to frame: %w", err)
			}
			ctx.Session.ClearRefs()
			ctx.Session.SetLastSnapshot("")
			return fmt.Sprintf("Switched to iframe: %s (%s)", iframeTarget.Title, iframeTarget.URL), nil
		})

	r.Register("state", CommandDesc{Category: "Meta", Description: "Save or load browser state", Usage: "state save|load <name>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) < 2 {
				return "", fmt.Errorf("usage: state save|load <name>")
			}
			action := ctx.Args[0]
			name := ctx.Args[1]

			if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(name) {
				return "", fmt.Errorf("state name must be alphanumeric (a-z, 0-9, _, -)")
			}

			cfg := config.Resolve(nil)
			stateDir := filepath.Join(cfg.StateDir, "browse-states")
			_ = fs.MkdirSecure(stateDir)
			statePath := filepath.Join(stateDir, name+".json")

			switch action {
			case "save":
				state, err := ctx.BM.SaveState()
				if err != nil {
					return "", fmt.Errorf("save state failed: %w", err)
				}
				saveData := struct {
					Version string `json:"version"`
					SavedAt string `json:"savedAt"`
					Cookies []*network.Cookie `json:"cookies"`
					Pages   []struct {
						URL      string `json:"url"`
						IsActive bool   `json:"isActive"`
					} `json:"pages"`
				}{
					Version: "1",
					SavedAt: time.Now().UTC().Format(time.RFC3339),
					Cookies: state.Cookies,
					Pages:   make([]struct{ URL string `json:"url"`; IsActive bool `json:"isActive"` }, len(state.Pages)),
				}
				for i, p := range state.Pages {
					saveData.Pages[i].URL = p.URL
					saveData.Pages[i].IsActive = p.IsActive
				}
				b, _ := json.MarshalIndent(saveData, "", "  ")
				if err := os.WriteFile(statePath, b, 0600); err != nil {
					return "", fmt.Errorf("write state failed: %w", err)
				}
				return fmt.Sprintf("State saved: %s (%d cookies, %d pages)\n⚠️ Cookies stored in plaintext. Delete when no longer needed.", statePath, len(state.Cookies), len(state.Pages)), nil

			case "load":
				b, err := os.ReadFile(statePath)
				if err != nil {
					return "", fmt.Errorf("state not found: %s", statePath)
				}
				var data struct {
					Version string              `json:"version"`
					SavedAt string              `json:"savedAt"`
					Cookies []*network.Cookie      `json:"cookies"`
					Pages   []struct {
						URL      string `json:"url"`
						IsActive bool   `json:"isActive"`
					} `json:"pages"`
				}
				if err := json.Unmarshal(b, &data); err != nil {
					return "", fmt.Errorf("invalid state file: %w", err)
				}
				if data.Cookies == nil || data.Pages == nil {
					return "", fmt.Errorf("invalid state file: expected cookies and pages arrays")
				}

				// Validate and filter cookies
				var validCookies []*network.Cookie
				for _, c := range data.Cookies {
					if c == nil || c.Name == "" || c.Domain == "" {
						continue
					}
					d := strings.TrimPrefix(c.Domain, ".")
					if d == "localhost" || strings.HasSuffix(d, ".internal") || d == "169.254.169.254" {
						continue
					}
					validCookies = append(validCookies, c)
				}

				// Warn on old state files
				if data.SavedAt != "" {
					if t, err := time.Parse(time.RFC3339, data.SavedAt); err == nil {
						if time.Since(t) > 7*24*time.Hour {
							// Non-fatal warning — just logged, not returned
						}
					}
				}

				// Close existing pages, then restore
				if err := ctx.BM.CloseAllTabs(); err != nil {
					return "", fmt.Errorf("close tabs failed: %w", err)
				}

				// Build restore state (strip loadedHtml/owner for security)
				restoreState := &browser.BrowserState{
					Cookies: validCookies,
					Pages:   make([]browser.PageState, len(data.Pages)),
				}
				for i, p := range data.Pages {
					restoreState.Pages[i] = browser.PageState{
						URL:      p.URL,
						IsActive: p.IsActive,
					}
				}
				if err := ctx.BM.RestoreState(restoreState); err != nil {
					return "", fmt.Errorf("restore state failed: %w", err)
				}
				return fmt.Sprintf("State loaded: %d cookies, %d pages", len(validCookies), len(data.Pages)), nil

			default:
				return "", fmt.Errorf("usage: state save|load <name>")
			}
		})

	r.Register("tab-each", CommandDesc{Category: "Meta", Description: "Run a command on every tab", Usage: "tab-each <command> [args...]"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: tab-each <command> [args...]")
			}
			innerName := Canonicalize(ctx.Args[0])
			innerArgs := ctx.Args[1:]

			// Check that the inner command exists
			if _, ok := r.Get(innerName); !ok {
				return "", fmt.Errorf("unknown command: %s", innerName)
			}

			originalActive := ctx.BM.ActiveTabId()
			tabs := ctx.BM.TabList()

			type resultEntry struct {
				TabID  int    `json:"tabId"`
				URL    string `json:"url"`
				Title  string `json:"title"`
				Status int    `json:"status"`
				Output string `json:"output"`
			}
			var results []resultEntry

			for _, tab := range tabs {
				// Skip internal chrome pages
				if strings.HasPrefix(tab.URL, "chrome://") || strings.HasPrefix(tab.URL, "chrome-extension://") {
					results = append(results, resultEntry{
						TabID:  tab.ID,
						URL:    tab.URL,
						Title:  tab.Title,
						Status: 0,
						Output: "skipped: internal page",
					})
					continue
				}

				// Switch to tab
				if err := ctx.BM.SwitchTab(tab.ID); err != nil {
					results = append(results, resultEntry{
						TabID:  tab.ID,
						URL:    tab.URL,
						Title:  tab.Title,
						Status: 500,
						Output: err.Error(),
					})
					continue
				}

				// Execute inner command
				output, err := r.Execute(ctx.BM, innerName, innerArgs)
				status := 200
				if err != nil {
					status = 500
					output = err.Error()
				}
				results = append(results, resultEntry{
					TabID:  tab.ID,
					URL:    tab.URL,
					Title:  tab.Title,
					Status: status,
					Output: output,
				})
			}

			// Restore original active tab
			_ = ctx.BM.SwitchTab(originalActive)

			out := struct {
				Command string        `json:"command"`
				Args    []string      `json:"args"`
				Total   int           `json:"total"`
				Results []resultEntry `json:"results"`
			}{
				Command: innerName,
				Args:    innerArgs,
				Total:   len(results),
				Results: results,
			}
			b, _ := json.MarshalIndent(out, "", "  ")
			return string(b), nil
		})

	r.Register("archive", CommandDesc{Category: "Meta", Description: "Save page as MHTML archive", Usage: "archive [path]"},
		func(ctx *ExecContext) (string, error) {
			outputPath := filepath.Join(os.TempDir(), fmt.Sprintf("browse-archive-%d.mhtml", time.Now().Unix()))
			if len(ctx.Args) > 0 && !startsWithDash(ctx.Args[0]) {
				outputPath = ctx.Args[0]
			}

			var data string
			if err := chromedp.Run(ctx.Session.Context(), chromedp.ActionFunc(func(c context.Context) error {
				var err error
				data, err = page.CaptureSnapshot().Do(c)
				return err
			})); err != nil {
				return "", fmt.Errorf("archive failed: %w", err)
			}

			if err := os.WriteFile(outputPath, []byte(data), 0644); err != nil {
				return "", fmt.Errorf("write archive failed: %w", err)
			}
			return "Archive saved to " + outputPath, nil
		})

	r.Register("handoff", CommandDesc{Category: "Meta", Description: "Open visible Chrome at current page for user takeover", Usage: "handoff [message]"},
		func(ctx *ExecContext) (string, error) {
			url := ctx.BM.CurrentURL()
			if url == "" || url == "about:blank" {
				return "", fmt.Errorf("no active page to hand off")
			}
			if ctx.BM.GetConnectionMode() == "headed" {
				return "HANDOFF: Already in headed mode at " + url, nil
			}

			// Get auth token from state file for extension bootstrap
			authToken := ""
			serverPort := ctx.Port
			if serverPort > 0 {
				cfg := config.Resolve(nil)
				if data, err := os.ReadFile(cfg.StateFile); err == nil {
					var sf struct {
						Token string `json:"token"`
						Port  int    `json:"port"`
					}
					if err := json.Unmarshal(data, &sf); err == nil {
						authToken = sf.Token
						if serverPort == 0 {
							serverPort = sf.Port
						}
					}
				}
			}

			opts := &browser.LaunchHeadedOptions{}
			if authToken != "" {
				opts.AuthToken = authToken
				opts.ServerPort = serverPort
			}
			if err := ctx.BM.LaunchHeadedWithOptions(url, opts); err != nil {
				return "", fmt.Errorf("handoff failed: %w", err)
			}
			msg := "Handed off to user at " + url
			if len(ctx.Args) > 0 {
				msg += "\nNote: " + strings.Join(ctx.Args, " ")
			}
			return msg, nil
		})

	r.Register("resume", CommandDesc{Category: "Meta", Description: "Resume AI control after handoff"},
		func(ctx *ExecContext) (string, error) {
			if ctx.BM.IsHeaded() {
				if err := ctx.BM.CloseHeaded(); err != nil {
					return "", fmt.Errorf("resume failed: %w", err)
				}
			}
			if ctx.Session != nil {
				ctx.Session.SetInFrame(false)
				ctx.Session.ClearRefs()
				ctx.Session.SetLastSnapshot("")
			}
			// Re-snapshot to give the agent fresh context
			result, err := r.Execute(ctx.BM, "snapshot", []string{"-i"})
			if err != nil {
				return "Resumed AI control (snapshot unavailable: " + err.Error() + ")", nil
			}
			return "Resumed AI control\n" + result, nil
		})

	r.Register("connect", CommandDesc{Category: "Meta", Description: "Connect to headed browser"},
		func(ctx *ExecContext) (string, error) {
			if ctx.BM.GetConnectionMode() == "headed" {
				return "Already in headed mode with extension.", nil
			}
			url := ctx.BM.CurrentURL()
			if url == "" {
				url = "about:blank"
			}
			if err := ctx.BM.LaunchHeaded(url); err != nil {
				return "", fmt.Errorf("connect failed: %w", err)
			}
			return "Connected to headed browser at " + url, nil
		})

	r.Register("disconnect", CommandDesc{Category: "Meta", Description: "Disconnect headed browser"},
		func(ctx *ExecContext) (string, error) {
			if ctx.BM.GetConnectionMode() != "headed" {
				return "Not in headed mode — nothing to disconnect.", nil
			}
			// Save state, close headed, relaunch headless, restore state
			state, err := ctx.BM.SaveState()
			if err != nil {
				state = nil // non-fatal: proceed without state save
			}
			if err := ctx.BM.CloseHeaded(); err != nil {
				return "", fmt.Errorf("disconnect failed: %w", err)
			}
			// Relaunch headless browser if it was closed
			if !ctx.BM.IsHealthy() {
				if err := ctx.BM.Launch(); err != nil {
					return "", fmt.Errorf("disconnect: failed to restart headless browser: %w", err)
				}
				if state != nil {
					if err := ctx.BM.RestoreState(state); err != nil {
						// Non-fatal: at least the browser is running
						_ = err
					}
				}
			}
			return "Disconnected headed browser. Server restarted in headless mode.", nil
		})

	r.Register("focus", CommandDesc{Category: "Meta", Description: "Bring browser window to foreground", Usage: "focus [sel|@ref]"},
		func(ctx *ExecContext) (string, error) {
			if ctx.BM.GetConnectionMode() != "headed" {
				return "", fmt.Errorf("focus requires headed mode. Run 'browse connect' first.")
			}

			var activatedApp string

			// ─── macOS: osascript ───────────────────────────────────
			if runtime.GOOS == "darwin" {
				activatedApp = focusMacOS(ctx.BM.HeadedExecPath())
			}

			// ─── Linux: xdotool ─────────────────────────────────────
			if runtime.GOOS == "linux" {
				activatedApp = focusLinux(ctx.BM.HeadedExecPath())
			}

			// ─── Windows: PowerShell ────────────────────────────────
			if runtime.GOOS == "windows" {
				activatedApp = focusWindows(ctx.BM.HeadedExecPath())
			}

			// Optional: scroll element into view
			if len(ctx.Args) > 0 {
				sel, err := ctx.Session.ResolveRef(ctx.Args[0])
				if err != nil {
					return "", err
				}
				_ = chromedp.Run(ctx.Session.Context(), chromedp.ScrollIntoView(sel))
			}

			if activatedApp != "" {
				return fmt.Sprintf("Browser window activated (%s).", activatedApp), nil
			}
			return "Browser window activated.", nil
		})

	r.Register("ux-audit", CommandDesc{Category: "Meta", Description: "Extract page structure for UX analysis"},
		func(ctx *ExecContext) (string, error) {
			script := `
			(() => {
				const HEADING_CAP = 50, INTERACTIVE_CAP = 200, TEXT_BLOCK_CAP = 50;

				const logoEl = document.querySelector('[class*="logo"], [id*="logo"], header img, [aria-label*="home"], a[href="/"]');
				const siteId = logoEl ? {
					found: true, text: (logoEl.textContent || '').trim().slice(0,100),
					tag: logoEl.tagName, alt: logoEl.alt || null
				} : { found: false, text: null, tag: null, alt: null };

				const h1 = document.querySelector('h1');
				const pageName = h1 ? { found: true, text: h1.textContent?.trim().slice(0,200) || '' } : { found: false, text: null };

				const navItems = [];
				document.querySelectorAll('nav, [role="navigation"]').forEach((nav, i) => {
					if (i >= 5) return;
					navItems.push({ text: (nav.getAttribute('aria-label') || 'nav-'+i).slice(0,50), links: nav.querySelectorAll('a').length });
				});

				const activeNavItems = document.querySelectorAll('nav [aria-current], nav .active, nav .current, [role="navigation"] [aria-current], [role="navigation"] .active, [role="navigation"] .current');
				const youAreHere = Array.from(activeNavItems).slice(0,5).map(el => ({
					text: (el.textContent || '').trim().slice(0,50), tag: el.tagName
				}));

				const searchEl = document.querySelector('input[type="search"], [role="search"], input[name*="search"], input[placeholder*="search" i], input[aria-label*="search" i]');
				const search = { found: !!searchEl };

				const breadcrumbEl = document.querySelector('[aria-label*="breadcrumb" i], .breadcrumb, .breadcrumbs, [class*="breadcrumb"]');
				const breadcrumbs = breadcrumbEl ? {
					found: true, items: Array.from(breadcrumbEl.querySelectorAll('a, span, li')).slice(0,10).map(el => (el.textContent || '').trim().slice(0,30))
				} : { found: false, items: [] };

				const headings = Array.from(document.querySelectorAll('h1,h2,h3,h4,h5,h6')).slice(0, HEADING_CAP).map(h => ({
					tag: h.tagName, text: (h.textContent || '').trim().slice(0,80), size: getComputedStyle(h).fontSize
				}));

				const interactive = Array.from(document.querySelectorAll('a, button, input, select, textarea, [role="button"], [tabindex]')).slice(0, INTERACTIVE_CAP).map(el => {
					const rect = el.getBoundingClientRect();
					return {
						tag: el.tagName, text: (el.textContent || el.placeholder || '').trim().slice(0,50),
						type: el.type || null, role: el.getAttribute('role'),
						w: Math.round(rect.width), h: Math.round(rect.height),
						visible: rect.width > 0 && rect.height > 0
					};
				}).filter(el => el.visible);

				const textBlocks = Array.from(document.querySelectorAll('p, [class*="description"], [class*="intro"], [class*="welcome"], [class*="hero"] p, main p')).slice(0, TEXT_BLOCK_CAP).map(el => ({
					text: (el.textContent || '').trim().slice(0,200),
					wordCount: (el.textContent || '').trim().split(/\s+/).filter(Boolean).length
				}));

				const totalWords = (document.body?.textContent || '').trim().split(/\s+/).filter(Boolean).length;

				return { url: window.location.href, title: document.title, siteId, pageName, navigation: navItems, youAreHere, search, breadcrumbs, headings, interactive, textBlocks, totalWords };
			})()
			`
			var result map[string]interface{}
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(script, &result)); err != nil {
				return "", fmt.Errorf("ux-audit failed: %w", err)
			}
			b, _ := json.MarshalIndent(result, "", "  ")
			return string(b), nil
		})

	r.Register("watch", CommandDesc{Category: "Meta", Description: "Observe user browsing with periodic snapshots", Usage: "watch [stop]"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) > 0 && ctx.Args[0] == "stop" {
				if !ctx.BM.IsWatching() {
					return "Not currently watching.", nil
				}
				result := ctx.BM.StopWatch()
				durationSec := int(result.Duration / 1000)
				lastSnap := "(none)"
				if len(result.Snapshots) > 0 {
					lastSnap = result.Snapshots[len(result.Snapshots)-1]
				}
				return fmt.Sprintf("WATCH STOPPED (%ds, %d snapshots)\n\nLast snapshot:\n%s", durationSec, len(result.Snapshots), lastSnap), nil
			}

			if ctx.BM.IsWatching() {
				return "Already watching. Run 'watch stop' to stop.", nil
			}
			ctx.BM.StartWatch()
			return "WATCHING — observing user browsing. Periodic snapshots every 5s.\nRun 'watch stop' to stop and get summary.", nil
		})
}

// getCleanTextJS is a self-contained script to extract clean page text.
const getCleanTextJS = `
(() => {
	const b = document.body;
	if (!b) return '';
	const c = b.cloneNode(true);
	c.querySelectorAll('script,style,noscript,svg').forEach(e => e.remove());
	return Array.from(c.innerText.split('\n')).map(l => l.trim()).filter(l => l.length > 0).join('\n');
})()
`

// ─── Frame switching helpers ─────────────────────────────

// resolveIframeURLBySelector runs JS on the page to find an iframe matching
// the CSS selector and returns its resolved URL (href preferred, src fallback).
func resolveIframeURLBySelector(ctx context.Context, selector string) (string, error) {
	var result struct {
		Src  string `json:"src"`
		Href string `json:"href"`
	}
	script := fmt.Sprintf(`
(() => {
	const el = document.querySelector(%s);
	if (!el || el.tagName !== 'IFRAME') return {src: '', href: ''};
	let src = el.src || '';
	let href = '';
	try { href = el.contentWindow.location.href; } catch(e) {}
	return {src, href};
})()
`, strconv.Quote(selector))
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &result)); err != nil {
		return "", fmt.Errorf("failed to locate iframe: %w", err)
	}
	if result.Src == "" && result.Href == "" {
		return "", fmt.Errorf("no iframe found matching selector %q", selector)
	}
	// Prefer href (same-origin, most current) over src (attribute, may be relative)
	if result.Href != "" {
		return result.Href, nil
	}
	return result.Src, nil
}

// resolveIframeURLByName resolves an iframe URL by its name attribute.
func resolveIframeURLByName(ctx context.Context, name string) (string, error) {
	script := fmt.Sprintf(`
(() => {
	const name = %s;
	const el = document.querySelector('iframe[name="' + name + '"]');
	if (!el) return {src: '', href: ''};
	let src = el.src || '';
	let href = '';
	try { href = el.contentWindow.location.href; } catch(e) {}
	return {src, href};
})()
`, strconv.Quote(name))
	var result struct {
		Src  string `json:"src"`
		Href string `json:"href"`
	}
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &result)); err != nil {
		return "", fmt.Errorf("failed to locate iframe by name: %w", err)
	}
	if result.Src == "" && result.Href == "" {
		return "", fmt.Errorf("no iframe found with name %q", name)
	}
	if result.Href != "" {
		return result.Href, nil
	}
	return result.Src, nil
}

// findTargetByURL searches the target list for an iframe target whose URL
// exactly or fuzzily matches the given URL.
func findTargetByURL(targets []*target.Info, url string) *target.Info {
	for _, t := range targets {
		if t.Type != "iframe" {
			continue
		}
		if t.URL == url {
			return t
		}
	}
	// Fallback: substring match (target URL may have extra query params)
	for _, t := range targets {
		if t.Type != "iframe" {
			continue
		}
		if strings.Contains(t.URL, url) || strings.Contains(url, t.URL) {
			return t
		}
	}
	return nil
}

// findTargetByURLPattern searches the target list for an iframe target whose
// URL matches the given pattern (regex if valid, otherwise substring).
func findTargetByURLPattern(targets []*target.Info, pattern string) *target.Info {
	if re, err := regexp.Compile(pattern); err == nil {
		for _, t := range targets {
			if t.Type != "iframe" {
				continue
			}
			if re.MatchString(t.URL) {
				return t
			}
		}
		return nil
	}
	// Fallback to substring match
	for _, t := range targets {
		if t.Type != "iframe" {
			continue
		}
		if strings.Contains(t.URL, pattern) {
			return t
		}
	}
	return nil
}

// tokenize splits a string respecting double-quoted segments.
func tokenize(s string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' {
			inQuote = !inQuote
		} else if ch == ' ' && !inQuote {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		} else {
			cur.WriteByte(ch)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// ─── Focus Helpers ────────────────────────────────────────

// focusMacOS brings the browser window to front on macOS using osascript.
// If execPath is known (e.g. "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"),
// it extracts the exact app bundle name for precise activation.
func focusMacOS(execPath string) string {
	// Try exact app name from exec path first
	if execPath != "" {
		if appName := extractMacOSAppName(execPath); appName != "" {
			cmd := exec.Command("osascript", "-e",
				fmt.Sprintf(`tell application "%s" to activate`, appName))
			if err := cmd.Run(); err == nil {
				// Also bring to absolute front via System Events
				_ = exec.Command("osascript", "-e",
					fmt.Sprintf(`tell application "System Events" to set frontmost of process "%s" to true`, appName)).Run()
				return appName
			}
		}
	}

	// Fallback: try common browser names
	appNames := []string{"Google Chrome", "Chromium", "Brave Browser", "Arc", "Microsoft Edge"}
	for _, appName := range appNames {
		cmd := exec.Command("osascript", "-e",
			fmt.Sprintf(`tell application "%s" to activate`, appName))
		if err := cmd.Run(); err == nil {
			_ = exec.Command("osascript", "-e",
				fmt.Sprintf(`tell application "System Events" to set frontmost of process "%s" to true`, appName)).Run()
			return appName
		}
	}
	return ""
}

// extractMacOSAppName extracts the display app name from a macOS bundle path.
// e.g. "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" → "Google Chrome"
func extractMacOSAppName(execPath string) string {
	// Look for "Something.app" in the path
	parts := strings.Split(execPath, string(filepath.Separator))
	for _, p := range parts {
		if strings.HasSuffix(p, ".app") {
			return strings.TrimSuffix(p, ".app")
		}
	}
	return ""
}

// focusLinux brings the browser window to front on Linux using xdotool.
func focusLinux(execPath string) string {
	// Try to find window by class name (from binary name)
	className := ""
	if execPath != "" {
		className = filepath.Base(execPath)
	}
	if className == "" {
		className = "chromium"
	}

	// Try xdotool search + activate
	cmd := exec.Command("xdotool", "search", "--class", className, "activatewindow")
	if err := cmd.Run(); err == nil {
		return className
	}

	// Fallback: try common class names
	for _, c := range []string{"google-chrome", "chromium-browser", "chromium", "brave"} {
		cmd := exec.Command("xdotool", "search", "--class", c, "activatewindow")
		if err := cmd.Run(); err == nil {
			return c
		}
	}
	return ""
}

// focusWindows brings the browser window to front on Windows using PowerShell.
func focusWindows(execPath string) string {
	processName := ""
	if execPath != "" {
		processName = strings.TrimSuffix(filepath.Base(execPath), ".exe")
	}
	if processName == "" {
		processName = "chrome"
	}

	ps := fmt.Sprintf(`
		$proc = Get-Process -Name %q -ErrorAction SilentlyContinue | Select-Object -First 1
		if ($proc) {
			Add-Type @"
				using System;
				using System.Runtime.InteropServices;
				public class Win32 {
					[DllImport("user32.dll")]
					public static extern bool SetForegroundWindow(IntPtr hWnd);
					[DllImport("user32.dll")]
					public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow);
				}
			"@
			Win32::ShowWindowAsync($proc.MainWindowHandle, 1)
			Win32::SetForegroundWindow($proc.MainWindowHandle)
		}
	`, processName)
	cmd := exec.Command("powershell", "-Command", ps)
	if err := cmd.Run(); err == nil {
		return processName
	}

	// Fallback: try common process names
	for _, name := range []string{"chrome", "chromium", "brave", "msedge"} {
		ps := fmt.Sprintf(`
			$proc = Get-Process -Name %q -ErrorAction SilentlyContinue | Select-Object -First 1
			if ($proc) {
				Add-Type @"
					using System;
					using System.Runtime.InteropServices;
					public class Win32 {
						[DllImport("user32.dll")]
						public static extern bool SetForegroundWindow(IntPtr hWnd);
						[DllImport("user32.dll")]
						public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow);
					}
				"@
				Win32::ShowWindowAsync($proc.MainWindowHandle, 1)
				Win32::SetForegroundWindow($proc.MainWindowHandle)
			}
		`, name)
		cmd := exec.Command("powershell", "-Command", ps)
		if err := cmd.Run(); err == nil {
			return name
		}
	}
	return ""
}
