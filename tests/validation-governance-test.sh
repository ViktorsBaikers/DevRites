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
run_ok "npm audit accepts one exact temporary exception" env DEVRITES_TODAY=2026-09-04 node "$ROOT/scripts/check-npm-audit.mjs" --input "$T/npm-audit.json" --exceptions "$T/npm-audit-exceptions.json"
node -e 'const fs=require("fs");const p=JSON.parse(fs.readFileSync(process.argv[1]));p[0].expires="2000-01-01";fs.writeFileSync(process.argv[2],JSON.stringify(p))' "$T/npm-audit-exceptions.json" "$T/npm-audit-expired.json"
run_fail_contains "npm audit rejects an expired exception" "expired" env DEVRITES_TODAY=2026-09-04 node "$ROOT/scripts/check-npm-audit.mjs" --input "$T/npm-audit.json" --exceptions "$T/npm-audit-expired.json"
node -e 'const fs=require("fs");const p=JSON.parse(fs.readFileSync(process.argv[1]));p[0].expires="2026-09-08";fs.writeFileSync(process.argv[2],JSON.stringify(p))' "$T/npm-audit-exceptions.json" "$T/npm-audit-soon.json"
run_fail_contains "npm audit rejects an exception inside the 7-day refresh horizon" "refresh or remove" env DEVRITES_TODAY=2026-09-04 node "$ROOT/scripts/check-npm-audit.mjs" --input "$T/npm-audit.json" --exceptions "$T/npm-audit-soon.json"
node -e 'const fs=require("fs");const p=JSON.parse(fs.readFileSync(process.argv[1]));p[0].expires="2026-09-12";fs.writeFileSync(process.argv[2],JSON.stringify(p))' "$T/npm-audit-exceptions.json" "$T/npm-audit-horizon-ok.json"
run_ok "npm audit accepts an exception outside the 7-day refresh horizon" env DEVRITES_TODAY=2026-09-04 node "$ROOT/scripts/check-npm-audit.mjs" --input "$T/npm-audit.json" --exceptions "$T/npm-audit-horizon-ok.json"
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

# Patched ancestors that retired the 2026-09 bundled-npm and fast-uri exceptions.
python3 - "$ROOT" <<'PY'
import json, sys
from pathlib import Path
root = Path(sys.argv[1])
pkg = json.loads((root / "package.json").read_text())
overrides = pkg.get("overrides") or {}
npm = ((overrides.get("@semantic-release/npm") or {}).get("npm"))
fast_uri = overrides.get("fast-uri")
if npm != "11.19.1":
    raise SystemExit(f"package.json must pin @semantic-release/npm.npm to 11.19.1, got {npm!r}")
if fast_uri != "3.1.6":
    raise SystemExit(f"package.json must pin fast-uri to 3.1.6, got {fast_uri!r}")
lock = json.loads((root / "package-lock.json").read_text())
packages = lock.get("packages") or {}
got_npm = (packages.get("node_modules/npm") or {}).get("version")
got_fast = (packages.get("node_modules/fast-uri") or {}).get("version")
if got_npm != "11.19.1":
    raise SystemExit(f"package-lock.json npm is {got_npm!r}, expected 11.19.1")
if got_fast != "3.1.6":
    raise SystemExit(f"package-lock.json fast-uri is {got_fast!r}, expected 3.1.6")
PY
if [ $? -eq 0 ]; then ok "release toolchain pins patched npm 11.19.1 and fast-uri 3.1.6"; else no "release toolchain pin missing"; fi

# The advertised local quality gate must be self-contained and pin the same
# three external analyzers used by CI.
make -C "$ROOT/engine" -n quality > "$T/make-quality" 2>&1 || true
for needle in 'staticcheck@2026.1' 'govulncheck@v1.6.0' 'gosec@v2.28.0'; do
  if grep -q "$needle" "$T/make-quality"; then ok "quality pins $needle"; else no "quality does not pin $needle"; fi
done

echo ""
[ "$fail" -eq 0 ] && echo "validation-governance-test: PASS" || echo "validation-governance-test: FAIL"
exit "$fail"
