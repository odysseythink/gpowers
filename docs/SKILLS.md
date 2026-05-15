# gpowers skills index

Complete inventory of gpowers skills across all four modules. See [ARCHITECTURE.md](ARCHITECTURE.md) for the module model.

The table below is auto-generated. To regenerate after `gpowers upgrade` or local edits:

```bash
gpowers docs regen          # uses _gpowers-docgen.sh
```

<!-- gpowers:generated:begin kind=skills -->
| Module | Skill | Description |
|---|---|---|
| core | brainstorming | stub fixture for brainstorming |
| core | dispatching-parallel-agents | stub fixture for dispatching-parallel-agents |
| core | executing-plans | stub fixture for executing-plans |
| core | finishing-a-development-branch | stub fixture for finishing-a-development-branch |
| core | receiving-code-review | stub fixture for receiving-code-review |
| core | requesting-code-review | stub fixture for requesting-code-review |
| core | subagent-driven-development | stub fixture for subagent-driven-development |
| core | systematic-debugging | stub fixture for systematic-debugging |
| core | test-driven-development | stub fixture for test-driven-development |
| core | using-git-worktrees | stub fixture for using-git-worktrees |
| core | using-gpowers | Entry skill — establishes the four-module model (core/roles/tools/business) and dual-track triggering. Invoked automatic |
| core | verification-before-completion | stub fixture for verification-before-completion |
| core | writing-plans | stub fixture for writing-plans |
| core | writing-skills | stub fixture for writing-skills |
| roles | autoplan | stub fixture for gstack role autoplan |
| roles | codex | stub fixture for gstack role codex |
| roles | cso | stub fixture for gstack role cso |
| roles | design-consultation | stub fixture for gstack role design-consultation |
| roles | design-html | stub fixture for gstack role design-html |
| roles | design-review | post-implementation visual review (gstack) |
| roles | design-shotgun | stub fixture for gstack role design-shotgun |
| roles | devex-review | stub fixture for gstack role devex-review |
| roles | document-release | stub fixture for gstack role document-release |
| roles | investigate | stub fixture for gstack role investigate |
| roles | learn | stub fixture for gstack role learn |
| roles | office-hours | stub fixture for gstack role office-hours |
| roles | pair-agent | stub fixture for gstack role pair-agent |
| roles | plan-ceo-review | stub fixture for gstack role plan-ceo-review |
| roles | plan-design-review | stub fixture for gstack role plan-design-review |
| roles | plan-devex-review | stub fixture for gstack role plan-devex-review |
| roles | plan-eng-review | stub fixture for gstack role plan-eng-review |
| roles | plan-tune | stub fixture for gstack role plan-tune |
| roles | pr-review | pre-merge PR review (gstack original) |
| roles | retro | stub fixture for gstack role retro |
| tools | aidesigner | stub fixture for aidesigner |
| tools | aidesigner-frontend | stub fixture for aidesigner-frontend |
| tools | benchmark | stub fixture for benchmark |
| tools | benchmark-models | stub fixture for gstack benchmark-models |
| tools | browse | stub fixture for browse |
| tools | canary | stub fixture for canary |
| tools | careful | stub fixture for gstack careful |
| tools | context-restore | stub fixture for gstack context-restore |
| tools | context-save | stub fixture for gstack context-save |
| tools | fewer-permission-prompts | stub fixture for gstack fewer-permission-prompts |
| tools | fix-the-roof | stub fixture for gstack fix-the-roof |
| tools | freeze | stub fixture for gstack freeze |
| tools | gpowers-upgrade | Pull upstream changes for any gpowers module (core / roles / tools / business) — git subtree mechanics, transform re-app |
| tools | guard | stub fixture for gstack guard |
| tools | health | stub fixture for gstack health |
| tools | land-and-deploy | stub fixture for gstack land-and-deploy |
| tools | landing-report | stub fixture for gstack landing-report |
| tools | make-pdf | stub fixture for gstack make-pdf |
| tools | open-gstack-browser | stub fixture for open-gstack-browser |
| tools | qa | stub fixture for qa |
| tools | qa-only | stub fixture for qa-only |
| tools | setup-browser-cookies | stub fixture for setup-browser-cookies |
| tools | setup-deploy | stub fixture for gstack setup-deploy |
| tools | setup-gbrain | stub fixture for setup-gbrain |
| tools | ship | stub fixture for gstack ship |
| tools | simplify | stub fixture for gstack simplify |
| tools | sync-gbrain | stub fixture for sync-gbrain |
| tools | unfreeze | stub fixture for gstack unfreeze |
| business | acquire-retain | stub fixture for gstack business skill acquire-retain |
| business | compounding-filter | stub fixture for gstack business skill compounding-filter |
| business | contrarian-timing | stub fixture for gstack business skill contrarian-timing |
| business | idea-evaluator | stub fixture for gstack business skill idea-evaluator |
| business | idea-generator | stub fixture for gstack business skill idea-generator |
| business | jtbd-mapping | stub fixture for gstack business skill jtbd-mapping |
| business | money | stub fixture for gstack business skill money |
| business | money-ads | stub fixture for gstack business skill money-ads |
| business | money-content | stub fixture for gstack business skill money-content |
| business | money-discover | stub fixture for gstack business skill money-discover |
| business | money-finance | stub fixture for gstack business skill money-finance |
| business | money-ops | stub fixture for gstack business skill money-ops |
| business | money-outreach | stub fixture for gstack business skill money-outreach |
| business | money-product | stub fixture for gstack business skill money-product |
| business | money-seo | stub fixture for gstack business skill money-seo |
| business | money-social | stub fixture for gstack business skill money-social |
| business | money-strategy | stub fixture for gstack business skill money-strategy |
| business | mvp-first | stub fixture for gstack business skill mvp-first |
| business | pain-archaeology | stub fixture for gstack business skill pain-archaeology |
| business | sell-the-outcome | stub fixture for gstack business skill sell-the-outcome |
<!-- gpowers:generated:end -->

## Adding a skill

See [CONTRIBUTING.md](CONTRIBUTING.md) for the contribution workflow. Briefly:

1. Decide the module (core / roles / tools / business).
2. Create `<module>/skills/<name>/SKILL.md` with frontmatter `name`, `description`, `namespace: <module>`, and (for roles/tools/business) `slash: /<command>`.
3. If browser-using: add `requires-driver: browser` and use only the 9-verb interface (see [DRIVERS.md](DRIVERS.md)).
4. Run `gpowers-platforms gen all` to refresh per-platform manifests.
