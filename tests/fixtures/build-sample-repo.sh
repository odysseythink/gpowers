#!/usr/bin/env bash
# Usage: build-sample-repo.sh <target-dir>
# Builds a deterministic sample git repo for tests. Idempotent: removes target first.
set -euo pipefail
T="${1:?target dir required}"
rm -rf "$T"
mkdir -p "$T"
cd "$T"
git init -q
git config user.email t@t
git config user.name t

cat > README.md <<'F'
# sample-repo

Fixture for gpowers tests. Do not edit by hand — regenerate with `tests/fixtures/build-sample-repo.sh`.
F
git add README.md
git commit -q -m "initial: README"

cat > package.json <<'F'
{
  "name": "sample-repo",
  "version": "0.1.0",
  "scripts": { "test": "echo no tests" }
}
F
git add package.json
git commit -q -m "feat: package.json"

mkdir -p src
cat > src/index.js <<'F'
// sample-repo entry point
export function add(a, b) { return a + b; }
F
git add src/index.js
git commit -q -m "feat: add() function"

cat > src/bug.js <<'F'
// sample bug for /investigate fixture
export function divide(a, b) { return a / b; }  // does not guard b === 0
F
git add src/bug.js
git commit -q -m "feat: divide() (intentional gap for /investigate fixture)"

# Branch we ship from
git checkout -q -b feature/sample
echo "// feature change" >> src/index.js
git add src/index.js
git commit -q -m "feat: feature/sample line"
git checkout -q main 2>/dev/null || git checkout -q master 2>/dev/null || true
