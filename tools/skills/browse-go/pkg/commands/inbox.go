package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"browse-go/pkg/config"
	"browse-go/pkg/security"
)

func (r *Registry) registerInbox() {
	r.Register("inbox", CommandDesc{Category: "Meta", Description: "Read sidebar inbox messages", Usage: "inbox [--clear]"},
		func(ctx *ExecContext) (string, error) {
			gitRoot := config.GitRoot()
			if gitRoot == "" {
				return "Not in a git repository — cannot locate inbox.", nil
			}

			inboxDir := filepath.Join(gitRoot, ".context", "sidebar-inbox")
			entries, err := os.ReadDir(inboxDir)
			if err != nil {
				if os.IsNotExist(err) {
					return "Inbox empty.", nil
				}
				return "", fmt.Errorf("cannot read inbox: %w", err)
			}

			var files []string
			for _, e := range entries {
				name := e.Name()
				if strings.HasSuffix(name, ".json") && !strings.HasPrefix(name, ".") {
					files = append(files, name)
				}
			}
			if len(files) == 0 {
				return "Inbox empty.", nil
			}

			// Sort newest first (filenames are timestamps)
			for i := 0; i < len(files)-1; i++ {
				for j := i + 1; j < len(files); j++ {
					if files[i] < files[j] {
						files[i], files[j] = files[j], files[i]
					}
				}
			}

			type inboxMsg struct {
				Timestamp   string `json:"timestamp"`
				URL         string `json:"url"`
				UserMessage string `json:"userMessage"`
			}
			var messages []inboxMsg
			for _, f := range files {
				data, err := os.ReadFile(filepath.Join(inboxDir, f))
				if err != nil {
					continue
				}
				var raw struct {
					Timestamp   string `json:"timestamp"`
					Page        struct {
						URL string `json:"url"`
					} `json:"page"`
					UserMessage string `json:"userMessage"`
				}
				if err := json.Unmarshal(data, &raw); err != nil {
					continue
				}
				messages = append(messages, inboxMsg{
					Timestamp:   raw.Timestamp,
					URL:         raw.Page.URL,
					UserMessage: raw.UserMessage,
				})
			}

			if len(messages) == 0 {
				return "Inbox empty.", nil
			}

			var lines []string
			lines = append(lines, fmt.Sprintf("SIDEBAR INBOX (%d message%s)", len(messages), plural(len(messages))))
			lines = append(lines, "────────────────────────────────")
			for _, msg := range messages {
				ts := msg.Timestamp
				if ts == "" {
					ts = "[unknown]"
				} else {
					ts = "[" + ts + "]"
				}
				lines = append(lines, fmt.Sprintf("%s %s", ts, security.WrapUntrusted(msg.URL, "inbox-url")))
				lines = append(lines, fmt.Sprintf("  \"%s\"", security.WrapUntrusted(msg.UserMessage, "inbox-message")))
				lines = append(lines, "")
			}
			lines = append(lines, "────────────────────────────────")

			// Handle --clear
			clear := false
			for _, a := range ctx.Args {
				if a == "--clear" {
					clear = true
					break
				}
			}
			if clear {
				cleared := 0
				for _, f := range files {
					if err := os.Remove(filepath.Join(inboxDir, f)); err == nil {
						cleared++
					}
				}
				lines = append(lines, fmt.Sprintf("Cleared %d message%s.", cleared, plural(cleared)))
			}

			return strings.Join(lines, "\n"), nil
		})
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}


