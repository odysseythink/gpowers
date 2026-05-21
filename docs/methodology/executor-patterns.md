# Executor Patterns

**Purpose:** Document the workflow patterns that oh-my-opencode encodes in its named executor personas (Sisyphus / Hephaestus / Prometheus / Librarian / Explore). gpowers ports them as *patterns the agent applies*, not as registered subagent types — because no mainstream host except OpenCode supports user-registered custom subagent types.

This is advisory content. The agent decides when to apply each pattern; no runtime mechanism enforces it.

## Pattern 1 — Orchestration (the Sisyphus pattern)

**When:** User request is non-trivial; multiple specialists could contribute; the agent is acting as primary.

**Apply:**
1. **Verbalize intent** before acting: one sentence mapping the user's surface form → true intent → routing decision. Example: "I detect implementation intent — explicit verb 'add' + concrete target. Approach: load `brainstorming` for scope check, then `writing-plans`, then delegate to a coder subagent."
2. **Delegation bias.** Default to dispatching specialists (via the host's Task/Agent tool) rather than working everything yourself. Trivial work stays in-session.
3. **Verification gate.** Before claiming done: load `core/skills/verification-before-completion`, run checks, paste evidence.
4. **High-stakes work.** Wrap in `/ultrawork` so Oracle re-verifies independently.

**Loads:** `brainstorming`, `dispatching-parallel-agents`, `verification-before-completion`, `executing-plans`.

## Pattern 2 — Deep Work (the Hephaestus pattern)

**When:** Single concrete task; depth matters more than breadth; no orchestration needed.

**Apply:**
1. **Single-task focus.** No tangential cleanup, no scope creep.
2. **Verify after every logical unit.** Re-run the relevant subset of tests/lints; paste output.
3. **3-strikes recovery.** After 3 consecutive failed fix attempts:
   - STOP all further edits.
   - REVERT to last known working state.
   - CONSULT Oracle with full failure context.
   - If Oracle cannot resolve → ask the user before proceeding.
4. **Never** delete failing tests to make a build pass; never suppress type errors with `as any` / `@ts-ignore`.

**Loads:** `executing-plans`, `verification-before-completion`, optional `ultrawork` for high-stakes work.

## Pattern 3 — Plan-Only (the Prometheus pattern)

**When:** User asks for a plan, design doc, or roadmap; implementation is explicitly deferred.

**Apply:**
1. **Interview first.** Ask 1–3 precise clarifying questions before writing anything.
2. **Write the plan.** Use `writing-plans` skill for shape.
3. **Refuse implementation.** If user later asks to implement, reply: "Plan delivered. Switch to a primary mode to implement." Do not silently start coding.

**Loads:** `brainstorming`, `writing-plans`.

## Pattern 4 — External Research (the Librarian pattern)

**When:** Question is about an external library, package, or open-source repo; not the current project.

**Apply:**
1. **Classify the request:** TYPE A conceptual / TYPE B implementation / TYPE C history / TYPE D comprehensive.
2. **Doc discovery first** (TYPE A/D only): find official docs URL → version check → sitemap → targeted reads.
3. **Cite permalinks** for every claim: `https://github.com/<owner>/<repo>/blob/<sha>/<path>#L<start>-L<end>`.
4. **Internal-code questions → out of scope.** Redirect: "this question is about the current project; use the codebase-search pattern instead."

## Pattern 5 — Internal Codebase Search (the Explore pattern)

**When:** Find files or code patterns in the current project.

**Apply:**
1. **Background + parallel.** Fire 2–5 explore searches simultaneously when angles differ.
2. **Specify thoroughness:** quick / medium / very thorough.
3. **Return file paths + pattern descriptions**, not commentary or recommendations.
4. **Stop when:** enough context to proceed / repeated results / 2 iterations yielded nothing new.

**Loads:** `dispatching-parallel-agents` for the calling-side discipline.

## Pattern 6 — Independent Verification (the Oracle pattern)

**Implemented as a real skill:** `roles/skills/oracle/SKILL.md`.

Unlike the other patterns, Oracle is a registered subagent type on platforms that support custom subagents (Kimi via `--agent-file`, OpenCode natively). On all other platforms it loads as a standard skill. See the SKILL body for the full protocol.

## When NOT to apply these patterns

- Trivial one-line tweaks. No pattern needed; just make the change.
- User explicitly asked for a different approach. User intent always wins.
- The host runtime already enforces an equivalent pattern (e.g., OpenCode running oh-my-opencode natively). Defer to the runtime.
