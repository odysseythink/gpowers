# gpowers

Unified distribution combining [gstack](https://github.com/garrytan/gstack)'s
23 role-based commands with [superpowers](https://github.com/obra/superpowers)'s
14 methodology skills.

## Install

```bash
git clone https://github.com/<your>/gpowers ~/.gpowers/repo && \
  ~/.gpowers/repo/install
```

See `docs/superpowers/specs/2026-05-14-gpowers-merge-design.md` for design.

## Development

```bash
# Run tests
./tests/run-all.sh

# Dry-run an install
./install --dry-run --with-business

# Detect platforms
./bin/gpowers detect-platforms
```

Requirements: bash 4+, jq, bats-core (for tests).
