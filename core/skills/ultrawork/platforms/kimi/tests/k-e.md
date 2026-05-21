# K-E — Install/Uninstall Idempotency (Kimi Native)

**Setup:** Clean project directory with no `.kimi/`.

**Steps:**
1. Run `./install.sh`.
2. Verify files exist and config has exactly one block.
3. Run `./install.sh` again.
4. Verify no duplicate entries in `config.toml` or `agent.yaml`.
5. Manually uninstall (follow README.md steps).
6. Verify `agent.yaml` has no `oracle:` key.
7. Verify `config.toml` has no hook block.
8. Run `./install.sh` again.
9. Verify clean re-install.
10. Run `./install.sh --force` with existing `oracle:` key.
11. Verify overwrite succeeds.

**Expected:**
- No duplicate markers in `config.toml`.
- No duplicate `oracle:` keys in `agent.yaml`.
- Without `--force`, duplicate key aborts with error message.
- With `--force`, overwrite succeeds.

**Evidence to capture:**
- `config.toml` before/after showing single block.
- `agent.yaml` before/after showing single `oracle:` key.
- Error message from abort.
