# Releasing gpowers

gpowers follows semantic versioning: `vMAJOR.MINOR.PATCH`.

- **MAJOR** — driver interface or module boundary change.
- **MINOR** — new skill, new platform, new opt-in module.
- **PATCH** — bug fix or upstream sync.

## Cutting a release

1. Verify `main` is green:
   ```bash
   ./tests/run.sh all
   ```

2. Bump the version in `manifest.json` (`.version`) and any READMEs that
   reference it. Commit:
   ```bash
   git commit -am "chore: bump version to v1.2.3"
   ```

3. Tag:
   ```bash
   git tag -a v1.2.3 -m "release v1.2.3"
   git push origin main --follow-tags
   ```

4. The `release.yml` workflow runs automatically. It:
   - Re-runs the full test suite
   - Builds `gpowers-1.2.3.tar.gz` + `gpowers-1.2.3.tar.gz.sha256`
   - Creates a GitHub Release with the tarball and auto-generated notes

5. Manually verify the release artifact is downloadable from the GitHub Releases page.

## Hotfix from a non-main branch

If a hotfix is needed off an older minor:
```bash
git checkout -b hotfix/v1.1.x v1.1.4
# make fix
git tag -a v1.1.5 -m "patch: hotfix"
git push origin hotfix/v1.1.x --follow-tags
```

The release workflow runs against the tag's checkout regardless of branch.
