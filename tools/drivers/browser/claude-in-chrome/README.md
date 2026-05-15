# claude-in-chrome driver

Maps gpowers 9-verb interface to Claude Code MCP server `claude-in-chrome` tools.

| gpowers verb | MCP tool |
|---|---|
| open | mcp__claude-in-chrome__tabs_create_mcp + resize_window |
| click | mcp__claude-in-chrome__click (via computer or find+click) |
| type | mcp__claude-in-chrome__form_input |
| read | mcp__claude-in-chrome__read_page (mode=text/dom) / read_console_messages (mode=console) |
| screenshot | mcp__claude-in-chrome__computer (action=screenshot) |
| wait | mcp__claude-in-chrome__find (with retry) |
| eval | mcp__claude-in-chrome__javascript_tool |
| cookies | mcp__claude-in-chrome__javascript_tool (document.cookie shim) |
| close | mcp__claude-in-chrome__tabs_close_mcp + tab_release |

Verb scripts emit a `GPOWERS_MCP_INSTRUCTION:` line on stderr that the agent reads and translates to the appropriate MCP tool call. The stdout payload is the verb's return value.

For automated tests (no live MCP), set `GPOWERS_BROWSER_MOCK=1` — scripts will skip the instruction emission and return canned success values.
