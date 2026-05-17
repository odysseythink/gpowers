package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"browse-go/pkg/browserskill"
	"browse-go/pkg/config"
	"browse-go/pkg/skilltoken"
)

const (
	defaultSkillTimeout = 60
	maxSkillStdout      = 1024 * 1024 // 1 MB
)

func (r *Registry) registerBrowserSkill() {
	r.Register("skill", CommandDesc{
		Category:    "System",
		Description: "Browser skills — per-task deterministic scripts",
		Usage:       "skill <list|show|run|test|rm> [args...]",
	}, func(ctx *ExecContext) (string, error) {
		return handleBrowserSkill(ctx)
	})
}

func handleBrowserSkill(ctx *ExecContext) (string, error) {
	if len(ctx.Args) == 0 {
		return formatBrowserSkillUsage(), nil
	}

	sub := ctx.Args[0]
	rest := ctx.Args[1:]

	switch sub {
	case "list":
		return handleBrowserSkillList()
	case "show":
		return handleBrowserSkillShow(rest)
	case "run":
		return handleBrowserSkillRun(ctx, rest)
	case "test":
		return handleBrowserSkillTest(ctx, rest)
	case "rm":
		return handleBrowserSkillRm(rest)
	case "help", "--help":
		return formatBrowserSkillUsage(), nil
	default:
		return "", fmt.Errorf("unknown skill subcommand: %s\n%s", sub, formatBrowserSkillUsage())
	}
}

func formatBrowserSkillUsage() string {
	return strings.Join([]string{
		"skill — browser skills (per-task deterministic scripts)",
		"",
		"Subcommands:",
		"  list              List all skills with resolved tier",
		"  show <name>       Print SKILL.md",
		"  run <name> [--arg k=v]... [--timeout=Ns]  Run the skill script",
		"  test <name>       Run the skill test entrypoint",
		"  rm <name> [--global]  Tombstone a user-tier skill",
	}, "\n")
}

func resolveTierPaths() browserskill.TierPaths {
	projectRoot := config.GitRoot()
	if projectRoot == "" {
		projectRoot, _ = os.Getwd()
	}
	home := config.Home()
	return browserskill.TierPaths{
		Project: filepath.Join(projectRoot, ".gstack", "browser-skills"),
		Global:  filepath.Join(home, "browser-skills"),
		Bundled: filepath.Join(home, "..", "browser-skills"),
	}
}

func handleBrowserSkillList() (string, error) {
	tierPaths := resolveTierPaths()
	skills := browserskill.List(tierPaths)
	if len(skills) == 0 {
		return "No browser-skills found.\n", nil
	}

	lines := []string{"NAME                          TIER     HOST                        DESC"}
	for _, s := range skills {
		desc := s.Frontmatter.Description
		if len(desc) > 40 {
			desc = desc[:40]
		}
		lines = append(lines, fmt.Sprintf("%-30s %-8s %-28s %s",
			s.Name, s.Tier, s.Frontmatter.Host, desc))
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func handleBrowserSkillShow(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("Usage: $B skill show <name>")
	}
	name := args[0]
	tierPaths := resolveTierPaths()
	skill, err := browserskill.Read(name, tierPaths)
	if err != nil {
		return "", err
	}
	mdPath := filepath.Join(skill.Dir, "SKILL.md")
	data, err := os.ReadFile(mdPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ─── run ────────────────────────────────────────────────────────

type skillRunArgs struct {
	passthrough    []string
	timeoutSeconds int
}

func parseSkillRunArgs(args []string) skillRunArgs {
	var out skillRunArgs
	out.timeoutSeconds = defaultSkillTimeout
	for _, a := range args {
		if strings.HasPrefix(a, "--timeout=") {
			if n, err := strconv.Atoi(a[len("--timeout="):]); err == nil && n > 0 {
				out.timeoutSeconds = n
			}
			continue
		}
		out.passthrough = append(out.passthrough, a)
	}
	return out
}

type spawnResult struct {
	stdout    string
	stderr    string
	exitCode  int
	timedOut  bool
	truncated bool
}

func handleBrowserSkillRun(ctx *ExecContext, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("Usage: $B skill run <name> [--arg k=v]... [--timeout=Ns]")
	}
	name := args[0]
	tierPaths := resolveTierPaths()
	skill, err := browserskill.Read(name, tierPaths)
	if err != nil {
		return "", err
	}

	runArgs := parseSkillRunArgs(args[1:])
	result, err := spawnSkill(spawnOpts{
		skill:          skill,
		skillArgs:      runArgs.passthrough,
		trusted:        skill.Frontmatter.Trusted,
		timeoutSeconds: runArgs.timeoutSeconds,
		port:           ctx.Port,
	})
	if err != nil {
		return "", err
	}

	if result.exitCode != 0 || result.timedOut || result.truncated {
		summary := ""
		switch {
		case result.truncated:
			summary = fmt.Sprintf("truncated stdout at %d bytes", maxSkillStdout)
		case result.timedOut:
			summary = fmt.Sprintf("timed out after %ds", runArgs.timeoutSeconds)
		default:
			summary = fmt.Sprintf("exit %d", result.exitCode)
		}
		stderrPreview := result.stderr
		if len(stderrPreview) > 4096 {
			stderrPreview = stderrPreview[:4096]
		}
		return "", fmt.Errorf("skill %q failed: %s\n--- stderr ---\n%s", name, summary, stderrPreview)
	}
	return result.stdout, nil
}

// spawnOpts configures a skill spawn.
type spawnOpts struct {
	skill          *browserskill.Skill
	skillArgs      []string
	trusted        bool
	timeoutSeconds int
	port           int
}

func spawnSkill(opts spawnOpts) (*spawnResult, error) {
	spawnID := generateSpawnID()
	tokenInfo := skilltoken.Mint(opts.skill.Name, spawnID, time.Duration(opts.timeoutSeconds)*time.Second, "")

	defer func() {
		skilltoken.Revoke(opts.skill.Name, spawnID)
	}()

	execPath, interpreter, err := browserskill.FindExecutable(opts.skill.Dir)
	if err != nil {
		return nil, err
	}

	env := buildSpawnEnv(buildEnvOpts{
		trusted:    opts.trusted,
		port:       opts.port,
		skillToken: tokenInfo.Token,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(opts.timeoutSeconds)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if interpreter == "" {
		cmd = exec.CommandContext(ctx, execPath)
		cmd.Args = append(cmd.Args, opts.skillArgs...)
	} else if interpreter == "go run" {
		cmd = exec.CommandContext(ctx, "go", append([]string{"run", execPath}, opts.skillArgs...)...)
	} else {
		parts := strings.Fields(interpreter)
		cmd = exec.CommandContext(ctx, parts[0], append(append(parts[1:], execPath), opts.skillArgs...)...)
	}
	cmd.Dir = opts.skill.Dir
	cmd.Env = env

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("spawn stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("spawn stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("spawn start: %w", err)
	}

	stdoutResult := readCapped(stdoutPipe, maxSkillStdout)
	stderrResult := readCapped(stderrPipe, maxSkillStdout)

	err = cmd.Wait()
	timedOut := ctx.Err() == context.DeadlineExceeded

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	if timedOut {
		exitCode = 124
	}

	return &spawnResult{
		stdout:    stdoutResult.text,
		stderr:    stderrResult.text,
		exitCode:  exitCode,
		timedOut:  timedOut,
		truncated: stdoutResult.truncated,
	}, nil
}

func generateSpawnID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type cappedRead struct {
	text      string
	truncated bool
}

func readCapped(r io.Reader, capBytes int) cappedRead {
	lr := &io.LimitedReader{R: r, N: int64(capBytes)}
	data, err := io.ReadAll(lr)
	truncated := err == nil && lr.N == 0
	return cappedRead{text: string(data), truncated: truncated}
}

// ─── env construction (security-critical) ───────────────────────

type buildEnvOpts struct {
	trusted    bool
	port       int
	skillToken string
}

var secretKeyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)TOKEN`),
	regexp.MustCompile(`(?i)KEY`),
	regexp.MustCompile(`(?i)SECRET`),
	regexp.MustCompile(`(?i)PASSWORD`),
	regexp.MustCompile(`(?i)CREDENTIAL`),
	regexp.MustCompile(`^AWS_`),
	regexp.MustCompile(`^AZURE_`),
	regexp.MustCompile(`^GCP_`),
	regexp.MustCompile(`^GOOGLE_APPLICATION_`),
	regexp.MustCompile(`^ANTHROPIC_`),
	regexp.MustCompile(`^OPENAI_`),
	regexp.MustCompile(`^GITHUB_`),
	regexp.MustCompile(`^GH_`),
	regexp.MustCompile(`^SSH_`),
	regexp.MustCompile(`^GPG_`),
	regexp.MustCompile(`^NPM_TOKEN`),
	regexp.MustCompile(`^PYPI_`),
}

var untrustedAllowlist = map[string]bool{
	"LANG": true, "LC_ALL": true, "LC_CTYPE": true,
	"TERM": true,
	"TZ":   true,
}

func buildSpawnEnv(opts buildEnvOpts) []string {
	out := make(map[string]string)

	if opts.trusted {
		for _, e := range os.Environ() {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) != 2 {
				continue
			}
			k := parts[0]
			if k == "GSTACK_TOKEN" {
				continue // never propagate root token
			}
			out[k] = parts[1]
		}
		if out["PATH"] == "" {
			out["PATH"] = "/usr/local/bin:/usr/bin:/bin"
		}
	} else {
		for k := range untrustedAllowlist {
			if v := os.Getenv(k); v != "" {
				out[k] = v
			}
		}
		out["PATH"] = resolveMinimalPath()
	}

	if !opts.trusted {
		for k := range out {
			for _, p := range secretKeyPatterns {
				if p.MatchString(k) {
					delete(out, k)
					break
				}
			}
		}
	}

	// Inject daemon connection (always last so callers cannot override)
	out["GSTACK_PORT"] = strconv.Itoa(opts.port)
	out["GSTACK_SKILL_TOKEN"] = opts.skillToken

	var env []string
	for k, v := range out {
		env = append(env, k+"="+v)
	}
	return env
}

func resolveMinimalPath() string {
	fallback := "/usr/local/bin:/usr/bin:/bin"
	if goPath, err := exec.LookPath("go"); err == nil {
		return filepath.Dir(goPath) + ":" + fallback
	}
	return fallback
}

// ─── test ───────────────────────────────────────────────────────

func handleBrowserSkillTest(ctx *ExecContext, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("Usage: $B skill test <name>")
	}
	name := args[0]
	tierPaths := resolveTierPaths()
	skill, err := browserskill.Read(name, tierPaths)
	if err != nil {
		return "", err
	}

	execPath, interpreter, err := browserskill.FindTestExecutable(skill.Dir)
	if err != nil {
		return "", err
	}

	ctxBg, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if interpreter == "" {
		cmd = exec.CommandContext(ctxBg, execPath)
	} else if interpreter == "go test" {
		cmd = exec.CommandContext(ctxBg, "go", "test", execPath)
	} else {
		parts := strings.Fields(interpreter)
		cmd = exec.CommandContext(ctxBg, parts[0], append(parts[1:], execPath)...)
	}
	cmd.Dir = skill.Dir
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			_ = exitErr.ExitCode()
		}
		return "", fmt.Errorf("skill %q tests failed.\n%s", name, output)
	}
	if output == "" {
		return fmt.Sprintf("tests passed for %q", name), nil
	}
	return output, nil
}

func handleBrowserSkillRm(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("Usage: $B skill rm <name> [--global]")
	}
	name := args[0]
	tier := browserskill.TierProject
	for _, a := range args[1:] {
		if a == "--global" {
			tier = browserskill.TierGlobal
		}
	}
	tierPaths := resolveTierPaths()
	if err := browserskill.Tombstone(name, tier, tierPaths); err != nil {
		return "", err
	}
	return fmt.Sprintf("Tombstoned %s (%s tier).", name, tier), nil
}
