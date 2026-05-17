package commands

import (
	"fmt"
	"strings"
	"time"
)

func (r *Registry) registerServer() {
	r.Register("status", CommandDesc{Category: "Server", Description: "Health check"},
		func(ctx *ExecContext) (string, error) {
			healthy := ctx.BM.IsHealthy()
			url := ctx.BM.CurrentURL()
			tabs := ctx.BM.TabCount()
			status := "healthy"
			if !healthy {
				status = "unhealthy"
			}
			if url == "" || url == "about:blank" {
				url = "(none)"
			}
			return fmt.Sprintf("Status: %s\nURL: %s\nTabs: %d", status, url, tabs), nil
		})

	r.Register("stop", CommandDesc{Category: "Server", Description: "Shutdown server"},
		func(ctx *ExecContext) (string, error) {
			// Actual shutdown is handled by the HTTP server layer
			return "Shutting down...", nil
		})

	r.Register("help", CommandDesc{Category: "Server", Description: "List all commands", Usage: "help [command]"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) > 0 {
				name := Canonicalize(ctx.Args[0])
				desc, ok := r.GetDesc(name)
				if !ok {
					return "", fmt.Errorf("unknown command: %s", name)
				}
				usage := desc.Usage
				if usage == "" {
					usage = name
				}
				return fmt.Sprintf("%s (%s)\n  %s\n  Usage: %s", name, desc.Category, desc.Description, usage), nil
			}

			cmds := r.ListCommands()
			var out strings.Builder
			out.WriteString("Available commands:\n")
			var lastCat string
			for _, c := range cmds {
				if c.Desc.Category != lastCat {
					out.WriteString(fmt.Sprintf("\n  %s:\n", c.Desc.Category))
					lastCat = c.Desc.Category
				}
				line := fmt.Sprintf("    %-15s %s", c.Name, c.Desc.Description)
				if c.Desc.Usage != "" {
					line += fmt.Sprintf(" (%s)", c.Desc.Usage)
				}
				out.WriteString(line + "\n")
			}
			return out.String(), nil
		})

	r.Register("restart", CommandDesc{Category: "Server", Description: "Restart the daemon"},
		func(ctx *ExecContext) (string, error) {
			// Signal shutdown; the CLI wrapper will restart the daemon
			go func() {
				time.Sleep(100 * time.Millisecond)
				_ = ctx.BM.Close()
			}()
			return "Restarting...", nil
		})
}

// StopRequested is a sentinel error used by the stop command handler
// to signal the HTTP server that it should initiate shutdown.
type StopRequested struct{}

func (StopRequested) Error() string { return "stop requested" }
