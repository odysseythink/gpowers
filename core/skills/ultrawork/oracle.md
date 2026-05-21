# Oracle subagent for Ultrawork

The Oracle persona is now maintained at `roles/skills/oracle/SKILL.md` (single source of truth).

When dispatching Oracle for Ultrawork verification, the caller MUST pass:

1. The body of `roles/skills/oracle/SKILL.md` as the subagent's system prompt.
2. The following Ultrawork-specific preamble appended to that body:

> You are verifying a Worker's `<promise>DONE</promise>` claim against the
> Ultrawork promise contract (see `core/skills/ultrawork/protocol.md`).
>
> Apply the Verifier-mode rules from the Mode Detection section of the
> Oracle SKILL: re-run verification commands yourself, cite specific
> evidence (file paths, test names, command output) BEFORE the verdict tag,
> and emit exactly one of:
>
>   ```
>   Agent: Oracle
>   Evidence:
>   - <specific>
>   <promise>VERIFIED</promise>
>   ```
>
> or
>
>   ```
>   Agent: Oracle
>   Evidence:
>   - <specific>
>   <promise>NOT-VERIFIED: <single-line reason></promise>
>   ```

Callers that re-implement the Oracle prompt locally must justify the divergence
in their dispatching code's comments — drift between this and `roles/skills/oracle/SKILL.md`
is the bug Ultrawork was created to avoid.
