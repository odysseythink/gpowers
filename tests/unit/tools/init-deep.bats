#!/usr/bin/env bats
# Install regression for tools/skills/init-deep

setup() {
  REPO="$(cd "$(dirname "$BATS_TEST_FILENAME")/../../.." && pwd)"
}

@test "init-deep SKILL.md exists and has valid frontmatter" {
  [ -f "$REPO/tools/skills/init-deep/SKILL.md" ]
  python3 -c "
import yaml
with open('$REPO/tools/skills/init-deep/SKILL.md') as f:
    lines = f.readlines()
start = end = None
for i, line in enumerate(lines):
    if line.strip() == '---':
        if start is None: start = i
        elif end is None: end = i; break
yaml.safe_load(''.join(lines[start+1:end]))
"
}

@test "init-deep SKILL.md contains scoring matrix" {
  grep -q "Scoring Matrix" "$REPO/tools/skills/init-deep/SKILL.md"
  grep -q "Decision rules" "$REPO/tools/skills/init-deep/SKILL.md"
}

@test "init-deep SKILL.md contains fan-out table" {
  grep -q "Dynamic Agent Fan-out" "$REPO/tools/skills/init-deep/SKILL.md"
}

@test "init-deep SKILL.md contains root template" {
  grep -q "Root AGENTS.md Template" "$REPO/tools/skills/init-deep/SKILL.md"
}

@test "init-deep SKILL.md contains subdir template" {
  grep -q "Subdir AGENTS.md Template" "$REPO/tools/skills/init-deep/SKILL.md"
}

@test "init-deep SKILL.md contains 4-phase workflow" {
  grep -q "Phase 1" "$REPO/tools/skills/init-deep/SKILL.md"
  grep -q "Phase 2" "$REPO/tools/skills/init-deep/SKILL.md"
  grep -q "Phase 3" "$REPO/tools/skills/init-deep/SKILL.md"
  grep -q "Phase 4" "$REPO/tools/skills/init-deep/SKILL.md"
}

@test "init-deep SKILL.md contains AI-slop deny-list" {
  grep -q "comprehensive" "$REPO/tools/skills/init-deep/SKILL.md"
  grep -q "enterprise-grade" "$REPO/tools/skills/init-deep/SKILL.md"
}

@test "init-deep tests directory exists" {
  [ -d "$REPO/tools/skills/init-deep/tests" ]
  [ -f "$REPO/tools/skills/init-deep/tests/scenarios.md" ]
}

@test "scenarios.md contains all 7 scenarios" {
  for s in I-A I-B I-C I-D I-E I-F I-G; do
    grep -q "$s" "$REPO/tools/skills/init-deep/tests/scenarios.md"
  done
}

@test "Kimi adapter contains init-deep" {
  [ -f "$REPO/platforms/kimi/adapters/gpowers-init-deep/SKILL.md" ]
}

@test "skills.json contains init-deep entry" {
  python3 -c "
import json
data = json.load(open('$REPO/platforms/kimi/kimi-skills.json'))
assert 'gpowers-init-deep' in data.get('adapters', []), 'missing gpowers-init-deep'
"
}
