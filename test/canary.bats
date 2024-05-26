setup() {
  load 'test_helper/bats-support/load'
  load 'test_helper/bats-assert/load'
  cd "$BATS_TEST_TMPDIR" || return 1
}

teardown() {
  cd "$PROJECT_ROOT" || return 1
}

@test "can run dorothy" {
  run dorothy
  assert_output --partial "A stab at data management"
}
