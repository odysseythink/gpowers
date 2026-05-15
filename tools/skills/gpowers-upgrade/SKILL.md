---
name: gpowers-upgrade
description: Pull upstream changes for any gpowers module (core / roles / tools / business) — git subtree mechanics, transform re-application, test re-run, platform manifest refresh.
namespace: tools
slash: /gpowers-upgrade
---

# gpowers-upgrade

When the user wants to refresh gpowers from upstream:

## Decide scope first

- **All four modules**: `gpowers upgrade` (no argument)
- **One module**: `gpowers upgrade core` (or `roles`, `tools`, `business`)
- **Just check what's new**: `gpowers upgrade --check` (read-only, no merge)

## Recommend a check before pulling

Suggest the user run `gpowers upgrade --check` first. It prints a table of
remote SHAs versus locally recorded SHAs and labels each row "up-to-date" or
"new version available". Use this to decide which modules actually need
pulling.

## Pull workflow

```bash
gpowers upgrade core            # pulls from github.com/obra/superpowers
gpowers upgrade tools           # pulls from github.com/garrytan/gstack
gpowers upgrade                 # all four
```

For each pulled module the runner:

1. Verifies `~/.gpowers/` working tree is clean (git subtree requirement).
2. Runs `git subtree pull --squash` from the upstream listed in
   `~/.gpowers/upstream-sources.json`.
3. Captures the new SHA and runs the module's `_upgrade-transform.sh` —
   re-applies `namespace:` and `upstream:` frontmatter, `~/.gstack/` path
   rewrites, `superpowers:` → `gpowers:` reference rewrites, and (for browser
   skills) the abstract-verb rewriter.
4. Regenerates all 7 platform manifests via `gpowers-platforms gen all`.
5. Runs the module's bats tests under `tests/unit/<module>/` and
   `tests/integration/<module>/`.
6. Bumps the SHA in `<module>/upstream-source.json`.
7. Commits the transformed state.

## Conflicts

`git subtree pull` may produce a merge conflict if you've made local edits
inside `~/.gpowers/<module>/`. The runner stops, prints `git status`, and
exits non-zero. Guide the user through:

```bash
cd ~/.gpowers
# Resolve conflicts in the listed files
git add <resolved-files>
gpowers upgrade --resume
```

`--resume` finishes the merge commit, runs the transform, regenerates
manifests, runs tests, and bumps the SHA — picking up where the conflict
interrupted things.

## Dry run

`gpowers upgrade --dry-run` prints the plan without acting. Use this to show
the user what would happen before they commit to a pull.

## Why each module has its own transform

The transform encodes how gpowers normalizes upstream content. Each module
ships a `_upgrade-transform.sh` that wraps the import helper used at first
install. Changing the normalization in one place (the helper) auto-applies
to upgrades — no separate code path to maintain.

## Telemetry

Upgrade events are recorded under `$(gpowers-path analytics)/upgrade.jsonl`.
Disable with `GPOWERS_ANALYTICS=off`.
