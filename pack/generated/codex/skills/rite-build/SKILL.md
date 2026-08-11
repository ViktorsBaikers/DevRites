---
name: rite-build
description: Build the next approved vertical slice with evidence. HITL stops after one; explicit AFK may chain bounded green slices.
argument-hint: "[slice number or name]"
user-invocable: true
---

# $rite-build: one verified slice

Build and prove one slice. HITL stops; a later user invocation starts the next.
Explicit `.devrites/AFK` alone lets the controlling root chain pending slices,
only under existing green-proof, cap, and pause rules. **Read the active workspace
first**; if none, tell the user to run `$rite-spec <feature>`.

The root owns gates/workspace; a
fresh-context [`devrites-slice-wright`](.codex/agents/devrites-slice-wright.toml)
writes product source and tests. Exact Vet-ready executable workflow artifacts use
[`workflow-artifacts.md`](../devrites-lib/reference/standards/workflow-artifacts.md).
Apply the readiness, selection, host, HITL/AFK,
dispatch, doubt, fail-on-red, record, and stop checks. See
[`reference/wright-dispatch.md`](reference/wright-dispatch.md).

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
Read `.agents/skills/devrites-lib/reference/standards/core.md` first. The wright
reads named on-demand rules while writing; root reads them for doubt/record gates:
- `coding-style.md`: naming, function shape, guard clauses, comments, reuse-first.
- `error-handling.md`: fail fast, no silent catches, fail closed.
- `testing.md`: pyramid, behaviour over implementation, see-it-fail-first.
- [`reference/tdd.md`](reference/tdd.md): the slice-level Red → Green → Refactor
  and Prove-It contract.
- `patterns.md`: composition over inheritance, avoid premature abstraction.
- `principles.md`: the project invariants (`.devrites/principles.md`) the slice must honor; the wright reads them as **binding**, not priors.
- `security.md`: when the slice touches user input, auth, data, or external integrations.
- `repository-topology.md`: when targets span roots/languages/services or touch generated/vendor surfaces.
- `data-integrity.md`: when the slice writes durable state or changes migration/concurrency/tenant/retention behavior.
- `integration-reliability.md`: when the slice changes an API/webhook/queue/job/cache/cross-service boundary.
- `definition-of-done.md`: standing Done bar: acceptance mapped, fresh proof, no open hard gates, scoped edits, rollback/docs where needed.


## Operating rules
- **One slice per wright dispatch.** Every wright returns after it. The controlling
  root applies the mode rule above before another dispatch.
- Evidence over confidence. Prefer existing conventions. Feature scope only: no
  drive-by refactors.
- **Record adjacent issues; do not edit them.** An issue outside the exact paths
  stated in the dispatch task becomes an FYI follow-up in `decisions.md`. The slice summary states
  what it deliberately left alone ([`git-workflow.md`](../devrites-lib/reference/standards/git-workflow.md)
  "Things I didn't touch"). The root rejects a returned diff outside this boundary.
- **Don't re-run an unchanged check.** The same build or test on unchanged code provides
  no new evidence. Re-verify after an edit.
- Surface material assumptions. Do not introduce an unplanned dependency or second design
  system: route the objective plan gap to `$rite-vet` (or bounded recovery), not to a human.
  Ask only if the newly exposed choice changes licensing/cost/security, product behavior,
  or an explicit architecture policy. The
  [Spec Drift Guard](reference/spec-drift-guard.md) is active throughout.
- **Avoid AI slop while writing.** `devrites-slice-wright` applies the anti-slop charter
  while writing. The canonical list is `rite-polish/reference/anti-ai-slop.md`; do not
  duplicate it here. The wright follows project idioms and reuses existing code first.
  **Verify the charter on return.** Do not correct source from the orchestrator.
  The **prose you write yourself** (`evidence.md`, `decisions.md`, the slice report) follows
  the human-voice charter (`.agents/skills/devrites-lib/reference/standards/prose-style.md`; depth in `devrites-prose-craft`): no
  filler openers or marketing adjectives; preserve exact commands and identifiers.
- **Honor declared project principles.** The wright reads `.devrites/principles.md` and treats
  each invariant as **binding** (not a prior to weigh like a convention): a slice it cannot build
  without breaking one is an **Escalation**, not a silent violation. On return **you verify no
  principle was broken**; a fresh violation is handled like any irreversible-risk item: a
  human-approved, scoped exception in the register or a stop, never folded into the slice. No
  `.devrites/principles.md` → none declared → nothing to honor.
- **You never edit product source/tests.** You write `.devrites/` bookkeeping and,
  only through `workflow-artifacts.md`, an exact Vet-ready executable workflow-artifact set.
  Follow the
  host gate in `wright-dispatch.md` before every build or recovery dispatch. On a
  supported host, the wright is the only writer of code and tests. Codex gives
  the root workspace permission only so that native writer dispatch can execute;
  never patch code from the root. Put the exact project-relative source/test path
  list directly in the task, dispatch the exact `devrites-slice-wright`, then
  compare `git diff --name-only` with those task paths. Any extra source file is
  a hard STOP.
- **Executable workflow-artifact branch.** If every implementation target is an
  exact admitted path under the active `.devrites/work/<slug>/`, the controlling
  root materializes and proves that atomic set itself. It does not dispatch the wright,
  decrement/count a product slice, update the candidate manifest, or execute the
  consumptive action. After narrow Vet, restore the caller cursor.

## Workflow

Run the full execution contract in
[`reference/phase-contract.md`](reference/phase-contract.md). It is not optional:
it contains the gated one-slice workflow, including readiness, HITL/AFK handling,
wright dispatch, doubt, fail-on-red, record gates, and stop behavior.

## Output

Use [`reference/output.md`](reference/output.md) and the shared
[`reply contract`](../devrites-lib/reference/reply-contract.md). They keep the
HITL stop, bounded AFK chaining, and no-automatic-`$rite-prove` boundary explicit.
