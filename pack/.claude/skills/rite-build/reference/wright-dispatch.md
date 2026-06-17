# Slice-wright dispatch

How `/rite-build` hands the **build core** of one slice to `devrites-slice-wright` — the
fresh-context, **write-capable** executor under `.claude/agents/`. Loaded on demand by
`/rite-build`; not a skill itself. Sibling of
[`../../rite-seal/reference/parallel-dispatch.md`](../../rite-seal/reference/parallel-dispatch.md)
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
by design: **one** wright per slice, never a parallel fan-out of writers — concurrent writers
make conflicting implicit decisions and produce incoherent code.

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
Scope boundary: <what it WILL and will NOT touch>
Mode: <HITL | AFK>   (+ AFK budget note if a cap is set)

Targets (stay inside these — from touched-files.md): <paths>
Interfaces / signatures to match: <if any>
Read yourself: spec.md, plan.md, decisions.md, assumptions.md, rite-polish/reference/anti-ai-slop.md<, design-brief.md if UI>
Rules in scope (.claude/rules/): coding-style, error-handling, testing, patterns<, security if input/auth/data><, performance if hot path / query / large payload>

Apply your documented discipline (orient → RED → implement smallest complete → verify →
return). Frontend slice → build to design-brief.md with devrites-frontend-craft. Uncertain
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
   built** — blocking question (AFK) or blocking gate (HITL); `Next: /rite-plan unblock`.
4. **Record.** Persist the wright's artifact to `state.md`, `evidence.md`, `touched-files.md`
   (and `browser-evidence.md` for UI) per [`evidence-standard.md`](evidence-standard.md).
   Evidence is the wright's real command output, not its say-so. Then tick AFK if `.devrites/AFK`
   is present (`tick-afk.sh`; exit 3 → STOP).

## Fallback
If the `Task` tool / sub-agent dispatch is unavailable, `/rite-build` runs the wright's
discipline **inline** in its own context and flags it as a fallback (no clean-context benefit).
The slice still gets the full one-slice cycle — orient → RED → implement → verify — under the
same anti-slop charter; it just doesn't get the isolation. Mirrors the reviewer-dispatch
fallback in
[`../../rite-seal/reference/parallel-dispatch.md`](../../rite-seal/reference/parallel-dispatch.md).
