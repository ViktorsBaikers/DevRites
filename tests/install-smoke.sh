#!/usr/bin/env bash
# Smoke-test the default dry run, a temporary install, an idempotent reinstall,
# and the generated Codex mirrors.
set -u
export DEVRITES_NO_BINARY=1   # pack smoke: the engine binary has its own lifecycle test (binary-lifecycle-test.sh)
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok()   { printf '  ok: %s\n' "$*"; }
no()   { printf '  FAIL: %s\n' "$*"; fail=1; }
T="$(mktemp -d)"
GEN="$(mktemp -d)"
cleanup() { rm -rf "$T" "$GEN"; }
trap cleanup EXIT
if [ -n "${DEVRITES_HOST_ARTIFACT_DIR:-}" ]; then
  cp -R "$DEVRITES_HOST_ARTIFACT_DIR"/. "$GEN"/ \
    || { echo "  FAIL: could not copy host artifacts"; exit 1; }
else
  DEVRITES_HOST_ARTIFACT_DIR="$GEN" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
    || { echo "  FAIL: could not build host artifacts"; exit 1; }
fi
export DEVRITES_HOST_ARTIFACT_DIR="$GEN"

# A partially present tracked payload must rebuild before install. On
# case-insensitive filesystems a local AGENTS.md exclude can hide agents.md.
rm -f \
  "$GEN/claude/skills/devrites-lib/reference/standards/agents.md" \
  "$GEN/codex/skills/devrites-lib/reference/standards/agents.md"
bash "$ROOT/install.sh" --target "$T" --dry-run >/dev/null 2>&1 || no "partial payload repair exited non-zero"
if [ -f "$GEN/claude/skills/devrites-lib/reference/standards/agents.md" ] \
  && [ -f "$GEN/codex/skills/devrites-lib/reference/standards/agents.md" ]; then
  ok "partial payload rebuild restores shared agent standard"
else
  no "partial payload accepted without shared agent standard"
fi
printf '\n<!-- install-smoke-generated-sentinel -->\n' >> "$GEN/codex/skills/rite/SKILL.md"

echo "== install-smoke (target: $T) =="

# 1) dry-run writes nothing
bash "$ROOT/install.sh" --target "$T" --dry-run >/dev/null 2>&1
[ -e "$T/.claude" ] && no "dry-run created .claude" || ok "dry-run changed nothing"
[ -e "$T/.agents" ] && no "dry-run created .agents" || true
[ -e "$T/.codex" ] && no "dry-run created .codex" || true
[ -e "$T/AGENTS.md" ] && no "dry-run created AGENTS.md" || true

# 2) real install
bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "install exited non-zero"
[ -f "$T/.claude/devrites.manifest" ] && ok "manifest written" || no "no manifest"
for f in \
  ".claude/skills/rite/SKILL.md" \
  ".agents/skills/rite/SKILL.md" \
  ".claude/skills/rite-define/SKILL.md" \
  ".agents/skills/rite-define/SKILL.md" \
  ".agents/skills/devrites-lib/reference/standards/core.md" \
  ".agents/skills/devrites-lib/reference/standards/agents.md" \
  ".agents/skills/devrites-lib/reference/standards/security.md" \
  ".claude/skills/rite-seal/SKILL.md" \
  ".claude/skills/rite-polish/SKILL.md" \
  ".claude/skills/rite-polish/reference/code.md" \
  ".claude/skills/rite-polish/reference/ui.md" \
  ".claude/skills/rite-pressure-test/SKILL.md" \
  ".claude/skills/devrites-doubt/SKILL.md" \
  ".claude/skills/devrites-lib/reference/parallel-dispatch.md" \
  ".claude/skills/devrites-frontend-craft/reference/shape.md" \
  ".claude/skills/devrites-debug-recovery/reference/build-the-loop.md" \
  ".claude/agents/devrites-code-reviewer.md" \
  ".codex/agents/devrites-code-reviewer.toml" \
  ".codex/config.toml" \
  ".claude/agents/devrites-spec-reviewer.md" \
  ".claude/skills/devrites-lib/reference/standards/security.md" \
  ".claude/skills/devrites-lib/reference/standards/agents.md" \
  ".claude/skills/devrites-lib/reference/standards/anti-patterns.md" \
  ".claude/skills/devrites-lib/reference/standards/README.md" \
  "AGENTS.md" \
  ".devrites/README.md" \
  ".devrites/ACTIVE" ; do
  [ -f "$T/$f" ] && ok "present: $f" || no "missing: $f"
done
[ ! -e "$T/.codex/hooks.json" ] && ok "clean install creates no Codex root hooks" || no "clean install created Codex root hooks"

# 2b) Deleted, merged, or renamed skills must not be present.
for f in \
  ".claude/skills/rite-polish-code/SKILL.md" \
  ".claude/skills/rite-polish-ui/SKILL.md" \
  ".claude/skills/devrites-context-pack/SKILL.md" \
  ".claude/skills/devrites-idea-refine/SKILL.md" \
  ".claude/skills/idea-pressure-test/SKILL.md" \
  ".claude/skills/devrites-prototype/SKILL.md" \
  ".claude/skills/devrites-handoff/SKILL.md" \
  ".claude/skills/devrites-zoom-out/SKILL.md" \
  ".claude/skills/devrites-code-simplifier/SKILL.md" \
  ".claude/skills/devrites-security-hardening/SKILL.md" \
  ".claude/skills/devrites-performance-check/SKILL.md" \
  ".claude/skills/devrites-parallel-review/SKILL.md" \
  ".claude/skills/devrites-rules/SKILL.md" ; do
  [ -f "$T/$f" ] && no "stale skill still installed: $f" || ok "removed: $f"
done

# /polish and /normalize aliases are removed in favor of /rite-polish modes.
[ -d "$T/.claude/skills/polish" ]    && no "stale /polish alias still installed"    || ok "no /polish alias (removed)"
[ -d "$T/.claude/skills/normalize" ] && no "stale /normalize alias still installed" || ok "no /normalize alias (removed)"

# 3) Check the manifest size and keep ACTIVE outside manifest management.
n="$(grep -vc '^#' "$T/.claude/devrites.manifest")"
[ "$n" -ge 80 ] && ok "manifest lists $n files" || no "manifest too small ($n)"
grep -q '\.devrites/ACTIVE' "$T/.claude/devrites.manifest" && no "ACTIVE should not be manifest-managed" || ok "ACTIVE excluded from manifest"
grep -q '^\.agents/skills/rite/SKILL.md$' "$T/.claude/devrites.manifest" && ok "manifest tracks Codex skill mirror" || no "manifest missing Codex skill mirror"
grep -q '^\.agents/skills/devrites-lib/reference/standards/core.md$' "$T/.claude/devrites.manifest" && ok "manifest tracks Codex rules mirror" || no "manifest missing Codex rules mirror"
grep -q '^\.codex/agents/devrites-code-reviewer.toml$' "$T/.claude/devrites.manifest" && ok "manifest tracks Codex custom agent" || no "manifest missing Codex custom agent"
grep -q '^AGENTS.md$' "$T/.claude/devrites.manifest" && no "AGENTS.md should be marker-managed, not file-managed" || ok "AGENTS.md not file-managed"
grep -q '^\.claude/devrites\.agents-merge$' "$T/.claude/devrites.manifest" && ok "AGENTS merge marker managed" || no "AGENTS merge marker missing"
grep -q '^\.codex/config.toml$' "$T/.claude/devrites.manifest" && no ".codex/config.toml should be marker-managed, not file-managed" || ok ".codex/config.toml not file-managed"
grep -q '^\.claude/devrites\.codex-config-merge$' "$T/.claude/devrites.manifest" && ok "Codex permission config merge marker managed" || no "Codex permission config merge marker missing"
grep -q '^\.codex/hooks.json$\\|^\.claude/devrites\.codex-hooks-merge$' "$T/.claude/devrites.manifest" && no "obsolete Codex hooks are manifest-managed" || ok "obsolete Codex hooks are absent from manifest"
grep -q '^\.codex/hooks/devrites-.*\.sh$' "$T/.claude/devrites.manifest" && no "post-cutover: Codex hook scripts should no longer be shipped" || ok "Codex hook scripts no longer shipped (engine binary is the control plane)"
grep -q '^\.codex/mcp/' "$T/.claude/devrites.manifest" && no "manifest tracks Codex MCP files" || ok "manifest does not track Codex MCP files"
grep -q 'install-smoke-generated-sentinel' "$T/.agents/skills/rite/SKILL.md" && ok "installer consumes generated Codex skill payload" || no "installer did not use generated Codex skill payload"
grep -q '## Codex compatibility' "$T/.agents/skills/rite-build/SKILL.md" && no "Codex skill mirror duplicates project-wide guidance" || ok "Codex skill mirror relies on the AGENTS bridge"
if grep -q 'fresh subagent thread' "$T/AGENTS.md" \
  && grep -q 'Dispatch every exact named `devrites-<role>`' "$T/AGENTS.md" \
  && grep -q 'default_permissions = ":read-only"' "$T/AGENTS.md" \
  && grep -q 'default_permissions = ":workspace"' "$T/AGENTS.md" \
  && grep -q 'never substitute a generic/default child' "$T/AGENTS.md" \
  && grep -q 'root must never edit product source or tests itself' "$T/AGENTS.md" \
  && grep -q 'exact path-bounded executable workflow artifacts' "$T/AGENTS.md" \
  && grep -q '`git diff --name-only`' "$T/AGENTS.md" \
  && grep -q 'reject any extra path' "$T/AGENTS.md" \
  && grep -q 'never recreate an engine dispatch bridge' "$T/AGENTS.md" \
  && ! grep -Eq 'agent_type=devrites-|fork_turns=|task_name=' "$T/AGENTS.md" \
  && ! grep -Eq 'MultiAgent V1|MultiAgent V2|required-agent-roles|agent-dispatch' "$T/.agents/skills/rite-build/SKILL.md"; then
  ok "AGENTS bridge owns native named fresh-context dispatch"
else
  no "AGENTS bridge missing fresh-context mapping"
fi
grep -q 'Read `.agents/skills/devrites-lib/reference/standards/core.md`' "$T/.agents/skills/rite-build/SKILL.md" && ok "Codex skill mirror loads DevRites rules mirror" || no "Codex skill mirror missing rules instruction"
grep -q 'repository-aware file tool refuses an ignored path.*native filesystem command.*not a completed task' "$T/AGENTS.md" && ok "installed AGENTS bridge recovers from ignored mirror refusals" || no "installed AGENTS bridge can return an ignored mirror refusal"
grep -q 'Engram calls.*omit optional `project` and `session_id`.*Never derive either from `task_name`.*mem_session_summary.*unknown_session.*unknown_project.*both optional fields omitted.*ambiguous.*ask the user' "$T/AGENTS.md" && ok "installed AGENTS bridge preserves exact Engram identifiers" || no "installed AGENTS bridge can invent Engram identifiers"
grep -q 'Codex custom-agent version\|repository-aware file tool refuses an ignored path\|For automatic Engram calls' "$T/.codex/agents/devrites-code-reviewer.toml" && no "installed Codex agent duplicates project-wide guidance" || ok "installed Codex agent contains only its role contract"
grep -q '\.claude/agents' "$T/.agents/skills/rite-build/SKILL.md" && no "Codex skill mirror still points at .claude/agents" || ok "Codex skill mirror does not point at .claude/agents"
grep -q '\.claude/skills/devrites-lib/reference/standards' "$T/.agents/skills/rite-build/SKILL.md" && no "Codex skill mirror still points at .claude/skills/devrites-lib/reference/standards" || ok "Codex skill mirror does not point at .claude/skills/devrites-lib/reference/standards"
grep -q '\.claude/skills' "$T/.agents/skills/rite-build/SKILL.md" && no "Codex skill mirror still points at .claude/skills" || ok "Codex skill mirror does not point at .claude/skills"
grep -q 'pack/\.claude' "$T/.agents/skills/rite-build/SKILL.md" && no "Codex skill mirror still points at pack/.claude" || ok "Codex skill mirror does not point at pack/.claude"
grep -q 'F=.agents/skills/rite-$V/SKILL.md' "$T/.agents/skills/rite/SKILL.md" && ok "Codex rite router dispatches inside .agents" || no "Codex rite router not rewritten to .agents"
grep -q '\.claude/skills/rite-$V' "$T/.agents/skills/rite/SKILL.md" && no "Codex rite router still points at .claude/skills" || ok "Codex rite router does not point at .claude/skills"
grep -R -q '\.\./.*agents/devrites-.*\.md' "$T/.agents/skills" && no "Codex skill mirrors still contain relative agent links" || ok "Codex skill mirrors rewrite relative agent links"
grep -q '\.codex/agents/devrites-slice-wright.toml' "$T/.agents/skills/rite-build/SKILL.md" && ok "Codex skill root points at Codex agent TOML" || no "Codex skill root missing Codex agent TOML path"
grep -q '\.claude/agents' "$T/.agents/skills/rite-build/reference/wright-dispatch.md" && no "Codex reference still points at .claude/agents" || ok "Codex reference rewrites .claude/agents"
grep -q '\.claude/skills/devrites-lib/reference/standards' "$T/.agents/skills/rite-build/reference/wright-dispatch.md" && no "Codex reference still points at .claude/skills/devrites-lib/reference/standards" || ok "Codex reference rewrites .claude/skills/devrites-lib/reference/standards"
grep -q 'standards/agents.md#source-writing-boundary' "$T/.agents/skills/rite-build/reference/wright-dispatch.md" \
  && grep -q 'exact `:workspace` wright' "$T/.agents/skills/rite-build/reference/wright-dispatch.md" \
  && grep -q 'substitute a generic agent' "$T/.agents/skills/rite-build/reference/wright-dispatch.md" \
  && grep -q 'never writes source/tests' "$T/.agents/skills/rite-build/reference/wright-dispatch.md" \
  && ok "Codex reference executes the exact writer while preserving the root boundary" \
  || no "Codex reference has the wrong host writer boundary"
grep -q 'exact project-relative source/test path list' "$T/.agents/skills/rite-build/reference/wright-dispatch.md" \
  && grep -q 'git diff --name-only' "$T/.agents/skills/rite-build/reference/wright-dispatch.md" \
  && ok "Codex reference preserves instruction-backed exact writer scope" \
  || no "Codex reference missing exact writer scope"
grep -q '\.claude/skills/devrites-lib/reference/standards' "$T/.codex/agents/devrites-code-reviewer.toml" && no "Codex agent still points at .claude/skills/devrites-lib/reference/standards" || ok "Codex agent uses mirrored rules paths"
grep -q '\$rite-build' "$T/.agents/skills/rite-define/reference/gates.md" && ok "Codex skill mirror rewrites slash rite invocations" || no "Codex skill mirror missing dollar rite invocation rewrite"
grep -q '\$rite-build' "$T/.codex/agents/devrites-slice-wright.toml" && ok "Codex agent descriptions rewrite slash rite invocations" || no "Codex agent descriptions missing dollar rite invocation rewrite"
grep -q 'exact project-relative source/test path list directly in the task' "$T/.agents/skills/rite-build/reference/phase-contract.md" \
  && grep -q 'git diff --name-only' "$T/.agents/skills/rite-build/reference/phase-contract.md" \
  && grep -Fq 'test hunks for deletion, skipping/focus, tautology, or weaker expectations' "$T/.agents/skills/rite-build/reference/phase-contract.md" \
  && grep -Fq 'repository/CI commands' "$T/.agents/skills/rite-build/reference/phase-contract.md" \
  && grep -q 'devrites-engine check readiness <slug>' "$T/.agents/skills/rite-build/reference/phase-contract.md" \
  && ! grep -q 'devrites-engine reconcile\\|devrites-engine test-integrity' "$T/.agents/skills/rite-build/reference/phase-contract.md" \
  && ok "Codex build contract keeps native boundary review with root" \
  || no "Codex build contract assigns deterministic gates inconsistently"
if grep -R -nE 'devrites-engine (readiness|seal|spec-validate|check-acceptance|evidence-fresh|coverage|doubt-coverage|test-integrity|review-integrity|build-readiness|readiness-digest|analyze|ledger|resolve|clarify-return|tick-afk|recovery|close-out|migrate)([[:space:]`]|$)' \
  "$T/.claude/skills" "$T/.claude/agents" "$T/.agents/skills" "$T/.codex/agents" \
  >/tmp/dr_install_retired_engine 2>/dev/null; then
  no "installed guidance retains retired engine commands"
  sed -n '1,20p' /tmp/dr_install_retired_engine
else
  ok "installed guidance uses only nested thin-engine commands"
fi
if grep -R -nE 'devrites-engine[[:space:]]+(check[[:space:]]+spec|state[[:space:]]+(clarify|tick-afk|recovery)([[:space:]`]|$)|state[[:space:]]+resolve[[:space:]]+next-qid|doctor([[:space:]`]|$))' \
  "$T/.claude/skills" "$T/.claude/agents" "$T/.agents/skills" "$T/.codex/agents" \
  >"$T/removed-policy-commands.log" 2>/dev/null; then
  no "installed guidance retains removed engine policy commands"
  sed -n '1,20p' "$T/removed-policy-commands.log"
else
  ok "installed guidance keeps spec, clarify, AFK, recovery, qid, and doctor policy native"
fi
if grep -q '^tools = .*Bash' "$T/.codex/agents/devrites-proof-runner.toml" \
   || grep -q 'Run only packet-approved' "$T/.codex/agents/devrites-proof-runner.toml"; then
  no "Codex proof runner still executes root-owned proof commands"
elif grep -q 'root-produced test, build, lint, typecheck, and browser evidence' "$T/.codex/agents/devrites-proof-runner.toml" \
   && grep -q 'The root runs only commands declared by `test-plan.md`' "$T/.agents/skills/rite-prove/SKILL.md"; then
  ok "Codex proof runner validates immutable root-owned proof"
else
  no "Codex proof ownership contract is incomplete"
fi
if grep -R -nE '(^|[^A-Za-z0-9_./-])/rite(-[a-z0-9-]+)?([^A-Za-z0-9_-]|$)' "$T/.agents/skills" "$T/.codex/agents" >/tmp/dr_codex_slash_rite 2>/dev/null; then
  no "Codex mirrors still contain slash rite invocations"
  sed -n '1,20p' /tmp/dr_codex_slash_rite
else
  ok "Codex mirrors contain no slash rite invocations"
fi
python3 - <<PY
import json, pathlib
try:
    import tomllib
except ModuleNotFoundError:
    tomllib = None
if tomllib is not None:
    tomllib.loads(pathlib.Path("$T/.codex/agents/devrites-code-reviewer.toml").read_text())
    tomllib.loads(pathlib.Path("$T/.codex/agents/devrites-slice-wright.toml").read_text())
    tomllib.loads(pathlib.Path("$T/.codex/config.toml").read_text())
PY
if [ "$?" -eq 0 ]; then
  if python3 - <<'PY' >/dev/null 2>&1
try:
    import tomllib  # noqa: F401
except ModuleNotFoundError:
    raise SystemExit(1)
PY
  then
    ok "generated Codex agent/root config TOML parses"
  else
    ok "generated Codex TOML parse skipped (python has no tomllib)"
  fi
else
  no "generated Codex agent/root config TOML invalid"
fi
grep -q 'default_permissions = "devrites-orchestrator"' "$T/.codex/config.toml" \
  && grep -q 'extends = ":workspace"' "$T/.codex/config.toml" \
  && ok "Codex root uses the native workspace-capable permission profile" \
  || no "Codex native root permission profile is incomplete"
grep -q 'default_permissions = ":workspace"' "$T/.codex/agents/devrites-slice-wright.toml" \
  && ! grep -q '\[\[hooks\.' "$T/.codex/agents/devrites-slice-wright.toml" \
  && ok "Codex slice-wright is the hook-free workspace writer" \
  || no "Codex slice-wright is not the native writer"
if grep -R -q '\[\[hooks\.' "$T/.codex/agents"; then
  no "Codex agents duplicate native permission policy with engine hooks"
else
  ok "Codex agents rely on native permission profiles"
fi

# 4) idempotent reinstall
out="$(bash "$ROOT/install.sh" --target "$T" 2>&1)"
echo "$out" | grep -q 'installed: 0' && ok "reinstall installs 0 new (idempotent)" || no "reinstall not idempotent"
echo "$out" | grep -q 'skipped(conflict): 0' && ok "reinstall skips 0 (all managed)" || no "reinstall reported conflicts"

# 5) Preserve managed customization by default and report forced dry runs accurately.
managed="$T/.claude/skills/rite/SKILL.md"
printf 'local customization\n' > "$managed"
out="$(bash "$ROOT/install.sh" --target "$T" 2>&1)" \
  && no "customized managed file was silently overwritten" \
  || ok "customized managed file aborts reinstall"
[ "$(cat "$managed")" = "local customization" ] && ok "default reinstall preserved customization" || no "default reinstall changed customization"
echo "$out" | grep -q -- 'rerun with --force' && ok "customization error gives force remediation" || no "customization error missing force remediation"
out="$(bash "$ROOT/install.sh" --target "$T" --force --dry-run 2>&1)" || no "forced dry-run failed"
echo "$out" | grep -q '\[overwrite(force-customized)\] .claude/skills/rite/SKILL.md' \
  && ok "forced dry-run predicts customized overwrite" || no "forced dry-run output inaccurate"
[ "$(cat "$managed")" = "local customization" ] && ok "forced dry-run preserved customization" || no "forced dry-run wrote customization"
bash "$ROOT/install.sh" --target "$T" --force >/dev/null 2>&1 || no "forced reinstall failed"
cmp -s "$managed" "$GEN/claude/skills/rite/SKILL.md" && ok "forced reinstall replaced customization" || no "forced reinstall did not replace customization"

echo ""
[ "$fail" -eq 0 ] && echo "install-smoke: PASS" || echo "install-smoke: FAIL"
exit "$fail"
