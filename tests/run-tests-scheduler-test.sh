#!/usr/bin/env bash
# A test waiting on a busy exclusive chain must not hold a worker slot: free
# tests run in the meantime. Exercised only against synthetic stubs in a
# scratch root with two workers. Each stub logs its own start and end
# timestamps, so the assertions are about ordering and overlap, not wall time.
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
scratch="$(mktemp -d "${TMPDIR:-/tmp}/run-tests-scheduler.XXXXXX")"
trap 'rm -rf "$scratch"' EXIT

root="$scratch/root"
log="$scratch/log"
mkdir -p "$root/scripts" "$root/tests" "$scratch/artifacts" "$log"
cp "$repo/scripts/run-tests.mjs" "$root/scripts/run-tests.mjs"

# stub NAME SLEEP_SECONDS: sleeps, logging epoch-ms start and end.
stub() {
  cat > "$root/tests/$1" <<STUB
#!/usr/bin/env bash
now() { node -e 'process.stdout.write(String(Date.now()))'; }
now > "$log/$1.start"
sleep $2
now > "$log/$1.end"
STUB
}

# Both share the install chain; the heavier one is scheduled first.
stub install-smoke.sh 5
stub install-shared-file-merge-smoke.sh 1
for n in a b c d; do stub "free-$n-test.sh" 1; done

out="$scratch/out.txt"
rc=0
DEVRITES_HOST_ARTIFACT_DIR="$scratch/artifacts" node "$root/scripts/run-tests.mjs" --jobs 2 >"$out" 2>&1 || rc=$?

fail() { echo "FAIL: $1" >&2; cat "$out" >&2; exit 1; }
ts() { cat "$log/$1"; }

[ "$rc" -eq 0 ] || fail "run should pass"

holder_end="$(ts install-smoke.sh.end)"
waiter_start="$(ts install-shared-file-merge-smoke.sh.start)"

[ "$waiter_start" -ge "$holder_end" ] || fail "chain tests overlapped (waiter started ${waiter_start}, holder ended ${holder_end})"

overlap=0
for n in a b c d; do
  [ "$(ts "free-$n-test.sh.start")" -lt "$holder_end" ] && overlap=$((overlap + 1))
done
[ "$overlap" -ge 1 ] || fail "no free test ran while the chain holder ran: the waiting test held a worker slot"
echo "run-tests scheduler test passed ($overlap free test(s) overlapped the busy chain)"
