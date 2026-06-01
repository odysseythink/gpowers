---
name: writing-plans
description: Use when you have a spec or requirements for a multi-step task, before touching code
namespace: core
upstream: superpowers@v5.1.0
---

# Writing Plans

## Overview

Write comprehensive implementation plans assuming the engineer has zero context for our codebase and questionable taste. Document everything they need to know: which files to touch for each task, code, testing, docs they might need to check, how to test it. Give them the whole plan as bite-sized tasks. DRY. YAGNI. TDD. Frequent commits.

Assume they are a skilled developer, but know almost nothing about our toolset or problem domain. Assume they don't know good test design very well.

**Specify, don't illustrate.** This skill is run by different models (Claude, Kimi, and others). Anything left to the engineer's judgment gets filled differently each time. Where this skill shows a structure, treat it as a **MUST**, not a suggestion — copy the skeletons verbatim and fill the blanks. Every rule below exists so that two different models produce the *same* high-quality plan from the same spec.

**Announce at start:** "I'm using the writing-plans skill to create the implementation plan."

**First action, every invocation:** check whether you are resuming a split plan — Glob for an existing `*-<feature>-index.md`. If one exists, do NOT re-plan from scratch; jump to "Generating a Split Plan (one part per invocation)" and continue from its Parts manifest. Only when no index exists do you start fresh with Scope Check and Plan Size.

**Context:** If working in an isolated worktree, it should have been created via the `gpowers:using-git-worktrees` skill at execution time.

**Save plans to:** `$(gpowers-path project)/plans/YYYY-MM-DD-<feature-name>.md` (single-file) or, when splitting (see Plan Size & File Layout), `$(gpowers-path project)/plans/YYYY-MM-DD-<feature-name>-index.md` plus `…-<subsystem>.md` siblings.
- (User preferences for plan location override this default)

## Scope Check

If the spec covers multiple independent subsystems, it should have been broken into sub-project specs during brainstorming. If it wasn't, suggest breaking this into separate plans — one per subsystem. Each plan should produce working, testable software on its own.

## Plan Size & File Layout

Before writing, count the tasks you are about to produce, then pick a layout. A 2000-line plan generated in one model turn loses its header and coherence; a set of ~400-line files does not.

- **≤ 8 tasks → one file**, but built incrementally (see How to Write the Plan File) — never emitted in a single shot.
- **> 8 tasks, OR the work spans more than one repo/subsystem → SPLIT into multiple flat files**: one index file plus one file per phase/subsystem.

**Flat multi-file layout (no subdirectories):**
- `YYYY-MM-DD-<feature>-index.md` — the global overview ONLY: Goal, Architecture, Tech Stack, the whole-system File Structure, the cross-file Dependency Overview, Risks & Open Questions, the global Self-Review **spec-coverage table** (item 1, mapping every spec section → `file: Task`), and a **Parts manifest** (see below). **No tasks live here.**
- `YYYY-MM-DD-<feature>-<subsystem>.md` — a short local header (one-line goal + `Depends on file: …` if any) then its tasks, then its own Self-Review items 2–7. Each sub-plan must be independently shippable.
- Cross-file dependencies: a task's `Depends on:` may name another file, e.g. `Depends on: <feature>-pantheon.md: Task P0`. The index's Dependency Overview shows the file-level graph.

**The Parts manifest is the durable state that lets generation resume across `/compact`.** Put this table in the index, ordered, with a `Status` column:

```markdown
## Parts (generate one per invocation, in order)

> ▶ To generate the next `pending` part: run `/compact`, then re-invoke the `/writing-plans` slash command. Do NOT type "continue" — it skips the rule reload and batch-generates everything.

| # | File | Scope | Status |
|---|---|---|---|
| 1 | <feature>-core.md | models + persistence + factory | pending |
| 2 | <feature>-chat.md | chat path + endpoint | pending |
| 3 | <feature>-agent.md | agent wiring + events | pending |
| 4 | <feature>-pantheon.md | upstream engine gaps | pending |
```

Mark a row `done` only after that sub-plan file is fully written. The next invocation reads this table to know what to do next — it is the single source of truth, not the conversation (which `/compact` may have erased).

## Generating a Split Plan (one part per invocation)

A large plan generated in a single session degrades: by the later files the model's context is bloated and the output rambles. So a split plan (> 8 tasks) is generated **one part per invocation**, with the user running `/compact` between parts to keep each generation's context clean. This skill is **re-entrant** — the filesystem holds the state, so every invocation starts by figuring out where it is.

**On every invocation, run this decision tree FIRST, before writing anything:**

1. **Glob for `*-<feature>-index.md`.**
2. **No index exists → this is part 0.** Write ONLY the index file (header, File Structure, Dependency Overview, Risks & Open Questions, spec-coverage table, and the Parts manifest with every row `pending`). Do not write any sub-plan. Then **stop and hand off** (see below) pointing at part 1.
3. **Index exists → read its Parts manifest.** Find the first row whose `Status` is `pending`:
   - Write that ONE sub-plan file (scaffold-then-append, per How to Write the Plan File).
   - Edit the index: set that row's `Status` to `done`.
   - **Stop and hand off**, pointing at the next `pending` row.
4. **Index exists and every row is `done` → generation is complete.** Don't write a sub-plan. Do the final cross-file Self-Review (is every `Depends on: <file>: Task` satisfied? does the coverage table still map every spec section?), then go to Execution Handoff.

**Iron rule: exactly one file per invocation.** Write the index, OR one sub-plan — never two sub-plans, never "I'll just finish the rest while I'm here." After the one file, you MUST end the turn (emit no further tool calls). This holds for **every** part, not just the first, and overrides **any** prompt that looks like a green light to keep going — a bare `"continue"`, `"go on"`, `"yes"`, auto-approve / YOLO mode, or a post-`/compact` summary that says "continue generating the parts". Treat all of those as "do the NEXT ONE part, then stop and hand off again." Finishing the remaining parts in one turn is the exact failure this protocol exists to prevent: after `/compact` these rules are gone from context, so a single "continue" generates everything in one degrading session.

**Hand-off message (end every part with this, then STOP — no more tool calls):**

> ✅ Part N of M written: `<feature>-<subsystem>.md` (index updated).
>
> To generate the next part (`<next-file>`) in a clean context, do exactly:
> 1. Run `/compact`
> 2. Re-invoke the **`/writing-plans`** slash command — it reloads these rules and resumes at part N+1 from the index manifest.
>
> ⚠️ Do **not** reply `"continue"`. After `/compact` has erased the one-part rules, a plain "continue" makes the model batch-generate every remaining part in one bloated session — the very thing this split exists to avoid. Re-invoking the slash command is what reloads the rules.

A single-file plan (≤ 8 tasks) skips all of this: write the one file incrementally in this invocation and go straight to Execution Handoff. No manifest, no stopping.

## File Structure

Before defining tasks, map out which files will be created or modified and what each one is responsible for. This is where decomposition decisions get locked in.

- Design units with clear boundaries and well-defined interfaces. Each file should have one clear responsibility.
- You reason best about code you can hold in context at once, and your edits are more reliable when files are focused. Prefer smaller, focused files over large ones that do too much.
- Files that change together should live together. Split by responsibility, not by technical layer.
- In existing codebases, follow established patterns. If the codebase uses large files, don't unilaterally restructure - but if a file you're modifying has grown unwieldy, including a split in the plan is reasonable.

This structure informs the task decomposition. Each task should produce self-contained changes that make sense independently.

## Task Ordering & Dependencies

Tasks are not a flat list — they form a dependency graph. Make it explicit so any model orders them the same way.

- **Every task MUST declare its prerequisites.** Put `**Depends on:** Task N` (or `none`) in the task header. A task may only use symbols (functions, methods, types, fields, files) that an earlier task or a declared prerequisite has already created — never something defined "later" or "elsewhere."
- **When there are more than ~8 tasks, or some tasks are independent / separately shippable, add a dependency overview at the top of the plan** — an ASCII graph or named phases (e.g. Phase A / B / C). Mark which tasks can run in parallel.
- **Each phase must produce working, testable software on its own.** If a phase can't be delivered without a not-yet-done task, the ordering is wrong — reorder, don't paper over it.

## Shared Signatures & the Build-Green Invariant

The most common way a plan compiles in your head but breaks on execution: a task changes something other code already calls, and the callers are left stale.

- **Changing a shared signature/type/interface/struct ripples in the same task.** If a task edits a constructor, function signature, interface, or struct that other code uses, that task MUST include a step that finds and updates *every* caller — show the search command (e.g. `grep -rn "NewChatService(" backend/`) and say what each caller passes (including test files). Don't defer caller updates to a later task.
- **Consolidate churn.** If several tasks would each modify the *same* shared signature, fold those modifications into one task (or do them first). Changing one constructor across three separate tasks — each re-updating all callers — is a decomposition smell.
- **Build-green invariant: every task ends buildable and test-passing.** The final verification step of each task MUST build the whole affected tree (e.g. `go build ./...`, `tsc --noEmit`, the project's full typecheck), not just the one new package — that's what catches stale callers. A new unit test that can't compile because an unrelated caller is broken is a false signal. Never commit a knowingly-broken intermediate state.

## External & Cross-Cutting Dependencies

The hardest plans depend on something that doesn't exist yet in *another* repo, package, or upstream library. Models improvise wildly here unless told how — so follow these rules:

- **Pull the minimal unblocking change to the front.** If the main work needs a method/field/endpoint that an upstream or sibling component lacks, add it as a **Phase 0 / prerequisite task** that creates exactly that, with its own failing test. Do not let later tasks call it as if it already exists.
- **Never express a deferred dependency as a `TODO` comment or dead code** (e.g. `// TODO: after upstream PR` plus an unused `_ = prev`). If something genuinely must be deferred, model it as a **typed shim**: a small interface with a real implementation now and the upstream-backed implementation swapped in later. A shim is testable; a TODO is a placeholder (see No Placeholders).
- **State the integration mechanism concretely** — the exact dependency bump, `replace` directive, branch, or PR the executor uses to wire the prerequisite in. No vague "once merged."

## Unknowns & Risks

When a task's approach is uncertain — it relies on a hook/API you haven't confirmed exists, or the only way you can see to do it is a fragile workaround — **surface it; never bury it inside an implementation step as if it were settled.**

- **Confirm before you build on it.** If the plan assumes a callback, hook, or API exists (e.g. "the framework fires `onStepFinish`"), add a verification/spike step that proves it exists *before* the task that depends on it — or cite the exact file/line where it's defined.
- **No fragile hacks smuggled in as fact.** Inventing a "poll every 5 seconds" ticker, a `sleep`, a global mutable, or a goroutine that watches shared state because you couldn't find the real hook is a design decision in disguise. Stop and flag it, don't write it into a step.
- **Put real unknowns in a "Risks & Open Questions" block at the top of the plan.** Each entry: what's uncertain, what you assumed, and what would change if the assumption is wrong.
- **Justified deferral is allowed when it's explicit and bound to a task.** A blockquote like "V1 logs `before=0/after=0` because the engine doesn't surface counts at this boundary; real numbers wired in Task N via the usage callback" is fine — it states the limitation, the reason, and where the real fix lands. A silent workaround is not.

## Bite-Sized Task Granularity

**Each step is one action (2-5 minutes):**
- "Write the failing test" - step
- "Run it to make sure it fails" - step
- "Implement the minimal code to make the test pass" - step
- "Run the tests and make sure they pass" - step
- "Commit" - step

**Test-first is per-task and non-negotiable for code:**
- Every task that produces testable code MUST begin with its own failing test and end with that test passing — the test and the implementation live in the *same* task.
- MUST NOT collect tests into a trailing "write the tests" / "test coverage" task. If deleting one task would remove another task's tests, the split is wrong.
- The only code exempt from test-first is code that can't be unit-tested (UI, config, dependency wiring) — handle it with the non-test task skeleton below.

**Test the risk, not the easy part.** Engineers (and models) tend to lavish tests on trivial pure helpers and leave the dangerous code bare. Invert that:
- Any task whose core is a **state mutation** — DB write/delete, soft-delete, boundary/offset computation, cache update, money/permission logic — MUST end with a behavioral test that asserts the mutation (the rows that changed, the boundary that was picked), not just `go build`. This is exactly where the bugs live.
- A pure, side-effect-free helper (e.g. a token estimator) needs only a light test or two — don't over-invest there.
- If a task's most error-prone function ships with only a compile check, the plan is testing the wrong thing.

## Plan Document Header

**Every plan MUST start with this header:**

```markdown
# [Feature Name] Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use gpowers:subagent-driven-development (recommended) or gpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** [One sentence describing what this builds]

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies/libraries]

---
```

## How to Write the Plan File (incremental — never one shot)

Do NOT generate the whole plan as a single Write. Build it through multiple tool calls, so the scaffold lands first and can never be crowded out by task detail. Each tool call is a fresh model generation — no single generation has to hold the entire plan in its head, and the header becomes structurally impossible to drop.

1. **Call 1 — scaffold only.** Write the file with JUST: the header above, the File Structure, the Dependency Overview, and Risks & Open Questions. Save it. Stop there — do not write any task yet.
2. **Calls 2…N — one phase per call.** For each phase, a SEPARATE Edit that appends that phase's tasks to the file. Do not batch all phases into one Edit.
3. **Final call — Self-Review.** Append the seven-item checklist last.

This incremental sequence governs the writing of **one file**. It composes with the split protocol, which governs **across files**: in a split plan each invocation produces exactly one file (the index, or one sub-plan), and that file is written with this scaffold-then-append sequence. Index invocation: scaffold = the global header + File Structure + Dependency Overview + Risks + Parts manifest. Sub-plan invocation: scaffold = the local header, then append its tasks, then its Self-Review. Do not start a second file in the same invocation.

**Why this matters:** a model that emits 2000 lines in one generation spends its attention satisfying per-task rules and silently omits the cheap document-level scaffolding (header, file map, dependency graph). Writing the scaffold in its own call removes that competition entirely.

## Task Structure

````markdown
### Task N: [Component Name]

**Depends on:** Task M (or `none`)

**Files:**
- Create: `exact/path/to/file.py`
- Modify: `exact/path/to/existing.py:123-145`
- Test: `tests/exact/path/to/test.py`

- [ ] **Step 1: Write the failing test**

```python
def test_specific_behavior():
    result = function(input)
    assert result == expected
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/path/test.py::test_name -v`
Expected: FAIL with "function not defined"

- [ ] **Step 3: Write minimal implementation**

```python
def function(input):
    return expected
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/path/test.py::test_name -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/path/test.py src/path/file.py
git commit -m "feat: add specific feature"
```
````

### Task Skeleton for non-testable code (UI, config, wiring)

Some code can't be driven by a unit test. It still gets **complete code** (never "add a section here") and a **manual verification step** with an exact action and the expected observation — not a skipped test.

````markdown
### Task N: [Component Name]

**Depends on:** Task M

**Files:**
- Modify: `exact/path/to/Component.jsx`

- [ ] **Step 1: Write the complete component code**

```jsx
<CompressionSettings
  value={settings.compressEnabled}
  onChange={(v) => setSettings((s) => ({ ...s, compressEnabled: v }))}
/>
```

- [ ] **Step 2: Build to verify it compiles**

Run: `cd frontend && npm run build`
Expected: build succeeds, no type/lint errors

- [ ] **Step 3: Manual verification**

Action: open Workspace Settings, toggle the control, save.
Expected: PUT body contains `compressEnabled: true`; reload shows the saved value.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/path/Component.jsx
git commit -m "feat: add compression settings UI"
```
````

### Task Skeleton for changing a shared signature (constructor, interface, struct)

When a task edits something other code already calls, the callers go stale. This skeleton bakes in the two steps models forget: **find every caller (including `_test.go` files)** and **typecheck the whole tree including tests, not one package**. Use it whenever a task touches a function signature, interface, struct field other code reads, or exported type used elsewhere.

**Critical:** `go build ./...` does NOT compile `_test.go` files, so it silently misses stale callers in tests — which is the most common place they hide. The whole-tree verification MUST use a command that typechecks test files too: `go vet ./...` (fast, compiles tests) or `go test ./... -run=^$` (compiles everything, runs nothing). For non-Go stacks use the equivalent that includes test sources (e.g. `tsc --noEmit` over the whole project, not one file).

````markdown
### Task N: [what changes — e.g. "add compStore + sysSvc to NewChatService"]

**Depends on:** Task M

**Files:**
- Modify: `path/to/def.go` (the signature itself)
- Modify: every caller found in Step 2

- [ ] **Step 1: Change the signature + write/adjust the behavioral test**

Show the new signature and the test that drives the new behavior. The test's own call MUST use the new arity from the start — don't write it against the old shape and break it in this same task.

- [ ] **Step 2: Find and update EVERY caller (prod + tests)**

Run: `grep -rn "NewChatService(" backend/`
Update each hit to the new shape (pass `nil`/zero for new params in tests). List the files you expect to change so the executor can confirm none were missed.

- [ ] **Step 3: Whole-tree typecheck (incl. tests) + targeted test**

Run: `cd backend && go vet ./... && go test ./internal/services/ -run TestNewBehavior -v`
Expected: vet passes (proves no stale caller anywhere, including `_test.go` files), test passes.
(Plain `go build ./...` is NOT enough — it skips `_test.go` files, which is exactly where stale callers hide. The targeted `go test` only covers one package, so use `go vet ./...` for the cross-tree check.)

- [ ] **Step 4: Commit**

```bash
git add path/to/def.go <callers>
git commit -m "refactor: extend NewChatService signature + update callers"
```
````

**Batch shared-signature changes into ONE task.** If two tasks would each add a param to the same constructor, do both in this one task. Re-updating all callers twice across separate tasks is a decomposition smell.

## No Placeholders

Every step must contain the actual content an engineer needs. These are **plan failures** — never write them:
- "TBD", "TODO", "implement later", "fill in details"
- "Add appropriate error handling" / "add validation" / "handle edge cases"
- "Write tests for the above" (without actual test code)
- "Similar to Task N" (repeat the code — the engineer may be reading tasks out of order)
- Steps that describe what to do without showing how (code blocks required for code steps)
- References to types, functions, or methods not defined in any task
- Author deliberation left in the plan body — "Wait, I already did X…", "Let me adjust…", "OK, I'll add a note…", "Actually, on second thought…". The plan is the finished artifact, not your scratchpad.

**"Deferred because of a dependency" is not an exception.** A dependency on unfinished upstream work is handled by a prerequisite task or a typed shim (see External & Cross-Cutting Dependencies) — not by a `TODO` comment, and not by dead code like `_ = prev`. If you're tempted to write `// TODO: after the upstream PR`, you have an ordering problem, not a documentation problem.

**When you realize an earlier task was wrong, go back and fix it — don't leave a note.** If, while writing a later task, you discover an earlier task needs to change (e.g. it "already committed" a constructor that now needs a second param), edit that earlier task so it's correct the first time. Never append reconciliation notes like "this means Task H7 must also update main.go" — the executor runs each task in isolation and in order, and will not cross-reference a future task's correction. Two tasks that mutate the same shared signature is the same churn smell: merge them.

**Every task must produce a verifiable change — no exceptions, no self-granted ones.** No empty tasks, no `git commit --allow-empty` filler, no task whose body is "already done in Task N." This rule is binary; "it's acceptable here because…" is the rationalization it exists to stop. If a spec item needs no code (a behavior already falls out of existing wiring), do **not** manufacture a task for it — record it in the self-review coverage table as `no-op (handled by Task N)`. The pressure to mint one task per spec section is exactly what produces phantom tasks; resist it.

## Remember
- Exact file paths always
- Complete code in every step — if a step changes code, show the code
- Exact commands with expected output
- DRY, YAGNI, TDD, frequent commits

## Self-Review

After writing the complete plan, look at the spec with fresh eyes and check the plan against it. This is a checklist you run yourself — not a subagent dispatch.

**Reproduce all seven items below as a checklist in the plan — verbatim, with the `- [ ]` boxes. Do not drop, merge, or renumber any of them.** A self-review that lists only five items is itself a failure: the two most-skipped checks (caller & build soundness, test-the-risk) are exactly the ones that catch the bugs that ship.

- [ ] **1. Spec coverage (build the table).** Map every spec section/requirement to the task that implements it. Render it as a table so gaps are visible at a glance:

| Spec section | Task(s) | Status |
|---|---|---|
| §4 Data model | Task 1 | covered |
| §9 Observability | — | GAP |
| §7 Thread binding | Task 5 | no-op (handled by existing JSON binding) |

`GAP` means missing — add the task. `no-op` is only for spec items that genuinely need zero code; never satisfy one with an empty-commit task.

- [ ] **2. Placeholder scan:** Search your plan for red flags — any pattern from "No Placeholders": `TODO`/`TBD`, deferred-by-dependency excuses, and dead-code placeholders.

- [ ] **3. No phantom tasks (binary):** Every task produces a verifiable change. Zero `--allow-empty`, zero "already done in Task N" bodies. A no-code spec item belongs in the coverage table as `no-op`, not as a task. If you wrote "this is acceptable because…" next to an empty commit, delete the task.

- [ ] **4. Dependency soundness:** Every task's `Depends on:` is satisfied by an *earlier* task. No task references a symbol, file, or endpoint that only a later task — or unfinished external work — creates.

- [ ] **5. Caller & build soundness:** For every task that changed a shared signature/type/interface, did it update all callers (run the `grep` mentally — including `_test.go` files) and end with a *whole-tree typecheck that includes tests* (`go vet ./...` / `go test ./... -run=^$` / project-wide `tsc --noEmit`), not `go build ./...` (which skips tests) and not a single-package build? A test written at the old arity and then broken by the same task is the classic miss. Also: did the same shared signature get changed in more than one task? If so, consolidate — that churn is the bug's source.

- [ ] **6. Test-the-risk:** Does every state-mutating task (DB write/delete, boundary calc, cache, permissions) have a behavioral test asserting the mutation — not just a compile check?

- [ ] **7. Type consistency:** Do the types, method signatures, and property names used in later tasks match what earlier tasks defined? `clearLayers()` in Task 3 but `clearFullLayers()` in Task 7 is a bug.

If you find issues, fix them inline. No need to re-review — just fix and move on.

## Execution Handoff

After saving the plan, offer execution choice:

**"Plan complete and saved to `$(gpowers-path project)/plans/<filename>.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?"**

**If Subagent-Driven chosen:**
- **REQUIRED SUB-SKILL:** Use gpowers:subagent-driven-development
- Fresh subagent per task + two-stage review

**If Inline Execution chosen:**
- **REQUIRED SUB-SKILL:** Use gpowers:executing-plans
- Batch execution with checkpoints for review

## Appendix: Copyable Plan Skeleton

Fill the blanks. Keep the header, the dependency overview, the per-task `Depends on:`, and the self-review table.

````markdown
# [Feature Name] Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use gpowers:subagent-driven-development (recommended) or gpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** [one sentence]
**Architecture:** [2-3 sentences]
**Tech Stack:** [key tech]

---

## File Structure
[created/modified files — one clear responsibility each]

## Dependency Overview
```
Phase 0 (prereqs / upstream unblockers)
  -> Phase A (...)  -> Phase B (...)    [Phase C independent, runs in parallel]
```

## Risks & Open Questions
[only if any: each = what's uncertain | what you assumed | what changes if wrong. Omit the section if none.]

---

### Task 0: [minimal upstream/external unblocker, if any]
**Depends on:** none
[failing test -> implement -> pass -> commit]

### Task N: [testable code]
**Depends on:** Task M
[Step 1 failing test -> Step 2 verify fail -> Step 3 implement -> Step 4 verify pass -> Step 5 commit]

### Task N: [non-testable code: UI / config / wiring]
**Depends on:** Task M
[Step 1 complete code -> Step 2 build -> Step 3 manual verification (action + expected) -> Step 4 commit]

### Task N: [shared-signature change]   (change a given signature in ONE task, not several)
**Depends on:** Task M
[Step 1 change sig + test at new arity -> Step 2 grep ALL callers (incl _test.go) + update -> Step 3 `go vet ./...` (typechecks tests too — not `go build ./...`) + targeted test -> Step 4 commit]

---

## Self-Review
Reproduce all seven as `- [ ]` checkboxes — do not shrink to five:
- [ ] 1. spec-coverage table (covered / GAP / no-op)
- [ ] 2. placeholder scan
- [ ] 3. no phantom tasks (zero --allow-empty)
- [ ] 4. dependency soundness
- [ ] 5. caller & build soundness (`go vet ./...` — typechecks tests; one signature changed in one task)
- [ ] 6. test-the-risk
- [ ] 7. type consistency
````
