package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/sergi/go-diff/diffmatchpatch"

	"browse-go/pkg/browser"
)

// snapshotScriptTemplate is the base JS function for building an accessibility tree.
// It is wrapped at runtime to inject options.
const snapshotScriptTemplate = `
(function(opts) {
	const INTERACTIVE = new Set(['button','link','textbox','checkbox','radio','combobox',
		'listbox','menuitem','menuitemcheckbox','menuitemradio','option','searchbox',
		'slider','spinbutton','switch','tab','treeitem']);

	function getRole(el) {
		const explicit = el.getAttribute('role');
		if (explicit) return explicit;
		const tag = el.tagName.toLowerCase();
		if (tag === 'a' && el.href) return 'link';
		if (tag === 'button' || tag === 'input' && ['button','submit','reset'].includes(el.type)) return 'button';
		if (tag === 'input') {
			if (el.type === 'checkbox') return 'checkbox';
			if (el.type === 'radio') return 'radio';
			if (el.type === 'range') return 'slider';
			if (el.type === 'number') return 'spinbutton';
			return 'textbox';
		}
		if (tag === 'select') return 'combobox';
		if (tag === 'textarea') return 'textbox';
		if (tag === 'img' && el.alt) return 'img';
		return tag;
	}

	function getName(el) {
		const aria = el.getAttribute('aria-label');
		if (aria) return aria;
		if (el.labels && el.labels.length > 0) return el.labels[0].textContent.trim();
		if (el.placeholder) return el.placeholder;
		const text = el.textContent.trim();
		if (text.length > 0 && text.length < 200) return text;
		return '';
	}

	function isInteractive(el) {
		const role = getRole(el);
		return INTERACTIVE.has(role) || el.onclick || el.tabIndex >= 0 ||
			getComputedStyle(el).cursor === 'pointer';
	}

	let eCounter = 0;
	let cCounter = 0;
	const refs = [];
	const lines = [];

	const maxDepth = opts.depth >= 0 ? opts.depth : 10;
	const root = opts.selector ? document.querySelector(opts.selector) : document.body;
	if (!root) return {text: '', refs: []};

	function walk(el, depth) {
		if (depth > maxDepth) return;
		const role = getRole(el);
		const name = getName(el);
		const interactive = isInteractive(el);

		if (role === 'script' || role === 'style' || role === 'noscript') return;

		if (opts.interactive && !interactive) {
			// Still recurse into children in case they are interactive
			for (const child of el.children) {
				walk(child, depth + 1);
			}
			return;
		}

		let prefix = (opts.compact ? '' : '  '.repeat(depth)) + '- ' + role;
		if (name) prefix += ' "' + name.slice(0,120) + '"';

		let ref = '';
		if (interactive) {
			if (INTERACTIVE.has(role)) {
				if (!opts.cursorInteractive) {
					eCounter++;
					ref = '@e' + eCounter;
				}
			} else {
				cCounter++;
				ref = '@c' + cCounter;
			}
			if (ref) {
				prefix += ' ' + ref;
				refs.push({ref: ref.slice(1), role, name: name || '', selector: generateSelector(el)});
			}
		}

		lines.push(prefix);

		for (const child of el.children) {
			walk(child, depth + 1);
		}
	}

	function generateSelector(el) {
		if (el.id) return '#' + el.id;
		const tag = el.tagName.toLowerCase();
		if (el.className) {
			const cls = el.className.toString().split(' ').filter(c => c).slice(0,2).join('.');
			if (cls) return tag + '.' + cls;
		}
		let idx = 1;
		let sib = el.previousElementSibling;
		while (sib) { if (sib.tagName === el.tagName) idx++; sib = sib.previousElementSibling; }
		return tag + ':nth-of-type(' + idx + ')';
	}

	walk(root, 0);
	return {text: lines.join('\n'), refs};
})
`

func (r *Registry) registerSnapshot() {
	r.Register("snapshot", CommandDesc{Category: "Snapshot", Description: "Accessibility tree with @e/@c refs", Usage: "snapshot [-i] [-c] [-d N] [-s sel] [--diff] [-a] [-o path] [-C] [-H <json>]"},
		func(ctx *ExecContext) (string, error) {
			interactive, compact, depth, selector, diff, cursorInteractive, annotate, outputPath, heatmap := parseSnapshotArgsFull(ctx.Args)

			// Build the JS invocation with inline options
			script := fmt.Sprintf(`%s({interactive:%v,compact:%v,depth:%d,selector:%q,cursorInteractive:%v})`,
				snapshotScriptTemplate, interactive, compact, depth, selector, cursorInteractive)

			var result struct {
				Text string `json:"text"`
				Refs []struct {
					Ref      string `json:"ref"`
					Role     string `json:"role"`
					Name     string `json:"name"`
					Selector string `json:"selector"`
				} `json:"refs"`
			}
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(script, &result)); err != nil {
				return "", fmt.Errorf("snapshot failed: %w", err)
			}

			// Store refs on the session
			refMap := make(map[string]browser.RefEntry, len(result.Refs))
			for _, r := range result.Refs {
				refMap[r.Ref] = browser.RefEntry{
					Selector: r.Selector,
					Role:     r.Role,
					Name:     r.Name,
				}
			}
			ctx.Session.SetRefMap(refMap)

			out := result.Text

			// ─── Annotated screenshot (-a) ────────────────────────────
			if annotate {
				path := outputPath
				if path == "" {
					path = filepath.Join(os.TempDir(), fmt.Sprintf("browse-annotated-%d.png", time.Now().Unix()))
				}
				if err := snapshotAnnotate(ctx.Session.Context(), result.Refs, path); err != nil {
					out += "\n\n[annotate failed: " + err.Error() + "]"
				} else {
					out += "\n\n[annotated screenshot: " + path + "]"
				}
			}

			// ─── Heatmap mode (-H) ────────────────────────────────────
			if heatmap != "" {
				path := outputPath
				if path == "" {
					path = filepath.Join(os.TempDir(), fmt.Sprintf("browse-heatmap-%d.png", time.Now().Unix()))
				}
				if err := snapshotHeatmap(ctx.Session.Context(), result.Refs, heatmap, path); err != nil {
					out += "\n\n[heatmap failed: " + err.Error() + "]"
				} else {
					out += "\n\n[heatmap screenshot: " + path + "]"
				}
			}

			if diff {
				baseline := ctx.Session.GetLastSnapshot()
				if baseline == "" {
					ctx.Session.SetLastSnapshot(result.Text)
					return out + "\n\n[no previous snapshot to diff against]", nil
				}
				diffText := textDiff(baseline, result.Text)
				ctx.Session.SetLastSnapshot(result.Text)
				return out + "\n\n" + diffText, nil
			}

			// Store baseline for future diffing
			ctx.Session.SetLastSnapshot(result.Text)
			return out, nil
		})
}

// parseSnapshotArgs extracts flags from snapshot command args (backward compat).
func parseSnapshotArgs(args []string) (interactive, compact bool, depth int, selector string) {
	interactive, compact, depth, selector, _, _, _, _, _ = parseSnapshotArgsFull(args)
	return
}

// parseSnapshotArgsFull extracts all flags including --diff, --annotate, --output, -C, --heatmap.
func parseSnapshotArgsFull(args []string) (interactive, compact bool, depth int, selector string, diff bool, cursorInteractive bool, annotate bool, outputPath string, heatmap string) {
	depth = -1
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-i", "--interactive":
			interactive = true
		case "-c", "--compact":
			compact = true
		case "-d", "--depth":
			if i+1 < len(args) {
				depth, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-s", "--selector":
			if i+1 < len(args) {
				selector = args[i+1]
				i++
			}
		case "--diff":
			diff = true
		case "-a", "--annotate":
			annotate = true
		case "-o", "--output":
			if i+1 < len(args) {
				outputPath = args[i+1]
				i++
			}
		case "-C", "--cursor-interactive":
			cursorInteractive = true
		case "-H", "--heatmap":
			if i+1 < len(args) {
				heatmap = args[i+1]
				i++
			}
		}
	}
	return
}

// textDiff returns a unified-style diff between two strings.
func textDiff(oldText, newText string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldText, newText, true)
	var out []string
	out = append(out, "--- previous snapshot", "+++ current snapshot", "")
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
	return strings.Join(out, "\n")
}

// snapshotAnnotate draws red overlay boxes + ref labels on the page and takes a full-page screenshot.
func snapshotAnnotate(ctx context.Context, refs []struct {
	Ref      string `json:"ref"`
	Role     string `json:"role"`
	Name     string `json:"name"`
	Selector string `json:"selector"`
}, path string) error {
	refsJSON, _ := json.Marshal(refs)
	script := fmt.Sprintf(`
		(() => {
			const refs = %s;
			let count = 0;
			refs.forEach(r => {
				const el = document.querySelector(r.selector);
				if (!el) return;
				const rect = el.getBoundingClientRect();
				const box = document.createElement('div');
				box.className = '__browse_annotate__';
				box.style.cssText = 'position:fixed;left:' + rect.left + 'px;top:' + rect.top + 'px;width:' + rect.width + 'px;height:' + rect.height + 'px;border:2px solid red;z-index:99999;pointer-events:none;box-sizing:border-box;';
				const label = document.createElement('div');
				label.className = '__browse_annotate__';
				label.textContent = r.ref;
				label.style.cssText = 'position:fixed;left:' + rect.left + 'px;top:' + (rect.top - 18) + 'px;background:red;color:white;font-size:11px;padding:1px 4px;z-index:99999;pointer-events:none;font-family:monospace;';
				document.body.appendChild(box);
				document.body.appendChild(label);
				count++;
			});
			return count;
		})()
	`, string(refsJSON))
	var count int
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &count)); err != nil {
		return err
	}
	var buf []byte
	if err := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 90)); err != nil {
		_ = chromedp.Run(ctx, chromedp.Evaluate(`document.querySelectorAll('.__browse_annotate__').forEach(el => el.remove())`, nil))
		return err
	}
	_ = chromedp.Run(ctx, chromedp.Evaluate(`document.querySelectorAll('.__browse_annotate__').forEach(el => el.remove())`, nil))
	return os.WriteFile(path, buf, 0644)
}

var validHeatmapColors = map[string]string{
	"green": "rgba(0,255,0,0.3)", "yellow": "rgba(255,255,0,0.3)", "red": "rgba(255,0,0,0.3)",
	"blue": "rgba(0,0,255,0.3)", "orange": "rgba(255,165,0,0.3)", "gray": "rgba(128,128,128,0.3)",
}

// snapshotHeatmap draws color-coded overlays from a JSON map and takes a full-page screenshot.
func snapshotHeatmap(ctx context.Context, refs []struct {
	Ref      string `json:"ref"`
	Role     string `json:"role"`
	Name     string `json:"name"`
	Selector string `json:"selector"`
}, heatmapJSON, path string) error {
	var colorMap map[string]string
	if err := json.Unmarshal([]byte(heatmapJSON), &colorMap); err != nil {
		return fmt.Errorf("invalid heatmap JSON: %w", err)
	}
	for ref, color := range colorMap {
		if _, ok := validHeatmapColors[color]; !ok {
			return fmt.Errorf("invalid heatmap color %q for %s", color, ref)
		}
		_ = ref
	}

	refsJSON, _ := json.Marshal(refs)
	colorMapJSON, _ := json.Marshal(colorMap)
	colorsJSON, _ := json.Marshal(validHeatmapColors)
	script := fmt.Sprintf(`
		(() => {
			const refs = %s;
			const colorMap = %s;
			const colors = %s;
			let count = 0;
			refs.forEach(r => {
				const colorName = colorMap[r.ref];
				if (!colorName) return;
				const el = document.querySelector(r.selector);
				if (!el) return;
				const rect = el.getBoundingClientRect();
				const overlay = document.createElement('div');
				overlay.className = '__browse_heatmap__';
				overlay.style.cssText = 'position:fixed;left:' + rect.left + 'px;top:' + rect.top + 'px;width:' + rect.width + 'px;height:' + rect.height + 'px;background:' + colors[colorName] + ';z-index:99999;pointer-events:none;box-sizing:border-box;';
				document.body.appendChild(overlay);
				count++;
			});
			return count;
		})()
	`, string(refsJSON), string(colorMapJSON), string(colorsJSON))
	var count int
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &count)); err != nil {
		return err
	}
	var buf []byte
	if err := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 90)); err != nil {
		_ = chromedp.Run(ctx, chromedp.Evaluate(`document.querySelectorAll('.__browse_heatmap__').forEach(el => el.remove())`, nil))
		return err
	}
	_ = chromedp.Run(ctx, chromedp.Evaluate(`document.querySelectorAll('.__browse_heatmap__').forEach(el => el.remove())`, nil))
	return os.WriteFile(path, buf, 0644)
}
