# Acknowledgements

The Oracle persona is adapted from **oh-my-opencode v3.17.10** (`src/agents/oracle.ts`,
`ORACLE_DEFAULT_PROMPT`), with the following changes:

- De-OpenCode-ified — vendor-specific tool names abstracted to generic verbs.
- Dual-mode added: general advisor (source default) + Ultrawork verifier (gpowers extension).
- Tool Discipline section explicitly enumerates the read-only contract that the
  upstream encodes via `createAgentToolRestrictions`.
- AI-slop deny-list explicit (upstream relies on prompt voice).

Upstream version pinned in `roles/upstream-source.json` under the `personas:` entry.
Manual refresh during gpowers maintenance windows; not auto-synced.

License terms from oh-my-opencode (`LICENSE.md`) apply to the adapted prompt content.
