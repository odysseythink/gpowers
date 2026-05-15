# Source me. Defines the migration map.
# Each rule is: <type>|<src-pattern>|<dst-template>|<comment>
# type: file | dir | project-glob | worktree-glob
# Templates expand:
#   ${GPOWERS_HOME}           = gpowers home (default ~/.gpowers)
#   ${HOME}
#   ${slug}                   = matched project slug (project-glob only)
#   <project_repo>            = resolved repo path or fallback
#   ${remainder}              = leftover path after pattern strip

GPOWERS_MIGRATION_RULES=(
  # ---------- gstack global config ----------
  "file|${HOME}/.config/gstack/compact.toml|${GPOWERS_HOME}/config/compact.toml|XDG compact config"
  "dir|${HOME}/.config/gstack/compact-rules|${GPOWERS_HOME}/config/compact-rules|XDG compact rules"
  "file|${HOME}/.gstack/builder-profile|${GPOWERS_HOME}/config/builder-profile|builder profile"
  "file|${HOME}/.gstack/developer-profile|${GPOWERS_HOME}/config/developer-profile|developer profile"
  "file|${HOME}/.gstack/gbrain-repo-policy|${GPOWERS_HOME}/config/gbrain-repo-policy|gbrain policy"
  "file|${HOME}/.gstack/plan-tune.toml|${GPOWERS_HOME}/config/plan-tune.toml|plan-tune config"

  # ---------- gstack state ----------
  "file|${HOME}/.gstack/installation-id|${GPOWERS_HOME}/state/installation-id|install id"
  "file|${HOME}/.gstack/last-update-check|${GPOWERS_HOME}/state/last-update-check|update check timestamp"
  "file|${HOME}/.gstack/update-snoozed|${GPOWERS_HOME}/state/update-snoozed|snooze marker"
  "file|${HOME}/.gstack/just-upgraded-from|${GPOWERS_HOME}/state/just-upgraded-from|prev version pointer"
  "dir|${HOME}/.gstack/security|${GPOWERS_HOME}/state/security|security state"

  # ---------- gstack cache ----------
  "dir|${HOME}/.gstack/browse|${GPOWERS_HOME}/cache/browse|browser cache"
  "dir|${HOME}/.gstack/cache/chromium-profile|${GPOWERS_HOME}/cache/browser/chromium-profile|chromium profile"
  "dir|${HOME}/.gstack/models|${GPOWERS_HOME}/cache/models|AI models"
  "dir|${HOME}/.gstack/repos|${GPOWERS_HOME}/cache/repos|clone mirrors"
  "dir|${HOME}/.gstack/cache|${GPOWERS_HOME}/cache|catch-all cache"

  # ---------- gstack analytics ----------
  "dir|${HOME}/.gstack/analytics|${GPOWERS_HOME}/analytics|telemetry"

  # ---------- gstack global data ----------
  "dir|${HOME}/.gstack/data/browser-skills|${GPOWERS_HOME}/data/browser-skills|user browser skills"
  "dir|${HOME}/.gstack/data/global-domain-skills|${GPOWERS_HOME}/data/global-domain-skills|domain skills"

  # ---------- gstack project-scoped data (slug-based) ----------
  "project-glob|${HOME}/.gstack/projects/<slug>/ceo-plans|<project_repo>/.gpowers/plans/ceo|per-project ceo plans"
  "project-glob|${HOME}/.gstack/projects/<slug>/eng-plans|<project_repo>/.gpowers/plans/eng|per-project eng plans"
  "project-glob|${HOME}/.gstack/projects/<slug>/design-plans|<project_repo>/.gpowers/plans/design|per-project design plans"
  "project-glob|${HOME}/.gstack/projects/<slug>/devex-plans|<project_repo>/.gpowers/plans/devex|per-project devex plans"
  "project-glob|${HOME}/.gstack/projects/<slug>/autoplans|<project_repo>/.gpowers/plans/autoplan|per-project autoplans"
  "project-glob|${HOME}/.gstack/projects/<slug>/designs|<project_repo>/.gpowers/designs|per-project designs"
  "project-glob|${HOME}/.gstack/projects/<slug>/evals|<project_repo>/.gpowers/evals|per-project evals"
  "project-glob|${HOME}/.gstack/projects/<slug>/canary|<project_repo>/.gpowers/canary|per-project canary"
  "project-glob|${HOME}/.gstack/projects/<slug>/health|<project_repo>/.gpowers/health|per-project health"
  "project-glob|${HOME}/.gstack/projects/<slug>/benchmark|<project_repo>/.gpowers/benchmark|per-project benchmark"
  "project-glob|${HOME}/.gstack/projects/<slug>/learnings|<project_repo>/.gpowers/learnings|per-project learnings"

  # ---------- superpowers worktrees ----------
  "worktree-glob|${HOME}/.config/superpowers/worktrees/<project>/<branch>|${GPOWERS_HOME}/state/worktrees/<project>/<branch>|worktree state"

  # ---------- legacy catch-all ----------
  "dir|${HOME}/.gstack/sessions|${GPOWERS_HOME}/data/sessions|global session catch-all"
  "dir|${HOME}/.gstack/retros|${GPOWERS_HOME}/data/retros/global|global retros"
  "dir|${HOME}/.gstack/learnings|${GPOWERS_HOME}/data/learnings/global|global learnings"
  "dir|${HOME}/.gstack/investigate-sessions|${GPOWERS_HOME}/data/investigate-sessions|global investigations"
)
