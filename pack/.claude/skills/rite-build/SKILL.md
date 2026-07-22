---
name: rite-build
description: Build one approved vertical slice, then stop with evidence. Use when the next planned slice should be implemented. Not for multiple slices.
argument-hint: "[slice number or name]"
user-invocable: true
---

# /rite-build: one verified slice

Build the next single slice, leave it working and proven, then **stop**. **Read the
active workspace first**; if none, tell the user to run `/rite-spec <feature>`.

This skill is the **orchestrator**: it owns the gates and the workspace; a fresh-context
[`devrites-slice-wright`](../../agents/devrites-slice-wright.md) owns the **writing**. You run
pre-flight (readiness, slice select, HITL pause), dispatch the wright for the build core, then
run the post-return gates (doubt, fail-on-red, record, stop). See
[`reference/wright-dispatch.md`](reference/wright-dispatch.md).

## Rules consulted (read on demand from `.claude/skills/devrites-lib/reference/standards/`)
Read `.claude/skills/devrites-lib/reference/standards/core.md` first (workflow step 0). The
following load on demand: **the wright reads them** (they are named in its contract) while it
writes; read them yourself for the doubt/record gates or in the inline fallback:
- `coding-style.md`: naming, function shape, guard clauses, comments, reuse-first.
- `error-handling.md`: fail fast, no silent catches, fail closed.
- `testing.md`: pyramid, behaviour over implementation, see-it-fail-first.
- `patterns.md`: composition over inheritance, avoid premature abstraction.
- `principles.md`: the project invariants (`.devrites/principles.md`) the slice must honor; the wright reads them as **binding**, not priors.
- `security.md`: when the slice touches user input, auth, data, or external integrations.
- `definition-of-done.md`: standing Done bar: acceptance mapped, fresh proof, no open hard gates, scoped edits, rollback/docs where needed.


## Operating rules
- **One slice at a time. DO NOT** start the next slice without the user asking.
- Evidence over confidence. Prefer existing conventions. Feature scope only: no
  drive-by refactors.
- **Noticed, not touched.** An adjacent smell the wright sees outside `touched-files.md` is
  recorded as an FYI follow-up in `decisions.md`, never fixed inline: the slice's change summary
  states what it *deliberately left alone* ([`git-workflow.md`](../devrites-lib/reference/standards/git-workflow.md) "Things
  I didn't touch"), so the reviewer reads a feature-scoped diff, not a renovation. The `devrites-engine reconcile`
  gate (step 6) enforces this by exit code.
- **Don't re-run an unchanged check.** Re-running the same build/test command on code that hasn't
  changed since proves nothing new. It's motion, not evidence. Re-verify after an edit, not before.
- Surface material assumptions; ask before adding dependencies or a second design
  system. The [Spec Drift Guard](reference/spec-drift-guard.md) is active throughout.
- **Avoid AI slop while writing.** `devrites-slice-wright` enforces the anti-slop charter **at
  the source**: the canonical do-not list is `rite-polish/reference/anti-ai-slop.md` (the
  wright reads it; don't restate it here). It writes the code the *project* would write, in its
  idiom, reusing before building; **you verify the charter held on return**. You do not re-list
  it and you do not fix slop by editing source. Polish catches what slips; build prevents.
  The **prose you write yourself** (`evidence.md`, `decisions.md`, the slice report) follows
  the human-voice charter (`.claude/skills/devrites-lib/reference/standards/prose-style.md`; depth in `devrites-prose-craft`): no
  filler openers, no marketing adjectives, exact commands and identifiers kept verbatim.
- **Honor declared project principles.** The wright reads `.devrites/principles.md` and treats
  each invariant as **binding** (not a prior to weigh like a convention): a slice it cannot build
  without breaking one is an **Escalation**, not a silent violation. On return **you verify no
  principle was broken**; a fresh violation is handled like any irreversible-risk item: a
  human-approved, scoped exception in the register or a stop, never folded into the slice. No
  `.devrites/principles.md` → none declared → nothing to honor.
- **You never edit source: the wright is the only writer of code + tests.** You write only
  `.devrites/` bookkeeping. On any red gate, doubt finding, or coverage gap your only remedies
  are **continue the same wright once** (it fixes in its own context) or **stop + escalate**:
  never patch the code yourself. The `devrites-engine reconcile` gate (step 6) enforces this by exit code:
  any source file changed outside the wright's claimed set is a hard STOP.
- **A `Forge: yes` slice competes candidates: one author still lands.** When `/rite-vet`
  flagged the slice a genuine architecture fork, step 3 runs K=2-3 candidate wrights in
  **isolated worktrees** and lands exactly one winner's diff; the single-writer invariant holds
  because no tree ever has two authors and only the winner reaches the working tree. You still
  never edit source, and reconcile runs against the winner's claimed set. The default slice is
  single-path: forge is the rare exception ([`reference/forge.md`](reference/forge.md)).

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
