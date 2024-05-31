setup() {
  load 'test_helper/bats-support/load'
  load 'test_helper/bats-assert/load'
  load 'test_helper/load'

  cd "$BATS_TEST_TMPDIR" || return 1

  # Prevents loading the global config on the testing box
  export XDG_CONFIG_HOME="$BATS_TEST_TMPDIR/global_config"

  require_ipfs_command
}

teardown() {
  cd "$PROJECT_ROOT" || return 1
}

@test "canary" {
  run echo "$XDG_CONFIG_HOME"
  assert_output "$BATS_TEST_TMPDIR/global_config"
}

@test "local initialization" {
  disallow_ipfs_daemon

  export IPFS_PATH=".dorothy"

  run dorothy init
  [[ "$status" -eq 0 ]]
  assert_output "Dorothy initialized"

  [[ -d .dorothy ]]

  for dir in blocks datastore keystore; do
    [[ -d ".dorothy/$dir" ]]
  done

  for file in config config.toml datastore_spec manifest repo.lock version; do
    [[ -f ".dorothy/$file" ]]
  done

  run cat .dorothy/config.toml
  assert_output ""

  run cat .dorothy/manifest
  assert_regex "^Q."

  local MANIFEST_HASH="$output"
  run ipfs pin ls
  assert_output "$MANIFEST_HASH recursive"
}

@test "global initalization" {
  require_ipfs_daemon

  run dorothy init -g
  [[ "$status" -eq 0 ]]
  assert_output "Dorothy initialized"

  [[ -d .dorothy ]]

  for dir in blocks datastore keystore; do
    ! [[ -e ".dorothy/$dir" ]]
  done

  for file in config datastore_spec repo.lock version; do
    ! [[ -e ".dorothy/$file" ]]
  done

  run dorothy config get ipfs.global
  assert_output "true"

  run cat .dorothy/manifest
  assert_regex "^Q."

  local MANIFEST_HASH="$output"
  run ipfs pin ls
  assert_output --partial "$MANIFEST_HASH recursive"
}
