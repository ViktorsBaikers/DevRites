#!/usr/bin/env bash
# supply-chain-iocs-test.sh — fixtures for scan-supply-chain-iocs.py:
# a lockfile pinning a known-malicious version → finding; a safe version → clean.
# Offline only (bundled IOC dataset) so it's deterministic in CI.
set -u

HERE="$(cd "$(dirname "$0")" && pwd)"
SCAN="python3 $HERE/../scripts/scan-supply-chain-iocs.py"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
fail=0

finds() { if $SCAN "$2" >/dev/null 2>&1; then echo "FAIL [$1]: expected a finding"; fail=1; else echo "ok   [$1]"; fi; }
clean() { if $SCAN "$2" >/dev/null 2>&1; then echo "ok   [$1]"; else echo "FAIL [$1]: expected clean:"; $SCAN "$2"; fail=1; fi; }

# v3 lockfile pinning a compromised ua-parser-js release → finding
cat > "$TMP/bad.json" <<'EOF'
{ "lockfileVersion": 3, "packages": {
  "": { "name": "x" },
  "node_modules/ua-parser-js": { "version": "0.7.29" },
  "node_modules/left-pad": { "version": "1.3.0" }
} }
EOF
finds "compromised version flagged" "$TMP/bad.json"

# v3 lockfile with a SAFE ua-parser-js version → clean
cat > "$TMP/clean.json" <<'EOF'
{ "lockfileVersion": 3, "packages": {
  "": { "name": "x" },
  "node_modules/ua-parser-js": { "version": "1.0.37" },
  "node_modules/left-pad": { "version": "1.3.0" }
} }
EOF
clean "safe version passes" "$TMP/clean.json"

# v1-style (dependencies tree) with event-stream@3.3.6 → finding (legacy path)
cat > "$TMP/v1.json" <<'EOF'
{ "lockfileVersion": 1, "dependencies": {
  "event-stream": { "version": "3.3.6" },
  "lodash": { "version": "4.17.21" }
} }
EOF
finds "v1 lockfile path flagged" "$TMP/v1.json"

# missing lockfile → clean (exit 0)
clean "missing lockfile → clean" "$TMP/does-not-exist.json"

# regression: the repo's real lockfile passes
clean "repo lockfile passes" "$HERE/../package-lock.json"

if [ "$fail" -ne 0 ]; then echo "SUPPLY-CHAIN-IOC TESTS: FAIL"; exit 1; fi
echo "SUPPLY-CHAIN-IOC TESTS: PASS"
exit 0
