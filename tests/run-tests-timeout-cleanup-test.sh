#!/usr/bin/env bash
# The runner must split the workflow-artifact-identity test into separately
# budgeted pieces and must let a timed-out child run its cleanup (SIGTERM
# before SIGKILL). Exercised only against a synthetic stub in a scratch root.
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
scratch="$(mktemp -d "${TMPDIR:-/tmp}/run-tests-timeout.XXXXXX")"
trap 'rm -rf "$scratch"' EXIT

mkdir -p "$scratch/root/scripts" "$scratch/root/tests" "$scratch/markers" "$scratch/artifacts"
cp "$repo/scripts/run-tests.mjs" "$scratch/root/scripts/run-tests.mjs"

cat > "$scratch/root/tests/workflow-artifact-identity-test.sh" <<'STUB'
#!/usr/bin/env bash
if [ -n "${DEVRITES_WAI_BOUNDARY_ONLY:-}" ] || [ -n "${DEVRITES_WAI_DELIVERY_MODEL_ONLY:-}" ]; then exit 0; fi
exec python3 -c '
import os, signal, sys, time
signal.signal(signal.SIGTERM, lambda *_: sys.exit(143))
marker = os.path.join(os.environ["MARKER_DIR"], os.environ.get("DEVRITES_WAI_CORE_SHARD", "whole").replace("/", "-"))
open(marker, "w").close()
try:
    time.sleep(60)
finally:
    os.remove(marker)
'
STUB

out="$scratch/out.txt"
rc=0
MARKER_DIR="$scratch/markers" DEVRITES_HOST_ARTIFACT_DIR="$scratch/artifacts" \
  DEVRITES_TEST_TIMEOUT_SEC=2 node "$scratch/root/scripts/run-tests.mjs" >"$out" 2>&1 || rc=$?

fail() { echo "FAIL: $1" >&2; cat "$out" >&2; exit 1; }

[ "$rc" -ne 0 ] || fail "timed-out core piece should fail the run"
grep -q 'timeout: killed after 2s' "$out" || fail "timeout not reported"
[ -z "$(ls -A "$scratch/markers")" ] || fail "timed-out child did not run its cleanup"
grep -q '^PASS: .*workflow-artifact-identity-test.sh#boundary' "$out" || fail "no separate boundary piece"
grep -q '^FAIL: .*workflow-artifact-identity-test.sh#core' "$out" || fail "no separate core piece"

# Sharding must partition the tests even when the workflow-artifact-identity
# test is not part of the selected set.
mkdir -p "$scratch/split/scripts" "$scratch/split/tests"
cp "$repo/scripts/run-tests.mjs" "$scratch/split/scripts/run-tests.mjs"
for n in a b c d e; do printf '#!/usr/bin/env bash\nexit 0\n' > "$scratch/split/tests/$n-test.sh"; done
for i in 1 2 3; do
  DEVRITES_HOST_ARTIFACT_DIR="$scratch/artifacts" node "$scratch/split/scripts/run-tests.mjs" --serial --shard "$i/3" 2>&1 \
    | sed -n 's/^PASS: //p' | sed 's/ .*//' | sort > "$scratch/shard-$i.txt" || true
done
[ "$(cat "$scratch"/shard-*.txt | wc -l | tr -d ' ')" -eq 5 ] || fail "shards must run each test exactly once (ran $(cat "$scratch"/shard-*.txt | wc -l | tr -d ' ') of 5)"
[ "$(cat "$scratch"/shard-*.txt | sort -u | wc -l | tr -d ' ')" -eq 5 ] || fail "shards overlap"
echo "run-tests timeout cleanup test passed"
