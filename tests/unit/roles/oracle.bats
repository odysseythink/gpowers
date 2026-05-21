#!/usr/bin/env bats
# Install regression for roles/skills/oracle

setup() {
  REPO="$(cd "$(dirname "$BATS_TEST_FILENAME")/../../.." && pwd)"
}

@test "oracle SKILL.md exists with valid frontmatter" {
  [ -f "$REPO/roles/skills/oracle/SKILL.md" ]
  python3 -c "
import yaml
with open('$REPO/roles/skills/oracle/SKILL.md') as f:
    lines = f.readlines()
start = end = None
for i, line in enumerate(lines):
    if line.strip() == '---':
        if start is None: start = i
        elif end is None: end = i; break
yaml.safe_load(''.join(lines[start+1:end]))
"
}

@test "oracle SKILL.md contains Mode Detection block" {
  grep -q "Mode Detection" "$REPO/roles/skills/oracle/SKILL.md"
  grep -q "Ultrawork Verifier mode" "$REPO/roles/skills/oracle/SKILL.md"
}

@test "oracle SKILL.md contains Three-Tier Response" {
  grep -q "Three-Tier Response" "$REPO/roles/skills/oracle/SKILL.md"
  grep -q "Bottom line" "$REPO/roles/skills/oracle/SKILL.md"
  grep -q "Action plan" "$REPO/roles/skills/oracle/SKILL.md"
  grep -q "Effort" "$REPO/roles/skills/oracle/SKILL.md"
  grep -q "Confidence" "$REPO/roles/skills/oracle/SKILL.md"
}

@test "oracle SKILL.md contains Tool Discipline forbidding Edit/Write/Agent" {
  grep -q "Tool Discipline" "$REPO/roles/skills/oracle/SKILL.md"
  grep -E "must NOT use:.*Edit.*Write.*Agent" "$REPO/roles/skills/oracle/SKILL.md"
}

@test "oracle SKILL.md contains AI-slop deny-list" {
  grep -q "AI-Slop Deny-List" "$REPO/roles/skills/oracle/SKILL.md"
  for w in comprehensive robust leverages powerful seamlessly enterprise-grade; do
    grep -q "$w" "$REPO/roles/skills/oracle/SKILL.md"
  done
}

@test "oracle SKILL.md contains both invocation examples" {
  grep -q "standalone advisor" "$REPO/roles/skills/oracle/SKILL.md"
  grep -q "Ultrawork verifier" "$REPO/roles/skills/oracle/SKILL.md"
}

@test "oracle ACKNOWLEDGEMENTS.md exists and credits oh-my-opencode" {
  [ -f "$REPO/roles/skills/oracle/ACKNOWLEDGEMENTS.md" ]
  grep -q "oh-my-opencode" "$REPO/roles/skills/oracle/ACKNOWLEDGEMENTS.md"
  grep -q "v3.17.10" "$REPO/roles/skills/oracle/ACKNOWLEDGEMENTS.md"
}

@test "oracle test scenarios file contains P-A, P-B, P-L" {
  f="$REPO/roles/skills/oracle/tests/scenarios.md"
  [ -f "$f" ]
  grep -q "P-A" "$f"
  grep -q "P-B" "$f"
  grep -q "P-L" "$f"
}

@test "ultrawork oracle.md is now a pointer at roles/skills/oracle/SKILL.md" {
  f="$REPO/core/skills/ultrawork/oracle.md"
  [ -f "$f" ]
  grep -q "roles/skills/oracle/SKILL.md" "$f"
  [ "$(wc -l < "$f")" -lt 80 ]
}

@test "ultrawork kimi oracle.yaml system_prompt_path points at roles/skills/oracle/SKILL.md" {
  f="$REPO/core/skills/ultrawork/platforms/kimi/oracle.yaml"
  [ -f "$f" ]
  grep -q "roles/skills/oracle/SKILL.md" "$f"
}

@test "platforms/kimi/agent.yaml exists and registers oracle subagent" {
  f="$REPO/platforms/kimi/agent.yaml"
  [ -f "$f" ]
  python3 -c "import yaml; yaml.safe_load(open('$f'))"
  grep -q "oracle:" "$f"
  grep -q "./oracle.yaml" "$f"
}

@test "platforms/kimi/oracle.yaml exists and excludes write/agent tools" {
  f="$REPO/platforms/kimi/oracle.yaml"
  [ -f "$f" ]
  python3 -c "import yaml; yaml.safe_load(open('$f'))"
  grep -q "WriteFile" "$f"
  grep -q "StrReplaceFile" "$f"
  grep -q "Agent" "$f"
  grep -q "AskUserQuestion" "$f"
}

@test "platforms/opencode/adapters/gpowers-oracle/agent.yaml exists and points at roles SKILL" {
  f="$REPO/platforms/opencode/adapters/gpowers-oracle/agent.yaml"
  [ -f "$f" ]
  python3 -c "import yaml; yaml.safe_load(open('$f'))"
  grep -q "roles/skills/oracle/SKILL.md" "$f"
}

@test "docs/methodology/executor-patterns.md exists and documents the 6 patterns" {
  f="$REPO/docs/methodology/executor-patterns.md"
  [ -f "$f" ]
  for p in Orchestration "Deep Work" Plan-Only "External Research" "Internal Codebase Search" "Independent Verification"; do
    grep -q "$p" "$f"
  done
}

@test "roles/upstream-source.json valid JSON with personas entry" {
  f="$REPO/roles/upstream-source.json"
  [ -f "$f" ]
  python3 -m json.tool "$f" > /dev/null
  python3 -c "import json; d=json.load(open('$f')); assert 'personas' in d, 'personas key missing'"
}

@test "every platform skills.json contains oracle" {
  for p in claude-code codex cursor copilot gemini opencode qoder; do
    f="$REPO/platforms/$p/skills.json"
    [ -f "$f" ] || continue
    grep -q '"name": "oracle"' "$f" || { echo "missing in $p"; return 1; }
  done
}

@test "kimi-skills.json contains gpowers-oracle" {
  f="$REPO/platforms/kimi/kimi-skills.json"
  [ -f "$f" ] || skip "kimi-skills.json absent"
  grep -q '"gpowers-oracle"' "$f"
}
