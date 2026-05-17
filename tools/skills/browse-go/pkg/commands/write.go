package commands

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"

	"browse-go/pkg/cookieimport"
	"browse-go/pkg/picker"
	"browse-go/pkg/util"
)

func (r *Registry) registerWrite() {
	// ─── Dialog Control ───────────────────────────────────────
	r.Register("dialog-accept", CommandDesc{Category: "Write", Description: "Auto-accept dialogs", Usage: "dialog-accept [promptText]"},
		func(ctx *ExecContext) (string, error) {
			text := ""
			if len(ctx.Args) > 0 {
				text = strings.Join(ctx.Args, " ")
			}
			ctx.BM.SetDialogAutoAccept(true)
			ctx.BM.SetDialogPromptText(text)
			if text != "" {
				return fmt.Sprintf("Dialogs will be accepted with text: %q", text), nil
			}
			return "Dialogs will be accepted", nil
		})

	r.Register("dialog-dismiss", CommandDesc{Category: "Write", Description: "Auto-dismiss dialogs"},
		func(ctx *ExecContext) (string, error) {
			ctx.BM.SetDialogAutoAccept(false)
			ctx.BM.SetDialogPromptText("")
			return "Dialogs will be dismissed", nil
		})

	// ─── Cookie ───────────────────────────────────────────────
	r.Register("cookie", CommandDesc{Category: "Write", Description: "Set a cookie", Usage: "cookie <name=value>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 || !strings.Contains(ctx.Args[0], "=") {
				return "", fmt.Errorf("usage: cookie <name=value>")
			}
			cookieStr := ctx.Args[0]
			eq := strings.Index(cookieStr, "=")
			name := cookieStr[:eq]
			value := cookieStr[eq+1:]

			pageURL := ctx.BM.CurrentURL()
			if pageURL == "" || pageURL == "about:blank" {
				return "", fmt.Errorf("no active page to set cookie for")
			}
			u, err := url.Parse(pageURL)
			if err != nil {
				return "", fmt.Errorf("invalid page URL: %w", err)
			}

			if err := chromedp.Run(ctx.Session.Context(), chromedp.ActionFunc(func(c context.Context) error {
				return network.SetCookie(name, value).
					WithDomain(u.Hostname()).
					WithPath("/").
					Do(c)
			})); err != nil {
				return "", fmt.Errorf("set cookie failed: %w", err)
			}
			return fmt.Sprintf("Cookie set: %s=****", name), nil
		})

	// ─── Cookie Import ────────────────────────────────────────
	r.Register("cookie-import", CommandDesc{Category: "Write", Description: "Import cookies from JSON file", Usage: "cookie-import <json-file>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: cookie-import <json-file>")
			}
			filePath := ctx.Args[0]

			// Resolve to absolute path
			resolved, err := filepath.Abs(filePath)
			if err != nil {
				return "", fmt.Errorf("invalid path: %w", err)
			}
			// Check file exists
			if _, err := os.Stat(resolved); err != nil {
				return "", fmt.Errorf("file not found: %s", filePath)
			}

			// Read and parse JSON
			raw, err := os.ReadFile(resolved)
			if err != nil {
				return "", fmt.Errorf("read file failed: %w", err)
			}
			var cookies []map[string]interface{}
			if err := json.Unmarshal(raw, &cookies); err != nil {
				return "", fmt.Errorf("invalid JSON: %w", err)
			}
			if len(cookies) == 0 {
				return "", fmt.Errorf("cookie file is empty")
			}

			// Get current page domain for auto-fill and validation
			pageURL := ctx.BM.CurrentURL()
			if pageURL == "" || pageURL == "about:blank" {
				return "", fmt.Errorf("no active page to import cookies for")
			}
			u, err := url.Parse(pageURL)
			if err != nil {
				return "", fmt.Errorf("invalid page URL: %w", err)
			}
			defaultDomain := u.Hostname()

			// Validate and set each cookie
			var imported int
			var importedDomains []string
			for _, c := range cookies {
				name, _ := c["name"].(string)
				value, _ := c["value"].(string)
				if name == "" || value == "" {
					continue
				}

				domain := defaultDomain
				if d, ok := c["domain"].(string); ok && d != "" {
					domain = d
					// Validate domain matches current page
					cookieDomain := domain
					if strings.HasPrefix(cookieDomain, ".") {
						cookieDomain = cookieDomain[1:]
					}
					if cookieDomain != defaultDomain && !strings.HasSuffix(defaultDomain, "."+cookieDomain) {
						return "", fmt.Errorf("cookie domain %q does not match current page domain %q", domain, defaultDomain)
					}
				}

				pathVal := "/"
				if p, ok := c["path"].(string); ok && p != "" {
					pathVal = p
				}

				var expires *cdp.TimeSinceEpoch
				if exp, ok := c["expires"].(float64); ok && exp > 0 {
					tse := cdp.TimeSinceEpoch(time.Unix(int64(exp), 0))
					expires = &tse
				}

				httpOnly := false
				if v, ok := c["httpOnly"].(bool); ok {
					httpOnly = v
				}

				secure := false
				if v, ok := c["secure"].(bool); ok {
					secure = v
				}

				if err := chromedp.Run(ctx.Session.Context(), chromedp.ActionFunc(func(c context.Context) error {
					return network.SetCookie(name, value).
						WithDomain(domain).
						WithPath(pathVal).
						WithExpires(expires).
						WithHTTPOnly(httpOnly).
						WithSecure(secure).
						Do(c)
				})); err != nil {
					return "", fmt.Errorf("set cookie %s failed: %w", name, err)
				}
				imported++
				importedDomains = append(importedDomains, domain)
			}

			return fmt.Sprintf("Loaded %d cookies from %s (%d domains)", imported, filePath, len(uniqueStrings(importedDomains))), nil
		})

	// ─── Cookie Import from Browser ───────────────────────────
	r.Register("cookie-import-browser", CommandDesc{Category: "Write", Description: "Import cookies from installed browser", Usage: "cookie-import-browser [browser] --domain <domain> | --all [--profile <profile>]"},
		func(ctx *ExecContext) (string, error) {
			return handleCookieImportBrowser(ctx)
		})

	// ─── Header ───────────────────────────────────────────────
	r.Register("header", CommandDesc{Category: "Write", Description: "Set extra HTTP header", Usage: "header <name:value>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 || !strings.Contains(ctx.Args[0], ":") {
				return "", fmt.Errorf("usage: header <name:value>")
			}
			headerStr := ctx.Args[0]
			sep := strings.Index(headerStr, ":")
			name := strings.TrimSpace(headerStr[:sep])
			value := strings.TrimSpace(headerStr[sep+1:])
			if name == "" {
				return "", fmt.Errorf("header name cannot be empty")
			}

			if err := ctx.BM.SetExtraHeader(name, value); err != nil {
				return "", fmt.Errorf("set header failed: %w", err)
			}

			sensitive := []string{"authorization", "cookie", "set-cookie", "x-api-key", "x-auth-token"}
			displayValue := value
			for _, s := range sensitive {
				if strings.EqualFold(name, s) {
					displayValue = "****"
					break
				}
			}
			return fmt.Sprintf("Header set: %s: %s", name, displayValue), nil
		})

	// ─── Style ────────────────────────────────────────────────
	r.Register("style", CommandDesc{Category: "Write", Description: "Modify element style", Usage: "style <sel|@ref> <property> <value> | style --undo [N]"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: style <sel|@ref> <property> <value> | style --undo [N]")
			}

			// Undo mode
			if ctx.Args[0] == "--undo" {
				return undoStyle(ctx)
			}

			if len(ctx.Args) < 3 {
				return "", fmt.Errorf("usage: style <sel|@ref> <property> <value>")
			}

			sel, err := ctx.Session.ResolveRef(ctx.Args[0])
			if err != nil {
				return "", err
			}
			property := ctx.Args[1]
			value := strings.Join(ctx.Args[2:], " ")

			// Validate property name
			if !isValidCSSProperty(property) {
				return "", fmt.Errorf("invalid CSS property name: %s", property)
			}

			// Validate value — block dangerous patterns
			dangerous := []string{"url(", "expression(", "@import", "javascript:", "data:"}
			lowerValue := strings.ToLower(value)
			for _, d := range dangerous {
				if strings.Contains(lowerValue, d) {
					return "", fmt.Errorf("CSS value rejected: contains potentially dangerous pattern")
				}
			}

			script := fmt.Sprintf(`
				(() => {
					const el = document.querySelector(%s);
					if (!el) return {ok: false, old: ''};
					const old = window.getComputedStyle(el).getPropertyValue(%s) || '';
					el.style.setProperty(%s, %s);
					return {ok: true, old: old};
				})()
			`, strconvQuote(sel), strconvQuote(property), strconvQuote(property), strconvQuote(value))

			var result struct {
				Ok  bool   `json:"ok"`
				Old string `json:"old"`
			}
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(script, &result)); err != nil {
				return "", fmt.Errorf("style failed: %w", err)
			}
			if !result.Ok {
				return "", fmt.Errorf("element not found: %s", sel)
			}

			oldDisplay := result.Old
			if oldDisplay == "" {
				oldDisplay = "(none)"
			}
			return fmt.Sprintf("Style modified: %s { %s: %s → %s }", sel, property, oldDisplay, value), nil
		})

	// ─── Cleanup ──────────────────────────────────────────────
	r.Register("cleanup", CommandDesc{Category: "Write", Description: "Remove ads, cookie banners, social widgets, etc.", Usage: "cleanup [--ads] [--cookies] [--sticky] [--social] [--overlays] [--clutter] [--all]"},
		func(ctx *ExecContext) (string, error) {
			doAds, doCookies, doSticky, doSocial, doOverlays, doClutter, doAll := parseCleanupFlags(ctx.Args)
			if doAll {
				doAds, doCookies, doSticky, doSocial, doOverlays, doClutter = true, true, true, true, true, true
			}

			var result struct {
				Removed     []string `json:"removed"`
				ScrollFixed int      `json:"scrollFixed"`
				AdLabels    int      `json:"adLabels"`
				Collapsed   int      `json:"collapsed"`
			}
			script := fmt.Sprintf(`
				(() => {
					const doAds = %v, doCookies = %v, doSticky = %v, doSocial = %v, doOverlays = %v, doClutter = %v;
					const removed = [];
					const AD_SELECTORS = [
						'[id*="ad"]', '[class*="ad"]', '[id*="banner"]', '[class*="banner"]',
						'[id*="sponsor"]', '[class*="sponsor"]', '[data-ad]',
						'.advertisement', '.promoted-content', '.taboola', '.outbrain',
						'[aria-label*="advertisement" i]', '[aria-label*="sponsored" i]'
					];
					const COOKIE_SELECTORS = [
						'[id*="cookie"]', '[class*="cookie"]', '[id*="consent"]', '[class*="consent"]',
						'[id*="gdpr"]', '[class*="gdpr"]', '[aria-label*="cookie" i]'
					];
					const SOCIAL_SELECTORS = [
						'[class*="facebook"]', '[class*="twitter"]', '[class*="linkedin"]',
						'[class*="share"]', '[class*="social"]', '.fb-like', '.twitter-share-button'
					];
					const OVERLAY_SELECTORS = [
						'[role="dialog"]', '.modal', '.popup', '.overlay', '[class*="modal"]',
						'[class*="popup"]', '[class*="overlay"]', '[id*="modal"]', '[id*="popup"]'
					];
					const CLUTTER_SELECTORS = [
						'[class*="newsletter"]', '[class*="subscribe"]', '[class*="related"]',
						'[class*="recommended"]', '[class*="trending"]', '.sidebar:not(:has(article))'
					];

					const selectors = [];
					if (doAds) selectors.push(...AD_SELECTORS);
					if (doCookies) selectors.push(...COOKIE_SELECTORS);
					if (doSocial) selectors.push(...SOCIAL_SELECTORS);
					if (doOverlays) selectors.push(...OVERLAY_SELECTORS);
					if (doClutter) selectors.push(...CLUTTER_SELECTORS);

					let hiddenCount = 0;
					for (const sel of selectors) {
						try {
							document.querySelectorAll(sel).forEach(el => {
								el.style.setProperty('display', 'none', 'important');
								hiddenCount++;
							});
						} catch (e) {
							// Invalid selector — skip
						}
					}
					if (hiddenCount > 0) {
						if (doAds) removed.push('ads');
						if (doCookies) removed.push('cookie banners');
						if (doSocial) removed.push('social widgets');
						if (doOverlays) removed.push('overlays/popups');
						if (doClutter) removed.push('clutter');
					}

					// Sticky/fixed elements
					let stickyCount = 0;
					if (doSticky) {
						const stickyEls = [];
						document.querySelectorAll('*').forEach(el => {
							const style = getComputedStyle(el);
							if (style.position === 'fixed' || style.position === 'sticky') {
								const rect = el.getBoundingClientRect();
								stickyEls.push({el, top: rect.top, width: rect.width, height: rect.height});
							}
						});
						stickyEls.sort((a, b) => a.top - b.top);
						let preservedTopNav = false;
						const vw = window.innerWidth;
						for (const {el, top, width, height} of stickyEls) {
							const tag = el.tagName.toLowerCase();
							if (tag === 'nav' || tag === 'header') continue;
							if (el.getAttribute('role') === 'navigation') continue;
							if (!preservedTopNav && top <= 50 && width > vw * 0.8 && height < 120) {
								preservedTopNav = true;
								continue;
							}
							el.style.setProperty('display', 'none', 'important');
							stickyCount++;
						}
						if (stickyCount > 0) removed.push(stickyCount + ' sticky/fixed elements');
					}

					// Unlock scroll
					let scrollFixed = 0;
					['body', 'html'].forEach(tag => {
						const el = document.querySelector(tag);
						if (!el) return;
						const style = getComputedStyle(el);
						if (style.overflow === 'hidden' || style.position === 'fixed') {
							el.style.setProperty('overflow', 'auto', 'important');
							el.style.setProperty('position', 'static', 'important');
							scrollFixed++;
						}
					});
					// Remove blur filters (paywall technique)
					document.querySelectorAll('[style*="blur"]').forEach(el => {
						el.style.setProperty('filter', 'none', 'important');
						scrollFixed++;
					});

					// Ad label text detection
					let adLabels = 0;
					const adPatterns = [/advertisement/i, /article continues/i, /paid content/i, /sponsored/i];
					document.querySelectorAll('div, span, p, figcaption, label').forEach(el => {
						const text = el.textContent.trim();
						if (adPatterns.some(p => p.test(text)) && text.length < 100) {
							el.style.setProperty('display', 'none', 'important');
							adLabels++;
						}
					});

					// Collapse empty ad placeholders
					let collapsed = 0;
					document.querySelectorAll('div[class*="ad"], aside[class*="ad"], div[class*="sidebar"]').forEach(el => {
						const rect = el.getBoundingClientRect();
						const textLen = el.textContent.trim().length;
						const hasImg = el.querySelector('img') !== null;
						const links = el.querySelectorAll('a').length;
						if (rect.height > 50 && textLen < 20 && !hasImg && links < 2) {
							el.style.setProperty('display', 'none', 'important');
							collapsed++;
						}
					});

					return {removed, scrollFixed, adLabels, collapsed};
				})()
			`, doAds, doCookies, doSticky, doSocial, doOverlays, doClutter)

			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(script, &result)); err != nil {
				return "", fmt.Errorf("cleanup failed: %w", err)
			}

			var parts []string
			parts = append(parts, result.Removed...)
			if result.ScrollFixed > 0 {
				parts = append(parts, "scroll unlocked")
			}
			if result.AdLabels > 0 {
				parts = append(parts, fmt.Sprintf("%d ad labels", result.AdLabels))
			}
			if result.Collapsed > 0 {
				parts = append(parts, fmt.Sprintf("%d empty placeholders", result.Collapsed))
			}
			if len(parts) == 0 {
				return "No clutter elements found to remove.", nil
			}
			return "Cleaned up: " + strings.Join(parts, ", "), nil
		})

	// ─── Upload ───────────────────────────────────────────────
	r.Register("upload", CommandDesc{Category: "Write", Description: "Upload file(s) to file input", Usage: "upload <sel|@ref> <file1> [file2...]"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) < 2 {
				return "", fmt.Errorf("usage: upload <sel|@ref> <file1> [file2...]")
			}
			sel, err := ctx.Session.ResolveRef(ctx.Args[0])
			if err != nil {
				return "", err
			}
			filePaths := ctx.Args[1:]

			// Validate files exist and paths are safe
			for _, fp := range filePaths {
				if _, err := os.Stat(fp); err != nil {
					return "", fmt.Errorf("file not found: %s", fp)
				}
				abs, err := filepath.Abs(fp)
				if err != nil {
					return "", fmt.Errorf("invalid path: %s", fp)
				}
				// Block path traversal
				if strings.Contains(filepath.Clean(abs), "..") {
					return "", fmt.Errorf("path traversal not allowed: %s", fp)
				}
			}

			// Resolve to absolute paths
			absPaths := make([]string, len(filePaths))
			for i, fp := range filePaths {
				abs, _ := filepath.Abs(fp)
				absPaths[i] = abs
			}

			// Use chromedp SetUploadFiles for proper multi-file support
			if err := chromedp.Run(ctx.Session.Context(), chromedp.SetUploadFiles(sel, absPaths)); err != nil {
				return "", fmt.Errorf("upload failed: %w", err)
			}

			var infos []string
			for _, fp := range filePaths {
				st, _ := os.Stat(fp)
				size := int64(0)
				if st != nil {
					size = st.Size()
				}
				infos = append(infos, fmt.Sprintf("%s (%dB)", filepath.Base(fp), size))
			}
			return "Uploaded: " + strings.Join(infos, ", "), nil
		})

	// ─── Download ─────────────────────────────────────────────
	r.Register("download", CommandDesc{Category: "Write", Description: "Download a URL or media element", Usage: "download <url|@ref> [path] [--base64] [--navigate]"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: download <url|@ref> [path] [--base64] [--navigate]")
			}
			isBase64 := false
			isNavigate := false
			filtered := make([]string, 0, len(ctx.Args))
			for _, a := range ctx.Args {
				switch a {
				case "--base64":
					isBase64 = true
				case "--navigate":
					isNavigate = true
				default:
					filtered = append(filtered, a)
				}
			}

			urlStr := filtered[0]
			outputPath := ""
			if len(filtered) > 1 {
				outputPath = filtered[1]
			}

			// ─── Browser-triggered download via navigation ─────────
			if isNavigate {
				return downloadViaNavigation(ctx, urlStr, outputPath, isBase64)
			}

			// Resolve @ref to element src
			if strings.HasPrefix(urlStr, "@") {
				sel, err := ctx.Session.ResolveRef(urlStr)
				if err != nil {
					return "", err
				}
				script := fmt.Sprintf(`
					(() => {
						const el = document.querySelector(%s);
						if (!el) return '';
						const tag = el.tagName.toLowerCase();
						if (tag === 'img') return el.currentSrc || el.src || el.dataset.src || '';
						if (tag === 'video' || tag === 'audio') return el.currentSrc || el.src || '';
						return el.getAttribute('src') || '';
					})()
				`, strconvQuote(sel))
				var src string
				if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(script, &src)); err != nil {
					return "", fmt.Errorf("resolve ref failed: %w", err)
				}
				if src == "" {
					return "", fmt.Errorf("could not extract URL from %s", urlStr)
				}
				urlStr = src
			}

			// HLS/DASH guard
			if strings.Contains(urlStr, ".m3u8") || strings.Contains(urlStr, ".mpd") {
				return "", fmt.Errorf("HLS/DASH stream detected; use yt-dlp or ffmpeg")
			}

			// Validate URL
			if _, err := util.ValidateNavigationURL(urlStr); err != nil {
				return "", err
			}

			// Blob URL strategy
			if strings.HasPrefix(urlStr, "blob:") {
				script := fmt.Sprintf(`
					(async () => {
						try {
							const resp = await fetch(%s);
							const blob = await resp.blob();
							if (blob.size > 100 * 1024 * 1024) return {ok: false, error: 'Blob too large (>100MB)'};
							return new Promise((resolve, reject) => {
								const reader = new FileReader();
								reader.onloadend = () => resolve({ok: true, contentType: blob.type, dataUrl: reader.result});
								reader.onerror = () => reject('Failed to read blob');
								reader.readAsDataURL(blob);
							});
						} catch (err) {
							return {ok: false, error: err.message || 'unknown'};
						}
					})()
				`, strconvQuote(urlStr))
				var result struct {
					OK          bool   `json:"ok"`
					Error       string `json:"error"`
					ContentType string `json:"contentType"`
					DataURL     string `json:"dataUrl"`
				}
				if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(script, &result)); err != nil {
					return "", fmt.Errorf("blob download failed: %w", err)
				}
				if !result.OK {
					return "", fmt.Errorf("blob download failed: %s", result.Error)
				}
				re := regexp.MustCompile(`^data:([^;]+);base64,(.+)$`)
				m := re.FindStringSubmatch(result.DataURL)
				if m == nil {
					return "", fmt.Errorf("failed to decode blob data")
				}
				data, _ := base64.StdEncoding.DecodeString(m[2])
				if isBase64 {
					return fmt.Sprintf("data:%s;base64,%s", m[1], m[2]), nil
				}
				dest := outputPath
				if dest == "" {
					dest = filepath.Join(os.TempDir(), fmt.Sprintf("browse-download-%d%s", time.Now().Unix(), mimeToExt(m[1])))
				}
				if err := validateOutputPath(dest); err != nil {
					return "", err
				}
				_ = os.WriteFile(dest, data, 0644)
				return fmt.Sprintf("Downloaded: %s (%dKB, %s)", dest, len(data)/1024, m[1]), nil
			}

			// Standard strategy: fetch via browser
			contentType, data, err := fetchViaBrowser(ctx.Session.Context(), urlStr)
			if err != nil {
				return "", err
			}
			if len(data) > 200*1024*1024 {
				return "", fmt.Errorf("file too large (>200MB)")
			}

			if isBase64 {
				if len(data) > 10*1024*1024 {
					return "", fmt.Errorf("file too large for --base64 (>10MB)")
				}
				return fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data)), nil
			}

			ext := mimeToExt(contentType)
			dest := outputPath
			if dest == "" {
				dest = filepath.Join(os.TempDir(), fmt.Sprintf("browse-download-%d%s", time.Now().Unix(), ext))
			}
			if err := validateOutputPath(dest); err != nil {
				return "", err
			}
			if err := os.WriteFile(dest, data, 0644); err != nil {
				return "", fmt.Errorf("write failed: %w", err)
			}
			return fmt.Sprintf("Downloaded: %s (%dKB, %s)", dest, len(data)/1024, contentType), nil
		})

	// ─── Scrape ───────────────────────────────────────────────
	r.Register("scrape", CommandDesc{Category: "Write", Description: "Bulk-download media from page", Usage: "scrape <images|videos|media> [--selector sel] [--dir path] [--limit N]"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: scrape <images|videos|media> [--selector sel] [--dir path] [--limit N]")
			}
			mediaType := ctx.Args[0]
			if mediaType != "images" && mediaType != "videos" && mediaType != "media" {
				return "", fmt.Errorf("invalid type: %s. Use: images, videos, or media", mediaType)
			}

			selector := ""
			dir := filepath.Join(os.TempDir(), fmt.Sprintf("browse-scrape-%d", time.Now().Unix()))
			limit := 50
			for i := 1; i < len(ctx.Args); i++ {
				switch ctx.Args[i] {
				case "--selector":
					if i+1 < len(ctx.Args) {
						selector = ctx.Args[i+1]
						i++
					}
				case "--dir":
					if i+1 < len(ctx.Args) {
						dir = ctx.Args[i+1]
						i++
					}
				case "--limit":
					if i+1 < len(ctx.Args) {
						l, _ := fmt.Sscanf(ctx.Args[i+1], "%d", &limit)
						if l == 1 && limit > 0 {
							if limit > 200 {
								limit = 200
							}
						} else {
							limit = 50
						}
						i++
					}
				}
			}

			if err := validateOutputPath(dir); err != nil {
				return "", err
			}
			_ = os.MkdirAll(dir, 0755)

			// Extract media URLs via JS
			filter := ""
			if mediaType == "images" {
				filter = "images"
			} else if mediaType == "videos" {
				filter = "videos"
			}
			script := fmt.Sprintf(`
				(() => {
					const root = %s ? document.querySelector(%s) || document : document;
					const images = [], videos = [], bgImages = [];
					if (!%q || %q === 'images') {
						root.querySelectorAll('img').forEach(el => {
							images.push(el.currentSrc || el.src || el.dataset.src || '');
						});
						const allEls = root.querySelectorAll('*');
						for (let i = 0; i < Math.min(allEls.length, 500); i++) {
							const style = getComputedStyle(allEls[i]);
							const bg = style.backgroundImage;
							if (bg && bg !== 'none') {
								const m = bg.match(/url\(["']?([^"')]+)["']?\)/);
								if (m && m[1] && !m[1].startsWith('data:')) bgImages.push(m[1]);
							}
						}
					}
					if (!%q || %q === 'videos') {
						root.querySelectorAll('video').forEach(el => {
							videos.push(el.currentSrc || el.src || '');
						});
					}
					return {images, videos, bgImages};
				})()
			`, strconvQuote(selector), strconvQuote(selector), filter, filter, filter, filter)

			var extracted struct {
				Images     []string `json:"images"`
				Videos     []string `json:"videos"`
				BgImages   []string `json:"bgImages"`
			}
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(script, &extracted)); err != nil {
				return "", fmt.Errorf("media extraction failed: %w", err)
			}

			// Collect unique URLs
			var urls []struct{ URL, Type string }
			seen := make(map[string]bool)
			for _, u := range extracted.Images {
				if u != "" && !seen[u] && !strings.HasPrefix(u, "data:") {
					seen[u] = true; urls = append(urls, struct{URL, Type string}{u, "image"})
				}
			}
			for _, u := range extracted.Videos {
				if u != "" && !seen[u] && !strings.HasPrefix(u, "blob:") {
					seen[u] = true; urls = append(urls, struct{URL, Type string}{u, "video"})
				}
			}
			for _, u := range extracted.BgImages {
				if u != "" && !seen[u] {
					seen[u] = true; urls = append(urls, struct{URL, Type string}{u, "image"})
				}
			}

			toDownload := urls
			if len(toDownload) > limit {
				toDownload = toDownload[:limit]
			}

			pageURL := ctx.BM.CurrentURL()
			manifest := struct {
				URL       string `json:"url"`
				ScrapedAt string `json:"scraped_at"`
				Files     []map[string]interface{} `json:"files"`
				TotalSize int    `json:"total_size"`
				Succeeded int    `json:"succeeded"`
				Failed    int    `json:"failed"`
			}{
				URL:       pageURL,
				ScrapedAt: time.Now().UTC().Format(time.RFC3339),
				Files:     []map[string]interface{}{},
			}
			var lines []string

			for i, item := range toDownload {
				tryURL := item.URL
				if _, err := util.ValidateNavigationURL(tryURL); err != nil {
					manifest.Files = append(manifest.Files, map[string]interface{}{
						"src": tryURL, "error": err.Error(),
					})
					manifest.Failed++
					lines = append(lines, fmt.Sprintf("  [%d/%d] FAILED: %s", i+1, len(toDownload), err.Error()))
					continue
				}
				ct, data, err := fetchViaBrowser(ctx.Session.Context(), tryURL)
				if err != nil {
					manifest.Files = append(manifest.Files, map[string]interface{}{
						"src": tryURL, "error": err.Error(),
					})
					manifest.Failed++
					lines = append(lines, fmt.Sprintf("  [%d/%d] FAILED: %s", i+1, len(toDownload), err.Error()))
					continue
				}
				ext := mimeToExt(ct)
				filename := fmt.Sprintf("%s-%03d%s", item.Type, i+1, ext)
				filePath := filepath.Join(dir, filename)
				_ = os.WriteFile(filePath, data, 0644)
				manifest.Files = append(manifest.Files, map[string]interface{}{
					"path": filename, "src": tryURL, "size": len(data), "type": ct,
				})
				manifest.TotalSize += len(data)
				manifest.Succeeded++
				lines = append(lines, fmt.Sprintf("  [%d/%d] %s (%dKB)", i+1, len(toDownload), filename, len(data)/1024))
				if i < len(toDownload)-1 {
					chromedp.Sleep(100 * time.Millisecond)
				}
			}

			mb, _ := json.MarshalIndent(manifest, "", "  ")
			_ = os.WriteFile(filepath.Join(dir, "manifest.json"), mb, 0644)
			return fmt.Sprintf("Scraped %d items to %s/\n%s\n\nSummary: %d succeeded, %d failed, %dKB total",
				len(toDownload), dir, strings.Join(lines, "\n"), manifest.Succeeded, manifest.Failed, manifest.TotalSize/1024), nil
		})
}

// ─── Style Undo Helpers ───────────────────────────────────

type styleMod struct {
	Selector string
	Property string
	OldValue string
	NewValue string
}

func undoStyle(ctx *ExecContext) (string, error) {
	count := 1
	if len(ctx.Args) > 1 {
		n, err := strconv.Atoi(ctx.Args[1])
		if err != nil || n < 1 {
			return "", fmt.Errorf("usage: style --undo [N] — N must be a positive integer")
		}
		count = n
	}

	var undone []string
	for i := 0; i < count; i++ {
		mod := ctx.Session.PopStyleMod()
		if mod == nil {
			if i == 0 {
				return "", fmt.Errorf("no style modifications to undo")
			}
			break
		}

		script := fmt.Sprintf(`
			(() => {
				const el = document.querySelector(%s);
				if (!el) return false;
				if (%s === '') {
					el.style.removeProperty(%s);
				} else {
					el.style.setProperty(%s, %s);
				}
				return true;
			})()
		`, strconvQuote(mod.Selector), strconvQuote(mod.OldValue), strconvQuote(mod.Property), strconvQuote(mod.Property), strconvQuote(mod.OldValue))

		var ok bool
		if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(script, &ok)); err != nil {
			return "", fmt.Errorf("undo failed for %s { %s }: %w", mod.Selector, mod.Property, err)
		}
		if !ok {
			return "", fmt.Errorf("element not found during undo: %s", mod.Selector)
		}

		oldDisplay := mod.OldValue
		if oldDisplay == "" {
			oldDisplay = "(none)"
		}
		undone = append(undone, fmt.Sprintf("%s { %s: %s → %s }", mod.Selector, mod.Property, mod.NewValue, oldDisplay))
	}

	return fmt.Sprintf("Style undone (%d): %s", len(undone), strings.Join(undone, "; ")), nil
}

func isValidCSSProperty(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '-') {
			return false
		}
	}
	return s != ""
}

func parseCleanupFlags(args []string) (ads, cookies, sticky, social, overlays, clutter, all bool) {
	if len(args) == 0 {
		return false, false, false, false, false, false, true
	}
	for _, a := range args {
		switch a {
		case "--ads":
			ads = true
		case "--cookies":
			cookies = true
		case "--sticky":
			sticky = true
		case "--social":
			social = true
		case "--overlays":
			overlays = true
		case "--clutter":
			clutter = true
		case "--all":
			all = true
		}
	}
	return
}

// ─── Download Helpers ─────────────────────────────────────

// fetchViaBrowser downloads a URL by running fetch() inside the browser context.
// This preserves cookies and extra headers. Returns (contentType, data, error).
func fetchViaBrowser(ctx context.Context, url string) (string, []byte, error) {
	script := fmt.Sprintf(`
		(async () => {
			const resp = await fetch(%s);
			if (!resp.ok) return {ok: false, status: resp.status, statusText: resp.statusText};
			const blob = await resp.blob();
			if (blob.size > 50 * 1024 * 1024) return {ok: false, error: 'File too large (>50MB)'};
			return new Promise((resolve, reject) => {
				const reader = new FileReader();
				reader.onloadend = () => resolve({ok: true, contentType: blob.type || 'application/octet-stream', dataUrl: reader.result});
				reader.onerror = () => reject('Failed to read blob');
				reader.readAsDataURL(blob);
			});
		})()
	`, strconvQuote(url))

	var result struct {
		OK          bool   `json:"ok"`
		Status      int    `json:"status"`
		StatusText  string `json:"statusText"`
		Error       string `json:"error"`
		ContentType string `json:"contentType"`
		DataURL     string `json:"dataUrl"`
	}
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &result)); err != nil {
		return "", nil, fmt.Errorf("fetch failed: %w", err)
	}
	if !result.OK {
		if result.Error != "" {
			return "", nil, fmt.Errorf("download failed: %s", result.Error)
		}
		return "", nil, fmt.Errorf("download failed: HTTP %d %s", result.Status, result.StatusText)
	}

	// Parse data:<mime>;base64,<data>
	re := regexp.MustCompile(`^data:([^;]+);base64,(.+)$`)
	m := re.FindStringSubmatch(result.DataURL)
	if m == nil {
		return "", nil, fmt.Errorf("failed to decode download data")
	}
	data, err := base64.StdEncoding.DecodeString(m[2])
	if err != nil {
		return "", nil, fmt.Errorf("base64 decode failed: %w", err)
	}
	return m[1], data, nil
}

// downloadViaNavigation triggers a browser download by navigating to a URL
// that serves a download (e.g. Content-Disposition: attachment).
func downloadViaNavigation(ctx *ExecContext, urlStr, outputPath string, isBase64 bool) (string, error) {
	if _, err := util.ValidateNavigationURL(urlStr); err != nil {
		return "", err
	}

	downloadDir := os.TempDir()
	if outputPath != "" {
		downloadDir = filepath.Dir(outputPath)
	}
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return "", fmt.Errorf("cannot create download dir: %w", err)
	}

	// Record existing files to detect new downloads
	existing := make(map[string]bool)
	entries, _ := os.ReadDir(downloadDir)
	for _, e := range entries {
		existing[e.Name()] = true
	}

	// Set browser download behavior via browser target
	tabCtx := ctx.Session.Context()
	if err := chromedp.Run(tabCtx, chromedp.ActionFunc(func(c context.Context) error {
		cc := chromedp.FromContext(c)
		if cc == nil || cc.Browser == nil {
			return fmt.Errorf("no browser context")
		}
		return browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).
			WithDownloadPath(downloadDir).
			Do(cdp.WithExecutor(c, cc.Browser))
	})); err != nil {
		return "", fmt.Errorf("set download behavior failed: %w", err)
	}

	// Navigate to trigger the download
	if err := chromedp.Run(tabCtx, chromedp.Navigate(urlStr)); err != nil {
		return "", fmt.Errorf("navigation failed: %w", err)
	}

	// Poll for downloaded file (max 15s)
	var downloadedFile string
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)
		entries, _ := os.ReadDir(downloadDir)
		for _, e := range entries {
			name := e.Name()
			if existing[name] {
				continue
			}
			if strings.HasSuffix(name, ".crdownload") {
				continue
			}
			downloadedFile = filepath.Join(downloadDir, name)
			break
		}
		if downloadedFile != "" {
			break
		}
	}

	if downloadedFile == "" {
		return "", fmt.Errorf("download did not complete within 15s timeout")
	}

	// Move/rename if output path specified
	if outputPath != "" && outputPath != downloadedFile {
		if err := os.Rename(downloadedFile, outputPath); err != nil {
			return "", fmt.Errorf("rename failed: %w", err)
		}
		downloadedFile = outputPath
	}

	data, err := os.ReadFile(downloadedFile)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}

	if isBase64 {
		if len(data) > 10*1024*1024 {
			return "", fmt.Errorf("file too large for --base64 (>10MB)")
		}
		return fmt.Sprintf("data:application/octet-stream;base64,%s", base64.StdEncoding.EncodeToString(data)), nil
	}

	return fmt.Sprintf("Downloaded: %s (%dKB)", downloadedFile, len(data)/1024), nil
}

// validateOutputPath ensures a path is within safe directories.
func validateOutputPath(p string) error {
	abs, err := filepath.Abs(p)
	if err != nil {
		return fmt.Errorf("invalid path: %s", p)
	}
	tmp := os.TempDir()
	cwd, _ := os.Getwd()
	safe := false
	for _, dir := range []string{tmp, cwd} {
		if dir != "" && strings.HasPrefix(abs, dir) {
			safe = true
			break
		}
	}
	if !safe {
		return fmt.Errorf("output path must be within temp dir or project root")
	}
	return nil
}

// mimeToExt maps common MIME types to file extensions.
func mimeToExt(mime string) string {
	m := map[string]string{
		"image/jpeg": ".jpg", "image/png": ".png", "image/gif": ".gif", "image/webp": ".webp",
		"video/mp4": ".mp4", "video/webm": ".webm",
		"audio/mpeg": ".mp3", "audio/wav": ".wav",
		"application/pdf": ".pdf", "application/zip": ".zip",
		"text/plain": ".txt", "text/html": ".html",
		"application/json": ".json",
	}
	if ext, ok := m[strings.ToLower(mime)]; ok {
		return ext
	}
	return ".bin"
}

// uniqueStrings returns deduplicated strings preserving order.
func uniqueStrings(ss []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// handleCookieImportBrowser implements the cookie-import-browser command.
// Modes:
//   1. Direct: cookie-import-browser <browser> --domain <domain> [--profile <profile>]
//   2. All:    cookie-import-browser <browser> --all [--profile <profile>]
//   3. Picker: cookie-import-browser [browser] (returns instructions — picker UI not yet ported)
func handleCookieImportBrowser(ctx *ExecContext) (string, error) {
	args := ctx.Args

	// Parse flags
	var browserArg string
	var domain string
	var profile string
	var hasAll bool

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--domain":
			if i+1 < len(args) {
				domain = args[i+1]
				i++
			}
		case "--profile":
			if i+1 < len(args) {
				profile = args[i+1]
				i++
			}
		case "--all":
			hasAll = true
		default:
			if !strings.HasPrefix(args[i], "-") && browserArg == "" {
				browserArg = args[i]
			}
		}
	}

	if profile == "" {
		profile = "Default"
	}

	// Mode 1: Direct import — scoped to specific domain
	if domain != "" {
		pageURL := ctx.BM.CurrentURL()
		if pageURL == "" || pageURL == "about:blank" {
			return "", fmt.Errorf("no active page to import cookies for")
		}
		u, err := url.Parse(pageURL)
		if err != nil {
			return "", fmt.Errorf("invalid page URL: %w", err)
		}
		pageHostname := u.Hostname()
		normalizedDomain := domain
		if strings.HasPrefix(normalizedDomain, ".") {
			normalizedDomain = normalizedDomain[1:]
		}
		if normalizedDomain != pageHostname && !strings.HasSuffix(pageHostname, "."+normalizedDomain) {
			return "", fmt.Errorf("--domain %q does not match current page domain %q. Navigate to the target site first.", domain, pageHostname)
		}

		browser := browserArg
		if browser == "" {
			browser = "comet"
		}
		result, err := cookieimport.ImportCookies(browser, []string{domain}, profile)
		if err != nil {
			if cie, ok := err.(*cookieimport.Error); ok {
				// If all failed and v20 detected, try CDP fallback (Windows only)
				if result != nil && result.Count == 0 && result.Failed > 0 && cookieimport.HasV20Cookies(browser, profile) {
					result, err = cookieimport.ImportCookiesViaCDP(browser, []string{domain}, profile)
					if err != nil {
						return "", fmt.Errorf("cookie import failed: %s", err.Error())
					}
				} else {
					return "", fmt.Errorf("cookie import failed [%s]: %s", cie.Code, cie.Msg)
				}
			} else {
				return "", fmt.Errorf("cookie import failed: %w", err)
			}
		}

		if result.Count > 0 {
			if err := setCDPCookies(ctx, result.Cookies); err != nil {
				return "", err
			}
			ctx.BM.TrackCookieImportDomains([]string{domain})
		}
		msg := fmt.Sprintf("Imported %d cookies for %s from %s", result.Count, domain, browser)
		if result.Failed > 0 {
			msg += fmt.Sprintf(" (%d failed to decrypt)", result.Failed)
		}
		return msg, nil
	}

	// Mode 2: Explicit all-cookies import
	if hasAll {
		browser := browserArg
		if browser == "" {
			browser = "comet"
		}
		listResult, err := cookieimport.ListDomains(browser, profile)
		if err != nil {
			if cie, ok := err.(*cookieimport.Error); ok {
				return "", fmt.Errorf("cookie import failed [%s]: %s", cie.Code, cie.Msg)
			}
			return "", fmt.Errorf("cookie import failed: %w", err)
		}
		if len(listResult.Domains) == 0 {
			return fmt.Sprintf("No cookies found in %s (profile: %s)", browser, profile), nil
		}
		allDomainNames := make([]string, len(listResult.Domains))
		for i, d := range listResult.Domains {
			allDomainNames[i] = d.Domain
		}
		result, err := cookieimport.ImportCookies(browser, allDomainNames, profile)
		if err != nil {
			if cie, ok := err.(*cookieimport.Error); ok {
				return "", fmt.Errorf("cookie import failed [%s]: %s", cie.Code, cie.Msg)
			}
			return "", fmt.Errorf("cookie import failed: %w", err)
		}
		if result.Count > 0 {
			if err := setCDPCookies(ctx, result.Cookies); err != nil {
				return "", err
			}
			ctx.BM.TrackCookieImportDomains(allDomainNames)
		}
		msg := fmt.Sprintf("Imported %d cookies across %d domains from %s", result.Count, len(result.DomainCounts), browser)
		msg += " (used --all: all browser cookies imported, consider --domain for tighter scoping)"
		if result.Failed > 0 {
			msg += fmt.Sprintf(" (%d failed to decrypt)", result.Failed)
		}
		return msg, nil
	}

	// Mode 3: Picker UI
	code := picker.GenerateCode()
	url := fmt.Sprintf("http://127.0.0.1:%d/cookie-picker?code=%s", ctx.Port, code)
	return fmt.Sprintf("Opening cookie picker...\n\n%s\n\nSelect domains to import from your browser.", url), nil
}

// setCDPCookies sets imported cookies on the current page context via CDP.
func setCDPCookies(ctx *ExecContext, cookies []*cookieimport.CDPCookie) error {
	if len(cookies) == 0 {
		return nil
	}
	params := make([]*network.CookieParam, 0, len(cookies))
	for _, cookie := range cookies {
		cp := &network.CookieParam{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HTTPOnly,
			SameSite: cookie.SameSite,
		}
		if cookie.Expires >= 0 {
			tse := cdp.TimeSinceEpoch(time.Unix(int64(cookie.Expires), 0))
			cp.Expires = &tse
		}
		params = append(params, cp)
	}
	return ctx.BM.SetCDPCookies(params)
}
