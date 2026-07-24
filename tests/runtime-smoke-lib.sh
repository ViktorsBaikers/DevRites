#!/usr/bin/env bash

runtime_explicit_roots_ready() {
  [ -n "${1:-}" ] && [ -n "${2:-}" ] && [ -d "$1" ] && [ -d "$2" ]
}

runtime_secure_auth_file_ready() {
  local path="${1:-}" mode
  [ -n "$path" ] && [ -f "$path" ] && [ ! -L "$path" ] || return 1
  if mode="$(stat -f '%Lp' "$path" 2>/dev/null)"; then
    :
  else
    mode="$(stat -c '%a' "$path" 2>/dev/null)" || return 1
  fi
  [ "$mode" = "600" ] && [ -s "$path" ]
}
