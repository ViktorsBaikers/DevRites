# $rite-fast check and triage

The lane has no gate between slices, so this is where misses are caught. It runs
once after the last slice, and again only after fixes (at most 2 rounds).

## Check run

Run, on the full tree and in this order, the detected typecheck or build command and
then the test suite. Record each command, exit code, and first decisive failure line
in `evidence.md` `## Check run`. Do not fix anything yourself; failures become gaps.

## Gap kinds (what `fast-checker` looks for)

| Kind | Means |
| --- | --- |
| `missing-ac` | An AC has no code that makes it true. |
| `untested-ac` | An AC has code but no test that would fail without it. |
| `unfinished-slice` | A ledger slice is unticked, or its files lack the promised change. |
| `out-of-scope` | A changed path or behavior no slice or AC asked for. |
| `failing-command` | The check run is red. |
| `marker` | A `TODO`, `TBD`, placeholder, stub, or `not implemented` left in changed code. |
| `broken-contract` | A caller, type, or interface the diff changed is now inconsistent. |

Gaps only. Style, naming, and simplification belong to `fast-critic` in REFINE.

## Admission

A gap is a claim. Before routing it:
1. Open the cited `file:line` and confirm it. A citation that does not hold is
   dropped and noted as `false`.
2. Merge gaps with one root cause into one item.
3. Never admit a "fix" that edits `spec.md` or `plan.md` to match the code.

## Routes

| Route | When | Action |
| --- | --- | --- |
| `patch` | The fix stays inside the approved intent and adds no public surface. | Dispatch `fast-builder` in `fix` mode with the gap, its AC, and exact paths. |
| `intent_gap` | The fix needs a decision the spec does not make, or the plan was wrong. | Ask the user one question with a recommended answer; record `Q → A` in `spec.md`; then `patch`. |
| `defer` | Real but outside this run. | Record in `plan.md` `## Deferred`. Deferring an AC needs the user's consent. |

`out-of-scope` changes default to `patch` as a revert of that change, unless the
user accepts them as a new AC.

## Round limits

- At most 2 check-and-fix rounds, and at most 2 proof-and-fix rounds in PROVE.
- A gap with the same cause twice is no progress; stop with `Stopped:` naming it.
- Proof rules that never bend: a skipped, filtered, or `.only` test counts as
  missing; a test expectation is never edited to match the code; a deleted or
  weakened assertion is a Critical finding.
