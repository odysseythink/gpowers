// gpowers-rewrite-browser — Rewrites browser-MCP and playwright-CLI references
// in a SKILL.md body to the abstract gpowers-browser <verb> interface.
// Stdin → stdout. No external deps.
//
// Build:
//   go build -o _gpowers-rewrite-browser gpowers-rewrite-browser.go
//
// Cross-compile for Windows:
//   GOOS=windows GOARCH=amd64 go build -o _gpowers-rewrite-browser.exe gpowers-rewrite-browser.go
package main

import (
	"io"
	"os"
	"regexp"
)

type rule struct {
	re   *regexp.Regexp
	repl string
}

// Order matters: longest / most specific patterns first.
var rules = []rule{
	// 1. tabs_create_mcp + bare URL → open
	{regexp.MustCompile("(?i)`mcp__claude-in-chrome__tabs_create_mcp`\\s+with\\s+URL\\s+((?:https?://)?(?:[^\\s.]+\\.)+[^\\s.]+)\\."),
		"`gpowers-browser open` with `{\"url\":\"$1\"}` on stdin."},

	// 2. navigate then read_page (combined sentence)
	{regexp.MustCompile("(?i)`mcp__claude-in-chrome__navigate`\\s+then\\s+`mcp__claude-in-chrome__read_page`\\s+for\\s+(\\w+)"),
		"`gpowers-browser open` then `gpowers-browser read` (mode: $1)"},

	// 3. find then click (combined sentence)
	{regexp.MustCompile("(?i)`mcp__claude-in-chrome__find`\\s+to\\s+locate,?\\s+then\\s+click(\\s+it)?"),
		"`gpowers-browser wait` (condition: selector:<css>) to locate, then `gpowers-browser click`"},

	// 4. computer action screenshot → screenshot
	{regexp.MustCompile("(?i)`mcp__claude-in-chrome__computer`\\s+action\\s+screenshot"),
		"`gpowers-browser screenshot`"},

	// 5. javascript_tool with `<expr>` → eval with code `<expr>`
	{regexp.MustCompile("`mcp__claude-in-chrome__javascript_tool`\\s+with\\s+`([^`]+)`"),
		"`gpowers-browser eval` with code `$1`"},

	// 6. read_console_messages → read mode console
	{regexp.MustCompile("`mcp__claude-in-chrome__read_console_messages`"),
		"`gpowers-browser read` (mode: console)"},
	{regexp.MustCompile(`\bmcp__claude-in-chrome__read_console_messages\b`),
		"gpowers-browser read (mode: console)"},

	// 7. tabs_close_mcp → close
	{regexp.MustCompile("`mcp__claude-in-chrome__tabs_close_mcp`"),
		"`gpowers-browser close`"},

	// 8. form_input → type (no surrounding context)
	{regexp.MustCompile("`mcp__claude-in-chrome__form_input`"),
		"`gpowers-browser type`"},
	{regexp.MustCompile(`\bmcp__claude-in-chrome__form_input\b`),
		"gpowers-browser type"},

	// 9. navigate (standalone)
	{regexp.MustCompile("`mcp__claude-in-chrome__navigate`"),
		"`gpowers-browser open`"},
	{regexp.MustCompile(`\bmcp__claude-in-chrome__navigate\b`),
		"gpowers-browser open"},

	// 10. read_page (standalone)
	{regexp.MustCompile("`mcp__claude-in-chrome__read_page`"),
		"`gpowers-browser read`"},
	{regexp.MustCompile(`\bmcp__claude-in-chrome__read_page\b`),
		"gpowers-browser read"},

	// 11. find (standalone) — rare on its own; map to wait selector
	{regexp.MustCompile("`mcp__claude-in-chrome__find`"),
		"`gpowers-browser wait` (condition: selector:<css>)"},
	{regexp.MustCompile(`\bmcp__claude-in-chrome__find\b`),
		"gpowers-browser wait (condition: selector:<css>)"},

	// 12. computer (standalone)
	{regexp.MustCompile("`mcp__claude-in-chrome__computer`"),
		"`gpowers-browser screenshot`"},
	{regexp.MustCompile(`\bmcp__claude-in-chrome__computer\b`),
		"gpowers-browser screenshot"},

	// 13. tabs_create_mcp (standalone)
	{regexp.MustCompile("`mcp__claude-in-chrome__tabs_create_mcp`"),
		"`gpowers-browser open`"},
	{regexp.MustCompile(`\bmcp__claude-in-chrome__tabs_create_mcp\b`),
		"gpowers-browser open"},

	// 14. javascript_tool (standalone)
	{regexp.MustCompile("`mcp__claude-in-chrome__javascript_tool`"),
		"`gpowers-browser eval`"},
	{regexp.MustCompile(`\bmcp__claude-in-chrome__javascript_tool\b`),
		"gpowers-browser eval"},

	// 15. Generic playwright fallback lines
	{regexp.MustCompile("`npx\\s+playwright[^`]*`"),
		"`gpowers-browser` (driver auto-selected)"},
	{regexp.MustCompile("`playwright[^`]*`"),
		"`gpowers-browser` (driver auto-selected)"},
	{regexp.MustCompile("(?i)(Non-CC:|On non[- ]Claude[- ]Code\\b[^\\n]*?:)[^\\n]*?\\bplaywright\\b[^\\n]*\\.?"),
		"The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh)."},
	{regexp.MustCompile("(?i)On non-Claude-Code platforms, fall back:\\s+`npx playwright[^`]*`\\."),
		"The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh)."},

	// 16. Bare verb words in pipeline descriptions (aidesigner-frontend style)
	{regexp.MustCompile(`\bnavigate,\s+form_input\b`),
		"gpowers-browser open, gpowers-browser type"},
	{regexp.MustCompile(`\bfind\+click\b`),
		"gpowers-browser wait + gpowers-browser click"},
	{regexp.MustCompile(`\bcomputer screenshot\b`),
		"gpowers-browser screenshot"},
	{regexp.MustCompile(`\bjavascript_tool eval\b`),
		"gpowers-browser eval"},
}

func rewrite(text string) string {
	for _, r := range rules {
		text = r.re.ReplaceAllString(text, r.repl)
	}
	return text
}

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Stderr.WriteString("read stdin: " + err.Error() + "\n")
		os.Exit(1)
	}
	out := rewrite(string(data))
	os.Stdout.WriteString(out)
	if len(out) == 0 || out[len(out)-1] != '\n' {
		os.Stdout.WriteString("\n")
	}
}
