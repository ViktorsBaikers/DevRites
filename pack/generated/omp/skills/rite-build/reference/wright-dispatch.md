# Slice-wright dispatch

> Root-only: not inlined in --role bundles.

`/rite-build` gives one bounded task to exact `devrites-slice-wright`.
Root owns scope/artifacts/inspection; never writes source/tests.
Wright alone writes; no agents.

## Host gate

Apply [source-writing boundary](skill://devrites-lib/reference/standards/agents.md#source-writing-boundary):
omp has no permission modes: only the wright's `tools` allowlist has `edit`/`write`, and the root stays source-read-only by instruction. Claude grants only the exact wright `acceptEdits`; Codex uses its workspace root,
the exact `:workspace` wright, and read-only specialists. Never bypass/substitute
the wright or recreate an engine bridge.

## Standalone correction

`/rite-quick`, `/rite-dogfood`, `/rite-harness`, and `/rite-pr-feedback` fixes have no `tasks.md`
slice, so skip Prepare's `check slice`, `context --slice`, and the build `dispatch`
wave. Give the wright an inline contract: goal, acceptance check, exact
project-relative paths, and proof command. With an active workspace, `claim add`
those paths first. On return, compare its file list and `git diff --name-only`
with the paths; reject any extra path.

## Isolated writer-worktree pilot

Use native isolation only when the current host exposes an explicit named-writer
worktree plus result reconciliation; separate agent threads or inherited sandboxes do
not qualify. Codex CLI custom subagents therefore use `same-worktree` unless a future
supported interface passes this capability gate—never create a manual worktree from the
read-only root to imitate it. Also require `git rev-parse
--show-superproject-working-tree` to be empty; resolved git/common dirs; committed,
clean candidate/index; all inputs at `HEAD`; no other writer; and green cheap isolated
baseline proof. Otherwise use serial same-worktree dispatch—never stash, commit, or
discard user work to qualify. Ask host for an explicit current-`HEAD` base when
supported. Regardless of host defaults, the wright's first command must prove actual `git rev-parse HEAD` equals supplied
`worktree_base`. Mismatch returns a gap with no write before project reads or baseline proof.

The isolated wright returns one local unpushed `transfer_commit` (contract
`WIP(<slug>):` subject, no slice id), its `worktree_base`, and exact files.
That commit is worker-branch transport, not the control-branch checkpoint. Root proves descendant base, exact `git diff --name-only`
`` `<base>..<transfer>` ``, no `.devrites/**`/submodule/symlink/unrelated delta, unchanged
source base, and no user-work overwrite. Use only host-native explicit reconciliation;
never ad hoc copy, cherry-pick, or merge from read-only root. Land on control
only per [`checkpoint.md`](checkpoint.md).

Conflict, extra/missing commit, moved base, or cleanup failure is `gap`/STOP:
preserve the worktree and commit. Without explicit reconciliation, use same-worktree serial.
Parallel isolated-writer worktrees remain forbidden until this serial pilot measures
transfer, conflict, proof, and review outcomes on both hosts; opt-in `/rite-build --parallel N`
(2≤N≤10) path-disjoint fan-out is the only sanctioned parallel mode. Same-worktree multi-writer / root-emulated
worktrees stay forbidden.

## Prepare

1. Run `devrites-engine check slice <slug> <SLICE-ID>` first — it fails a
   malformed contract (missing goal, empty/invalid Writer allowlist,
   unresolvable `Satisfies` AC ids, empty proof field) before a wright
   context is spent; a failure is a gap, not a dispatch. A slice with a
   `pre-dispatch <SLICE-ID>` preflight row next runs
   `devrites-engine check preconditions <slug> --slice <SLICE-ID>` before
   `dispatch open`; non-zero is the same gap. Then derive the
   smallest exact project-relative source/test path list; reject
   directories/globs, traversal, symlink escapes, duplicates, and `.devrites/**`.
   A target composed only of vetted executable workflow artifacts routes to the
   controlling root under
   [`workflow-artifacts.md`](skill://devrites-lib/reference/standards/workflow-artifacts.md);
   its rejection here is not a blocker.
2. Goal, verbatim acceptance, exclusions, context, `test-plan.md` proof
   commands, and applicable standards reach the wright through the slice bundle
   its contract names; only a native-worktree pilot contract adds the exact
   `WIP(<slug>):` subject (no slice id). Build the wright's read-set with
   `devrites-engine context --skills-root .omp/skills <slug> --phase build --role slice-wright --slice <SLICE-ID> --trigger <set>`
   and hand it the bundle path (over-budget artifacts arrive as anchor indexes;
   read owning sections live). Evaluate each trigger condition and pass those
   that fire (TDD slice → `tdd`, AFK run → `afk`, UI work → `frontend`, …).
   The command prints `unselected=[...]`; an omitted applicable trigger is a gap.
   For each triggered
   topology/data/integration standard, add as anchored `--focus` rows only the feature-specific owner/invariant,
   failure or partial-state case, recovery rule, and required proof from the vetted plan.
   Do not paste the whole standard or silently omit an applicable risk.
   Quote, as a `--focus` row, every active `decisions.md` carry-forward constraint whose `Files:`
   overlap the task paths (or that names no files).
   A new path requires a new bounded contract; the wright cannot widen the task.
   Name (`--checks` gate IDs) the writer-safe commands the wright must run (syntax/type/lint, unit and
   self-check suites, a narrow local smoke of the changed path) and never forbid
   them: a wright that cannot execute writes blind and its failures surface a
   whole review round later. Only artifact-producing, consumptive, or full
   canonical feature gates stay root-owned; when `gates.md` has a `RUN: writer`
   gate, the checks include `devrites-engine gates precheck <slug> --worktree <path> --slice <SLICE-ID>`
   (those gates, minus ones only other unbuilt slices own, plus `check windows`;
   never evidence, root still runs every gate).
3. Run `devrites-engine check diff-scope <slug> --snapshot` before dispatch so
   unrelated work remains distinguishable (it fingerprints every dirty path). For an isolated pilot, also record committed base SHA and exact
   baseline status before asking the host for isolation. **Failing case:** paths that
   were dirty at that baseline are listed as slice-owned after dispatch.

## Run

Same-worktree (non-isolated) dispatch claims before writing: the `dispatch open`
below adds `--claim-session <id>`, one `--claim <path>` per task path, and
`--claim-ttl <minutes>` sized to the write; it runs the
`claim check --session <id> <task-paths>` test, and a live foreign holder fails the
open with no wave and no claim and stops the dispatch, per
[`agents.md`](skill://devrites-lib/reference/standards/agents.md). `.devrites/**`
is never a task path; see [Workflow Artifact admission](skill://devrites-lib/reference/standards/workflow-artifacts.md).
Render the contract with `devrites-engine packet <slug> --wave <wave> --phase build
--role slice-wright --slice <SLICE-ID> --checks <G-ID,...> [--focus <file>]` (it
writes the linted same-worktree `packets/<wave>.contract.md`; a native-worktree pilot
writes its own contract with `Isolation`, `worktree_base`, and subject) and hand the wright that text verbatim. Reviewers read
that file, so `--focus` rows state each defect to fix with its anchor, never a reviewer's verdict,
count, or pass status ("code review is no-findings" voids the next review). Ask the host
for the exact writer in fresh context; a repair round on the same slice whose
returned wright the host can still continue goes to that wright on its existing
handle instead, so it does not re-read the slice from scratch. Wait, and track it
through the launch barrier — `devrites-engine dispatch <slug> open --phase build --wave
<slice-id> --role slice-wright --slice <SLICE-ID> --contract <file>` (it refuses a
contract that forbids writer-safe checks or names none), then `start` with the host handle
(the last start seals the wave) and `return` on completion (auto-records dispatch/return metrics). Every 5 wright
rounds on one slice, `open` blocks for a reslice review. First re-admit each
finding still driving rounds under the Proportionality rule of
[agents.md § Result admission](skill://devrites-lib/reference/standards/agents.md#result-admission): one without a broken anchor or a reachable trigger becomes a
Suggestion and drives no further round. Then propose an
acceptance-preserving split (`root_no_change_analysis` → `FOLD` per
[acceptance-preserving-reslice](skill://devrites-lib/reference/standards/acceptance-preserving-reslice.md);
open findings move to the new slice owning their paths), or record
`## <DRIFT-ID> — <SLICE-ID> reslice review` with why no smaller split exists and
a spec trace of the slice's obligations (hardening no AC/INV/PROH requires moves to
a sibling slice dependents do not need, still gated before Seal), and
reopen with `--reslice-review <DRIFT-ID>`. Use
at most one writer across all linked worktrees for this workspace. Never run two
writers in one worktree, run isolated and same-worktree writers concurrently, or
substitute a generic agent. Opt-in `/rite-build --parallel N` fans out only under
[`parallel-batch.md`](parallel-batch.md).

The `return` that completes the wave releases its claim; `dispatch abandon --release`
releases it on any other host-confirmed terminal path (a stop, a launch refused with no
handle, or a timed-out child later confirmed terminal). A `claim add` claim releases
with `devrites-engine claim release --session <id> --id <claim-id>` on every terminal path:
successful return, rejected result, stop, or a launch the host refused with no handle. A
timeout or unknown outcome keeps the claim ([`parallel-dispatch.md`](skill://devrites-lib/reference/parallel-dispatch.md#cancellation-and-terminal-reconciliation)).
TTL is crash recovery, not normal cleanup.
An `unattributed root edit` exit 3 (root-edit tripwire) routes the path through a wright wave naming it in
`--claim`, or, given evidence (owner work, gate output, a standalone wright, Polish ADR/`DESIGN.md`),
`dispatch <slug> attribute --reason <evidence> --path <path>`.

## Inspect and prove

1. Compare the returned file list and `git diff --name-only` with task paths.
   `devrites-engine check diff-scope <slug> --allow <task-paths> --baseline` is the
   mechanical subset gate (a step-3 path whose bytes are unchanged is not a
   violation; a changed one still is) — run it before any reviewer dispatch; a violation
   restores through the bounded wright without spending review, but only for a
   path the wright's `files_changed` or diff trace attributes to it. An
   unattributed extra path that was clean at the step-3 baseline is preserved
   (never restored), recorded as drift, and stops for the human. **Failing case:**
   a user edits `README.md` during a same-tree dispatch and the wright reverts it.
   Reject a result that omits any required key — the bookkeeping arrays count
   whether empty or filled — or adds a path. Root never widens scope or edits
   source.
2. Run `devrites-engine check windows <slug>` on the wright's diff (`--worktree`
   in the same tree, plus `--since-dispatch <wave>` naming a repair round's wave;
   `--base <B> --cwd <worktree>` for a parallel transfer commit). Each unwaived hit is repaired through the bounded wright or gets a
   reasoned `windows.md` row before review; never waive to reach green. Then
   inspect the test diff for deletion, skipping, focus markers, or loosened
   assertions. Dedicated test analysis treats weakening as Critical.
   Confirm a test for a data/integration/topology risk can actually exhibit that risk;
   mocks that erase it and one-root proof offered for another root are unproven.
3. Apply [independent Build review](phase-contract.md#independent-build-review)
   and, in the same window, run only repository proof already approved by
   `test-plan.md` against the same frozen candidate; neither waits for the
   other. Human choices return; technical failures follow
   [the canonical retry contract](skill://devrites-lib/reference/standards/afk-hitl.md#retry-cap-no-progress-loops-and-self-resolve).
   Derive accounting from recorded reproductions, corrections, and decisive rechecks.
4. After proof, inspect `git diff --name-only` again in case a proof tool changed
   source; a change rejects that candidate and voids the concurrent review. A
   repair wright voids both; rerun both on the new candidate.
   **Do not land on control until review and proof are both green.** Isolated
   `transfer_commit` stays on the worker branch; same-worktree diffs stay
   uncommitted. Serial: [`checkpoint.md`](checkpoint.md), then host may remove the
   worktree. Parallel: `record-green`, then `parallel integrate
   --slice <id> --apply-to-control`; `parallel cleanup` removes worktrees.
5. Reconcile returned reuse, conventions, principles, sources, assumptions,
   decisions, dead ends, follow-ups, gates, and touched files. Persist the
   relevant facts in canonical artifacts, not the wholesale report. A plan-gap
   escalation without the checked-assumption list (verified|falsified) is
   malformed; root sweeps the unchecked remainder per
   [`spec-drift-guard.md`](spec-drift-guard.md) before repair.
6. **Learning edge.** A correction accepted with a confirmed cause (a review,
   gate, or proof rejection plus its fix) becomes a carry-forward constraint so
   later slices start where this one ended: append one line under
   `decisions.md` `## Carry-forward constraints` —
   `- <rule> — from <SLICE-ID> (<evidence ref>); Files: <paths>`. Record the
   derived rule, not the output, at the scope its evidence supports: "this
   button is blue" never becomes "buttons are blue". Never edit `tasks.md` mid-build (it is a
   readiness input). **Failing case:** slice 3 repeats a defect slice 2 fixed.
