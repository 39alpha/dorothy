require_ipfs_command() {
  if ! which ipfs > /dev/null 2>&1 ; then
    skip
  fi
}

require_ipfs_daemon() {
  if ! pgrep ipfs ; then
    skip "requires ipfs daemon"
  fi
}

disallow_ipfs_daemon() {
  if pgrep ipfs ; then
    skip "ipfs daemon cannot be running"
  fi
}

