#!/usr/bin/env python3
"""
Rewrites browser-MCP and playwright-CLI references in a SKILL.md body to the
abstract `gpowers-browser <verb>` interface defined in tools/drivers/browser/.
Stdin → stdout. No external deps.
"""
import re
import sys

# Order matters: longest / most specific patterns first.
RULES = [
    # 1. tabs_create_mcp + bare URL in the same sentence → open
    (re.compile(
        r"`mcp__claude-in-chrome__tabs_create_mcp`\s+with\s+URL\s+((?:https?://)?(?:[^\s.]+\.)+[^\s.]+)\.",
        re.IGNORECASE),
     r'`gpowers-browser open` with `{"url":"\1"}` on stdin.'),

    # 2. navigate then read_page (combined sentence)
    (re.compile(
        r"`mcp__claude-in-chrome__navigate`\s+then\s+`mcp__claude-in-chrome__read_page`\s+for\s+(\w+)",
        re.IGNORECASE),
     r'`gpowers-browser open` then `gpowers-browser read` (mode: \1)'),

    # 3. find then click (combined sentence)
    (re.compile(
        r"`mcp__claude-in-chrome__find`\s+to\s+locate,?\s+then\s+click(\s+it)?",
        re.IGNORECASE),
     r'`gpowers-browser wait` (condition: selector:<css>) to locate, then `gpowers-browser click`'),

    # 4. computer action screenshot → screenshot
    (re.compile(
        r"`mcp__claude-in-chrome__computer`\s+action\s+screenshot",
        re.IGNORECASE),
     r'`gpowers-browser screenshot`'),

    # 5. javascript_tool with `<expr>` → eval with code `<expr>`
    (re.compile(
        r"`mcp__claude-in-chrome__javascript_tool`\s+with\s+`([^`]+)`"),
     r'`gpowers-browser eval` with code `\1`'),

    # 6. read_console_messages → read mode console
    (re.compile(
        r"`mcp__claude-in-chrome__read_console_messages`"),
     r'`gpowers-browser read` (mode: console)'),

    # 7. tabs_close_mcp → close
    (re.compile(
        r"`mcp__claude-in-chrome__tabs_close_mcp`"),
     r'`gpowers-browser close`'),

    # 8. form_input → type (no surrounding context)
    (re.compile(
        r"`mcp__claude-in-chrome__form_input`"),
     r'`gpowers-browser type`'),
    (re.compile(r"\bmcp__claude-in-chrome__form_input\b"),
     r'gpowers-browser type'),

    # 9. navigate (standalone)
    (re.compile(r"`mcp__claude-in-chrome__navigate`"),
     r'`gpowers-browser open`'),
    (re.compile(r"\bmcp__claude-in-chrome__navigate\b"),
     r'gpowers-browser open'),

    # 10. read_page (standalone)
    (re.compile(r"`mcp__claude-in-chrome__read_page`"),
     r'`gpowers-browser read`'),

    # 11. find (standalone) — rare on its own; map to wait selector
    (re.compile(r"`mcp__claude-in-chrome__find`"),
     r'`gpowers-browser wait` (condition: selector:<css>)'),

    # 12. computer (standalone)
    (re.compile(r"`mcp__claude-in-chrome__computer`"),
     r'`gpowers-browser screenshot`'),

    # 13. tabs_create_mcp (standalone)
    (re.compile(r"`mcp__claude-in-chrome__tabs_create_mcp`"),
     r'`gpowers-browser open`'),

    # 14. javascript_tool (standalone)
    (re.compile(r"`mcp__claude-in-chrome__javascript_tool`"),
     r'`gpowers-browser eval`'),

    # 15. Generic playwright fallback lines — replace whole line
    (re.compile(
        r"^(Non-CC:|On non[- ]Claude[- ]Code\b[^\n]*?:)\s+`?npx\s+playwright[^`\n]*`?\.?$",
        re.MULTILINE),
     r"The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh)."),
    (re.compile(
        r"^(Non-CC:|On non[- ]Claude[- ]Code\b[^\n]*?:)\s+`?playwright[^`\n]*`?\.?$",
        re.MULTILINE),
     r"The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh)."),
    # Fallback prefix: "fall back: <command>" → strip
    (re.compile(
        r"On non-Claude-Code platforms, fall back:\s+`npx playwright[^`]*`\.",
        re.IGNORECASE),
     r"The driver is selected automatically by `gpowers-browser` (see drivers/browser/select-driver.sh)."),
]


def rewrite(text: str) -> str:
    for pat, repl in RULES:
        text = pat.sub(repl, text)
    return text


def main() -> int:
    src = sys.stdin.read()
    sys.stdout.write(rewrite(src))
    if not src.endswith("\n"):
        sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
