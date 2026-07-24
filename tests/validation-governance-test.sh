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

# Every supporting reference is size-ratcheted, not only SKILL.md.
mkdir -p "$T/size/pack/.claude/skills/demo/reference" "$T/size/tests"
printf '# demo\n[details](reference/details.md)\n' > "$T/size/pack/.claude/skills/demo/SKILL.md"
printf '# details\nsmall\n' > "$T/size/pack/.claude/skills/demo/reference/details.md"
run_ok "instruction baseline writes references" node "$ROOT/scripts/check-instruction-size-baseline.mjs" --root "$T/size" --baseline "$T/size/tests/baseline.json" --write
printf 'unreviewed growth that must trip the ratchet\n' >> "$T/size/pack/.claude/skills/demo/reference/details.md"
run_fail_contains "instruction baseline catches reference growth" "reference/details.md grew" node "$ROOT/scripts/check-instruction-size-baseline.mjs" --root "$T/size" --baseline "$T/size/tests/baseline.json"
run_fail_contains "reference file budget is blocking" "reference/details.md" env DEVRITES_REFERENCE_FILE_BUDGET=20 node "$ROOT/scripts/check-generated-skill-budget.mjs" "$T/size/pack/.claude/skills"

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

# Dependency audit exceptions are exact, owner-bound, expiring, and stale-intolerant.
cat > "$T/npm-audit.json" <<'JSON'
{"auditReportVersion":2,"vulnerabilities":{"npm":{"severity":"moderate","via":["tar"],"nodes":["node_modules/npm"]},"tar":{"severity":"moderate","via":[{"name":"tar","url":"https://github.com/advisories/GHSA-r292-9mhp-454m","severity":"moderate","range":"<=7.5.20"}],"nodes":["node_modules/npm/node_modules/tar"]}}}
JSON
cat > "$T/npm-audit-exceptions.json" <<'JSON'
[{"id":"GHSA-r292-9mhp-454m","package":"tar","range":"<=7.5.20","nodes":["node_modules/npm/node_modules/tar"],"source":"https://github.com/advisories/GHSA-r292-9mhp-454m","owner":"security","reason":"fixture","expires":"2099-01-01"}]
JSON
run_ok "npm audit accepts one exact temporary exception" node "$ROOT/scripts/check-npm-audit.mjs" --input "$T/npm-audit.json" --exceptions "$T/npm-audit-exceptions.json"
node -e 'const fs=require("fs");const p=JSON.parse(fs.readFileSync(process.argv[1]));p[0].expires="2000-01-01";fs.writeFileSync(process.argv[2],JSON.stringify(p))' "$T/npm-audit-exceptions.json" "$T/npm-audit-expired.json"
run_fail_contains "npm audit rejects an expired exception" "expired" node "$ROOT/scripts/check-npm-audit.mjs" --input "$T/npm-audit.json" --exceptions "$T/npm-audit-expired.json"
node -e 'const fs=require("fs");const p=JSON.parse(fs.readFileSync(process.argv[1]));p.vulnerabilities.other={severity:"moderate",via:[{name:"other",url:"https://github.com/advisories/GHSA-aaaa-bbbb-cccc",severity:"moderate",range:"<2"}],nodes:["node_modules/other"]};fs.writeFileSync(process.argv[2],JSON.stringify(p))' "$T/npm-audit.json" "$T/npm-audit-extra.json"
run_fail_contains "npm audit rejects an unexcepted advisory" "not excepted" node "$ROOT/scripts/check-npm-audit.mjs" --input "$T/npm-audit-extra.json" --exceptions "$T/npm-audit-exceptions.json"

# The advertised local quality gate must be self-contained and pin the same
# three external analyzers used by CI.
make -C "$ROOT/engine" -n quality > "$T/make-quality" 2>&1 || true
for needle in 'staticcheck@2025.1.1' 'govulncheck@v1.5.0' 'gosec@v2.27.1'; do
  if grep -q "$needle" "$T/make-quality"; then ok "quality pins $needle"; else no "quality does not pin $needle"; fi
done

echo ""
[ "$fail" -eq 0 ] && echo "validation-governance-test: PASS" || echo "validation-governance-test: FAIL"
exit "$fail"
