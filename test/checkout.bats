setup() {
  load 'test_helper/bats-support/load'
  load 'test_helper/bats-assert/load'
  load 'test_helper/load'

  cd "$BATS_TEST_TMPDIR" || return 1

  # Prevents loading the global config on the testing box
  export XDG_CONFIG_HOME="$BATS_TEST_TMPDIR/global_config"

  # Prevents writing to global ipfs
  export IPFS_PATH=".dorothy"

  require_ipfs_command

  export NAME="John Doe"
  export EMAIL="john.doe@39alpharesearch.org"

  dorothy init >/dev/null 2>&1

  run dorothy config set user.name "$NAME"
  assert_output "wrote to \"$BATS_TEST_TMPDIR/.dorothy/config.toml\""

  run dorothy config set user.email "$EMAIL"
  assert_output "wrote to \"$BATS_TEST_TMPDIR/.dorothy/config.toml\""
}

teardown() {
  cd "$PROJECT_ROOT" || return 1
}

@test "canary" {
  run echo "$XDG_CONFIG_HOME"
  assert_output "$BATS_TEST_TMPDIR/global_config"

  run echo "$IPFS_PATH"
  assert_output ".dorothy"
}

@test "can checkout single file" {
  local content='# Welcome to Dorothy'

  echo "$content" > README.md
  [[ -f README.md ]]
  run cat README.md
  assert_output "$content"

  dorothy commit -m "Initial commit" README.md

  dorothy checkout "$(dorothy log | awk '/Hash/{ print $2}')" NEW_README.md
  [[ -f NEW_README.md ]]

  diff README.md NEW_README.md
}

@test "can checkout directory" {
  local content='# Welcome to Dorothy'

  mkdir data
  echo "$content" > data/README.md
  [[ -f data/README.md ]]
  run cat data/README.md
  assert_output "$content"

  dorothy commit -m "Initial commit" data

  dorothy checkout "$(dorothy log | awk '/Hash/{ print $2}')" data2
  [[ -d data2 ]]
  [[ -f data2/README.md ]]

  diff data/README.md data2/README.md
}

@test "can checkout multiple commit" {
  local content='# Welcome to Dorothy'

  mkdir data

  echo "$content 1" > README.md
  [[ -f README.md ]]
  run cat README.md
  assert_output "$content 1"

  echo "$content 2" > data/README.md
  [[ -f data/README.md ]]
  run cat data/README.md
  assert_output "$content 2"

  dorothy commit -m "Initial commit" README.md data

  dorothy checkout "$(dorothy log | awk '/Hash/{ print $2}')" data2
  [[ -d data2 ]]
  [[ -f data2/README.md ]]
  [[ -d data2/data ]]
  [[ -f data2/data/README.md ]]

  diff README.md data2/README.md
  diff data/README.md data2/data/README.md
}

