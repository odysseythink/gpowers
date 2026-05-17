package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"browse-go/pkg/config"
	"browse-go/pkg/domainskill"
	"browse-go/pkg/telemetry"
)

func (r *Registry) registerDomainSkill() {
	r.Register("domain-skill", CommandDesc{
		Category:    "System",
		Description: "Per-site notes the agent writes for itself",
		Usage:       "domain-skill <save|list|show|promote|rollback|rm> [args...]",
	}, func(ctx *ExecContext) (string, error) {
		return handleDomainSkill(ctx)
	})
}

func readFileSafe(path string) ([]byte, error) {
	// Validate path is absolute or within cwd to prevent traversal
	if filepath.IsAbs(path) {
		return os.ReadFile(path)
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	full := filepath.Join(wd, path)
	return os.ReadFile(full)
}

func handleDomainSkill(ctx *ExecContext) (string, error) {
	if len(ctx.Args) == 0 {
		return formatDomainSkillUsage(), nil
	}

	sub := ctx.Args[0]
	rest := ctx.Args[1:]
	slug := config.RemoteSlug()

	switch sub {
	case "save":
		return handleDomainSkillSave(rest, ctx)
	case "list":
		return handleDomainSkillList(slug)
	case "show":
		return handleDomainSkillShow(rest, slug)
	case "promote-to-global":
		return handleDomainSkillPromote(rest, slug)
	case "rollback":
		return handleDomainSkillRollback(rest, slug)
	case "rm", "remove", "delete":
		return handleDomainSkillDelete(rest, slug)
	case "help", "--help":
		return formatDomainSkillUsage(), nil
	default:
		return "", fmt.Errorf("unknown domain-skill subcommand: %s\n%s", sub, formatDomainSkillUsage())
	}
}

func formatDomainSkillUsage() string {
	return strings.Join([]string{
		"domain-skill — agent-authored per-site notes",
		"",
		"Subcommands:",
		"  save              save body from stdin or --from-file (host from active tab)",
		"  list              list all skills visible to current project",
		"  show <host>       print skill body",
		"  promote-to-global <host>  promote active skill to global scope",
		"  rollback <host> [--global]  restore prior version",
		"  rm <host> [--global]  tombstone",
	}, "\n")
}

func handleDomainSkillSave(args []string, ctx *ExecContext) (string, error) {
	var body string
	fromFileIdx := -1
	for i, a := range args {
		if a == "--from-file" && i+1 < len(args) {
			fromFileIdx = i
			break
		}
	}
	if fromFileIdx >= 0 {
		data, err := readFileSafe(args[fromFileIdx+1])
		if err != nil {
			return "", fmt.Errorf("cannot read --from-file: %w", err)
		}
		body = string(data)
	}
	if body == "" {
		return "", fmt.Errorf("save failed: empty body\n" +
			"Cause: no content provided via --from-file or stdin.\n" +
			"Action: pipe markdown into $B domain-skill save, or pass --from-file <path>.")
	}

	// Derive host from active tab
	var url string
	if ctx.BM != nil {
		url = ctx.BM.CurrentURL()
	}
	host, err := domainskill.DeriveHostFromURL(url)
	if err != nil {
		return "", err
	}

	row, err := domainskill.Write(domainskill.WriteInput{
		Host:        host,
		Body:        body,
		ProjectSlug: config.RemoteSlug(),
		Source:      "agent",
	})
	if err != nil {
		return "", err
	}

	telemetry.Log(telemetry.Event{"event": "domain_skill_saved", "host": row.Host, "scope": string(row.Scope), "state": string(row.State), "bytes": len(row.Body)})
	return fmt.Sprintf(
		"Saved (state: %s, scope: project).\n"+
			"Host: %s\n"+
			"Bytes: %d\n"+
			"Version: %d\n"+
			"Stored at: ~/.gstack/projects/%s/learnings.jsonl\n"+
			"\n"+
			"Next: skill is quarantined and won't fire in prompts until used 3 times\n"+
			"      without classifier flags. Run $B domain-skill list to see state.",
		row.State, row.Host, len(row.Body), row.Version, config.RemoteSlug(),
	), nil
}

func handleDomainSkillList(slug string) (string, error) {
	list, err := domainskill.List(slug)
	if err != nil {
		return "", err
	}
	if len(list.Project) == 0 && len(list.Global) == 0 {
		return "No domain-skills yet.\n\n" +
			"Next: navigate to a site, then $B domain-skill save with a markdown body to begin.", nil
	}

	var lines []string
	if len(list.Project) > 0 {
		lines = append(lines, "Project (per-project):")
		for _, r := range list.Project {
			lines = append(lines, fmt.Sprintf("  [%s] %s — v%d, %d bytes, used %d× (%d flags)",
				r.State, r.Host, r.Version, len(r.Body), r.UseCount, r.FlagCount))
		}
	}
	if len(list.Global) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Global (cross-project):")
		for _, r := range list.Global {
			lines = append(lines, fmt.Sprintf("  %s — v%d, %d bytes", r.Host, r.Version, len(r.Body)))
		}
	}
	return strings.Join(lines, "\n"), nil
}

func handleDomainSkillShow(args []string, slug string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("Usage: $B domain-skill show <host>")
	}
	host := args[0]
	result, err := domainskill.Read(host, slug)
	if err != nil {
		return "", err
	}
	if result == nil {
		return fmt.Sprintf("No active skill for %s.\n\nA quarantined skill may exist; run $B domain-skill list to see all states.", host), nil
	}
	return fmt.Sprintf("# %s (%s scope, %s)\n# version: %d, used: %d×, flags: %d\n\n%s",
		result.Row.Host, result.Source, result.Row.State,
		result.Row.Version, result.Row.UseCount, result.Row.FlagCount,
		result.Row.Body), nil
}

func handleDomainSkillPromote(args []string, slug string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("Usage: $B domain-skill promote-to-global <host>")
	}
	row, err := domainskill.PromoteToGlobal(args[0], slug)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Promoted %s to global scope (v%d).\n\n"+
		"This skill now fires for all projects unless they have a per-project skill for the same host.",
		row.Host, row.Version), nil
}

func handleDomainSkillRollback(args []string, slug string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("Usage: $B domain-skill rollback <host> [--global]")
	}
	scope := domainskill.ScopeProject
	for _, a := range args {
		if a == "--global" {
			scope = domainskill.ScopeGlobal
		}
	}
	row, err := domainskill.Rollback(args[0], slug, scope)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Rolled back %s (%s scope) to prior version.\nNew version: %d",
		row.Host, scope, row.Version), nil
}

func handleDomainSkillDelete(args []string, slug string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("Usage: $B domain-skill rm <host> [--global]")
	}
	scope := domainskill.ScopeProject
	for _, a := range args {
		if a == "--global" {
			scope = domainskill.ScopeGlobal
		}
	}
	if err := domainskill.Delete(args[0], slug, scope); err != nil {
		return "", err
	}
	return fmt.Sprintf("Tombstoned %s (%s scope). Use $B domain-skill rollback to restore.", args[0], scope), nil
}
