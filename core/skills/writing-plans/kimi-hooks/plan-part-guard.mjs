#!/usr/bin/env node
// PreToolUse hook (matcher: "Write|Edit") for kimi-code.
// Enforces the writing-plans "one part per turn" rule at the harness level:
// it BLOCKS the 2nd distinct plan sub-plan file written within a single user
// turn, so a bare "continue" (or YOLO mode) can no longer batch-generate every
// remaining part in one degrading session.
//
// Contract (confirmed against kimi-code 0.6.0 source):
//   - stdin: JSON with snake_case keys
//       { hook_event_name, session_id, cwd, tool_name, tool_input:{path,...}, tool_call_id }
//   - block: exit code 2 + reason on stderr  (allow: exit 0)
//   - matcher is a RegExp tested against the tool name
//
// Fail-open: ANY error, or any write that is not a plan sub-plan file, exits 0.
// This must never interfere with normal code edits.

import { readFileSync, writeFileSync, existsSync, statSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, isAbsolute, resolve, basename } from 'node:path';

// Stale-state cleanup only. The real per-turn reset is plan-part-reset.mjs
// (UserPromptSubmit). This just stops a crashed session from blocking forever.
const STALE_MS = 6 * 60 * 60 * 1000;

function readStdin() {
  try {
    return readFileSync(0, 'utf8');
  } catch {
    return '';
  }
}

try {
  const data = JSON.parse(readStdin() || '{}');
  const input = data.tool_input ?? {};
  const sessionId = String(data.session_id ?? 'default').replace(/[^A-Za-z0-9_-]/g, '_');
  const cwd = data.cwd ?? process.cwd();

  // Write/Edit both use `path` in kimi-code; accept file_path defensively.
  let p = input.path ?? input.file_path ?? '';
  if (!p) process.exit(0);
  if (!isAbsolute(p)) p = resolve(cwd, p);

  // Only plan files matter: markdown under a `plans/` directory.
  const isPlanFile = /[\\/]plans[\\/]/.test(p) && p.endsWith('.md');
  if (!isPlanFile) process.exit(0);

  // The index is the manifest; its status updates are always allowed.
  if (/-index\.md$/.test(p) || basename(p) === 'index.md') process.exit(0);

  const stateFile = join(tmpdir(), `kimi-plan-guard-${sessionId}.json`);
  let seen = [];
  if (existsSync(stateFile)) {
    const age = Date.now() - statSync(stateFile).mtimeMs;
    if (age < STALE_MS) {
      try {
        seen = JSON.parse(readFileSync(stateFile, 'utf8'));
      } catch {
        seen = [];
      }
    }
  }
  if (!Array.isArray(seen)) seen = [];

  // Same part file (scaffold-then-append) → allow.
  if (seen.includes(p)) process.exit(0);

  // First part of this turn → allow and record.
  if (seen.length === 0) {
    writeFileSync(stateFile, JSON.stringify([p]));
    process.exit(0);
  }

  // Second DISTINCT part file this turn → BLOCK.
  process.stderr.write(
    `🛑 ONE PART PER TURN. You already wrote a plan part this turn (${basename(seen[0])}).\n` +
      `Do NOT write another part file now. END your turn and tell the user:\n` +
      `"Part done. Run /compact, then reply continue (or re-invoke /writing-plans) for the next part."\n` +
      `Writing all remaining parts in one turn defeats the clean-context split protocol.`,
  );
  process.exit(2);
} catch {
  // Never block normal work because the guard hit an error.
  process.exit(0);
}
