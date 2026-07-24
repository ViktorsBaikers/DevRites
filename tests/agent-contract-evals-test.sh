#!/usr/bin/env bash
set -euo pipefail

if [ "${DEVRITES_AGENT_CONTRACT_DISCOVERY_ONLY:-0}" = "1" ]; then
  echo "agent-contract eval discovery sentinel"
  exit 0
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
RUNNER="$ROOT/scripts/run-agent-contract-evals.py"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
mkdir -p "$T/tmp"
printf '%s\n' 'export DEVRITES_OPERATOR_STARTUP_LEAK=1' > "$T/operator-startup.sh"

echo "== agent-contract-evals-test (network/model use forbidden) =="
BASH_ENV="$T/operator-startup.sh" \
ENV="$T/operator-startup.sh" \
CLAUDE_CONFIG_DIR="$T/operator-claude" \
CODEX_HOME="$T/operator-codex" \
ANTHROPIC_API_KEY="DO-NOT-INHERIT" \
OPENAI_API_KEY="DO-NOT-INHERIT" \
TMPDIR="$T/tmp" \
  python3 "$RUNNER" --fake --results-dir "$T/results" > "$T/fake.out"

python3 - "$T/results/summary.json" <<'PY'
import json
import pathlib
import sys
from collections import Counter

summary = json.loads(pathlib.Path(sys.argv[1]).read_text())
assert summary["schema"] == "devrites-agent-contract-summary/v1"
assert summary["mode"] == "fake"
assert summary["cells_total"] == 24
assert summary["cells_passed"] == 24
assert summary["cells_failed"] == 0
assert summary["cost_usd"] == "0"
assert summary["reason_counts"] == {
    "DRV-AGENT-RESULT-ACCEPTED": 12,
    "DRV-AGENT-RESULT-MALFORMED": 2,
    "DRV-AGENT-RESULT-OUT-OF-SCOPE": 4,
    "DRV-AGENT-RESULT-STALE": 2,
    "DRV-AGENT-UNAVAILABLE": 4,
}
rows = summary["results"]
assert Counter(row["host"] for row in rows) == {"claude": 12, "codex": 12}
assert Counter(row["execution_mode"] for row in rows) == {"named": 20, "generic": 2, "inline": 2}
assert Counter(row["guard_strength"] for row in rows) == {"enforced": 22, "observed": 2}
assert Counter(row["terminal_sentinel"] for row in rows) == {
    "complete": 16,
    "stale": 2,
    "malformed": 2,
    "missing": 2,
    "interrupted": 2,
}
assert sum(not row["independence"] for row in rows) == 2
wrights = [row for row in rows if row["scenario"] == "named-wright"]
assert len(wrights) == 2
assert all([item["path"] for item in row["repo_writes"]] == ["src/allowed.txt"] for row in wrights)
assert all(row["identity_matched"] for row in rows)
assert all(row["baseline_matched"] for row in rows if row["scenario"] != "stale-result")
assert all(not row["baseline_matched"] for row in rows if row["scenario"] == "stale-result")
assert all(row["usage"]["cost_provenance"] == "synthetic" for row in rows)

forbidden = {
    "absolute_path", "auth", "config", "home", "prompt", "raw",
    "scratch_root", "source", "trace",
}
def walk(value):
    if isinstance(value, dict):
        assert not (forbidden & set(value))
        for child in value.values():
            walk(child)
    elif isinstance(value, list):
        for child in value:
            walk(child)
    elif isinstance(value, str):
        assert not value.startswith("/")
walk(summary)
PY
echo "  ok: 24/24 fake Claude/Codex cells passed with exact typed outcomes"

if find "$T/tmp" -maxdepth 1 -name 'devrites-agent-contract-*' -print -quit | grep -q .; then
  echo "temporary contract roots were not cleaned" >&2
  exit 1
fi
if grep -Eq 'DO-NOT-INHERIT|fake-scoped-credential|agent-packet|Synthetic agent contract fixture|operator/private/path' "$T/fake.out"; then
  echo "fake summary retained private transport data" >&2
  exit 1
fi
echo "  ok: cleanup and metadata-only retention passed"

set +e
TMPDIR="$T/tmp" python3 "$RUNNER" --fake \
  --fake-fault claude:named-reviewer:identity-mismatch \
  --results-dir "$T/identity" > "$T/identity.out" 2> "$T/identity.err"
identity_rc=$?
set -e
[ "$identity_rc" -eq 1 ]
python3 - "$T/identity/summary.json" <<'PY'
import json, pathlib, sys
value = json.loads(pathlib.Path(sys.argv[1]).read_text())
bad = [row for row in value["results"] if not row["passed"]]
assert len(bad) == 1
assert bad[0]["result_reason_id"] == "DRV-AGENT-RESULT-IDENTITY-MISMATCH"
assert bad[0]["identity_matched"] is False
PY
echo "  ok: run/role/baseline identity mismatch fails typed"

set +e
TMPDIR="$T/tmp" python3 "$RUNNER" --fake \
  --fake-fault codex:named-reviewer:escape-write \
  --results-dir "$T/escape" > "$T/escape.out" 2> "$T/escape.err"
escape_rc=$?
set -e
[ "$escape_rc" -eq 1 ]
grep -Fq '"DRV-AGENT-RESULT-OUT-OF-SCOPE"' "$T/escape/summary.json"
echo "  ok: undeclared filesystem effect is denied and cleaned"

set +e
TMPDIR="$T/tmp" python3 "$RUNNER" --fake \
  --fake-fault claude:named-reviewer:privacy-leak \
  --results-dir "$T/privacy" > "$T/privacy.out" 2> "$T/privacy.err"
privacy_rc=$?
set -e
[ "$privacy_rc" -eq 1 ]
grep -Fq "DRV-AGENT-RESULT-MALFORMED" "$T/privacy/summary.json"
if grep -Fq "operator/private/path" "$T/privacy/summary.json"; then
  echo "privacy fault reached persisted metadata" >&2
  exit 1
fi
echo "  ok: scenario-closed signals reject unknown values before summary retention"

PYTHONDONTWRITEBYTECODE=1 python3 - "$RUNNER" "$ROOT/scripts/live-hosts/host-transport.py" <<'PY'
import importlib.util
import sys

def load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module

runner = load("agent_contract_runner_privacy", sys.argv[1])
transport = load("agent_contract_host_transport", sys.argv[2])

for value in (
    "OPENAI_API_KEY=sk-private-example",
    "prompt=retain this operator request",
    "source: print('private')",
):
    try:
        runner.assert_private_metadata({"signals": [value]})
    except runner.ContractError:
        pass
    else:
        raise AssertionError(f"private string was accepted: {value}")

assert runner.safe_id("claude-code 2.1.0 (stable)", "host version")
for value in ("/absolute/model", "bad\nmodel", "x" * 129):
    try:
        runner.safe_id(value, "host model")
    except runner.ContractError as exc:
        assert exc.reason_id == "DRV-AGENT-UNAVAILABLE"
    else:
        raise AssertionError(f"unsafe host identifier was accepted: {value!r}")

model_result = {
    "result_version": "agent-result/v1",
    "usage": {"input_tokens": 999, "output_tokens": 999, "cost_usd": "0"},
}
unknown = transport.observed_usage([
    {"usage": {"input_tokens": 7, "output_tokens": 3}, "result": model_result}
])
assert unknown == {
    "input_tokens": 7,
    "output_tokens": 3,
    "cost_usd": None,
    "cost_provenance": "unavailable",
}
observed = transport.observed_usage([
    {"total_cost_usd": "0.125", "usage": {"input_tokens": 7, "output_tokens": 3}}
])
assert observed["cost_usd"] == "0.125"
assert observed["cost_provenance"] == "provider-reported"
PY
echo "  ok: non-path secrets/text payloads fail privacy and unknown provider cost stays unknown"

python3 - "$ROOT/evals/agent-contract/contract-matrix.json" "$T/bad-matrix.json" <<'PY'
import json, pathlib, sys
value = json.loads(pathlib.Path(sys.argv[1]).read_text())
value["scenarios"][0]["expect"]["result_reason_id"] = "DRV-NOT-FROZEN"
pathlib.Path(sys.argv[2]).write_text(json.dumps(value))
PY
set +e
python3 "$RUNNER" --fake --matrix "$T/bad-matrix.json" > "$T/bad.out" 2> "$T/bad.err"
bad_rc=$?
set -e
[ "$bad_rc" -eq 2 ]
grep -Fq "frozen reason catalog" "$T/bad.err"
echo "  ok: matrix rejects non-canonical reason IDs"

set +e
python3 "$RUNNER" --live --preflight-only --results-dir "$T/live-missing" \
  > "$T/live-missing.out" 2> "$T/live-missing.err"
missing_rc=$?
set -e
[ "$missing_rc" -eq 2 ]
grep -Fq "DRV-AGENT-UNAVAILABLE" "$T/live-missing.err"
echo "  ok: live mode fails closed before any host call"

mkdir -p "$T/bin" "$T/host-artifacts"
cat > "$T/bin/claude" <<'EOF'
#!/usr/bin/env bash
[ "${1:-}" = "--version" ] && { echo "claude-test/1"; exit 0; }
exit 99
EOF
cat > "$T/bin/codex" <<'EOF'
#!/usr/bin/env bash
[ "${1:-}" = "--version" ] && { echo "codex-test/1"; exit 0; }
exit 99
EOF
chmod +x "$T/bin/claude" "$T/bin/codex"
printf '%s\n' "claude-test-credential" > "$T/claude.auth"
printf '%s\n' "codex-test-credential" > "$T/codex.auth"
chmod 600 "$T/claude.auth" "$T/codex.auth"
month="$(date -u +%Y-%m)"
printf '{"schema":"devrites-agent-contract-budget/v1","month":"%s","spent_usd":"0"}\n' "$month" > "$T/ledger.json"
chmod 600 "$T/ledger.json"

set +e
PATH="$T/bin:$PATH" python3 "$RUNNER" --live --preflight-only \
  --confirm-live RUN-PAID-HOST-CONTRACTS \
  --host-version claude=claude-test/1 \
  --host-version codex=codex-test/1 \
  --host-model claude=claude-test-model \
  --host-model codex=codex-test-model \
  --auth-file "claude=$T/claude.auth" \
  --auth-file "codex=$T/codex.auth" \
  --host-artifacts "$T/host-artifacts" \
  --max-cost-per-run-usd 0.50 \
  --max-monthly-cost-usd 12.00 \
  --budget-ledger "$T/ledger.json" \
  --results-dir "$T/live-ok" > "$T/live-ok.out" 2> "$T/live-ok.err"
live_cap_rc=$?
set -e
[ "$live_cap_rc" -eq 2 ]
[ ! -e "$T/live-ok/summary.json" ]
grep -Fq "lacks an enforced runtime/provider cost cap: codex" "$T/live-ok.err"
echo "  ok: paid Codex is ineligible without an enforced runtime/provider cost cap"

PYTHONDONTWRITEBYTECODE=1 python3 - "$RUNNER" "$T/ledger.json" <<'PY'
import importlib.util
import json
import pathlib
import sys
from decimal import Decimal

spec = importlib.util.spec_from_file_location("agent_contract_runner", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
ledger = pathlib.Path(sys.argv[2])
module.reserve_budget(ledger, Decimal("12"), Decimal("12"))
assert json.loads(ledger.read_text())["spent_usd"] == "12"
module.settle_budget(ledger, Decimal("12"), None)
assert json.loads(ledger.read_text())["spent_usd"] == "12"
module.settle_budget(ledger, Decimal("12"), Decimal("1.25"))
assert json.loads(ledger.read_text())["spent_usd"] == "1.25"
module.settle_budget(ledger, Decimal("1.25"), Decimal("0"))
assert Decimal(json.loads(ledger.read_text())["spent_usd"]) == 0
PY
echo "  ok: missing observed cost consumes the reservation; observed cost settles exactly"

ln "$T/claude.auth" "$T/linked.auth"
set +e
PATH="$T/bin:$PATH" python3 "$RUNNER" --live --preflight-only \
  --confirm-live RUN-PAID-HOST-CONTRACTS \
  --host-version claude=claude-test/1 \
  --host-version codex=codex-test/1 \
  --host-model claude=claude-test-model \
  --host-model codex=codex-test-model \
  --auth-file "claude=$T/claude.auth" \
  --auth-file "codex=$T/linked.auth" \
  --host-artifacts "$T/host-artifacts" \
  --max-cost-per-run-usd 0.50 \
  --max-monthly-cost-usd 12.00 \
  --budget-ledger "$T/ledger.json" \
  --results-dir "$T/live-linked" > "$T/live-linked.out" 2> "$T/live-linked.err"
linked_rc=$?
set -e
[ "$linked_rc" -eq 2 ]
grep -Fq "separate scoped files" "$T/live-linked.err"
echo "  ok: hard-linked host credentials are not treated as separate scopes"

set +e
PATH="$T/bin:$PATH" python3 "$RUNNER" --live --preflight-only \
  --confirm-live RUN-PAID-HOST-CONTRACTS \
  --host-version claude=claude-test/1 \
  --host-version codex=codex-test/1 \
  --host-model claude=claude-test-model \
  --host-model codex=codex-test-model \
  --auth-file "claude=$T/claude.auth" \
  --auth-file "codex=$T/codex.auth" \
  --host-artifacts "$T/host-artifacts" \
  --max-cost-per-run-usd 0.50 \
  --max-monthly-cost-usd 11.99 \
  --budget-ledger "$T/ledger.json" \
  --results-dir "$T/live-over" > "$T/live-over.out" 2> "$T/live-over.err"
budget_rc=$?
set -e
[ "$budget_rc" -eq 2 ]
grep -Fq "monthly ceiling" "$T/live-over.err"
echo "  ok: monthly budget exhaustion fails closed"

mkdir -p "$T/discovery-host-artifacts"
discovery="$(
  DEVRITES_AGENT_CONTRACT_DISCOVERY_ONLY=1 \
  DEVRITES_HOST_ARTIFACT_DIR="$T/discovery-host-artifacts" \
  DEVRITES_ENGINE_CLI=/usr/bin/true \
    node "$ROOT/scripts/run-tests.mjs" --serial agent-contract-evals-test
)"
grep -Fq "agent-contract eval discovery sentinel" <<<"$discovery"
grep -Fq "PASS: tests/agent-contract-evals-test.sh" <<<"$discovery"
echo "  ok: repository test runner discovers the contract gate"

echo "agent-contract-evals-test: PASS"
