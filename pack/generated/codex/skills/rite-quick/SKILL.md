---
name: rite-quick
description: Express lane for a small reversible change: typo/copy/config, rate-limit constant, error message, local variable rename, feature flag, README link, one-file/function fix.
argument-hint: "<what to change>"
user-invocable: true
---

# $rite-quick: express lane for small changes

The full DevRites lifecycle is right for real features; it is **ceremony** for a typo, a
copy tweak, a one-function fix, or a small config change. `$rite-quick` keeps the
discipline that matters (idiom, TDD, evidence, escape-to-full-lifecycle) and drops the
artifacts that don't, in a **single pass**. Senior-engineer "right step, right time" made
executable, not advisory. This lane uses the **Quick** depth profile from
[`devrites-lib/reference/orchestration-profiles.md`](../devrites-lib/reference/orchestration-profiles.md).

## The significance gate (the whole safety story)

**Run this first. If ANY of these holds, STOP and route to `$rite-spec`. Do NOT use the
express lane:**
- Touches **auth / authz**, a **data migration**, a **public API contract**, or any
  **destructive / data-loss** path in the irreversible-risk list. See `../devrites-lib/reference/standards/afk-hitl.md`.
- Spans **more than one vertical slice**, or more than a couple of files of real logic.
- **Ambiguous scope:** you'd have to guess what "done" means, or the ask hides a design
  decision (data model, new dependency, second design system).
- Security-sensitive input handling, or a measurable performance-critical path.
- Would **break a declared project principle** (`.devrites/principles.md`) with no recorded,
  human-approved exception: the express lane never relaxes a project gate; a needed exception is
  a deliberate human decision, so route it to `$rite-spec`.

If none hold, the change is small + reversible + unambiguous → proceed. **When in doubt,
escalate**: the cost of the full lifecycle on a small change is minutes; the cost of the
express lane on a risky one is the failure mode the lifecycle exists to prevent.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
Load this lane's conditional standards when needed:
- `coding-style.md`: naming, guard clauses, reuse-first.
- `testing.md`: TDD, **completeness** (every touched behavior/element asserted) +
  **assertion strength** (no tautological tests; see it fail first), scaled to the change.
- `error-handling.md` / `security.md`: only if the change touches input/errors.
- `principles.md`: when `.devrites/principles.md` exists; a change that breaks an invariant is a gate, not a quick fix.
- `definition-of-done.md`: standing Done bar: acceptance mapped, fresh proof, no open hard gates, scoped edits, rollback/docs where needed.


## Workflow
0. **Orient.** Read `core.md`. If a `.devrites/` workspace is active, read its
   `state.md` directly; `$rite-quick` does **not** require
   a workspace and does **not** create the full tree.
1. **Significance gate** (above). Fail → STOP, tell the user to run `$rite-spec <feature>`.
2. **One-line contract.** Restate in 1-3 lines: the change, its **acceptance** (how you'll
   know it works), and the **scope boundary** (what you will NOT touch). This is the entire
   "spec + plan" for a quick change: keep it in the chat; optionally drop a `brief.md` +
   `evidence.md` under an unused `.devrites/work/<slug>/` (no `state.md`; not a workspace
   cursor) if the user wants a record. This is the
   `$rite-frame` FRAME move: if the acceptance won't reduce to a check that could be *false*,
   the ask is ambiguous → escalate per the significance gate. Its AUDIT pass is the diff
   self-review in step 5.
3. **Build with TDD.** Failing test first when behavior changes (see it fail for the right
   reason) → smallest complete change in the **project's idiom** (reuse before you write,
   anti-AI-slop) → cover every touched behavior / interactive element with a **real**
   assertion. UI → check the states + a browser glance.
4. **Prove (scoped).** Run the **targeted** tests + typecheck / lint for what changed (not
   the whole suite) → green. Record the command + output. A tautological test that can't
   fail is not proof.
5. **Review-lite + ship.** Self-review the diff for correctness, scope, and idiom. If
   `.devrites/principles.md` exists, confirm that no declared invariant is broken. This is
   `$rite-frame`'s AUDIT pass: one pass, no subagent fan-out. Show the diff, then on the user's confirm commit it
   (Conventional Commits, atomic), or hand to `$rite-ship` only when the change belongs to the active
   feature's candidate; otherwise commit standalone. **Never push
   without the user asking.**

## Escalation (mid-flight): the Spec Drift Guard still applies
If the "small" change turns out to be not small: a second slice appears, a real design
decision surfaces, scope grows past the boundary, or you hit an irreversible-risk item:
**STOP, say so, and route to `$rite-spec` / `$rite-define`**. Don't quietly grow a quick
fix into an unreviewed feature; that's the exact drift the lifecycle guards against.

If the quick boundary does not hold or an escalation remains, use `Stopped / blocked`
and route to the full lifecycle; do not render `Done`.

**DO NOT** use this lane to dodge the gate: the express lane is for changes that are
genuinely small, not for shipping risky work faster.
