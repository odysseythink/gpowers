# Upgrading gpowers

`gpowers upgrade` keeps each module in sync with its upstream:

- `core/` ← `github.com/obra/superpowers`
- `roles/` `tools/` `business/` ← `github.com/garrytan/gstack`

## Commands

```bash
gpowers upgrade                  # All four modules
gpowers upgrade core             # Just core
gpowers upgrade tools roles      # Multiple modules in one run
gpowers upgrade --check          # Read-only: show what's new without pulling
gpowers upgrade --dry-run        # Plan-only: print what would happen
gpowers upgrade --resume         # Continue after manual conflict resolution
```

## How a pull works

Per module, the upgrader:

1. **Verifies clean working tree.** `git subtree pull` requires it. Commit or stash your local changes first.
2. **Runs `git subtree pull --squash`** against the upstream URL/ref listed in `~/.gpowers/upstream-sources.json`. Pulls just the subtree (e.g., the `skills/` directory in superpowers becomes the new `core/skills/`).
3. **Re-applies the install-time transform.** The pulled content is in upstream form (no `namespace:`, original `superpowers:` references intact). `<module>/_upgrade-transform.sh` re-injects frontmatter and rewrites `~/.gstack/` paths, `gstack-*` CLI names, `superpowers:` references, and (for browser skills) MCP/playwright literals → 9-verb abstraction.
4. **Refreshes platform manifests.** `gpowers-platforms gen all` rebuilds plugin.json, commands/, skills.json, hooks.json, Kimi adapters.
5. **Runs the module's tests.** Unit + integration. A failure exits non-zero but the pull is preserved for you to inspect.
6. **Updates `<module>/upstream-source.json`** with the new SHA and timestamp.
7. **Commits the transformed state** into `~/.gpowers/` (since it's a git repo).

## Conflicts

If `git subtree pull` produces merge conflicts (you've edited content under `~/.gpowers/<module>/` locally), the upgrader stops:

```
[upgrade:roles] subtree pull failed (likely conflict)
On branch main
You are currently in the middle of a merge.
Unmerged paths:
  both modified: roles/skills/pr-review/SKILL.md
Run `gpowers upgrade --resume` after resolving conflicts.
```

Resolution:

```bash
cd ~/.gpowers
# Open the listed files in your editor and resolve markers manually
git add roles/skills/pr-review/SKILL.md
gpowers upgrade --resume
```

`--resume` continues from where the conflict interrupted — applies the transform, regenerates manifests, runs tests, bumps the SHA.

## Telemetry

Upgrade events are logged to `$(gpowers-path analytics)/upgrade.jsonl`. To disable analytics globally, set `GPOWERS_ANALYTICS=off`.

## Versioning

gpowers follows semantic versioning. See [RELEASING.md](../RELEASING.md) for the release procedure.
