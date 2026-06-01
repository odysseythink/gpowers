#!/usr/bin/env node
// UserPromptSubmit hook for kimi-code.
// Resets the per-turn plan-part counter so EACH user message — including a bare
// "continue" — starts with a fresh one-part budget. Pairs with plan-part-guard.mjs.
//
// stdin: JSON with snake_case keys (we only need session_id). Fail-open always.

import { readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

let raw = '';
try {
  raw = readFileSync(0, 'utf8');
} catch {
  // ignore
}

try {
  const data = JSON.parse(raw || '{}');
  const sessionId = String(data.session_id ?? 'default').replace(/[^A-Za-z0-9_-]/g, '_');
  rmSync(join(tmpdir(), `kimi-plan-guard-${sessionId}.json`), { force: true });
} catch {
  // ignore — resetting is best-effort
}

process.exit(0);
