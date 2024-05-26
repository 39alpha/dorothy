setup() {
  load 'test_helper/bats-support/load'
  load 'test_helper/bats-assert/load'
  cd "$BATS_TEST_TMPDIR" || return 1
}

teardown() {
  cd "$PROJECT_ROOT" || return 1
}

@test "log fails if outside repository" {
  run dorothy log
  [ "$status" -eq 1 ]
  assert_output "fatal: not a dorothy repository"
}

@test "log successed in empty repository" {
  dorothy init
  run dorothy log
  assert_output "no versions"
}

@test "log prints versions" {
  local NAME="John Doe"
  local EMAIL="john.doe@39alpharesearch.org"
  local MESSAGE="John Doe"


  dorothy init
  dorothy config set user.name "$NAME"
  dorothy config set user.email "$EMAIL"
  touch README.md
  dorothy commit -m "$MESSAGE" README.md
  run dorothy log
  assert_output --partial "Hash:    QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQH"
  assert_output --partial "Author:  $NAME <$EMAIL>"
  assert_output --partial "Date:    "
  assert_output --partial "Type:    FILE"
  assert_output --partial "    $MESSAGE"
}
