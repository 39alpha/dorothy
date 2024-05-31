setup() {
  load 'test_helper/bats-support/load'
  load 'test_helper/bats-assert/load'
  load 'test_helper/load'

  cd "$BATS_TEST_TMPDIR" || return 1

  # Prevents loading the global config on the testing box
  export XDG_CONFIG_HOME="$BATS_TEST_TMPDIR/global_config"

  # Prevents writing to global ipfs
  export IPFS_PATH=".dorothy"
}

teardown() {
  cd "$PROJECT_ROOT" || return 1
}

@test "can run dorothy" {
  run dorothy
  assert_output --partial "A stab at data management"
}
