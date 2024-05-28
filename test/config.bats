setup() {
  load 'test_helper/bats-support/load'
  load 'test_helper/bats-assert/load'
  cd "$BATS_TEST_TMPDIR" || return 1

  mkdir -p global_config/dorothy
  export XDG_CONFIG_HOME="$BATS_TEST_TMPDIR/global_config"
  export DOROTHY_GLOBAL_CONFIG="$XDG_CONFIG_HOME/dorothy/config.toml"

  dorothy init >/dev/null 2>&1
}

teardown() {
  cd "$PROJECT_ROOT" || return 1
}

@test "canary" {
  run echo "$XDG_CONFIG_HOME"
  assert_output "$BATS_TEST_TMPDIR/global_config"

  run echo "$DOROTHY_GLOBAL_CONFIG"
  assert_output "$BATS_TEST_TMPDIR/global_config/dorothy/config.toml"
}

@test "loads global config" {
  run dorothy config get user.name
  assert_failure

  cat >"$DOROTHY_GLOBAL_CONFIG" <<EOL
[user]
name = "John Doe"
EOL

  run dorothy config get user.name
  assert_output "John Doe"
}

@test "sets global config" {
  run dorothy config get user.name
  assert_failure

  run dorothy config set -g user.name 'John Doe'
  assert_output "wrote to \"$DOROTHY_GLOBAL_CONFIG\""

  run dorothy config get user.name
  assert_output "John Doe"
}

@test "loads local config" {
  run dorothy config get user.name
  assert_failure

  cat >".dorothy/config.toml" <<EOL
[user]
name = "John Doe"
EOL

  run dorothy config get user.name
  assert_output "John Doe"
}

@test "sets local config" {
  run dorothy config get user.name
  assert_failure

  run dorothy config set user.name 'John Doe'
  assert_output "wrote to \"$BATS_TEST_TMPDIR/.dorothy/config.toml\""

  run dorothy config get user.name
  assert_output "John Doe"
}

@test "loads custom config" {
  run dorothy config get user.name
  assert_failure

  cat >"custom_config.toml" <<EOL
[user]
name = "John Doe"
EOL

  run dorothy config get -c custom_config.toml user.name
  assert_output "John Doe"
}

@test "sets custom config" {
  run dorothy config get -c custom_config.toml user.name
  assert_failure

  run dorothy config set -c custom_config.toml user.name 'John Doe'
  assert_output "wrote to \"custom_config.toml\""

  run dorothy config get -c custom_config.toml user.name
  assert_output "John Doe"
}
