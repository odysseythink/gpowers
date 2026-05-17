package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"browse-go/pkg/browserskill"
	"browse-go/pkg/skilltoken"
)

func TestParseSkillRunArgs(t *testing.T) {
	cases := []struct {
		input    []string
		wantArgs []string
		wantTo   int
	}{
		{[]string{"--timeout=30"}, []string{}, 30},
		{[]string{"--timeout=120", "--foo", "bar"}, []string{"--foo", "bar"}, 120},
		{[]string{"a", "b", "--timeout=10"}, []string{"a", "b"}, 10},
	}
	for _, c := range cases {
		got := parseSkillRunArgs(c.input)
		if len(got.passthrough) != len(c.wantArgs) {
			t.Errorf("passthrough mismatch for %v", c.input)
		}
		if got.timeoutSeconds != c.wantTo {
			t.Errorf("timeout mismatch: got %d, want %d", got.timeoutSeconds, c.wantTo)
		}
	}
}

func TestBuildSpawnEnvTrusted(t *testing.T) {
	env := buildSpawnEnv(buildEnvOpts{trusted: true, port: 12345, skillToken: "sk_test"})
	m := envMap(env)
	if m["GSTACK_PORT"] != "12345" {
		t.Errorf("GSTACK_PORT = %s, want 12345", m["GSTACK_PORT"])
	}
	if m["GSTACK_SKILL_TOKEN"] != "sk_test" {
		t.Error("GSTACK_SKILL_TOKEN mismatch")
	}
	if m["PATH"] == "" {
		t.Error("trusted env should have PATH")
	}
}

func TestBuildSpawnEnvUntrusted(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "secret123")
	t.Setenv("LANG", "en_US.UTF-8")
	env := buildSpawnEnv(buildEnvOpts{trusted: false, port: 12345, skillToken: "sk_test"})
	m := envMap(env)
	if m["GITHUB_TOKEN"] != "" {
		t.Error("untrusted env should strip GITHUB_TOKEN")
	}
	if m["LANG"] != "en_US.UTF-8" {
		t.Error("untrusted env should keep LANG")
	}
	if m["GSTACK_PORT"] != "12345" {
		t.Error("GSTACK_PORT mismatch")
	}
}

func TestSpawnSkillMissingExecutable(t *testing.T) {
	skilltoken.Reset()
	tmp := t.TempDir()
	skill := &browserskill.Skill{
		Name: "no-exec",
		Dir:  tmp,
		Frontmatter: browserskill.Frontmatter{
			Trusted: false,
		},
	}
	_, err := spawnSkill(spawnOpts{
		skill:          skill,
		skillArgs:      nil,
		trusted:        false,
		timeoutSeconds: 5,
		port:           9999,
	})
	if err == nil || !strings.Contains(err.Error(), "no executable found") {
		t.Fatalf("expected 'no executable found' error, got: %v", err)
	}
}

func TestSpawnSkillSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	skilltoken.Reset()
	tmp := t.TempDir()
	script := `#!/bin/sh
echo "hello from skill"
echo "$GSTACK_PORT"
echo "$GSTACK_SKILL_TOKEN" | cut -c1-3`
	scriptPath := filepath.Join(tmp, "run")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	skill := &browserskill.Skill{
		Name: "test-skill",
		Dir:  tmp,
		Frontmatter: browserskill.Frontmatter{
			Trusted: false,
		},
	}
	result, err := spawnSkill(spawnOpts{
		skill:          skill,
		skillArgs:      nil,
		trusted:        false,
		timeoutSeconds: 5,
		port:           9999,
	})
	if err != nil {
		t.Fatalf("spawn error: %v", err)
	}
	if result.exitCode != 0 {
		t.Fatalf("unexpected exit code: %d, stderr: %s", result.exitCode, result.stderr)
	}
	lines := strings.Split(strings.TrimSpace(result.stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), result.stdout)
	}
	if lines[0] != "hello from skill" {
		t.Errorf("line 0 = %q", lines[0])
	}
	if lines[1] != "9999" {
		t.Errorf("GSTACK_PORT = %q", lines[1])
	}
	if lines[2] != "sk_" {
		t.Errorf("token prefix = %q", lines[2])
	}
	if result.timedOut {
		t.Error("should not have timed out")
	}
}

func TestSpawnSkillTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	skilltoken.Reset()
	tmp := t.TempDir()
	script := `#!/bin/sh
exec sleep 10`
	scriptPath := filepath.Join(tmp, "run")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	skill := &browserskill.Skill{
		Name: "timeout-skill",
		Dir:  tmp,
		Frontmatter: browserskill.Frontmatter{
			Trusted: false,
		},
	}
	result, err := spawnSkill(spawnOpts{
		skill:          skill,
		skillArgs:      nil,
		trusted:        false,
		timeoutSeconds: 1,
		port:           9999,
	})
	if err != nil {
		t.Fatalf("spawn error: %v", err)
	}
	if !result.timedOut {
		t.Error("expected timed out")
	}
	if result.exitCode != 124 {
		t.Errorf("expected exit code 124, got %d", result.exitCode)
	}
}

func TestSpawnSkillStdoutCap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	skilltoken.Reset()
	tmp := t.TempDir()
	script := `#!/bin/sh
python3 -c "print('x'*2000000)" 2>/dev/null || perl -e 'print "x" x 2000000'`
	scriptPath := filepath.Join(tmp, "run")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	skill := &browserskill.Skill{
		Name: "cap-skill",
		Dir:  tmp,
		Frontmatter: browserskill.Frontmatter{
			Trusted: false,
		},
	}
	result, err := spawnSkill(spawnOpts{
		skill:          skill,
		skillArgs:      nil,
		trusted:        false,
		timeoutSeconds: 5,
		port:           9999,
	})
	if err != nil {
		t.Fatalf("spawn error: %v", err)
	}
	if !result.truncated {
		t.Error("expected truncated stdout")
	}
	if len(result.stdout) > maxSkillStdout+100 {
		t.Errorf("stdout too large: %d bytes", len(result.stdout))
	}
}

func envMap(env []string) map[string]string {
	m := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}
