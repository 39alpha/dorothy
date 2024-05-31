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

@test "can commit single file" {
  local content='# Welcome to Dorothy'

  echo "$content" > README.md
  [[ -f README.md ]]

  local MESSAGE="Initial commit"

  dorothy commit -m "$MESSAGE" README.md

  local MANIFEST_HASH
  MANIFEST_HASH=$(cat .dorothy/manifest)

  local MANIFEST
  MANIFEST=$(ipfs cat "$MANIFEST_HASH")
  echo "$MANIFEST"

  # Has a single version
  [[ $(echo "$MANIFEST" | jq ".versions | length") -eq 1 ]]

  # Has commit author
  assert_equal "$(echo "$MANIFEST" | jq '.versions[0].author')" "\"$NAME <$EMAIL>\""

  # Has commit message
  assert_equal "$(echo "$MANIFEST" | jq '.versions[0].message')" "\"$MESSAGE\""

  # Is a FILE
  assert_equal "$(echo "$MANIFEST" | jq '.versions[0].path_type')" "\"FILE\""

  # Has no parents
  [[ $(echo "$MANIFEST" | jq '.versions[0].parents | length') -eq 0 ]] 

  local VERSION_HASH
  VERSION_HASH=$(echo "$MANIFEST" | jq '.versions[0].hash' | sed 's/"//g')

  run ipfs cat "$VERSION_HASH"
  assert_output "$content"

  run ipfs pin ls 
  assert_output --partial "$MANIFEST_HASH recursive"
  assert_output --partial "$VERSION_HASH recursive"
}

@test "can commit directory" {
  local content='# Welcome to Dorothy'

  mkdir data

  echo "$content" > data/README.md
  [[ -f data/README.md ]]

  local MESSAGE="Initial commit"

  dorothy commit -m "$MESSAGE" data

  local MANIFEST_HASH
  MANIFEST_HASH=$(cat .dorothy/manifest)

  local MANIFEST
  MANIFEST=$(ipfs cat "$MANIFEST_HASH")
  echo "$MANIFEST"

  # Has a single version
  [[ $(echo "$MANIFEST" | jq ".versions | length") -eq 1 ]]

  # Has commit author
  assert_equal "$(echo "$MANIFEST" | jq '.versions[0].author')" "\"$NAME <$EMAIL>\""

  # Has commit message
  assert_equal "$(echo "$MANIFEST" | jq '.versions[0].message')" "\"$MESSAGE\""

  # Is a FILE
  assert_equal "$(echo "$MANIFEST" | jq '.versions[0].path_type')" "\"DIRECTORY\""

  # Has no parents
  [[ $(echo "$MANIFEST" | jq '.versions[0].parents | length') -eq 0 ]] 

  local VERSION_HASH
  VERSION_HASH=$(echo "$MANIFEST" | jq '.versions[0].hash' | sed 's/"//g')

  run ipfs pin ls 
  assert_output --partial "$MANIFEST_HASH recursive"
  assert_output --partial "$VERSION_HASH recursive"
}

@test "can commit multiple files" {
  local content='# Welcome to Dorothy'

  mkdir data

  echo "$content" > README.md
  [[ -f README.md ]]

  echo "$content" > data/README.md
  [[ -f data/README.md ]]

  local MESSAGE="Initial commit"

  dorothy commit -m "$MESSAGE" README.md data

  local MANIFEST_HASH
  MANIFEST_HASH=$(cat .dorothy/manifest)

  local MANIFEST
  MANIFEST=$(ipfs cat "$MANIFEST_HASH")
  echo "$MANIFEST"

  # Has a single version
  [[ $(echo "$MANIFEST" | jq ".versions | length") -eq 1 ]]

  # Has commit author
  assert_equal "$(echo "$MANIFEST" | jq '.versions[0].author')" "\"$NAME <$EMAIL>\""

  # Has commit message
  assert_equal "$(echo "$MANIFEST" | jq '.versions[0].message')" "\"$MESSAGE\""

  # Is a FILE
  assert_equal "$(echo "$MANIFEST" | jq '.versions[0].path_type')" "\"DIRECTORY\""

  # Has no parents
  [[ $(echo "$MANIFEST" | jq '.versions[0].parents | length') -eq 0 ]] 

  local VERSION_HASH
  VERSION_HASH=$(echo "$MANIFEST" | jq '.versions[0].hash' | sed 's/"//g')

  run ipfs pin ls 
  assert_output --partial "$MANIFEST_HASH recursive"
  assert_output --partial "$VERSION_HASH recursive"
}

