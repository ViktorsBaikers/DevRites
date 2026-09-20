#!/usr/bin/env bash
# Focused checks for the Claude-to-omp generator used by host packaging.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

TMP_GEN_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_GEN_DIR"' EXIT

# shellcheck source=../scripts/omp-generate.sh
. "$ROOT/scripts/omp-generate.sh"

echo "== omp-generator-test =="

NAV="ctx_read, ctx_ls, ctx_find, ctx_grep, ctx_glob, ctx_search, ctx_compose, ctx_callgraph, ctx_tree, symbol_search, project_report, module_report, read_symbol, read_enclosing, lens_diagnostics"

expect_tools() {
  local got="$1" want="$2" label="$3"
  [ "$got" = "$want" ] && ok "$label" || {
    no "$label"
    printf '    got:  %s\n    want: %s\n' "$got" "$want"
  }
}

expect_tools \
  "$(_omp_append_extension_tools "read, grep, glob, bash")" \
  "read, grep, glob, bash, ${NAV}, ctx_shell" \
  "bash base keeps order and appends nav plus ctx_shell"

expect_tools \
  "$(_omp_append_extension_tools "read, grep, glob")" \
  "read, grep, glob, ${NAV}" \
  "read-only base appends nav only"

expect_tools \
  "$(_omp_append_extension_tools "read, edit, write, bash, glob, grep")" \
  "read, edit, write, bash, glob, grep, ${NAV}, ctx_edit, ctx_patch, ctx_shell" \
  "wright base appends nav plus ctx_edit/ctx_patch/ctx_shell"

expect_tools \
  "$(_omp_append_extension_tools "read, write")" \
  "read, write, ${NAV}, ctx_edit, ctx_patch" \
  "write without bash gets ctx_edit/ctx_patch not ctx_shell"

expect_tools \
  "$(_omp_append_extension_tools "read, edit")" \
  "read, edit, ${NAV}, ctx_edit, ctx_patch" \
  "edit without write still gets ctx_edit/ctx_patch"

dup="$(_omp_append_extension_tools "read, ctx_read, bash")"
case ", ${dup}, " in
*", ctx_read, "*) ok "keeps existing ctx_read" ;;
*) no "keeps existing ctx_read" ;;
esac
printf '%s' "$dup" | grep -q 'ctx_read, ctx_read' && no "duplicates ctx_read" || ok "does not duplicate ctx_read"
printf '%s' "$dup" | grep -q 'ctx_shell' && ok "still appends ctx_shell after existing ctx_read" || no "dropped ctx_shell"

# Mapped Claude tools, then extension extras. Skill/Web* stay dropped.
expect_tools \
  "$(_omp_append_extension_tools "$(_omp_map_tools "Read, Grep, Glob, Bash, Skill, WebFetch" "devrites-code-reviewer")")" \
  "read, grep, glob, bash, ${NAV}, ctx_shell" \
  "maps Claude tools then appends extras"

expect_tools \
  "$(_omp_append_extension_tools "$(_omp_map_tools "Read, Grep, Glob" "devrites-proof-runner")")" \
  "read, grep, glob, ${NAV}" \
  "proof-runner map stays read-only"

wright_src="$TMP_GEN_DIR/devrites-slice-wright.md"
reviewer_src="$TMP_GEN_DIR/devrites-code-reviewer.md"
proof_src="$TMP_GEN_DIR/devrites-proof-runner.md"
cat >"$wright_src" <<'EOF'
---
name: devrites-slice-wright
description: Write-capable executor.
tools: Read, Edit, Write, Bash, Glob, Grep
---

Body.
EOF
cat >"$reviewer_src" <<'EOF'
---
name: devrites-code-reviewer
description: Reviews a diff.
tools: Read, Grep, Glob, Bash
---

Body.
EOF
cat >"$proof_src" <<'EOF'
---
name: devrites-proof-runner
description: Validates proof.
tools: Read, Grep, Glob
---

Body.
EOF

gen_omp_agent "$wright_src" "$TMP_GEN_DIR/out-wright.md"
gen_omp_agent "$reviewer_src" "$TMP_GEN_DIR/out-reviewer.md"
gen_omp_agent "$proof_src" "$TMP_GEN_DIR/out-proof.md"

grep -qxF "tools: read, edit, write, bash, glob, grep, ${NAV}, ctx_edit, ctx_patch, ctx_shell" \
  "$TMP_GEN_DIR/out-wright.md" &&
  ok "gen_omp_agent wright tools line" ||
  no "gen_omp_agent wright tools line"

grep -qxF "tools: read, grep, glob, bash, ${NAV}, ctx_shell" \
  "$TMP_GEN_DIR/out-reviewer.md" &&
  ok "gen_omp_agent reviewer tools line" ||
  no "gen_omp_agent reviewer tools line"

grep -qxF "tools: read, grep, glob, ${NAV}" \
  "$TMP_GEN_DIR/out-proof.md" &&
  ok "gen_omp_agent proof-runner tools line" ||
  no "gen_omp_agent proof-runner tools line"

readonly_ok=1
for f in "$TMP_GEN_DIR/out-reviewer.md" "$TMP_GEN_DIR/out-proof.md"; do
  line="$(grep '^tools:' "$f")"
  for bad in edit write ctx_edit ctx_patch; do
    case ", ${line#tools: }, " in
    *", ${bad}, "*)
      no "$(basename "$f") gained $bad"
      readonly_ok=0
      ;;
    esac
  done
done
[ "$readonly_ok" -eq 1 ] && ok "read-only generated agents stay read-only"

[ "$fail" -eq 0 ] && echo "omp-generator-test: PASS" || echo "omp-generator-test: FAIL"
exit "$fail"
