---
name: rite-quick
description: Express lane for a SMALL, reversible change. Use when the user says "quick fix" or "just do X", or for a typo / copy / config / single-function / one-file change that is low-risk and unambiguous. Escalates to `$rite-spec` (full lifecycle) the moment scope, risk, or ambiguity grows. Not for auth / data-model / migration / public-API / multi-slice / ambiguous work — those need the full lifecycle.
argument-hint: "<what to change>"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.


# $rite-quick — express lane for small changes

The full DevRites lifecycle is right for real features; it is **ceremony** for a typo, a
copy tweak, a one-function fix, or a small config change. `$rite-quick` keeps the
discipline that matters (idiom, TDD, evidence, escape-to-full-lifecycle) and drops the
artifacts that don't, in a **single pass**. Senior-engineer "right step, right time" made
executable, not advisory.

## The significance gate (the whole safety story)

**Run this first. If ANY of these holds, STOP and route to `$rite-spec` — do NOT use the
express lane:**
- Touches **auth / authz**, a **data migration**, a **public API contract**, or any
  **destructive / data-loss** path (the irreversible-risk list — `standards/afk-hitl.md`).
- Spans **more than one vertical slice**, or more than a couple of files of real logic.
- **Ambiguous scope** — you'd have to guess what "done" means, or the ask hides a design
  decision (data model, new dependency, second design system).
- Security-sensitive input handling, or a measurable performance-critical path.
- Would **break a declared project principle** (`.devrites/principles.md`) with no recorded,
  human-approved exception — the express lane never relaxes a project gate; a needed exception is
  a deliberate human decision, so route it to `$rite-spec`.

If none hold, the change is small + reversible + unambiguous → proceed. **When in doubt,
escalate** — the cost of the full lifecycle on a small change is minutes; the cost of the
express lane on a risky one is the failure mode the lifecycle exists to prevent.

## Rules consulted
Read `.agents/skills/devrites-lib/reference/standards/core.md` first. Then the small set this lane actually needs:
- `coding-style.md` — naming, guard clauses, reuse-first.
- `testing.md` — TDD, **completeness** (every touched behavior/element asserted) +
  **assertion strength** (no tautological tests; see it fail first), scaled to the change.
- `error-handling.md` / `security.md` — only if the change touches input/errors.
- `principles.md` — when `.devrites/principles.md` exists; a change that breaks an invariant is a gate, not a quick fix.
- `definition-of-done.md` — standing Done bar: acceptance mapped, fresh proof, no open hard gates, scoped edits, rollback/docs where needed.


## Workflow
0. **Orient.** Read `core.md`. If a `.devrites/` workspace is active, run the preamble to
   see it; `$rite-quick` does **not** require one and does **not** create the full tree.
1. **Significance gate** (above). Fail → STOP, tell the user to run `$rite-spec <feature>`.
2. **One-line contract.** Restate in 1–3 lines: the change, its **acceptance** (how you'll
   know it works), and the **scope boundary** (what you will NOT touch). This is the entire
   "spec + plan" for a quick change — keep it in the chat; optionally drop a `brief.md` +
   `evidence.md` under `.devrites/work/<slug>/` if the user wants a record. This is the
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
5. **Review-lite + ship.** Self-review the diff (correctness, scope, idiom, and — if
   `.devrites/principles.md` exists — no declared invariant broken; this is `$rite-frame`'s AUDIT
   pass — one pass, no subagent fan-out). Show the diff, then on the user's confirm commit it
   (Conventional Commits, atomic) — or hand to `$rite-ship` if a workspace is active. **Never push
   without the user asking.**

## Escalation (mid-flight) — the Spec Drift Guard still applies
If the "small" change turns out to be not small — a second slice appears, a real design
decision surfaces, scope grows past the boundary, or you hit an irreversible-risk item —
**STOP, say so, and route to `$rite-spec` / `$rite-define`**. Don't quietly grow a quick
fix into an unreviewed feature; that's the exact drift the lifecycle guards against.

## Output
Run `devrites-engine progress` when a workspace is active; otherwise skip it. Then use the
shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: quick change complete — <one line>.
Changed: <files touched>
Evidence: tests <cmd -> pass>; assertion check <real asserts saw red | n/a>; boundary held <yes|no>
Open: <none | escalation reason | tracked workspace needs ship>
Next: <single recommended command>
Record: <commit/PR path | .devrites/work/<slug>/evidence.md | not applicable>
↻ Hygiene: /clear after commit
```

**DO NOT** use this lane to dodge the gate — the express lane is for changes that are
*actually* small, not for shipping risky work faster.
