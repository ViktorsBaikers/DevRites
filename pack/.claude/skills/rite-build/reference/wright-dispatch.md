# Slice-wright dispatch

How `/rite-build` hands the **build core** of one slice to `devrites-slice-wright` — the
fresh-context, **write-capable** executor under `.claude/agents/`. Loaded on demand by
`/rite-build`; not a skill itself. Sibling of
[`../../devrites-lib/reference/parallel-dispatch.md`](../../devrites-lib/reference/parallel-dispatch.md)
(the read-only reviewer fan-out) — same isolation principle, opposite direction: this one
**writes**.

Pattern: the orchestrator owns the **gates and the workspace**; the wright owns the **writing**.
Brief it precisely, in a clean context, with only the slice contract — then doubt, record, and
gate its return.

## Why a fresh context writes the slice
The orchestrator's window is full of spec investigation, planning, prior slices, and tool
output — the exact "lost-in-the-middle" load that degrades instruction-following and pulls the
model toward generic code. The wright starts clean and sees only the contract, so it holds the
slice boundary strictly and writes to the *project's* idiom instead of drifting. Single-threaded
by design: **one** wright per slice, never a parallel fan-out of writers *sharing a tree* —
concurrent writers on one working tree make conflicting implicit decisions and produce incoherent
code. The one sanctioned exception is a **forge** slice, which competes K candidates in *isolated*
worktrees and lands exactly one — no tree ever has two authors, so the invariant holds (see
[Forge](#forge--competing-candidates-the-deliberate-exception)).

## The contract `/rite-build` sends
One `Task` call to `devrites-slice-wright` carrying everything the writer needs and nothing it
doesn't:

```
Build one slice of the active DevRites feature. You have a clean context — this contract is
the whole job.

Workspace: .devrites/work/<slug>/
Slice: <id — name>
Goal: <one or two sentences>
Acceptance criteria: <the slice's criteria, verbatim>
UI visual acceptance: <if UI — state × viewport × input + target R-id/brief rule, verbatim from tasks.md>
Scope boundary: <what it WILL and will NOT touch>
Mode: <HITL | AFK>   (+ AFK budget note if a cap is set)

Targets (stay inside these — from touched-files.md): <paths>
Interfaces / signatures to match: <if any>
Read yourself: spec.md, plan.md, decisions.md, assumptions.md, rite-polish/reference/anti-ai-slop.md<, design-brief.md if UI>
Rules in scope (.claude/skills/devrites-lib/reference/standards/): coding-style, error-handling, testing, patterns<, security if input/auth/data><, performance if hot path / query / large payload>
Test completeness: write ≥1 asserting test for EVERY interactive element + user flow in this
  slice's test-plan.md interaction inventory, each at the right level (fields/elements →
  unit/component; critical journeys → one E2E; never one-per-field). No element ships
  unverified — testing.md "Completeness". If the slice has no test-plan inventory, derive the
  element/flow list from the slice's own UI surface and cover it the same way.

Apply your documented discipline (orient → RED → implement smallest complete → verify →
return). Frontend slice → build to design-brief.md with devrites-frontend-craft and close
the slice's visual-acceptance deltas before returning. Uncertain
framework fact → verify at the source. Code + tests only — do NOT write the workspace
bookkeeping files; return that data. Return your structured artifact, not your transcript.
```

Rules:
- **One `Task` call, one wright.** Never dispatch two writers on the same slice.
- **No author reasoning beyond the contract.** Give the slice spec, not your analysis of *how*
  to code it — a clean, undirected read is the point.
- **Name the boundary explicitly.** The scope boundary is the single most load-bearing field; an
  underspecified one is the main cause of drift.

## On return — the orchestrator's job (don't delegate these)
**You never edit source here.** The wright is the only writer of code + tests; you write only
`.devrites/` bookkeeping. Every remedy below is **continue the same wright once** or **stop +
escalate** — never an inline patch. You snapshotted the tree before dispatch (`devrites-engine reconcile snapshot`); the reconcile check in step 4 proves no source changed outside the wright's claimed
set.

1. **Doubt the surfaced decisions.** For each entry in the wright's `Decisions stood`, apply
   `devrites-doubt` (→ `devrites-doubt-reviewer`) before accepting — the writer must not grade
   its own decisions. The wright's return is the not-yet-load-bearing moment (slice not `built`,
   not merged), so this post-return doubt is still pre-commit. Irreversible-risk items always
   pause — and an irreversible item that the wright filed under `Decisions stood` instead of
   `Escalation` is a protocol violation: pause and re-dispatch with it flagged out-of-bounds,
   don't doubt-and-accept.
2. **Honor escalations.** A non-empty `Escalation` → do **not** mark the slice built. Write the
   `questions.md` entry + `state.md` `Awaiting human` (blocking gate), or route a scope change
   through `/rite-plan repair` (Spec Drift Guard). You are the canonical writer of these files.
3. **Fail-on-red.** If `Gates` show red (or the wright couldn't verify), the slice is **not
   built** — and you do **not** fix the code. First remedy: **continue the same wright once**
   (`SendMessage`, carrying the failing gate + real output) so it fixes in its own context —
   objective failures only (red gate / type / lint / missing coverage / UI browser-proof fail),
   never a contested decision. Still red after that one retry → blocking question (AFK) or
   blocking gate (HITL); `Next: /rite-plan unblock`.
   An interactive element or user flow in the slice's test-plan interaction inventory left with
   **no asserting test** has the same standing as red: an unverified-element gap blocks the
   slice (don't mark it built). Continue the same wright to cover it, or record a blocker.
4. **Reconcile, then record.** First prove A1 held: write the wright's `Files changed` paths
   (one per line) to `.devrites/work/<slug>/.reconcile-claimed` and run `devrites-engine reconcile check`.
   **Exit 5 → STOP** — a source file changed outside the wright's claimed set (A1 breach); revert
   it and re-dispatch, don't mark the slice built. Then persist the wright's artifact to
   `state.md`, `evidence.md`, `touched-files.md` (and `browser-evidence.md` for UI) per
   [`evidence-standard.md`](evidence-standard.md). Evidence is the wright's real command output,
   not its say-so. Add a concern-ordered `## Review trail` to `touched-files.md` from the wright's changed paths and summary so a human can review by design intent instead of file order. **Persist every `Decisions stood` entry to a `## Decisions stood` section in
   `decisions.md`, one line each ending `— doubt: <accept | reject-resolved | MISSING>`** —
   independent of the doubt step (step 1 above), so a skipped decision still lands on record for
   the seal's doubt-coverage cross-check (`- none` when the wright stood nothing). Then tick AFK if
   `.devrites/AFK` is present (`devrites-engine tick-afk`; exit 3 → STOP).

## Forge — competing candidates (the deliberate exception)

The single-writer rule forbids parallel writers **sharing one tree**. A `Forge: yes` slice
(flagged by `/rite-vet` as a genuine architecture fork at Complexity ≥4) is the one sanctioned
fan-out, and it keeps the rule intact by **isolation**: each candidate wright works in its own
`git worktree`, sees the identical slice contract plus one **distinct strategy**, and never
touches another candidate's tree. A read-only [`devrites-forge-judge`](../../../agents/devrites-forge-judge.md)
then scores the finished candidates against acceptance + `test-plan.md` + `.devrites/principles.md`
+ the anti-slop charter, and the orchestrator lands **exactly one** winner's diff in the working
tree. No tree ever has two authors; exactly one author's work ships. Everything downstream (doubt,
fail-on-red, reconcile against the winner's claimed set, record) runs on the winner as if a single
wright had built it.

Full mechanics — strategy derivation, worktree setup, the judge contract, landing + grafting the
winner, `forge-report.md`, AFK budgeting, and the worktree-unavailable fallback — live in
[`forge.md`](forge.md).

## Fallback
If the `Task` tool / sub-agent dispatch is unavailable, `/rite-build` runs the wright's
discipline **inline** in its own context and flags it as a fallback (no clean-context benefit).
The slice still gets the full one-slice cycle — orient → RED → implement → verify — under the
same anti-slop charter; it just doesn't get the isolation. In this path the orchestrator is
legitimately the writer, so write `.devrites/work/<slug>/.reconcile-inline` before editing — the
reconcile gate (step 4) skips when that sentinel is present. Mirrors the reviewer-dispatch
fallback in
[`../../devrites-lib/reference/parallel-dispatch.md`](../../devrites-lib/reference/parallel-dispatch.md).

## Optional pre-block hook (defense in depth)
`devrites-engine reconcile` is the **post-hoc** gate — it always runs and catches an A1 breach at record time.
A companion **pre-block** hook, `devrites-engine hook a1-guard` (a `PreToolUse` matcher on
`Edit|Write|MultiEdit`), stops the breach *before* the write lands. It is armed only inside the
mid-build window (between `devrites-engine reconcile snapshot` and a clean `check`, keyed on `.reconcile-base`),
allows the wright (subagent calls carry `agent_id`), the inline fallback (`.reconcile-inline`),
and any `.devrites/` write — so it never touches `/rite-polish`, `/rite-quick`, or ordinary
manual edits. It ships **observe-only** (logs would-be blocks to `.a1-guard.log`, never blocks);
flip to enforce with `DEVRITES_A1_HOOK=enforce` or a `.devrites/work/<slug>/.a1-enforce` file once
the log confirms it never flags the wright's own edits (older Claude Code builds may not populate
`agent_id` — the log is the proof before you enforce). The post-hoc gate stands on its own; the
hook is belt-and-suspenders.
