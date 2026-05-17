package commands

import (
	"fmt"
	"strconv"
	"strings"
)

func (r *Registry) registerTabs() {
	r.Register("tabs", CommandDesc{Category: "Tabs", Description: "List open tabs"},
		func(ctx *ExecContext) (string, error) {
			list := ctx.BM.TabList()
			if len(list) == 0 {
				return "No tabs open", nil
			}
			lines := make([]string, len(list))
			for i, t := range list {
				marker := ""
				if t.Active {
					marker = " *"
				}
				title := t.Title
				if title == "" {
					title = "(no title)"
				}
				lines[i] = fmt.Sprintf("[%d]%s %s — %s", t.ID, marker, title, t.URL)
			}
			return strings.Join(lines, "\n"), nil
		})

	r.Register("tab", CommandDesc{Category: "Tabs", Description: "Switch to tab", Usage: "tab <id>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: tab <id>")
			}
			id, err := strconv.Atoi(ctx.Args[0])
			if err != nil {
				return "", fmt.Errorf("invalid tab id: %s", ctx.Args[0])
			}
			if err := ctx.BM.SwitchTab(id); err != nil {
				return "", err
			}
			return fmt.Sprintf("Switched to tab %d", id), nil
		})

	r.Register("newtab", CommandDesc{Category: "Tabs", Description: "Open new tab", Usage: "newtab [url] [--json]"},
		func(ctx *ExecContext) (string, error) {
			var url string
			jsonOut := false
			for _, a := range ctx.Args {
				if a == "--json" {
					jsonOut = true
				} else if url == "" {
					url = a
				}
			}
			id, err := ctx.BM.NewTab(url)
			if err != nil {
				return "", err
			}
			if jsonOut {
				return fmt.Sprintf(`{"tabId":%d,"url":%q}`, id, ctx.BM.TabURL(id)), nil
			}
			return fmt.Sprintf("Opened tab %d", id), nil
		})

	r.Register("closetab", CommandDesc{Category: "Tabs", Description: "Close tab", Usage: "closetab [id]"},
		func(ctx *ExecContext) (string, error) {
			id := 0
			if len(ctx.Args) > 0 {
				var err error
				id, err = strconv.Atoi(ctx.Args[0])
				if err != nil {
					return "", fmt.Errorf("invalid tab id: %s", ctx.Args[0])
				}
			}
			if err := ctx.BM.CloseTab(id); err != nil {
				return "", err
			}
			return fmt.Sprintf("Closed tab %d", id), nil
		})
}
