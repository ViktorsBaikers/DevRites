---
name: rite-build
description: Build one approved vertical slice, then stop with evidence. Use when the next planned slice should be implemented. Not for multiple slices.
argument-hint: "[slice number or name]"
user-invocable: true
required-agent-roles: devrites-slice-wright
---

# /rite-build: one verified slice

Build and prove one slice, then **stop**. **Read the active workspace first**; if none,
tell the user to run `/rite-spec <feature>`.

This skill owns the gates and workspace. A fresh-context
[`devrites-slice-wright`](../../agents/devrites-slice-wright.md) writes source and tests.
Run the readiness, slice-selection, and HITL checks; dispatch the wright; then run the
doubt, fail-on-red, recording, and stop checks. See
[`reference/wright-dispatch.md`](reference/wright-dispatch.md).

## Rules consulted (read on demand from `.claude/skills/devrites-lib/reference/standards/`)
Read `.claude/skills/devrites-lib/reference/standards/core.md` first (workflow step 0). The
following load on demand: **the wright reads them** (they are named in its contract) while it
writes; read them yourself for the doubt/record gates:
- `coding-style.md`: naming, function shape, guard clauses, comments, reuse-first.
- `error-handling.md`: fail fast, no silent catches, fail closed.
- `testing.md`: pyramid, behaviour over implementation, see-it-fail-first.
- [`reference/tdd.md`](reference/tdd.md): the slice-level Red → Green → Refactor
  and Prove-It contract.
- `patterns.md`: composition over inheritance, avoid premature abstraction.
- `principles.md`: the project invariants (`.devrites/principles.md`) the slice must honor; the wright reads them as **binding**, not priors.
- `security.md`: when the slice touches user input, auth, data, or external integrations.
- `definition-of-done.md`: standing Done bar: acceptance mapped, fresh proof, no open hard gates, scoped edits, rollback/docs where needed.


## Operating rules
- **One slice at a time. DO NOT** start the next slice without the user asking.
- Evidence over confidence. Prefer existing conventions. Feature scope only: no
  drive-by refactors.
- **Record adjacent issues; do not edit them.** An issue outside the captured
  `.wright-allowlist` becomes an FYI follow-up in `decisions.md`. The slice summary states
  what it deliberately left alone ([`git-workflow.md`](../devrites-lib/reference/standards/git-workflow.md)
  "Things I didn't touch"). The `devrites-engine reconcile` gate enforces this boundary.
- **Don't re-run an unchanged check.** The same build or test on unchanged code provides
  no new evidence. Re-verify after an edit.
- Surface material assumptions. Do not introduce an unplanned dependency or second design
  system: route the objective plan gap to `/rite-vet` (or bounded recovery), not to a human.
  Ask only if the newly exposed choice changes licensing/cost/security, product behavior,
  or an explicit architecture policy. The
  [Spec Drift Guard](reference/spec-drift-guard.md) is active throughout.
- **Avoid AI slop while writing.** `devrites-slice-wright` applies the anti-slop charter
  while writing. The canonical list is `rite-polish/reference/anti-ai-slop.md`; do not
  duplicate it here. The wright follows project idioms and reuses existing code first.
  **Verify the charter on return.** Do not correct source from the orchestrator.
  The **prose you write yourself** (`evidence.md`, `decisions.md`, the slice report) follows
  the human-voice charter (`.claude/skills/devrites-lib/reference/standards/prose-style.md`; depth in `devrites-prose-craft`): no
  filler openers or marketing adjectives; preserve exact commands and identifiers.
- **Honor declared project principles.** The wright reads `.devrites/principles.md` and treats
  each invariant as **binding** (not a prior to weigh like a convention): a slice it cannot build
  without breaking one is an **Escalation**, not a silent violation. On return **you verify no
  principle was broken**; a fresh violation is handled like any irreversible-risk item: a
  human-approved, scoped exception in the register or a stop, never folded into the slice. No
  `.devrites/principles.md` → none declared → nothing to honor.
- **You never edit source: the wright is the only writer of code + tests.** You write only
  `.devrites/` bookkeeping. On any red gate, doubt finding, or coverage gap your only remedies
  are to continue the same wright under bounded `devrites-debug-recovery` (three total attempts
  per root cause) or stop with the correctly classified blocker: never patch the code yourself
  and never ask the human to authorize agent-owned repair work. The `devrites-engine reconcile`
  gate enforces this against the root-owned pre-dispatch allowlist: any source file changed
  outside that exact set is a hard STOP.
- **Forge is manifest-owned, winner-takes-all.** For a fully typed `Forge: yes`
  slice, follow [`reference/forge.md`](reference/forge.md): `forge plan` precedes
  reconciliation; every candidate is bound to its isolated worktree and live worker;
  the judge records one winner; normal reconciliation and proof pass before manifest-only
  cleanup and the human report. Any stale/ineligible flag uses one serial wright before
  Forge creates state.

## Workflow

Run the full execution contract in
[`reference/phase-contract.md`](reference/phase-contract.md). It is not optional:
it contains the gated one-slice workflow, including readiness, HITL/AFK handling,
wright dispatch, forge, doubt, fail-on-red, record gates, and stop behavior.

## Output

Use the full output contract in [`reference/output.md`](reference/output.md).
It preserves the progress-footer-first response shape, uses the shared completion
reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)),
and keeps the explicit stop after one slice.
