#!/usr/bin/env bash
# Shared harness for the per-host generator tests; source it, do not run it.
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

NAV="ctx_read, ctx_ls, ctx_find, ctx_grep, ctx_glob, ctx_search, ctx_compose, ctx_callgraph, ctx_tree, symbol_search, project_report, module_report, read_symbol, read_enclosing, lens_diagnostics"

expect_tools() {
  local got="$1" want="$2" label="$3"
  [ "$got" = "$want" ] && ok "$label" || {
    no "$label"
    printf '    got:  %s\n    want: %s\n' "$got" "$want"
  }
}
