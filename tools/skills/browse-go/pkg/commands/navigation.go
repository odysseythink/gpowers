package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chromedp/chromedp"

	"browse-go/pkg/util"
)

func (r *Registry) registerNavigation() {
	r.Register("goto", CommandDesc{Category: "Navigation", Description: "Navigate to URL", Usage: "goto <url>"},
		func(ctx *ExecContext) (string, error) {
			if ctx.Session.InFrame() {
				return "", fmt.Errorf("cannot use goto inside a frame. Run 'frame main' first.")
			}
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: goto <url>")
			}
			ctx.Session.ClearLoadedHtml()
			normalized, err := util.ValidateNavigationURL(ctx.Args[0])
			if err != nil {
				return "", err
			}
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Navigate(normalized)); err != nil {
				return "", fmt.Errorf("navigation failed: %w", err)
			}
			return "Navigated to " + normalized, nil
		})

	r.Register("back", CommandDesc{Category: "Navigation", Description: "History back"},
		func(ctx *ExecContext) (string, error) {
			if ctx.Session.InFrame() {
				return "", fmt.Errorf("cannot use back inside a frame. Run 'frame main' first.")
			}
			ctx.Session.ClearLoadedHtml()
			if err := chromedp.Run(ctx.Session.Context(), chromedp.NavigateBack()); err != nil {
				return "", fmt.Errorf("back failed: %w", err)
			}
			return "Navigated back", nil
		})

	r.Register("forward", CommandDesc{Category: "Navigation", Description: "History forward"},
		func(ctx *ExecContext) (string, error) {
			if ctx.Session.InFrame() {
				return "", fmt.Errorf("cannot use forward inside a frame. Run 'frame main' first.")
			}
			ctx.Session.ClearLoadedHtml()
			if err := chromedp.Run(ctx.Session.Context(), chromedp.NavigateForward()); err != nil {
				return "", fmt.Errorf("forward failed: %w", err)
			}
			return "Navigated forward", nil
		})

	r.Register("reload", CommandDesc{Category: "Navigation", Description: "Reload page"},
		func(ctx *ExecContext) (string, error) {
			if ctx.Session.InFrame() {
				return "", fmt.Errorf("cannot use reload inside a frame. Run 'frame main' first.")
			}
			ctx.Session.ClearLoadedHtml()
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Reload()); err != nil {
				return "", fmt.Errorf("reload failed: %w", err)
			}
			return "Page reloaded", nil
		})

	r.Register("url", CommandDesc{Category: "Navigation", Description: "Print current URL"},
		func(ctx *ExecContext) (string, error) {
			var url string
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Location(&url)); err != nil {
				return "", err
			}
			return url, nil
		})

	r.Register("viewport", CommandDesc{Category: "Navigation", Description: "Set viewport size", Usage: "viewport <width> <height> [--scale <n>]"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) < 2 {
				return "", fmt.Errorf("usage: viewport <width> <height> [--scale <n>]")
			}
			w, err := strconv.Atoi(ctx.Args[0])
			if err != nil {
				return "", fmt.Errorf("invalid width: %s", ctx.Args[0])
			}
			h, err := strconv.Atoi(ctx.Args[1])
			if err != nil {
				return "", fmt.Errorf("invalid height: %s", ctx.Args[1])
			}
			scale := 0
			for i := 2; i < len(ctx.Args); i++ {
				if ctx.Args[i] == "--scale" && i+1 < len(ctx.Args) {
					scale, _ = strconv.Atoi(ctx.Args[i+1])
					i++
				}
			}
			if err := ctx.BM.SetViewport(w, h); err != nil {
				return "", fmt.Errorf("viewport failed: %w", err)
			}
			msg := fmt.Sprintf("Viewport set to %dx%d", w, h)
			if scale > 0 {
				_, err := ctx.BM.SetDeviceScaleFactor(scale)
				if err != nil {
					return msg + fmt.Sprintf(" (scale failed: %v)", err), nil
				}
				msg += fmt.Sprintf(" with scale %d", scale)
			}
			return msg, nil
		})

	r.Register("useragent", CommandDesc{Category: "Navigation", Description: "Set custom user agent", Usage: "useragent <ua>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return ctx.BM.UserAgent(), nil
			}
			ua := strings.Join(ctx.Args, " ")
			ctx.BM.SetUserAgent(ua)
			return "User agent set (requires relaunch to take effect)", nil
		})

	r.Register("load-html", CommandDesc{Category: "Navigation", Description: "Load HTML content", Usage: "load-html <html> [--wait-until load|domcontentloaded|networkidle] [--from-file <path>]"},
		func(ctx *ExecContext) (string, error) {
			if ctx.Session.InFrame() {
				return "", fmt.Errorf("cannot use load-html inside a frame. Run 'frame main' first.")
			}
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: load-html <html> [--wait-until ...] [--from-file <path>]")
			}

			html := ""
			waitUntil := "domcontentloaded"
			fromFile := ""

			for i := 0; i < len(ctx.Args); i++ {
				switch ctx.Args[i] {
				case "--wait-until":
					if i+1 < len(ctx.Args) {
						waitUntil = ctx.Args[i+1]
						i++
					}
				case "--from-file":
					if i+1 < len(ctx.Args) {
						fromFile = ctx.Args[i+1]
						i++
					}
				default:
					if html == "" && !strings.HasPrefix(ctx.Args[i], "-") {
						html = ctx.Args[i]
					}
				}
			}

			if fromFile != "" {
				payload, err := readLoadHtmlPayload(fromFile)
				if err != nil {
					return "", err
				}
				if payload.HTML != "" {
					html = payload.HTML
				}
				if payload.WaitUntil != "" {
					waitUntil = payload.WaitUntil
				}
			}

			if html == "" {
				return "", fmt.Errorf("no HTML content provided")
			}
			if err := setTabContent(ctx.Session.Context(), html, waitUntil); err != nil {
				return "", fmt.Errorf("load-html failed: %w", err)
			}
			ctx.Session.SetTabContent(html, waitUntil)
			return "HTML loaded", nil
		})
}

// loadHtmlPayload is the JSON structure accepted by --from-file.
type loadHtmlPayload struct {
	HTML      string `json:"html"`
	URL       string `json:"url"`
	WaitUntil string `json:"waitUntil"`
}

const maxLoadHtmlFileSize = 50 * 1024 * 1024 // 50 MB

// readLoadHtmlPayload reads and validates a JSON payload file for load-html.
// It guards against binary files (magic-byte check) and enforces a 50 MB cap.
func readLoadHtmlPayload(path string) (*loadHtmlPayload, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read payload file: %w", err)
	}
	if info.Size() > maxLoadHtmlFileSize {
		return nil, fmt.Errorf("payload file exceeds 50 MB limit (%d bytes)", info.Size())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read payload file: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("payload file is empty")
	}
	// Magic-byte guard: first byte must be '{' (JSON object) or '[' (JSON array)
	first := data[0]
	if first != '{' && first != '[' {
		return nil, fmt.Errorf("payload file does not look like JSON (first byte 0x%02x); expected '{' or '['", first)
	}
	var payload loadHtmlPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("cannot parse payload JSON: %w", err)
	}
	return &payload, nil
}

// setTabContent injects HTML into a tab using document.open/write/close.
func setTabContent(ctx context.Context, html string, waitUntil string) error {
	if err := chromedp.Run(ctx, chromedp.Navigate("about:blank")); err != nil {
		return err
	}
	expr := fmt.Sprintf(`document.open(); document.write(%s); document.close();`, strconv.Quote(html))
	if err := chromedp.Run(ctx, chromedp.Evaluate(expr, nil)); err != nil {
		return err
	}
	// Simple wait — chromedp.WaitReady covers domcontentloaded well enough
	return chromedp.Run(ctx, chromedp.WaitReady("body"))
}
