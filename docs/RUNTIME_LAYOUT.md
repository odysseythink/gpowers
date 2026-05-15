# gpowers runtime layout

gpowers splits runtime data between a global layer (cross-project, machine-wide) and a per-project layer (committable, team-shared decision memory).

## Global layer — `~/.gpowers/`

```
~/.gpowers/
├── core/  roles/  tools/  business/   ← installed module content (git managed)
├── platforms/                          ← per-platform adapters (generated)
├── bin/                                ← CLI entry points (on PATH)
├── manifest.json                       ← module + version state
├── upstream-sources.json               ← per-module upstream pointers
├── .gitignore
│
├── config/                             ← user preferences (cross-project)
│   ├── compact.toml, compact-rules/
│   ├── builder-profile, developer-profile
│   ├── gbrain-repo-policy, plan-tune.toml
│
├── state/                              ← mutable state
│   ├── installation-id, last-update-check, update-snoozed
│   ├── security/
│   ├── worktrees/<project-slug>/<branch>/
│   ├── browser/tabs/<tab_id>/
│   └── migrate-journal.jsonl
│
├── cache/                              ← rebuildable
│   ├── browser/chromium-profile/
│   ├── models/                         ← AI models (large, cross-project)
│   ├── repos/                          ← clone mirrors
│
├── data/                               ← global artifacts (cross-project)
│   ├── browser-skills/  global-domain-skills/
│   ├── retros/global/  learnings/global/
│   ├── sessions/  investigate-sessions/
│   ├── benchmarks/global/
│   └── legacy-projects/<slug>/         ← migration fallback bucket
│
├── analytics/                          ← telemetry (default on, off via GPOWERS_ANALYTICS=off)
├── logs/                               ← global error logs
└── tmp/                                ← short-lived
```

## Project layer — `<repo>/.gpowers/`

```
<repo>/
├── .gpowers/
│   ├── plans/
│   │   ├── ceo/<slug>.md
│   │   ├── eng/<slug>.md
│   │   ├── design/<slug>.md
│   │   ├── devex/<slug>.md
│   │   └── autoplan/<slug>.md
│   ├── designs/                        ← /design-shotgun /design-html output
│   ├── evals/                          ← evaluation results
│   ├── sessions/                       ← project session snapshots
│   ├── investigate/                    ← root-cause analysis records
│   ├── retros/                         ← project retros
│   ├── learnings/                      ← project lessons learned (PROJECT.learn.md)
│   ├── canary/                         ← canary history
│   ├── health/                         ← /health score history
│   ├── benchmark/                      ← performance baseline
│   ├── ship-queue.json                 ← /landing-report state
│   ├── browser-skills/                 ← project-specific browser skills
│   ├── logs/                           ← project-level logs
│   └── README.md                       ← auto-generated team note
│
└── docs/gpowers/specs/                 ← committed design specs (outside .gpowers/)
```

## Project detection

`gpowers-path project` resolves the project root by:

1. Honoring `$GPOWERS_PROJECT_DIR` if set.
2. Walking up from `cwd` to find a `.gpowers/` directory.
3. Walking up from `cwd` to find a `.git/` directory.
4. Falling back to the global layer (`$GPOWERS_HOME/data/...`).

Initialize a project explicitly with `gpowers init`, which creates `<repo>/.gpowers/` and writes a `.gpowers/.gitignore` template.

## Commit vs. ignore

`<repo>/.gpowers/.gitignore` defaults:

```
# Team-shared (commit):
plans/ designs/ evals/ retros/ learnings/ investigate/
canary/ health/ benchmark/ ship-queue.json
browser-skills/

# Local-only (ignore):
logs/ tmp/
sessions/*.pid sessions/*.lock
*.local.*
.cache/
ship-queue.lock
```

Rationale per subdir is in [ARCHITECTURE.md](ARCHITECTURE.md) — short version: anything that captures a *decision* should commit; anything ephemeral or local-machine-specific should ignore.

## Environment variables

Every path is overridable:

| Variable | Default | Meaning |
|---|---|---|
| `GPOWERS_HOME` | `~/.gpowers` | global root |
| `GPOWERS_CONFIG` | `$GPOWERS_HOME/config` | config |
| `GPOWERS_STATE` | `$GPOWERS_HOME/state` | mutable state |
| `GPOWERS_CACHE` | `$GPOWERS_HOME/cache` | cache |
| `GPOWERS_DATA` | `$GPOWERS_HOME/data` | global artifacts |
| `GPOWERS_ANALYTICS` | `$GPOWERS_HOME/analytics` | telemetry (set `off` to disable) |
| `GPOWERS_TMP` | `$GPOWERS_HOME/tmp` | temp |
| `GPOWERS_PROJECT_DIR` | auto-detect | project root override |
| `GPOWERS_PROJECT_DATA` | `$GPOWERS_PROJECT_DIR/.gpowers` | project data |

## XDG interoperability

If you prefer XDG paths:

```bash
export GPOWERS_DATA="${XDG_DATA_HOME:-$HOME/.local/share}/gpowers"
export GPOWERS_CACHE="${XDG_CACHE_HOME:-$HOME/.cache}/gpowers"
export GPOWERS_STATE="${XDG_STATE_HOME:-$HOME/.local/state}/gpowers"
export GPOWERS_CONFIG="${XDG_CONFIG_HOME:-$HOME/.config}/gpowers"
```

`gpowers-path` honors all of these.

## Migration

If you previously used gstack or superpowers, `gpowers migrate` transitions you:

```bash
gpowers migrate --scan-only       # See what would move
gpowers migrate --plan-only       # Full mapping JSON
gpowers migrate                    # Interactive apply
gpowers migrate --apply --yes      # Non-interactive
```

The migration:
1. Scans `~/.gstack/` (gstack) and `~/.config/superpowers/` (superpowers).
2. Moves global config / state / cache / analytics to the new gpowers layout.
3. Resolves gstack project slugs to repository paths (via recorded `.repo-path` or filesystem search) and moves project-scoped data to `<repo>/.gpowers/`. Unresolved slugs land in `~/.gpowers/data/legacy-projects/<slug>/` so nothing is lost.
4. Aliases `/review` to `/pr-review` with a deprecation banner valid until 2026-11-14.

Migration is journaled and reversible: a failure during apply triggers automatic rollback. See [the migrate plan](../docs/superpowers/plans/2026-05-14-gpowers-migrate.md) for full details.

## Path resolver

`bin/gpowers-path` is the single API skills use to find paths:

```bash
gpowers-path home                  # $GPOWERS_HOME
gpowers-path config                # $GPOWERS_CONFIG
gpowers-path state browser/tabs    # $GPOWERS_STATE/browser/tabs
gpowers-path project plans         # <repo>/.gpowers/plans (or fallback)
gpowers-path cache models          # $GPOWERS_CACHE/models
```

Skills MUST use this helper. Direct concatenation of `~/.gpowers/` in a skill body is a lint violation (caught by `tests/unit/tools/no-gstack-paths.bats` and friends).
