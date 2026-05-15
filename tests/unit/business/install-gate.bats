#!/usr/bin/env bats

setup() {
  REPO="$BATS_TEST_DIRNAME/../../.."
  export PATH="$REPO/bin:$PATH"
  TMP="$BATS_TEST_TMPDIR/gp"
  mkdir -p "$TMP"
}

@test "install --dry-run without --with-business skips business module" {
  out=$(GPOWERS_HOME="$TMP" "$REPO/install" --dry-run --non-interactive)
  echo "$out" | grep -qi "skip.*business\|business.*skipped"
  ! echo "$out" | grep -qi "copy.*business/skills"
}

@test "install --dry-run --with-business --non-interactive shows business activation" {
  out=$(GPOWERS_HOME="$TMP" "$REPO/install" --dry-run --with-business --non-interactive)
  echo "$out" | grep -qi "business"
  echo "$out" | grep -qi "DISCLAIMER\|disclaimer"
}

@test "install --with-business in interactive mode requires confirmation" {
  # Send 'n' (no) — install should abort business
  out=$(echo n | GPOWERS_HOME="$TMP" "$REPO/install" --dry-run --with-business)
  echo "$out" | grep -qi "abort\|cancel\|skipping business"
}

@test "install --with-business interactive 'y' proceeds" {
  out=$(echo y | GPOWERS_HOME="$TMP" "$REPO/install" --dry-run --with-business)
  echo "$out" | grep -qi "business.*activat\|installing business"
}
