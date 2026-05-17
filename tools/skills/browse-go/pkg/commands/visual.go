package commands

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func (r *Registry) registerVisual() {
	r.Register("screenshot", CommandDesc{Category: "Visual", Description: "Save screenshot", Usage: "screenshot [path] [--base64] [--selector <css>] [--viewport]"},
		func(ctx *ExecContext) (string, error) {
			var outputPath string
			base64Out := false
			selector := ""
			viewportOnly := false

			for i := 0; i < len(ctx.Args); i++ {
				a := ctx.Args[i]
				switch a {
				case "--base64":
					base64Out = true
				case "--selector":
					if i+1 < len(ctx.Args) {
						selector = ctx.Args[i+1]
						i++
					}
				case "--viewport":
					viewportOnly = true
				default:
					if outputPath == "" && !startsWithDash(a) {
						outputPath = a
					}
				}
			}

			if outputPath == "" && !base64Out {
				outputPath = filepath.Join(os.TempDir(), "browse-screenshot.png")
			}

			var buf []byte
			if selector != "" {
				if err := chromedp.Run(ctx.Session.Context(), chromedp.Screenshot(selector, &buf)); err != nil {
					return "", fmt.Errorf("screenshot failed: %w", err)
				}
			} else if viewportOnly {
				if err := chromedp.Run(ctx.Session.Context(), chromedp.CaptureScreenshot(&buf)); err != nil {
					return "", fmt.Errorf("screenshot failed: %w", err)
				}
			} else {
				if err := chromedp.Run(ctx.Session.Context(), chromedp.FullScreenshot(&buf, 90)); err != nil {
					return "", fmt.Errorf("screenshot failed: %w", err)
				}
			}

			if base64Out {
				return base64.StdEncoding.EncodeToString(buf), nil
			}

			if err := os.WriteFile(outputPath, buf, 0644); err != nil {
				return "", fmt.Errorf("write screenshot failed: %w", err)
			}
			return "Screenshot saved to " + outputPath, nil
		})

	r.Register("prettyscreenshot", CommandDesc{Category: "Visual", Description: "Screenshot with cleanup and scroll", Usage: "prettyscreenshot [path] [--cleanup] [--scroll-to <sel|text>] [--hide <sel>] [--width <px>]"},
		func(ctx *ExecContext) (string, error) {
			var outputPath string
			doCleanup := false
			scrollTo := ""
			var hideSelectors []string
			viewportWidth := 0

			for i := 0; i < len(ctx.Args); i++ {
				a := ctx.Args[i]
				switch a {
				case "--cleanup":
					doCleanup = true
				case "--scroll-to":
					if i+1 < len(ctx.Args) {
						scrollTo = ctx.Args[i+1]
						i++
					}
				case "--hide":
					i++
					for i < len(ctx.Args) && !startsWithDash(ctx.Args[i]) {
						hideSelectors = append(hideSelectors, ctx.Args[i])
						i++
					}
					i--
				case "--width":
					if i+1 < len(ctx.Args) {
						w, _ := fmt.Sscanf(ctx.Args[i+1], "%d", &viewportWidth)
						if w != 1 {
							viewportWidth = 0
						}
						i++
					}
				default:
					if outputPath == "" && !startsWithDash(a) {
						outputPath = a
					}
				}
			}

			if outputPath == "" {
				ts := time.Now().UTC().Format("2006-01-02T15-04-05")
				outputPath = filepath.Join(os.TempDir(), fmt.Sprintf("browse-pretty-%s.png", ts))
			}

			// Set viewport width if specified
			if viewportWidth > 0 {
				if err := ctx.BM.SetViewport(viewportWidth, 720); err != nil {
					return "", fmt.Errorf("viewport failed: %w", err)
				}
			}

			// Run cleanup if requested
			if doCleanup {
				cleanupScript := `
					(() => {
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
						[...AD_SELECTORS, ...COOKIE_SELECTORS, ...SOCIAL_SELECTORS].forEach(sel => {
							try {
								document.querySelectorAll(sel).forEach(el => {
									el.style.setProperty('display', 'none', 'important');
								});
							} catch (e) {}
						});
						document.querySelectorAll('*').forEach(el => {
							const style = getComputedStyle(el);
							if (style.position === 'fixed' || style.position === 'sticky') {
								const tag = el.tagName.toLowerCase();
								if (tag === 'nav' || tag === 'header') return;
								if (el.getAttribute('role') === 'navigation') return;
								el.style.setProperty('display', 'none', 'important');
							}
						});
					})()
				`
				_ = chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(cleanupScript, nil))
			}

			// Hide specific selectors
			if len(hideSelectors) > 0 {
				script := fmt.Sprintf(`
					(() => {
						const sels = %s;
						for (const sel of sels) {
							try {
								document.querySelectorAll(sel).forEach(el => {
									el.style.setProperty('display', 'none', 'important');
								});
							} catch (e) {}
						}
					})()
				`, strconvQuote(strings.Join(hideSelectors, ",")))
				_ = chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(script, nil))
			}

			// Scroll to target
			if scrollTo != "" {
				scrollScript := fmt.Sprintf(`
					(() => {
						const target = %s;
						let el = document.querySelector(target);
						if (el) {
							el.scrollIntoView({behavior: 'instant', block: 'center'});
							return true;
						}
						const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT, null);
						let node;
						while ((node = walker.nextNode())) {
							if (node.textContent && node.textContent.includes(target)) {
								const parent = node.parentElement;
								if (parent) {
									parent.scrollIntoView({behavior: 'instant', block: 'center'});
									return true;
								}
							}
						}
						return false;
					})()
				`, strconvQuote(scrollTo))
				var scrolled bool
				if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(scrollScript, &scrolled)); err != nil {
					return "", fmt.Errorf("scroll failed: %w", err)
				}
				if !scrolled {
					return "", fmt.Errorf("could not find element or text to scroll to: %s", scrollTo)
				}
				// Brief wait for scroll to settle
				chromedp.Sleep(300 * time.Millisecond)
			}

			// Take screenshot
			var buf []byte
			if scrollTo != "" {
				// Viewport screenshot when scrolled to specific element
				if err := chromedp.Run(ctx.Session.Context(), chromedp.CaptureScreenshot(&buf)); err != nil {
					return "", fmt.Errorf("screenshot failed: %w", err)
				}
			} else {
				if err := chromedp.Run(ctx.Session.Context(), chromedp.FullScreenshot(&buf, 90)); err != nil {
					return "", fmt.Errorf("screenshot failed: %w", err)
				}
			}

			if err := os.WriteFile(outputPath, buf, 0644); err != nil {
				return "", fmt.Errorf("write screenshot failed: %w", err)
			}

			parts := []string{"Screenshot saved"}
			if doCleanup {
				parts = append(parts, "(cleaned)")
			}
			if scrollTo != "" {
				parts = append(parts, fmt.Sprintf("(scrolled to: %s)", scrollTo))
			}
			parts = append(parts, ": "+outputPath)
			return strings.Join(parts, " "), nil
		})

	r.Register("responsive", CommandDesc{Category: "Visual", Description: "Screenshot at multiple viewports", Usage: "responsive [path-prefix]"},
		func(ctx *ExecContext) (string, error) {
			prefix := filepath.Join(os.TempDir(), "browse-responsive")
			if len(ctx.Args) > 0 && !startsWithDash(ctx.Args[0]) {
				prefix = ctx.Args[0]
			}

			breakpoints := []struct {
				name   string
				width  int
				height int
			}{
				{"mobile", 375, 812},
				{"tablet", 768, 1024},
				{"desktop", 1280, 720},
			}

			var outputs []string
			for _, bp := range breakpoints {
				if err := ctx.BM.SetViewport(bp.width, bp.height); err != nil {
					return "", fmt.Errorf("viewport %s failed: %w", bp.name, err)
				}
				var buf []byte
				if err := chromedp.Run(ctx.Session.Context(), chromedp.FullScreenshot(&buf, 90)); err != nil {
					return "", fmt.Errorf("screenshot %s failed: %w", bp.name, err)
				}
				path := fmt.Sprintf("%s-%s.png", prefix, bp.name)
				if err := os.WriteFile(path, buf, 0644); err != nil {
					return "", fmt.Errorf("write %s failed: %w", bp.name, err)
				}
				outputs = append(outputs, fmt.Sprintf("%s (%dx%d): %s", bp.name, bp.width, bp.height, path))
			}

			// Restore default viewport
			_ = ctx.BM.SetViewport(1280, 720)
			return strings.Join(outputs, "\n"), nil
		})
}

func startsWithDash(s string) bool {
	return len(s) > 0 && s[0] == '-'
}
