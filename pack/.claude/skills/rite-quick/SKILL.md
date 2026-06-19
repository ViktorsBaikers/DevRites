---
name: rite-quick
description: Express lane for a SMALL, reversible change — one short artifact, then build → prove → ship, collapsing the full spec → define → vet → build → prove → review → seal lifecycle into one pass. Use when the user says "quick fix", "small change", "tiny tweak", "just do X", or for a typo / copy / config / single-function / one-file change that is low-risk and unambiguous. Escalates to `/rite-spec` (full lifecycle) the moment scope, risk, or ambiguity grows. Not for auth / data-model / migration / public-API / multi-slice / ambiguous work — those need the full lifecycle.
argument-hint: "<what to change>"
user-invocable: true
---

# /rite-quick — express lane for small changes

The full DevRites lifecycle is right for real features; it is **ceremony** for a typo, a
copy tweak, a one-function fix, or a small config change. `/rite-quick` keeps the
discipline that matters (idiom, TDD, evidence, escape-to-full-lifecycle) and drops the
artifacts that don't, in a **single pass**. Senior-engineer "right step, right time" made
executable, not advisory.

## The significance gate (the whole safety story)

**Run this first. If ANY of these holds, STOP and route to `/rite-spec` — do NOT use the
express lane:**
- Touches **auth / authz**, a **data migration**, a **public API contract**, or any
  **destructive / data-loss** path (the irreversible-risk list — `rules/afk-hitl.md`).
- Spans **more than one vertical slice**, or more than a couple of files of real logic.
- **Ambiguous scope** — you'd have to guess what "done" means, or the ask hides a design
  decision (data model, new dependency, second design system).
- Security-sensitive input handling, or a measurable performance-critical path.

If none hold, the change is small + reversible + unambiguous → proceed. **When in doubt,
escalate** — the cost of the full lifecycle on a small change is minutes; the cost of the
express lane on a risky one is the failure mode the lifecycle exists to prevent.

## Rules consulted
Read `.claude/rules/core.md` first. Then the small set this lane actually needs:
- `coding-style.md` — naming, guard clauses, reuse-first.
- `testing.md` — TDD, **completeness** (every touched behavior/element asserted) +
  **assertion strength** (no tautological tests; see it fail first), scaled to the change.
- `error-handling.md` / `security.md` — only if the change touches input/errors.

## Workflow
0. **Orient.** Read `core.md`. If a `.devrites/` workspace is active, run the preamble to
   see it; `/rite-quick` does **not** require one and does **not** create the full tree.
1. **Significance gate** (above). Fail → STOP, tell the user to run `/rite-spec <feature>`.
2. **One-line contract.** Restate in 1–3 lines: the change, its **acceptance** (how you'll
   know it works), and the **scope boundary** (what you will NOT touch). This is the entire
   "spec + plan" for a quick change — keep it in the chat; optionally drop a `brief.md` +
   `evidence.md` under `.devrites/work/<slug>/` if the user wants a record.
3. **Build with TDD.** Failing test first when behavior changes (see it fail for the right
   reason) → smallest complete change in the **project's idiom** (reuse before you write,
   anti-AI-slop) → cover every touched behavior / interactive element with a **real**
   assertion. UI → check the states + a browser glance.
4. **Prove (scoped).** Run the **targeted** tests + typecheck / lint for what changed (not
   the whole suite) → green. Record the command + output. A tautological test that can't
   fail is not proof.
5. **Review-lite + ship.** Self-review the diff (correctness, scope, idiom — one pass, no
   subagent fan-out). Show the diff, then on the user's confirm commit it (Conventional
   Commits, atomic) — or hand to `/rite-ship` if a workspace is active. **Never push without
   the user asking.**

## Escalation (mid-flight) — the Spec Drift Guard still applies
If the "small" change turns out to be not small — a second slice appears, a real design
decision surfaces, scope grows past the boundary, or you hit an irreversible-risk item —
**STOP, say so, and route to `/rite-spec` / `/rite-define`**. Don't quietly grow a quick
fix into an unreviewed feature; that's the exact drift the lifecycle guards against.

## Output
**Footer first** — render the flow ribbon by running the progress footer (`progress.sh`,
resolved like the step-0 preamble — canonical snippet in `devrites-lib/SKILL.md`) when a
workspace is active; otherwise skip it. Then:
```
Quick change: <one line>
Scope: <files touched>   (boundary held? yes)
Tests: <cmd → pass>   ·   Assertion check: <real asserts, saw red | n/a>
Diff: <summary>
Next  ▸ commit (Conventional) on your confirm  ·  or /rite-ship if tracked
↻ Hygiene: /clear after commit — the change is small and self-contained.
```

**DO NOT** use this lane to dodge the gate — the express lane is for changes that are
*actually* small, not for shipping risky work faster.
