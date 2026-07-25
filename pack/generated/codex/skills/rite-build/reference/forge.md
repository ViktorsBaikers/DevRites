# Forge: isolated candidate builds

Forge is the rare `$rite-build` branch for a vetted architecture fork. Two or
three wrights implement the same slice in separate, manifest-owned worktrees; a
read-only judge selects one; only that candidate lands.

The engine manifest (`devrites-forge/v1`) is the sole authority for run,
candidate, branch, worktree, worker, merge, verification, and cleanup identity.
`forge-report.md`, names, paths, and prompts are records or inputs, never
ownership.

## Entry contract

`$rite-vet` may set `Forge: yes` only when the slice has all three:

```markdown
Forge: yes — <why building the wrong architecture costs more than 2–3 attempts>
Forge strategies: A=<complete approach> | B=<complete approach> [| C=<complete approach>]
Forge scorecard: acceptance=<AC-### list>; test-plan=<exact test-plan.md rows/commands>
```

Strategies are distinct seams, data shapes, reuse choices, or algorithms, not
wording variants. The scorecard covers every slice AC and every required test.
Otherwise set `Forge: no`, both Forge detail fields to `none`, and dispatch one
wright.

Before `forge plan`, also require a Git checkout and a real host adapter that can
declare `manifest-env-v1` and:

- start each candidate in its exact worktree;
- expose its stable worker ID, live PID, and matching process-start token before
  its first tool call;
- set all five Forge binding variables; and
- end that process before extraction.

Never claim that binding on the adapter's behalf. Without it, an unbound
`forge plan` returns typed `degraded` / `serial` JSON and creates nothing; accept
that result and use the normal serial wright.

## Lifecycle

### 1. Plan, then snapshot

Write the normal root-owned `.wright-allowlist`. Hash the complete scorecard
owners, not excerpts:

```bash
SLUG="$(cat .devrites/ACTIVE)"
WORK=".devrites/work/$SLUG"
hash_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    return 127
  fi
}
ACCEPTANCE_HASH="$(hash_file "$WORK/spec.md")"
TEST_PLAN_HASH="$(hash_file "$WORK/test-plan.md")"

devrites-engine forge plan "<SLICE-###>" "$SLUG" \
  --strategy "A=<strategy A>" \
  --strategy "B=<strategy B>" \
  --acceptance-hash "$ACCEPTANCE_HASH" \
  --test-plan-hash "$TEST_PLAN_HASH" \
  --worker-binding=manifest-env-v1
```

Add `--strategy "C=<strategy C>"` only for a third vetted strategy. Both hashes
must be the complete 64-hex SHA-256 values of the current files. The command
shown is only for an adapter that really provides the binding. Otherwise omit
`--worker-binding`; the engine returns the side-effect-free serial result.

Read the JSON result:

- `status: planned`: take the run ID and every candidate path from its
  manifest; then run `devrites-engine reconcile snapshot`.
- `status: degraded, mode: serial`: record the reason and use one normal
  wright. The engine created no Forge state.
- non-zero exit: classify it as a technical defect. Preserve any manifest
  state and recover it; never infer paths or silently start a serial writer.

`forge plan` must precede `reconcile snapshot`: validated operational manifests
are reconciliation-private, while `forge-report.md` and every other `.forge`
path remain visible.

### 2. Bind and dispatch every candidate

For each candidate, the host starts a fresh worker at the manifest's exact
`worktree`, pauses it before its first tool call, and obtains the engine token
for its live PID:

```bash
devrites-engine forge process-token "$WORKER_PID"
```

Read `process_start` from the JSON; never reproduce the token algorithm in
shell. Record that exact identity, then release the worker:

```bash
devrites-engine forge record "$RUN_ID" "$CANDIDATE" started \
  --worker-id "$WORKER_ID" \
  --pid "$WORKER_PID" \
  --process-start "$PROCESS_START"
```

The candidate process receives this all-or-none environment:

```text
DEVRITES_FORGE_RUN_ID=<manifest run_id>
DEVRITES_FORGE_CANDIDATE=<A|B|C>
DEVRITES_FORGE_WORKER_ID=<recorded worker id>
DEVRITES_FORGE_WORKER_PID=<recorded live pid>
DEVRITES_FORGE_PROCESS_START=<recorded start token>
```

Its cwd is the exact manifest worktree. Give it the normal
[`wright-dispatch.md`](wright-dispatch.md) packet plus its one recorded
strategy. The same project-relative allowlist applies inside that candidate
tree. Any missing, partial, stale, wrong-tree, wrong-branch, or wrong-worker
binding is a denial, not a reason to widen scope.

After the worker returns and its process exits:

```bash
devrites-engine forge record "$RUN_ID" "$CANDIDATE" finished \
  --worker-id "$WORKER_ID"
devrites-engine forge extract "$RUN_ID" "$CANDIDATE"
```

Extraction makes the complete tracked, staged, unstaged, untracked, deletion,
mode, and symlink delta reachable by the recorded candidate commit and binds
its SHA-256. Ignored, dirty, mismatched, live, or ambiguous content is
preserved. A crashed worker is recorded `failed`; the run stays preserved for
technical recovery and does not degrade in place.

### 3. Judge, record, and merge

After **every** candidate is extracted, dispatch
[`devrites-forge-judge`](.codex/agents/devrites-forge-judge.toml) in fresh,
read-only context. Give it the manifest, slice scorecard, base commit, and each
recorded candidate commit/tree/delta hash. It scores the immutable diffs against
the same acceptance, tests, principles, reuse, simplicity, and anti-slop bar.

For one qualifying winner:

```bash
devrites-engine forge record "$RUN_ID" winner "<A|B|C>" \
  --worker-id "<judge result id>"
devrites-engine forge merge "$RUN_ID" "<A|B|C>"
```

Merge revalidates every extract and the unchanged primary baseline, performs a
no-side-effect conflict preflight, then fast-forwards to exactly the recorded
winner. A mismatch or conflict leaves the primary tree unchanged. Runner-up
ideas remain report notes; never hand-merge candidates or mutate the landed
winner inside this run.

If no candidate qualifies, preserve the run and route the actual finding:
product/acceptance ambiguity returns to planning; implementation or proof
defects use bounded technical recovery. Retry authorization is not a human
decision.

### 4. Verify, clean, then record

Hand the winner's structured artifact to the normal cycle. Run immediate
reconciliation, doubt every stood decision, and complete test-integrity,
targeted, browser, and other slice proof. Only after every required gate is
green:

```bash
devrites-engine forge record "$RUN_ID" verification verified \
  --worker-id "<verifier result id>"
devrites-engine forge cleanup "$RUN_ID"
devrites-engine reconcile close
```

Cleanup reads only the manifest. It removes an exact terminal, dead, clean,
reachable worktree; preserves live, dirty, mismatched, unreachable, or
ambiguous state with a reason; and never deletes candidate branches.

Now write `.devrites/work/<slug>/forge-report.md` alongside the normal Build
records. The report is written after reconciliation and verification, includes
cleanup preservation results, and never authorizes merge, cleanup, or reap.

```markdown
# Forge report: <SLICE-### — name>
Run: <run-id> · scorecard: <acceptance hash> / <test-plan hash>

## Candidates
| # | Strategy | Gates | Delta SHA-256 | Judge result |
|---|---|---|---|---|
| A | <approach> | <green/red> | <hash> | <score + decisive evidence> |
| B | <approach> | <green/red> | <hash> | <score + decisive evidence> |

## Verdict
Winner: <A|B|C> — <rubric-based reason>
Discarded: <one reason per loser>
Runner-up notes: <ideas for later | none>
Cleanup: <complete | preserved candidate + exact engine reason>
```

## Interrupted runs

Use `devrites-engine forge reap [slug]` only for recovery. It enumerates
validated manifests rather than matching directories or branch names. It applies the
same exact identity, liveness, cleanliness, and reachability checks as cleanup.
Foreign, live, dirty, malformed, mismatched, or ambiguous state is reported and
preserved. Never infer a path, suppress a failure, or delete a branch.
