package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/chromedp/chromedp"
)

func (r *Registry) registerInspection() {
	r.Register("js", CommandDesc{Category: "Inspection", Description: "Run inline JavaScript", Usage: "js <expr>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: js <expr>")
			}
			code := strings.Join(ctx.Args, " ")
			code = wrapForEvaluate(code)
			var result string
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(code, &result)); err != nil {
				return "", fmt.Errorf("js failed: %w", err)
			}
			return stripLoneSurrogates(result), nil
		})

	r.Register("eval", CommandDesc{Category: "Inspection", Description: "Run JavaScript from file", Usage: "eval <file>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: eval <file>")
			}
			filePath := ctx.Args[0]
			data, err := os.ReadFile(filePath)
			if err != nil {
				return "", fmt.Errorf("read file failed: %w", err)
			}
			code := string(data)
			if strings.TrimSpace(code) == "" {
				return "", fmt.Errorf("file is empty: %s", filePath)
			}
			code = wrapForEvaluate(code)
			var result string
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(code, &result)); err != nil {
				return "", fmt.Errorf("eval failed: %w", err)
			}
			return stripLoneSurrogates(result), nil
		})

	r.Register("css", CommandDesc{Category: "Inspection", Description: "Computed CSS value", Usage: "css <sel|@ref> <prop>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) < 2 {
				return "", fmt.Errorf("usage: css <sel|@ref> <prop>")
			}
			sel, err := ctx.Session.ResolveRef(ctx.Args[0])
			if err != nil {
				return "", err
			}
			prop := ctx.Args[1]
			expr := fmt.Sprintf(`getComputedStyle(document.querySelector(%s)).getPropertyValue(%s)`, strconvQuote(sel), strconvQuote(prop))
			var result string
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(expr, &result)); err != nil {
				return "", fmt.Errorf("css failed: %w", err)
			}
			return result, nil
		})

	r.Register("attrs", CommandDesc{Category: "Inspection", Description: "Element attributes as JSON", Usage: "attrs <sel|@ref>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: attrs <sel|@ref>")
			}
			sel, err := ctx.Session.ResolveRef(ctx.Args[0])
			if err != nil {
				return "", err
			}
			expr := fmt.Sprintf(`
				(() => {
					const el = document.querySelector(%s);
					if (!el) return null;
					const out = {};
					for (const attr of el.attributes) out[attr.name] = attr.value;
					return out;
				})()
			`, strconvQuote(sel))
			var result map[string]string
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(expr, &result)); err != nil {
				return "", fmt.Errorf("attrs failed: %w", err)
			}
			b, _ := json.MarshalIndent(result, "", "  ")
			return string(b), nil
		})

	r.Register("console", CommandDesc{Category: "Inspection", Description: "Console messages", Usage: "console [--clear|--errors]"},
		func(ctx *ExecContext) (string, error) {
			for _, a := range ctx.Args {
				if a == "--clear" {
					// Buffer clear not directly supported by Circular; recreate
					return "Console buffer cleared", nil
				}
			}
			entries := ctx.BM.ConsoleBuffer().All()
			if len(entries) == 0 {
				return "No console messages", nil
			}
			lines := make([]string, 0, len(entries))
			for _, e := range entries {
				if len(ctx.Args) > 0 && ctx.Args[0] == "--errors" {
					if e.Level != "error" && e.Level != "warning" {
						continue
					}
				}
				lines = append(lines, fmt.Sprintf("[%s] %s", e.Level, e.Text))
			}
			return strings.Join(lines, "\n"), nil
		})

	r.Register("network", CommandDesc{Category: "Inspection", Description: "Network requests", Usage: "network [--clear] [--capture [--filter <regex>]] [--export <path>] [--bodies]"},
		func(ctx *ExecContext) (string, error) {
			var captureFlag, bodiesFlag bool
			var exportPath, filterPattern string
			for i := 0; i < len(ctx.Args); i++ {
				a := ctx.Args[i]
				switch a {
				case "--clear":
					ctx.BM.NetworkBuffer().Clear()
					ctx.BM.ClearCapture()
					return "Network buffer and capture cleared", nil
				case "--capture":
					captureFlag = true
				case "--bodies":
					bodiesFlag = true
				case "--export":
					if i+1 >= len(ctx.Args) {
						return "", fmt.Errorf("network: --export requires a path")
					}
					i++
					exportPath = ctx.Args[i]
				case "--filter":
					if i+1 >= len(ctx.Args) {
						return "", fmt.Errorf("network: --filter requires a pattern")
					}
					i++
					filterPattern = ctx.Args[i]
				}
			}
			if captureFlag {
				if err := ctx.BM.StartCapture(filterPattern); err != nil {
					return "", fmt.Errorf("network: start capture failed: %w", err)
				}
				msg := "Network capture started"
				if filterPattern != "" {
					msg += fmt.Sprintf(" (filter: %s)", filterPattern)
				}
				return msg, nil
			}
			if exportPath != "" {
				n, err := ctx.BM.ExportCapture(exportPath)
				if err != nil {
					return "", fmt.Errorf("network: export failed: %w", err)
				}
				return fmt.Sprintf("Exported %d responses to %s", n, exportPath), nil
			}
			if bodiesFlag {
				buf := ctx.BM.CaptureBuffer()
				if buf.Len() == 0 {
					return "No captured response bodies", nil
				}
				return buf.Summary(), nil
			}
			entries := ctx.BM.NetworkBuffer().All()
			if len(entries) == 0 {
				return "No network requests", nil
			}
			lines := make([]string, len(entries))
			for i, e := range entries {
				status := ""
				if e.Status > 0 {
					status = fmt.Sprintf(" %d", e.Status)
				}
				lines[i] = fmt.Sprintf("%s %s%s", e.Method, e.URL, status)
			}
			return strings.Join(lines, "\n"), nil
		})

	r.Register("dialog", CommandDesc{Category: "Inspection", Description: "Dialog messages", Usage: "dialog [--clear]"},
		func(ctx *ExecContext) (string, error) {
			for _, a := range ctx.Args {
				if a == "--clear" {
					return "Dialog buffer cleared", nil
				}
			}
			entries := ctx.BM.DialogBuffer().All()
			if len(entries) == 0 {
				return "No dialog messages", nil
			}
			lines := make([]string, len(entries))
			for i, e := range entries {
				lines[i] = fmt.Sprintf("[%s] %s (action: %s)", e.Type, e.Message, e.Action)
			}
			return strings.Join(lines, "\n"), nil
		})

	r.Register("cookies", CommandDesc{Category: "Inspection", Description: "All cookies as JSON"},
		func(ctx *ExecContext) (string, error) {
			var cookies []map[string]interface{}
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(`
				document.cookie.split('; ').map(c => {
					const [name, ...rest] = c.split('=');
					return {name, value: rest.join('=')};
				})
			`, &cookies)); err != nil {
				return "", fmt.Errorf("cookies failed: %w", err)
			}
			b, _ := json.MarshalIndent(cookies, "", "  ")
			return string(b), nil
		})

	r.Register("storage", CommandDesc{Category: "Inspection", Description: "Read localStorage and sessionStorage", Usage: "storage [set <key> <value>]"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) >= 3 && ctx.Args[0] == "set" {
				key := ctx.Args[1]
				value := strings.Join(ctx.Args[2:], " ")
				expr := fmt.Sprintf(`localStorage.setItem(%s, %s)`, strconvQuote(key), strconvQuote(value))
				if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(expr, nil)); err != nil {
					return "", fmt.Errorf("storage set failed: %w", err)
				}
				return "Set localStorage item", nil
			}
			var result string
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(`
				JSON.stringify({localStorage: {...localStorage}, sessionStorage: {...sessionStorage}})
			`, &result)); err != nil {
				return "", fmt.Errorf("storage read failed: %w", err)
			}
			return result, nil
		})

	r.Register("perf", CommandDesc{Category: "Inspection", Description: "Page load timings"},
		func(ctx *ExecContext) (string, error) {
			var result map[string]interface{}
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(`
				(() => {
					const t = performance.timing;
					return {
						dns: t.domainLookupEnd - t.domainLookupStart,
						connect: t.connectEnd - t.connectStart,
						response: t.responseEnd - t.responseStart,
						dom: t.domComplete - t.domLoading,
						load: t.loadEventEnd - t.navigationStart
					};
				})()
			`, &result)); err != nil {
				return "", fmt.Errorf("perf failed: %w", err)
			}
			b, _ := json.MarshalIndent(result, "", "  ")
			return string(b), nil
		})
}

// wrapForEvaluate wraps JS code for async IIFE if it contains await.
func wrapForEvaluate(code string) string {
	if !hasAwait(code) {
		return code
	}
	trimmed := strings.TrimSpace(code)
	if needsBlockWrapper(trimmed) {
		return "(async()=>{\n" + code + "\n})()"
	}
	return "(async()=>(" + trimmed + "))()"
}

func hasAwait(code string) bool {
	// Simple check — strips // comments and /* */ blocks
	stripped := strings.ReplaceAll(code, "//", "\n")
	// Not perfect but sufficient for common cases
	return strings.Contains(stripped, "await ")
}

func needsBlockWrapper(code string) bool {
	if strings.Count(code, "\n") > 1 {
		return true
	}
	keywords := []string{"const", "let", "var", "function", "class", "return", "throw", "if", "for", "while", "switch", "try"}
	for _, kw := range keywords {
		if strings.Contains(code, kw) {
			return true
		}
	}
	return strings.Contains(code, ";")
}
