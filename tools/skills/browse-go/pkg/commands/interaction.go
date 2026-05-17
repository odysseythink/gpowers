package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

func (r *Registry) registerInteraction() {
	r.Register("click", CommandDesc{Category: "Interaction", Description: "Click element", Usage: "click <sel|@ref>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: click <sel|@ref>")
			}
			sel, err := ctx.Session.ResolveRef(ctx.Args[0])
			if err != nil {
				return "", err
			}
			if err := chromedp.Run(ctx.Session.Context(),
				chromedp.WaitVisible(sel),
				chromedp.Click(sel),
			); err != nil {
				return "", fmt.Errorf("click failed: %w", err)
			}
			return "Clicked", nil
		})

	r.Register("fill", CommandDesc{Category: "Interaction", Description: "Fill input", Usage: "fill <sel|@ref> <value>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) < 2 {
				return "", fmt.Errorf("usage: fill <sel|@ref> <value>")
			}
			sel, err := ctx.Session.ResolveRef(ctx.Args[0])
			if err != nil {
				return "", err
			}
			value := strings.Join(ctx.Args[1:], " ")
			if err := chromedp.Run(ctx.Session.Context(),
				chromedp.Clear(sel),
				chromedp.SendKeys(sel, value),
			); err != nil {
				return "", fmt.Errorf("fill failed: %w", err)
			}
			return "Filled", nil
		})

	r.Register("type", CommandDesc{Category: "Interaction", Description: "Type into focused element", Usage: "type <text>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: type <text>")
			}
			text := strings.Join(ctx.Args, " ")
			if err := chromedp.Run(ctx.Session.Context(), chromedp.KeyEvent(text)); err != nil {
				return "", fmt.Errorf("type failed: %w", err)
			}
			return "Typed", nil
		})

	r.Register("press", CommandDesc{Category: "Interaction", Description: "Press keyboard key", Usage: "press <key>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: press <key>")
			}
			key := normalizeKey(ctx.Args[0])
			if err := chromedp.Run(ctx.Session.Context(), chromedp.KeyEvent(key)); err != nil {
				return "", fmt.Errorf("press failed: %w", err)
			}
			return "Pressed " + key, nil
		})

	r.Register("scroll", CommandDesc{Category: "Interaction", Description: "Scroll element into view or page bottom", Usage: "scroll [sel|@ref] [--times N] [--wait ms]"},
		func(ctx *ExecContext) (string, error) {
			target := ""
			times := 1
			waitMs := 0
			for i := 0; i < len(ctx.Args); i++ {
				switch ctx.Args[i] {
				case "--times":
					if i+1 < len(ctx.Args) {
						if n, err := strconv.Atoi(ctx.Args[i+1]); err == nil && n > 0 {
							times = n
						}
						i++
					}
				case "--wait":
					if i+1 < len(ctx.Args) {
						if n, err := strconv.Atoi(ctx.Args[i+1]); err == nil && n >= 0 {
							waitMs = n
						}
						i++
					}
				default:
					if target == "" && !strings.HasPrefix(ctx.Args[i], "-") {
						target = ctx.Args[i]
					}
				}
			}

			for i := 0; i < times; i++ {
				if target != "" {
					sel, err := ctx.Session.ResolveRef(target)
					if err != nil {
						return "", err
					}
					if err := chromedp.Run(ctx.Session.Context(), chromedp.ScrollIntoView(sel)); err != nil {
						return "", fmt.Errorf("scroll failed: %w", err)
					}
				} else {
					if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight)`, nil)); err != nil {
						return "", fmt.Errorf("scroll failed: %w", err)
					}
				}
				if waitMs > 0 && i < times-1 {
					time.Sleep(time.Duration(waitMs) * time.Millisecond)
				}
			}

			if target != "" {
				if times > 1 {
					return fmt.Sprintf("Scrolled into view %d times", times), nil
				}
				return "Scrolled into view", nil
			}
			if times > 1 {
				return fmt.Sprintf("Scrolled to bottom %d times", times), nil
			}
			return "Scrolled to bottom", nil
		})

	r.Register("wait", CommandDesc{Category: "Interaction", Description: "Wait for element, network idle, page load, or DOMContentLoaded", Usage: "wait <sel|--networkidle|--load|--domcontentloaded>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: wait <sel|--networkidle|--load|--domcontentloaded>")
			}
			target := ctx.Args[0]
			ctxSession, cancel := context.WithTimeout(ctx.Session.Context(), 15*time.Second)
			defer cancel()

			switch target {
			case "--networkidle":
				// Simplified: wait for body to be ready
				if err := chromedp.Run(ctxSession, chromedp.WaitReady("body")); err != nil {
					return "", fmt.Errorf("wait networkidle failed: %w", err)
				}
			case "--load":
				if err := chromedp.Run(ctxSession, chromedp.WaitReady("body")); err != nil {
					return "", fmt.Errorf("wait load failed: %w", err)
				}
			case "--domcontentloaded":
				if err := chromedp.Run(ctxSession, chromedp.Poll(`document.readyState === 'interactive' || document.readyState === 'complete'`, nil)); err != nil {
					return "", fmt.Errorf("wait domcontentloaded failed: %w", err)
				}
			default:
				sel, err := ctx.Session.ResolveRef(target)
				if err != nil {
					return "", err
				}
				if err := chromedp.Run(ctxSession, chromedp.WaitVisible(sel)); err != nil {
					return "", fmt.Errorf("wait failed: %w", err)
				}
			}
			return "Wait complete", nil
		})

	r.Register("hover", CommandDesc{Category: "Interaction", Description: "Hover element", Usage: "hover <sel|@ref>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: hover <sel|@ref>")
			}
			sel, err := ctx.Session.ResolveRef(ctx.Args[0])
			if err != nil {
				return "", err
			}
			expr := fmt.Sprintf(`document.querySelector(%s).dispatchEvent(new MouseEvent('mouseover', {bubbles: true}))`, strconvQuote(sel))
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(expr, nil)); err != nil {
				return "", fmt.Errorf("hover failed: %w", err)
			}
			return "Hovered", nil
		})

	r.Register("select", CommandDesc{Category: "Interaction", Description: "Select dropdown option", Usage: "select <sel|@ref> <value>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) < 2 {
				return "", fmt.Errorf("usage: select <sel|@ref> <value>")
			}
			sel, err := ctx.Session.ResolveRef(ctx.Args[0])
			if err != nil {
				return "", err
			}
			value := strings.Join(ctx.Args[1:], " ")
			expr := fmt.Sprintf(`document.querySelector(%s).value = %s`, strconvQuote(sel), strconvQuote(value))
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(expr, nil)); err != nil {
				return "", fmt.Errorf("select failed: %w", err)
			}
			return "Selected", nil
		})
}

// normalizeKey maps Playwright-style key names to chromedp key events.
func normalizeKey(key string) string {
	key = strings.TrimSpace(key)
	switch strings.ToLower(key) {
	case "enter":
		return "\n"
	case "tab":
		return "\t"
	case "escape", "esc":
		return "\u001b"
	case "backspace":
		return "\b"
	case "delete", "del":
		return "\u007f"
	case "arrowup":
		return kb.ArrowUp
	case "arrowdown":
		return kb.ArrowDown
	case "arrowleft":
		return kb.ArrowLeft
	case "arrowright":
		return kb.ArrowRight
	case "home":
		return kb.Home
	case "end":
		return kb.End
	case "pageup":
		return kb.PageUp
	case "pagedown":
		return kb.PageDown
	}
	return key
}

func strconvQuote(s string) string {
	return `"` + s + `"`
}
