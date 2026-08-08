#!/usr/bin/env bash
# host-artifacts-test.sh: validate prebuilt Claude/Codex host artifacts.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
OUT="$T/generated"

echo "== host-artifacts-test =="

DEVRITES_HOST_ARTIFACT_DIR="$OUT" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
  && ok "build-host-artifacts completed" \
  || no "build-host-artifacts failed"

if diff -qr "$ROOT/pack/generated" "$OUT" >"$T/tree-parity.log" 2>&1; then
  ok "generated host artifact tree matches repository"
else
  no "generated host artifact tree differs from repository"
  sed -n '1,20p' "$T/tree-parity.log"
fi

touch "$T/not-a-directory"
if DEVRITES_HOST_ARTIFACT_DIR="$T/not-a-directory/generated" \
  bash "$ROOT/scripts/build-host-artifacts.sh" >"$T/failed-build.log" 2>&1; then
  no "build-host-artifacts reported success after an output failure"
else
  ok "build-host-artifacts propagates output failures"
fi

for f in \
  "claude/skills/rite-build/SKILL.md" \
  "claude/skills/devrites-lib/reference/standards/workflow-artifacts.md" \
  "claude/skills/rite-upgrade/SKILL.md" \
  "claude/agents/devrites-code-reviewer.md" \
  "claude/agents/devrites-upgrade-planner.md" \
  "claude/settings.json" \
  "codex/skills/rite-build/SKILL.md" \
  "codex/skills/rite-build/reference/wright-dispatch.md" \
  "codex/skills/devrites-lib/reference/standards/core.md" \
  "codex/skills/devrites-lib/reference/standards/workflow-artifacts.md" \
  "codex/skills/devrites-lib/reference/reply-contract.md" \
  "codex/skills/rite-prove/SKILL.md" \
  "codex/skills/rite-prove/reference/test-command-discovery.md" \
  "codex/skills/rite-spec/SKILL.md" \
  "codex/skills/rite-customize/SKILL.md" \
  "codex/skills/rite-upgrade/SKILL.md" \
  "codex/agents/devrites-slice-wright.toml" \
  "codex/agents/devrites-proof-runner.toml" \
  "codex/agents/devrites-code-reviewer.toml" \
  "codex/agents/devrites-upgrade-planner.toml" \
  "codex/AGENTS.md" \
  "codex/config.toml" \
  "README.md" ; do
  [ -f "$OUT/$f" ] && ok "artifact present: $f" || no "artifact missing: $f"
done
[ ! -e "$OUT/codex/hooks.json" ] && ok "Codex root hooks artifact is absent" || no "Codex root hooks artifact survived"

src_skills="$(find "$ROOT/pack/.claude/skills" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')"
claude_skills="$(find "$OUT/claude/skills" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')"
codex_skills="$(find "$OUT/codex/skills" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')"
[ "$claude_skills" = "$src_skills" ] && ok "Claude artifact skill count matches source" || no "Claude artifact skill count mismatch"
[ "$codex_skills" = "$src_skills" ] && ok "Codex artifact skill count matches source" || no "Codex artifact skill count mismatch"

grep -q '## Codex compatibility' "$OUT/codex/skills/rite-build/SKILL.md" \
  && no "Codex skill artifact duplicates project-wide guidance" \
  || ok "Codex skill artifact relies on the AGENTS bridge"
grep -q '.agents/skills/devrites-lib/reference/standards/core.md' "$OUT/codex/skills/rite-build/SKILL.md" \
  && ok "Codex skill artifact uses mirrored rules path" \
  || no "Codex skill artifact missing mirrored rules path"
grep -q 'repository-aware file tool refuses an ignored path.*native filesystem command.*not a completed task' "$OUT/codex/AGENTS.md" \
  && ok "Codex AGENTS artifact recovers from ignored mirror refusals" \
  || no "Codex AGENTS artifact can return an ignored mirror refusal"
grep -q 'Engram calls.*omit optional `project` and `session_id`.*Never derive either from `task_name`.*mem_session_summary.*unknown_session.*unknown_project.*both optional fields omitted.*ambiguous.*ask the user' "$OUT/codex/AGENTS.md" \
  && ok "Codex AGENTS artifact preserves exact Engram identifiers" \
  || no "Codex AGENTS artifact can invent Engram identifiers"
grep -q 'exact path-bounded executable workflow artifacts under the active `.devrites/work/<slug>/`' "$OUT/codex/AGENTS.md" \
  && ok "Codex root owns bounded executable workflow artifacts" \
  || no "Codex root cannot materialize executable workflow artifacts"
grep -q 'Codex custom-agent version\|repository-aware file tool refuses an ignored path\|For automatic Engram calls' "$OUT/codex/agents/devrites-code-reviewer.toml" \
  && no "Codex agent artifact duplicates project-wide guidance" \
  || ok "Codex agent artifact contains only its converted role contract"
grep -q '.codex/agents/devrites-slice-wright.toml' "$OUT/codex/skills/rite-build/SKILL.md" \
  && ok "Codex skill artifact references Codex agent TOML" \
  || no "Codex skill artifact missing Codex agent TOML"

grep -q 'Immediately before its final response' "$OUT/codex/skills/devrites-lib/reference/standards/core.md" \
  && ok "Codex core preserves the universal reply boundary" \
  || no "Codex core lost the universal reply boundary"
grep -q 'devrites-engine check readiness <slug>' "$OUT/codex/skills/devrites-lib/reference/standards/core.md" \
  && ok "Codex core preserves lifecycle rest points" \
  || no "Codex core lost lifecycle rest points"
if grep -R -nE 'devrites-engine (readiness|seal|spec-validate|check-acceptance|evidence-fresh|coverage|doubt-coverage|test-integrity|review-integrity|build-readiness|readiness-digest|analyze|ledger|resolve|clarify-return|tick-afk|recovery|close-out|migrate)([[:space:]`]|$)' \
  "$OUT/claude" "$OUT/codex" >/tmp/dr_host_artifacts_retired 2>/dev/null; then
  no "generated host artifacts retain retired engine commands"
  sed -n '1,20p' /tmp/dr_host_artifacts_retired
else
  ok "generated host artifacts use only nested thin-engine commands"
fi
if grep -R -nE 'devrites-engine[[:space:]]+(check[[:space:]]+spec|state[[:space:]]+(clarify|tick-afk|recovery)([[:space:]`]|$)|state[[:space:]]+resolve[[:space:]]+next-qid|doctor([[:space:]`]|$))' \
  "$OUT/claude" "$OUT/codex" >"$T/removed-policy-commands.log" 2>/dev/null; then
  no "generated host artifacts retain removed engine policy commands"
  sed -n '1,20p' "$T/removed-policy-commands.log"
else
  ok "generated host artifacts keep spec, clarify, AFK, recovery, qid, and doctor policy native"
fi
if grep -R -nE 'ADR-[0-9]{4}' \
  "$OUT/claude/skills" "$OUT/claude/agents" \
  "$OUT/codex/skills" "$OUT/codex/agents" "$OUT/codex/AGENTS.md" \
  >"$T/source-adr.log" 2>/dev/null; then
  no "generated model-visible artifacts contain source ADR identifiers"
  sed -n '1,20p' "$T/source-adr.log"
else
  ok "generated model-visible artifacts contain no source ADR identifiers"
fi
for host in claude codex; do
  authoring="$OUT/$host/skills/devrites-lib/reference/standards/skill-authoring.md"
  if grep -q 'Source-checkout only' "$authoring" \
    && grep -q 'where `pack/\.claude/` exists' "$authoring" \
    && grep -q 'Installed generated mirrors are not authoring surfaces' "$authoring"; then
    ok "$host skill-authoring guard survives generation"
  else
    no "$host skill-authoring guard missing after generation"
  fi
  grep -q 'exact standalone token in the current invocation arguments' \
    "$OUT/$host/skills/devrites-lib/reference/standards/core.md" \
    && ok "$host core preserves literal-only optional flags" \
    || no "$host core can infer optional flags from context"
  grep -q 'deleted or retired ID remains consumed' \
    "$OUT/$host/skills/devrites-lib/reference/workspace-artifact-schema.md" \
    && ok "$host workspace schema preserves append-only IDs" \
    || no "$host workspace schema lost append-only ID lifecycle"
  if grep -q 'at most 64 characters' \
      "$OUT/$host/skills/devrites-lib/reference/workspace-artifact-schema.md" \
    && grep -q 'After the final shortening or suffix step' \
      "$OUT/$host/skills/devrites-lib/reference/workspace-artifact-schema.md"; then
    ok "$host workspace schema preserves boundary-safe slug identity"
  else
    no "$host workspace schema lost boundary-safe slug identity"
  fi
  if grep -q 'base-10 integer; missing, repeated, malformed, or conflicting values stop' \
      "$OUT/$host/skills/rite-autocomplete/SKILL.md" \
    && grep -q 'one-write AFK contract' \
      "$OUT/$host/skills/rite-autocomplete/SKILL.md" \
    && grep -q 'never rewrite it after' \
      "$OUT/$host/skills/rite-autocomplete/reference/loop.md"; then
    ok "$host autocomplete preserves flag and AFK-state hardening"
  else
    no "$host autocomplete lost flag or AFK-state hardening"
  fi
done
for key in reuse conventions principles sources assumptions follow_ups; do
  grep -q "$key: \\[\\]" "$OUT/codex/agents/devrites-slice-wright.toml" \
    && ok "Codex wright preserves $key bookkeeping" \
    || no "Codex wright lost $key bookkeeping"
done
grep -q 'sole approved runtime' "$OUT/codex/skills/rite-prove/SKILL.md" \
  && ok "Codex prove preserves test-plan authority" \
  || no "Codex prove lost test-plan authority"
grep -q 'reject missing, synthesized, or unapproved commands' "$OUT/codex/agents/devrites-proof-runner.toml" \
  && ok "Codex proof runner rejects unapproved commands" \
  || no "Codex proof runner accepts commands outside test-plan"
grep -q 'Acceptance delta' "$OUT/codex/skills/rite-spec/SKILL.md" \
  && ok "Codex spec preserves existing-workspace deltas" \
  || no "Codex spec lost existing-workspace deltas"
grep -q -- '--import-legacy' "$OUT/codex/skills/rite-customize/SKILL.md" \
  && ok "Codex customize preserves legacy import mode" \
  || no "Codex customize lost legacy import mode"
grep -q 'earlier context cannot activate it' "$OUT/codex/skills/rite-customize/SKILL.md" \
  && ok "Codex customize preserves literal-only legacy mode" \
  || no "Codex customize can infer legacy mode from context"
for host in claude codex; do
  grep -q 'Older provenance, cursor form, or pack version alone is never a defect' \
    "$OUT/$host/skills/rite-upgrade/SKILL.md" \
    && ok "$host upgrade requires an observed current-contract defect" \
    || no "$host upgrade can infer staleness from age"
done
grep -q 'Outcome: <current | repairable | unsupported | gap>' \
  "$OUT/codex/agents/devrites-upgrade-planner.toml" \
  && ok "Codex upgrade planner preserves typed fail-closed outcomes" \
  || no "Codex upgrade planner lost its typed outcome contract"

grep -q 'allow_implicit_invocation: false' "$OUT/codex/skills/rite-status/agents/openai.yaml" \
  && ok "Codex preserves public explicit-only policy" \
  || no "Codex public explicit-only skill missing native invocation policy"
[ ! -e "$OUT/codex/skills/devrites-doubt/agents/openai.yaml" ] \
  && ok "Codex internal model-invoked skill has no explicit-only policy" \
  || no "Codex internal model-invoked skill got an explicit-only policy"
grep -q '^description: User-invoked read-only active-feature report' "$OUT/codex/skills/rite-status/SKILL.md" \
  && ok "Codex preserves public explicit-only description" \
  || no "Codex public explicit-only description was stubbed"

if { grep -R -nE '\.claude/skills|\.claude/agents|(^|[^A-Za-z0-9_./-])/rite(-[a-z0-9-]+)?([^A-Za-z0-9_-]|$)' "$OUT/codex/skills" "$OUT/codex/agents" \
  || grep -R --exclude='skill-authoring.md' -nE 'pack/\.claude' "$OUT/codex/skills" "$OUT/codex/agents"; } >/tmp/dr_host_artifacts_paths 2>/dev/null; then
  no "Codex artifacts contain stale runtime Claude paths or slash invocations"
  sed -n '1,40p' /tmp/dr_host_artifacts_paths
else
  ok "Codex artifacts contain no stale runtime Claude paths or slash invocations"
fi

if command -v python3 >/dev/null 2>&1; then
  python3 - "$OUT" <<'PY'
import json, pathlib, sys, tomllib
root = pathlib.Path(sys.argv[1])
tomllib.loads((root / "codex/agents/devrites-code-reviewer.toml").read_text())
tomllib.loads((root / "codex/config.toml").read_text())
PY
  rc="$?"
  if [ "$rc" -eq 0 ]; then
    ok "Codex agent and root config TOML artifacts parse"
  else
    if python3 - <<'PY' >/dev/null 2>&1
try:
    import tomllib  # noqa: F401
except ModuleNotFoundError:
    raise SystemExit(1)
PY
    then
      no "Codex agent/root config TOML artifacts invalid"
    else
      ok "Codex TOML parse skipped (python has no tomllib)"
    fi
  fi
else
  ok "Codex TOML parse skipped (python3 not found)"
fi

echo ""
[ "$fail" -eq 0 ] && echo "host-artifacts-test: PASS" || echo "host-artifacts-test: FAIL"
exit "$fail"
