#!/usr/bin/env bash
# Ultrawork Kimi Native Path — Opt-in Installer
# Idempotent. Scope: project (./.kimi/) or user (~/.kimi/).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GPOWERS_HOME="${GPOWERS_HOME:-$(cd "$SCRIPT_DIR/../../../.." && pwd)}"

# Parse args
SCOPE="project"
FORCE=false
while [ $# -gt 0 ]; do
  case "$1" in
    --user) SCOPE="user" ;;
    --force) FORCE=true ;;
    --help|-h)
      echo "Usage: $0 [--user] [--force]"
      echo "  --user   Install to ~/.kimi/ instead of ./.kimi/"
      echo "  --force  Overwrite existing oracle: subagent key"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
  shift
done

if [ "$SCOPE" = "user" ]; then
  KIMI_DIR="${HOME}/.kimi"
else
  KIMI_DIR="${PWD}/.kimi"
fi

echo "Installing Ultrawork Kimi native path to: $KIMI_DIR"

# 1. Copy skill adapter (text-only, harmless)
mkdir -p "$KIMI_DIR/skills/ultrawork"
cp "$GPOWERS_HOME/core/skills/ultrawork/SKILL.md" "$KIMI_DIR/skills/ultrawork/SKILL.md"
echo "  [OK] skill → $KIMI_DIR/skills/ultrawork/SKILL.md"

# 2. Copy Oracle subagent files
mkdir -p "$KIMI_DIR/agents/oracle"
cp "$SCRIPT_DIR/oracle.md" "$KIMI_DIR/agents/oracle/oracle.md"

EXTEND_PATH=""
if [ -f "$KIMI_DIR/agent.yaml" ]; then
  EXTEND_PATH="$KIMI_DIR/agent.yaml"
else
  EXTEND_PATH="builtin:default"
fi

sed "s|{{RESOLVED_AT_INSTALL_TIME}}|$EXTEND_PATH|" "$SCRIPT_DIR/oracle.yaml" \
  > "$KIMI_DIR/agents/oracle/oracle.yaml"
echo "  [OK] oracle spec → $KIMI_DIR/agents/oracle/oracle.yaml"
echo "  [OK] oracle prompt → $KIMI_DIR/agents/oracle/oracle.md"

# 3. Copy Stop hook script
mkdir -p "$KIMI_DIR/hooks"
cp "$SCRIPT_DIR/stop-hook.sh" "$KIMI_DIR/hooks/ultrawork-stop.sh"
chmod +x "$KIMI_DIR/hooks/ultrawork-stop.sh"
echo "  [OK] stop hook → $KIMI_DIR/hooks/ultrawork-stop.sh"

# 4. Register hook in config.toml (idempotent markers)
CONFIG_FILE="$KIMI_DIR/config.toml"
mkdir -p "$(dirname "$CONFIG_FILE")"

HOOK_BLOCK=$(cat <<'HOOK'
# >>> gpowers:ultrawork >>>
[hooks.stop]
command = "${KIMI_DIR}/hooks/ultrawork-stop.sh"
# <<< gpowers:ultrawork <<<
HOOK
)

HOOK_BLOCK="${HOOK_BLOCK//\$\{KIMI_DIR\}/$KIMI_DIR}"

if [ ! -f "$CONFIG_FILE" ]; then
  echo "$HOOK_BLOCK" > "$CONFIG_FILE"
else
  if grep -q ">>> gpowers:ultrawork >>>" "$CONFIG_FILE"; then
    BLOCK_TMP=$(mktemp)
    printf '%s\n' "$HOOK_BLOCK" > "$BLOCK_TMP"
    awk -v blockfile="$BLOCK_TMP" '
      /# >>> gpowers:ultrawork >>>/ {
        while ((getline line < blockfile) > 0) print line
        close(blockfile)
        skip=1
        next
      }
      /# <<< gpowers:ultrawork <<</ { skip=0; next }
      !skip { print }
    ' "$CONFIG_FILE" > "$CONFIG_FILE.tmp" && mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"
    rm -f "$BLOCK_TMP"
    echo "  [OK] updated existing hook block in config.toml"
  else
    echo "" >> "$CONFIG_FILE"
    echo "$HOOK_BLOCK" >> "$CONFIG_FILE"
    echo "  [OK] appended hook block to config.toml"
  fi
fi

# 5. Register Oracle subagent in agent.yaml
AGENT_FILE="$KIMI_DIR/agent.yaml"
mkdir -p "$(dirname "$AGENT_FILE")"

ORACLE_ENTRY=$(cat <<'ENTRY'
  oracle:
    spec: ./agents/oracle/oracle.yaml
ENTRY
)

if [ ! -f "$AGENT_FILE" ]; then
  cat > "$AGENT_FILE" <<YAML
subagents:
$ORACLE_ENTRY
YAML
  echo "  [OK] created agent.yaml with oracle subagent"
else
  if grep -q "^\s*oracle:" "$AGENT_FILE"; then
    if [ "$FORCE" = true ]; then
      awk -v entry="$ORACLE_ENTRY" '
        /^\s*oracle:/ {
          print entry
          skip=1
          next
        }
        skip && /^[a-zA-Z]/ && !/^[ ]/ { skip=0 }
        !skip { print }
      ' "$AGENT_FILE" > "$AGENT_FILE.tmp" && mv "$AGENT_FILE.tmp" "$AGENT_FILE"
      echo "  [OK] overwrote existing oracle: key (forced)"
    else
      echo "  [ERROR] oracle: key already present in agent.yaml"
      echo "          Rename existing or pass --force to overwrite."
      exit 1
    fi
  else
    if grep -q "^subagents:" "$AGENT_FILE"; then
      awk -v entry="$ORACLE_ENTRY" '
        /^subagents:/ { print; print entry; next }
        { print }
      ' "$AGENT_FILE" > "$AGENT_FILE.tmp" && mv "$AGENT_FILE.tmp" "$AGENT_FILE"
    else
      echo "" >> "$AGENT_FILE"
      echo "subagents:" >> "$AGENT_FILE"
      echo "$ORACLE_ENTRY" >> "$AGENT_FILE"
    fi
    echo "  [OK] registered oracle subagent in agent.yaml"
  fi
fi

# 6. Validate hook script
if bash -n "$KIMI_DIR/hooks/ultrawork-stop.sh"; then
  echo "  [OK] hook script syntax valid"
else
  echo "  [ERROR] hook script has syntax errors"
  exit 1
fi

# 7. Run hook once with sample input for heartbeat check
echo '{"session_id":"test-validation"}' | \
  KIMI_SESSION_DIR="/tmp/.kimi-test-$$" \
  "$KIMI_DIR/hooks/ultrawork-stop.sh" >/dev/null 2>&1 || true
echo "  [OK] hook dry-run completed"

# Summary
echo ""
echo "Ultrawork Kimi native path installed successfully."
echo ""
echo "Next steps:"
echo "  1. Restart Kimi if it is already running (config.toml changes require restart)."
echo "  2. Invoke with: /flow:ultrawork \"<your task>\""
echo ""
echo "Files installed:"
echo "  $KIMI_DIR/skills/ultrawork/SKILL.md"
echo "  $KIMI_DIR/agents/oracle/oracle.yaml"
echo "  $KIMI_DIR/agents/oracle/oracle.md"
echo "  $KIMI_DIR/hooks/ultrawork-stop.sh"
echo "  $KIMI_DIR/config.toml (modified)"
echo "  $KIMI_DIR/agent.yaml (modified)"
