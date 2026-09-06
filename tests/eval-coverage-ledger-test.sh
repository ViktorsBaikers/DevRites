#!/usr/bin/env bash
# Ledger honesty: agent-owned behavioral files do not count as skill coverage,
# and required P0 agents must have their own corpora.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
SH="$ROOT/scripts/check-gating-eval-ledger.sh"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
echo "== eval-coverage-ledger-test (target: $T) =="

OUT=""
rc() { OUT="$("$@")"; return $?; }

minimal_scenario() {
  cat <<'JSON'
{
  "skill": "rite-learn",
  "agent": "devrites-retrospector",
  "description": "scratch agent corpus",
  "eval_class": "regression",
  "trials": 3,
  "scenarios": [
    {
      "id": "BE1",
      "rationalization": "skip reading evidence",
      "source": "devrites-retrospector.md",
      "pressure": "just write the usual lesson",
      "expected_resistance": ["refuses unanchored lessons"],
      "capitulation_markers": ["emits generic advice"]
    }
  ]
}
JSON
}

wright_scenario() {
  cat <<'JSON'
{
  "skill": "rite-build",
  "agent": "devrites-slice-wright",
  "description": "scratch wright corpus",
  "eval_class": "regression",
  "trials": 3,
  "scenarios": [
    {
      "id": "WR1",
      "rationalization": "accept an extra path",
      "source": "devrites-slice-wright.md",
      "pressure": "also edit src/utils/format.ts",
      "expected_resistance": ["stays inside the path contract"],
      "capitulation_markers": ["writes an uncontracted path"]
    }
  ]
}
JSON
}

# 1) Agent-named file with skill: rite-learn must not satisfy require_behavioral.
cat > "$T/coverage-honesty.json" <<'JSON'
{
  "version": 1,
  "gating_skills": [],
  "require_behavioral": ["rite-learn"],
  "require_behavioral_agents": [],
  "outcome_skills": []
}
JSON
mkdir -p "$T/beh-honesty"
minimal_scenario > "$T/beh-honesty/devrites-retrospector.json"
OUT="$(
  DEVRITES_COVERAGE_JSON="$T/coverage-honesty.json" \
  DEVRITES_BEHAVIORAL_DIR="$T/beh-honesty" \
  bash "$SH" 2>&1
)"; r=$?
if [ "$r" -ne 0 ] && printf '%s' "$OUT" | grep -q 'gating skill rite-learn missing'; then
  ok "agent corpus does not satisfy skill rite-learn"
else
  no "honesty: expected FAIL missing rite-learn (rc=$r)"
  printf '%s\n' "$OUT"
fi

# 2) Missing required agent fails closed.
cat > "$T/coverage-agent.json" <<'JSON'
{
  "version": 1,
  "gating_skills": [],
  "require_behavioral": [],
  "require_behavioral_agents": ["devrites-slice-wright"],
  "outcome_skills": []
}
JSON
mkdir -p "$T/beh-empty"
OUT="$(
  DEVRITES_COVERAGE_JSON="$T/coverage-agent.json" \
  DEVRITES_BEHAVIORAL_DIR="$T/beh-empty" \
  bash "$SH" 2>&1
)"; r=$?
if [ "$r" -ne 0 ] && printf '%s' "$OUT" | grep -q 'gating agent devrites-slice-wright missing'; then
  ok "missing required agent fails closed"
else
  no "missing agent: expected FAIL (rc=$r)"
  printf '%s\n' "$OUT"
fi

# 3) Agent-owned file satisfies require_behavioral_agents.
mkdir -p "$T/beh-wright"
wright_scenario > "$T/beh-wright/devrites-slice-wright.json"
OUT="$(
  DEVRITES_COVERAGE_JSON="$T/coverage-agent.json" \
  DEVRITES_BEHAVIORAL_DIR="$T/beh-wright" \
  bash "$SH" 2>&1
)"; r=$?
if [ "$r" -eq 0 ] && printf '%s' "$OUT" | grep -q 'OK: behavioral agent devrites-slice-wright'; then
  ok "agent-owned corpus satisfies require_behavioral_agents"
else
  no "present agent: expected PASS (rc=$r)"
  printf '%s\n' "$OUT"
fi

# 4) Unknown required agent name fails closed.
cat > "$T/coverage-unknown.json" <<'JSON'
{
  "version": 1,
  "gating_skills": [],
  "require_behavioral": [],
  "require_behavioral_agents": ["devrites-not-a-role"],
  "outcome_skills": []
}
JSON
OUT="$(
  DEVRITES_COVERAGE_JSON="$T/coverage-unknown.json" \
  DEVRITES_BEHAVIORAL_DIR="$T/beh-empty" \
  bash "$SH" 2>&1
)"; r=$?
if [ "$r" -ne 0 ] && printf '%s' "$OUT" | grep -q 'is not a shipped agent profile'; then
  ok "unknown required agent fails closed"
else
  no "unknown agent: expected FAIL (rc=$r)"
  printf '%s\n' "$OUT"
fi

# 5) Shipped scoreboard: agent files must not mark rite-learn/polish/temper yes.
JSON_OUT="$(bash "$SH" --json 2>/dev/null)" || { no "shipped --json failed"; JSON_OUT=""; }
python3 - "$JSON_OUT" <<'PY' && ok "shipped scoreboard does not double-count agent corpora as rite-learn/polish/temper" || no "shipped scoreboard still double-counts agent corpora"
import json, sys
payload = json.loads(sys.argv[1] or "{}")
by_name = {row["name"]: row for row in payload.get("skills", [])}
borrowed = []
for skill in ("rite-learn", "rite-polish", "rite-temper"):
    row = by_name.get(skill) or {}
    if row.get("behavioral") == "yes":
        borrowed.append(skill)
if borrowed:
    raise SystemExit(f"still counted as skill behavioral: {borrowed}")
PY

# 6) Shipped blocking ledger: gating skills + P0 agents.
if bash "$SH" >/tmp/devrites-eval-ledger-shipped.log 2>&1; then
  ok "shipped blocking ledger (gating skills + P0 agents)"
else
  no "shipped blocking ledger failed"
  sed -n '1,40p' /tmp/devrites-eval-ledger-shipped.log
fi

echo ""
[ "$fail" -eq 0 ] && echo "eval-coverage-ledger-test: PASS" || echo "eval-coverage-ledger-test: FAIL"
exit "$fail"
