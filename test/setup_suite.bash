setup_suite() {
  make >/dev/null 2>&1
  cp dorothy "$BATS_SUITE_TMPDIR/dorothy"
  PATH="$BATS_SUITE_TMPDIR:$PATH"
}
