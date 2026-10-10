#!/usr/bin/env bash
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
fail=0

ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }
run_ok() {
  label="$1"; shift
  if "$@" >"$T/out" 2>&1; then ok "$label"; else no "$label"; sed -n '1,80p' "$T/out"; fi
}
run_fail_contains() {
  label="$1"; needle="$2"; shift 2
  if "$@" >"$T/out" 2>&1; then no "$label accepted invalid fixture"
  elif grep -q "$needle" "$T/out"; then ok "$label"
  else no "$label wrong failure"; sed -n '1,80p' "$T/out"; fi
}

echo "== validation-governance-test =="

# Cross-reference resolution distinguishes shipped repo docs and declared
# runtime artifacts from genuinely dead pointers.
mkdir -p "$T/cross/pack/.claude/skills/demo" "$T/cross/docs"
printf '# command map\n' > "$T/cross/docs/command-map.md"
cat > "$T/cross/pack/.claude/skills/demo/SKILL.md" <<'MD'
Read `docs/command-map.md` and write `ai-spec.md`.
MD
run_ok "cross refs accept repo docs and runtime artifacts" python3 "$ROOT/scripts/check-cross-refs.py" --root "$T/cross"
printf 'Read `definitely-dead.md`.\n' >> "$T/cross/pack/.claude/skills/demo/SKILL.md"
run_fail_contains "cross refs still reject unknown documents" "definitely-dead.md" python3 "$ROOT/scripts/check-cross-refs.py" --root "$T/cross"

# Bare shared-standard basenames must not pass merely because the file exists
# under another skill (hosts open <skill>/reference/<name>.md for bare names).
mkdir -p "$T/bare/pack/.claude/skills/demo" \
  "$T/bare/pack/.claude/skills/devrites-lib/reference/standards"
printf '# afk\n' > "$T/bare/pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md"
printf 'Read `afk-hitl.md` first.\n' > "$T/bare/pack/.claude/skills/demo/SKILL.md"
run_fail_contains "cross refs reject bare shared-standard basename" "not local" \
  python3 "$ROOT/scripts/check-cross-refs.py" --root "$T/bare"
printf 'Read [`afk-hitl.md`](../devrites-lib/reference/standards/afk-hitl.md) first.\n' \
  > "$T/bare/pack/.claude/skills/demo/SKILL.md"
run_ok "cross refs accept relative link to shared standard" \
  python3 "$ROOT/scripts/check-cross-refs.py" --root "$T/bare"

# Unquoted skill/reference paths are join-hazards: hosts attach the token to
# the current skill (devrites-lib/reference/afk-discipline.md) even when the
# file exists under the named skill.
mkdir -p "$T/join/pack/.claude/skills/devrites-lib/reference/standards" \
  "$T/join/pack/.claude/skills/rite-build/reference"
printf '# disc\n' > "$T/join/pack/.claude/skills/rite-build/reference/afk-discipline.md"
printf 'examples: rite-build/reference/afk-discipline.md\n' \
  > "$T/join/pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md"
run_fail_contains "cross refs reject unquoted cross-skill join-hazard" "join-hazard" \
  python3 "$ROOT/scripts/check-cross-refs.py" --root "$T/join"
printf 'examples: .claude/skills/rite-build/reference/afk-discipline.md\n' \
  > "$T/join/pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md"
run_ok "cross refs accept host-prefixed cross-skill path" \
  python3 "$ROOT/scripts/check-cross-refs.py" --root "$T/join"
printf 'See [loop](../../../rite-build/reference/afk-discipline.md).\n' \
  > "$T/join/pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md"
run_ok "cross refs accept relative markdown link to another skill" \
  python3 "$ROOT/scripts/check-cross-refs.py" --root "$T/join"

# Codex install paths must map onto pack/.claude, never a missing pack/.agents tree.
mkdir -p "$T/agents/pack/.claude/agents" \
  "$T/agents/pack/.claude/skills/rite-review/reference"
printf '# checklist\n' > "$T/agents/pack/.claude/skills/rite-review/reference/performance-checklist.md"
printf 'Read `.agents/skills/rite-review/reference/performance-checklist.md`.\n' \
  > "$T/agents/pack/.claude/agents/devrites-performance-reviewer.md"
run_ok "cross refs resolve Codex .agents/skills install paths" \
  python3 "$ROOT/scripts/check-cross-refs.py" --root "$T/agents"

# Permission profile names are not skill invocations; undeclared devrites-* names remain errors.
printf 'Use the devrites-orchestrator permission profile.\n' > "$T/non-skill-profile.md"
printf 'Invoke devrites-definitely-missing.\n' > "$T/missing-invocation.md"
run_ok "invocation scanner distinguishes a permission profile from a missing name" python3 - "$ROOT/scripts/check-invocation-integrity.py" "$T/non-skill-profile.md" "$T/missing-invocation.md" <<'PY'
import importlib.util
import sys

spec = importlib.util.spec_from_file_location("invocation_integrity", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

problems = []
module.scan(sys.argv[2], set(), set(), problems)
assert problems == [], problems

module.scan(sys.argv[3], set(), set(), problems)
assert len(problems) == 1, problems
assert "unresolved skill/agent name 'devrites-definitely-missing'" in problems[0], problems
PY

# Every supporting reference is size-ratcheted, not only SKILL.md.
mkdir -p "$T/size/pack/.claude/skills/demo/reference" "$T/size/tests"
printf '# demo\n[details](reference/details.md)\n' > "$T/size/pack/.claude/skills/demo/SKILL.md"
printf '# details\nsmall\n' > "$T/size/pack/.claude/skills/demo/reference/details.md"
run_ok "instruction baseline writes references" node "$ROOT/scripts/check-instruction-size-baseline.mjs" --root "$T/size" --baseline "$T/size/tests/baseline.json" --write
printf 'unreviewed growth that must trip the ratchet\n' >> "$T/size/pack/.claude/skills/demo/reference/details.md"
run_fail_contains "instruction baseline catches reference growth" "reference/details.md grew" node "$ROOT/scripts/check-instruction-size-baseline.mjs" --root "$T/size" --baseline "$T/size/tests/baseline.json"
run_fail_contains "reference file budget is blocking" "reference/details.md" env DEVRITES_REFERENCE_FILE_BUDGET=20 node "$ROOT/scripts/check-generated-skill-budget.mjs" "$T/size/pack/.claude/skills"

# Model-visible routing metadata has its own aggregate budget.
mkdir -p "$T/routing/demo"
cat > "$T/routing/demo/SKILL.md" <<'MD'
---
name: demo
description: Route this model-visible demo skill.
---
# Demo
MD
run_fail_contains "model-visible routing budget is blocking" "shorten name/description frontmatter" env DEVRITES_SKILL_ROUTING_BUDGET=1 node "$ROOT/scripts/check-generated-skill-budget.mjs" "$T/routing"

# Agent descriptions are model-visible routing text with their own budget.
mkdir -p "$T/agent-routing/skills/demo" "$T/agent-routing/agents"
cp "$T/routing/demo/SKILL.md" "$T/agent-routing/skills/demo/SKILL.md"
cat > "$T/agent-routing/agents/reviewer.md" <<'MD'
---
name: reviewer
description: Reviews one diff from a fresh context.
tools: Read
---
# Reviewer
MD
run_ok "agent routing budget accepts the default" node "$ROOT/scripts/check-generated-skill-budget.mjs" "$T/agent-routing/skills"
run_fail_contains "agent routing budget is blocking" "agent routing metadata" env DEVRITES_AGENT_ROUTING_BUDGET=1 node "$ROOT/scripts/check-generated-skill-budget.mjs" "$T/agent-routing/skills"

# Devin skills express invocation through triggers, not disable-model-invocation.
mkdir -p "$T/triggers-user/user-only" "$T/triggers-model/model-visible"
printf -- '---\nname: user-only\ndescription: Slash-only skill.\ntriggers:\n  - user\n---\n# Demo\n' > "$T/triggers-user/user-only/SKILL.md"
printf -- '---\nname: model-visible\ndescription: Model skill.\ntriggers:\n  - user\n  - model\n---\n# Demo\n' > "$T/triggers-model/model-visible/SKILL.md"
run_ok "user-only triggers do not count toward routing" env DEVRITES_SKILL_ROUTING_BUDGET=20 node "$ROOT/scripts/check-generated-skill-budget.mjs" "$T/triggers-user"
run_fail_contains "model triggers count toward routing" "model-visible skill routing metadata" env DEVRITES_SKILL_ROUTING_BUDGET=20 node "$ROOT/scripts/check-generated-skill-budget.mjs" "$T/triggers-model"

# Module URLs must be decoded before they are used as filesystem paths.
SPACE_ROOT="$T/repository with spaces"
mkdir -p "$SPACE_ROOT/scripts" "$SPACE_ROOT/tests" "$SPACE_ROOT/pack/.claude/skills/demo" "$T/shared-artifacts"
cp "$ROOT/scripts/run-tests.mjs" "$ROOT/scripts/check-generated-skill-budget.mjs" "$SPACE_ROOT/scripts/"
cat > "$SPACE_ROOT/tests/path-smoke.sh" <<'SH'
#!/usr/bin/env bash
exit 0
SH
printf '# demo\npayload\n' > "$SPACE_ROOT/pack/.claude/skills/demo/SKILL.md"
run_ok "test runner decodes module URL paths" env DEVRITES_HOST_ARTIFACT_DIR="$T/shared-artifacts" node "$SPACE_ROOT/scripts/run-tests.mjs" --serial path-smoke
run_fail_contains "default skill path survives spaces" "SKILL.md" env DEVRITES_SKILL_FILE_BUDGET=1 node "$SPACE_ROOT/scripts/check-generated-skill-budget.mjs"

# A budget guard that cannot measure its tree or parse its limits must fail
# closed instead of passing vacuously.
run_ok "skill budget accepts a valid tree" node "$ROOT/scripts/check-generated-skill-budget.mjs" "$T/routing"
run_fail_contains "skill budget rejects a nonexistent path" "missing" node "$ROOT/scripts/check-generated-skill-budget.mjs" "$T/no-such-skills"
for budget in DEVRITES_SKILL_TOTAL_BUDGET DEVRITES_SKILL_ROUTING_BUDGET DEVRITES_SKILL_FILE_BUDGET DEVRITES_REFERENCE_FILE_BUDGET; do
  run_fail_contains "skill budget rejects non-numeric $budget" "not a finite number" env "$budget=abc" node "$ROOT/scripts/check-generated-skill-budget.mjs" "$T/routing"
done

# Reachability is blocking unless an orphan has an owned, expiring exception.
mkdir -p "$T/refs/demo/reference"
printf '# demo\n' > "$T/refs/demo/SKILL.md"
printf '# orphan\n' > "$T/refs/demo/reference/orphan.md"
printf '{}\n' > "$T/orphans.json"
run_fail_contains "reference governance rejects an orphan" "unreachable reference" node "$ROOT/scripts/check-reference-governance.mjs" --skills-dir "$T/refs" --allowlist "$T/orphans.json"
cat > "$T/orphans.json" <<'JSON'
{"demo/reference/orphan.md":{"owner":"validation","reason":"fixture","expires":"2099-01-01"}}
JSON
run_ok "reference governance accepts owned expiring exception" node "$ROOT/scripts/check-reference-governance.mjs" --skills-dir "$T/refs" --allowlist "$T/orphans.json"

# Reference files over ~300 lines need a ## Contents table of contents.
mkdir -p "$T/toc/demo/reference"
printf '# demo\n[long](reference/long.md)\n' > "$T/toc/demo/SKILL.md"
{
  printf '# long\n\n'
  i=1
  while [ "$i" -le 301 ]; do
    printf 'line %s\n' "$i"
    i=$((i + 1))
  done
} > "$T/toc/demo/reference/long.md"
printf '{}\n' > "$T/toc-allow.json"
run_fail_contains "reference governance requires TOC over 300 lines" "needs a ## Contents" node "$ROOT/scripts/check-reference-governance.mjs" --skills-dir "$T/toc" --allowlist "$T/toc-allow.json"
printf '# long\n\n## Contents\n\n- [One](#one)\n\n## One\n\n' > "$T/toc/demo/reference/long.md"
i=1
while [ "$i" -le 301 ]; do
  printf 'line %s\n' "$i" >> "$T/toc/demo/reference/long.md"
  i=$((i + 1))
done
run_ok "reference governance accepts TOC over 300 lines" node "$ROOT/scripts/check-reference-governance.mjs" --skills-dir "$T/toc" --allowlist "$T/toc-allow.json"

# One-hop: a skill-local reference must not hide another file in that skill.
mkdir -p "$T/hop/demo/reference"
printf '# demo\n[mid](reference/mid.md)\n' > "$T/hop/demo/SKILL.md"
printf '# mid\n[hidden](hidden.md)\n' > "$T/hop/demo/reference/mid.md"
printf '# hidden\n' > "$T/hop/demo/reference/hidden.md"
printf '{}\n' > "$T/hop-allow.json"
run_fail_contains "reference governance rejects a planted two-hop" "two-hop via" node "$ROOT/scripts/check-reference-governance.mjs" --skills-dir "$T/hop" --allowlist "$T/hop-allow.json"
printf '# demo\n[mid](reference/mid.md)\n[hidden](reference/hidden.md)\n' > "$T/hop/demo/SKILL.md"
run_ok "reference governance accepts a SKILL that also links the hidden file" node "$ROOT/scripts/check-reference-governance.mjs" --skills-dir "$T/hop" --allowlist "$T/hop-allow.json"
mkdir -p "$T/hop-index/demo/reference"
printf '# demo\n[core](reference/core.md)\n' > "$T/hop-index/demo/SKILL.md"
printf '# core\n[hidden](hidden.md)\n' > "$T/hop-index/demo/reference/core.md"
printf '# hidden\n' > "$T/hop-index/demo/reference/hidden.md"
printf '{}\n' > "$T/hop-index-allow.json"
run_ok "reference governance allows two-hops through core.md as an index" node "$ROOT/scripts/check-reference-governance.mjs" --skills-dir "$T/hop-index" --allowlist "$T/hop-index-allow.json"

# Dependency audit exceptions are exact, owner-bound, expiring, and stale-intolerant.
cat > "$T/npm-audit.json" <<'JSON'
{"auditReportVersion":2,"vulnerabilities":{"npm":{"severity":"moderate","via":["tar"],"nodes":["node_modules/npm"]},"tar":{"severity":"moderate","via":[{"name":"tar","url":"https://github.com/advisories/GHSA-r292-9mhp-454m","severity":"moderate","range":"<=7.5.20"}],"nodes":["node_modules/npm/node_modules/tar"]}}}
JSON
cat > "$T/npm-audit-exceptions.json" <<'JSON'
[{"id":"GHSA-r292-9mhp-454m","package":"tar","range":"<=7.5.20","nodes":["node_modules/npm/node_modules/tar"],"source":"https://github.com/advisories/GHSA-r292-9mhp-454m","owner":"security","reason":"fixture","expires":"2099-01-01"}]
JSON
run_ok "npm audit accepts one exact temporary exception" env DEVRITES_TEST_TODAY=2026-09-04 node "$ROOT/scripts/check-npm-audit.mjs" --test --input "$T/npm-audit.json" --exceptions "$T/npm-audit-exceptions.json"
node -e 'const fs=require("fs");const p=JSON.parse(fs.readFileSync(process.argv[1]));p[0].expires="2000-01-01";fs.writeFileSync(process.argv[2],JSON.stringify(p))' "$T/npm-audit-exceptions.json" "$T/npm-audit-expired.json"
run_fail_contains "npm audit rejects an expired exception" "expired" env DEVRITES_TEST_TODAY=2026-09-04 node "$ROOT/scripts/check-npm-audit.mjs" --test --input "$T/npm-audit.json" --exceptions "$T/npm-audit-expired.json"
node -e 'const fs=require("fs");const p=JSON.parse(fs.readFileSync(process.argv[1]));p[0].expires="2026-09-08";fs.writeFileSync(process.argv[2],JSON.stringify(p))' "$T/npm-audit-exceptions.json" "$T/npm-audit-soon.json"
run_fail_contains "npm audit rejects an exception inside the 7-day refresh horizon" "refresh or remove" env DEVRITES_TEST_TODAY=2026-09-04 node "$ROOT/scripts/check-npm-audit.mjs" --test --input "$T/npm-audit.json" --exceptions "$T/npm-audit-soon.json"
node -e 'const fs=require("fs");const p=JSON.parse(fs.readFileSync(process.argv[1]));p[0].expires="2026-09-12";fs.writeFileSync(process.argv[2],JSON.stringify(p))' "$T/npm-audit-exceptions.json" "$T/npm-audit-horizon-ok.json"
run_ok "npm audit accepts an exception outside the 7-day refresh horizon" env DEVRITES_TEST_TODAY=2026-09-04 node "$ROOT/scripts/check-npm-audit.mjs" --test --input "$T/npm-audit.json" --exceptions "$T/npm-audit-horizon-ok.json"
run_fail_contains "npm audit ignores DEVRITES_TODAY without the test gate" "expired" env DEVRITES_TODAY=2026-09-04 node "$ROOT/scripts/check-npm-audit.mjs" --input "$T/npm-audit.json" --exceptions "$T/npm-audit-horizon-ok.json"
printf '[]\n' > "$T/npm-audit-empty.json"
printf '{"auditReportVersion":2,"vulnerabilities":{}}\n' > "$T/npm-audit-clean.json"
run_ok "npm audit accepts an empty exception list on a clean graph" node "$ROOT/scripts/check-npm-audit.mjs" --input "$T/npm-audit-clean.json" --exceptions "$T/npm-audit-empty.json"
node -e 'const fs=require("fs");const p=JSON.parse(fs.readFileSync(process.argv[1]));p.vulnerabilities.other={severity:"moderate",via:[{name:"other",url:"https://github.com/advisories/GHSA-aaaa-bbbb-cccc",severity:"moderate",range:"<2"}],nodes:["node_modules/other"]};fs.writeFileSync(process.argv[2],JSON.stringify(p))' "$T/npm-audit.json" "$T/npm-audit-extra.json"
run_fail_contains "npm audit rejects an unexcepted advisory" "not excepted" node "$ROOT/scripts/check-npm-audit.mjs" --input "$T/npm-audit-extra.json" --exceptions "$T/npm-audit-exceptions.json"

# Every npm-audit exception ID must appear in osv-scanner.toml with a matching
# ignoreUntil. Extra OSV ignores (below the npm moderate+ gate) must still expire.
python3 - "$ROOT" <<'PY'
import json, re, sys
from pathlib import Path
root = Path(sys.argv[1])
exceptions = json.loads((root / "scripts/npm-audit-exceptions.json").read_text())
text = (root / "osv-scanner.toml").read_text()
blocks = re.findall(r"(?m)^\[\[IgnoredVulns\]\](.*?)(?=\n\[\[|\Z)", text, re.S)
osv = {}
for block in blocks:
    mid = re.search(r'id\s*=\s*"([^"]+)"', block)
    until = re.search(r"ignoreUntil\s*=\s*(\d{4}-\d{2}-\d{2})", block)
    if not mid:
        raise SystemExit("osv-scanner.toml IgnoredVulns block is missing id")
    if not until:
        raise SystemExit(f"{mid.group(1)}: osv ignore is missing ignoreUntil")
    osv[mid.group(1)] = until.group(1)
for exception in exceptions:
    advisory = exception["id"]
    if advisory not in osv:
        raise SystemExit(f"{advisory}: missing from osv-scanner.toml")
    if osv[advisory] != exception["expires"]:
        raise SystemExit(f"{advisory}: ignoreUntil {osv[advisory]} != expires {exception['expires']}")
PY
if [ $? -eq 0 ]; then ok "osv-scanner.toml ignoreUntil matches npm-audit exceptions"; else no "osv-scanner.toml ignoreUntil mismatch"; fi

# The local quality gate must run the same external analyzers, at the same
# versions, as CI. Compare the two sources directly so a version lives only in
# engine/Makefile and .github/workflows/ci.yml.
make -C "$ROOT/engine" -n quality > "$T/make-quality" 2>&1 || true
python3 - "$T/make-quality" "$ROOT/.github/workflows/ci.yml" <<'PY'
import re, sys
from pathlib import Path
tools = ("govulncheck", "golangci-lint")
def versions(text):
    found = {t: set(re.findall(rf"/cmd/{re.escape(t)}@(\S+)", text)) for t in tools}
    # CI installs golangci-lint through golangci-lint-action's `version:` input.
    action = re.search(r"golangci/golangci-lint-action@\S+.*?\n\s+with:\s*\n\s+version:\s*(\S+)", text, re.S)
    if action:
        found["golangci-lint"].add(action.group(1))
    return found
make = versions(Path(sys.argv[1]).read_text())
ci = versions(Path(sys.argv[2]).read_text())
for t in tools:
    if len(make[t]) != 1 or make[t] != ci[t]:
        raise SystemExit(f"{t}: make quality {sorted(make[t])} != ci.yml {sorted(ci[t])}")
PY
if [ $? -eq 0 ]; then ok "make quality and CI pin the same analyzer versions"; else no "make quality and CI analyzer versions differ"; fi

# The validate-tools installer runs in CI with the job's GITHUB_TOKEN, so the
# osv-scanner release asset must be checksum-verified before it is made executable.
# A stand-in curl plus pre-satisfied actionlint/zizmor isolate that fetch: a
# matching checksum must install, a tampered payload must abort with nothing
# installed, and no fetched file may reach an install without verification.
ci_stub="$T/ci-stub"
ci_assets="$T/ci-assets"
mkdir -p "$ci_stub" "$ci_assets"
case "$(uname -m)" in
  x86_64|amd64) ci_goarch=amd64 ;;
  aarch64|arm64) ci_goarch=arm64 ;;
  *) ci_goarch=unknown ;;
esac
case "$(uname -s)" in
  Linux) ci_goos=linux ;;
  Darwin) ci_goos=darwin ;;
  *) ci_goos=unknown ;;
esac
ci_asset="osv-scanner_${ci_goos}_${ci_goarch}"
printf '#!/usr/bin/env bash\necho "osv-scanner stub"\n' > "$ci_assets/good"
shasum -a 256 "$ci_assets/good" | awk -v f="$ci_asset" '{ print $1 "  " f }' > "$ci_assets/SHA256SUMS"
cp "$ci_assets/good" "$ci_assets/tampered"
printf '# substituted payload\n' >> "$ci_assets/tampered"
cat > "$ci_stub/curl" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
out=""
url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$url" in
  */osv-scanner_SHA256SUMS) cp "$CI_STUB_ASSETS/SHA256SUMS" "$out" ;;
  */osv-scanner_*) cp "$CI_STUB_ASSETS/${CI_STUB_PAYLOAD}" "$out" ;;
  *) echo "stub curl: unexpected URL: $url" >&2; exit 22 ;;
esac
SH
chmod +x "$ci_stub/curl"
ci_prepare_dest() {
  local dest="$1"
  rm -rf "$dest"
  mkdir -p "$dest/tools" "$dest/venv/bin"
  printf '#!/usr/bin/env bash\necho "actionlint ci-stub"\n' > "$dest/tools/actionlint"
  printf '#!/usr/bin/env bash\necho "zizmor ci-stub"\n' > "$dest/venv/bin/zizmor"
  chmod +x "$dest/tools/actionlint" "$dest/venv/bin/zizmor"
}
ci_run() {
  env ACTIONLINT_VERSION=ci-stub OSV_SCANNER_VERSION=ci-stub \
    CI_VALIDATE_TOOLS_DIR="$1" CI_STUB_ASSETS="$ci_assets" CI_STUB_PAYLOAD="$2" \
    PATH="$ci_stub:$PATH" bash "$ROOT/scripts/ci-install-validate-tools.sh"
}
ci_prepare_dest "$T/ci-verified"
if ci_run "$T/ci-verified" good >"$T/out" 2>&1; then
  if [ -x "$T/ci-verified/tools/osv-scanner" ] && "$T/ci-verified/tools/osv-scanner" --version >/dev/null 2>&1; then
    ok "ci-install-validate-tools installs a checksum-verified osv-scanner"
  else
    no "ci-install-validate-tools exited 0 without installing osv-scanner"; sed -n '1,80p' "$T/out"
  fi
else
  no "ci-install-validate-tools rejected the published checksum"; sed -n '1,80p' "$T/out"
fi
ci_prepare_dest "$T/ci-tampered"
if ci_run "$T/ci-tampered" tampered >"$T/out" 2>&1; then
  no "ci-install-validate-tools installed a tampered osv-scanner"
else
  if [ -e "$T/ci-tampered/tools/osv-scanner" ]; then
    no "ci-install-validate-tools aborted after installing the tampered osv-scanner"
  else
    ok "ci-install-validate-tools refuses a tampered osv-scanner before install"
  fi
fi
if python3 - "$ROOT/scripts/ci-install-validate-tools.sh" >"$T/out" 2>&1 <<'PY'
import re, sys
from pathlib import Path

checksum = re.compile(r"\bshasum\b[^\n]*\s-c\b|\bsha256sum\b[^\n]*\s-c\b|\bopenssl\s+dgst\b|\bcosign\s+verify\b")
fetch = re.compile(r"\bcurl\b[^\n]*\s-o\s+\S")
install = re.compile(r"^\s*install\s+-m\b")
pending = False
verified = False
for number, line in enumerate(Path(sys.argv[1]).read_text().splitlines(), 1):
    if fetch.search(line):
        pending, verified = True, False
    elif pending and checksum.search(line):
        verified = True
    elif pending and install.search(line):
        if not verified:
            raise SystemExit(f"line {number}: fetched file reaches install with no checksum verification: {line.strip()}")
        pending = False
PY
then
  ok "remote fetches are checksum-verified before install"
else
  no "remote fetch reaches install without verification"; sed -n '1,80p' "$T/out"
fi

# Every row of the tool-coverage table maps a tool state onto both fields the
# standard requires per row: a status and a result, taken from the enums it defines.
if python3 - "$ROOT/pack/.claude/skills/devrites-lib/reference/standards/verification-methods.md" >"$T/out" 2>&1 <<'PY'
import re, sys
from pathlib import Path

text = Path(sys.argv[1]).read_text()
def enum(field):
    m = re.search(r"^- \*\*" + field + r"\*\* [^\n]*(?:\n  [^\n]*)*", text, re.M)
    return set(re.findall(r"`(\w+)`", m.group(0))) if m else set()
statuses, results = enum("status"), enum("result")
if len(statuses) != 5 or len(results) != 3:
    raise SystemExit(f"status/result enums not found: {sorted(statuses)} {sorted(results)}")
section = text.split("### Tool coverage states", 1)[1]
rows = section.split("\n\n", 3)[2].splitlines()[2:]
bad = []
for row in rows:
    cell = row.split("|")[2]
    ticks = set(re.findall(r"`(\w+)`", cell))
    if not (ticks & statuses and ticks & results):
        bad.append(row)
if not rows or bad:
    raise SystemExit("rows missing a status or a result:\n" + "\n".join(bad))
PY
then
  ok "every tool-coverage row names a status and a result"
else
  no "a tool-coverage row lacks a status or a result"; sed -n '1,80p' "$T/out"
fi

# The test runner documents its flags and reports a bad --shard as a one-line
# usage error instead of an uncaught stack trace.
run_ok "test runner --help exits 0 with usage" node "$ROOT/scripts/run-tests.mjs" --help
if grep -qi '^usage' "$T/out"; then ok "test runner --help prints usage"; else no "test runner --help prints usage"; fi
for shard in 0/3 4/3 x 1/0 15/14 ''; do
  node "$ROOT/scripts/run-tests.mjs" --shard "$shard" >"$T/out" 2>&1; rc=$?
  if [ "$rc" -eq 2 ] && [ "$(wc -l <"$T/out" | tr -d ' ')" -eq 1 ] && ! grep -q ' at ' "$T/out"; then
    ok "test runner rejects --shard '$shard' with exit 2 and one line"
  else
    no "test runner rejects --shard '$shard' (exit $rc)"; sed -n '1,10p' "$T/out"
  fi
done
node "$ROOT/scripts/run-tests.mjs" --shard >"$T/out" 2>&1; rc=$?
if [ "$rc" -eq 2 ]; then ok "test runner rejects --shard without a value"; else no "test runner rejects --shard without a value (exit $rc)"; fi

echo ""
[ "$fail" -eq 0 ] && echo "validation-governance-test: PASS" || echo "validation-governance-test: FAIL"
exit "$fail"
