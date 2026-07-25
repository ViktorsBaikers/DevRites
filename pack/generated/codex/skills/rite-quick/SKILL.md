---
name: rite-quick
description: Express lane for a small reversible change: typo/copy/config, rate-limit constant, error message, local variable rename, feature flag, README link, one-file/function fix.
argument-hint: "<what to change>"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- Inspect the current `spawn_agent` role list. When the named `devrites-<role>` is exposed, dispatch it with `fork_turns="none"`; full-history forks inherit the parent type. The matching project contract is `.codex/agents/devrites-<role>.toml`.
- If a named role is not exposed, use generic `explorer` for every read-only role with `fork_turns="none"`. Tell it to read `.codex/agents/devrites-<role>.toml`, follow its `developer_instructions`, and execute the unchanged packet. Trusted `.codex/hooks.json` binds `agent_type=explorer` to the fail-closed reviewer read-only guard.
- For `devrites-slice-wright`, trusted `.codex/hooks.json` binds generic `worker` (`agent_type=worker`) to the active reconcile window and exact `.wright-allowlist`. Dispatch that worker with `fork_turns="none"`, tell it to read `.codex/agents/devrites-slice-wright.toml`, and execute the unchanged packet. Never create `.reconcile-inline` when this safe rung is available.
- A missing custom role is not evidence that spawning is unavailable. Only when the project hooks are unavailable or untrusted, no spawn primitive exists, or higher-priority policy rejects a safe spawn may the root run the documented discipline inline. Label it `independence: fallback`, never call it independent, create `.reconcile-inline` only for that path, and apply every fallback risk gate.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-quick: express lane for small changes

The full DevRites lifecycle is right for real features; it is **ceremony** for a typo, a
copy tweak, a one-function fix, or a small config change. `$rite-quick` keeps the
discipline that matters (idiom, TDD, evidence, escape-to-full-lifecycle) and drops the
artifacts that don't, in a **single pass**. Senior-engineer "right step, right time" made
executable, not advisory.

## The significance gate (the whole safety story)

**Run this first. If ANY of these holds, STOP and route to `$rite-spec`. Do NOT use the
express lane:**
- Touches **auth / authz**, a **data migration**, a **public API contract**, or any
  **destructive / data-loss** path in the irreversible-risk list. See `standards/afk-hitl.md`.
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

## Rules consulted
Load this lane's conditional standards when needed:
- `coding-style.md`: naming, guard clauses, reuse-first.
- `testing.md`: TDD, **completeness** (every touched behavior/element asserted) +
  **assertion strength** (no tautological tests; see it fail first), scaled to the change.
- `error-handling.md` / `security.md`: only if the change touches input/errors.
- `principles.md`: when `.devrites/principles.md` exists; a change that breaks an invariant is a gate, not a quick fix.
- `definition-of-done.md`: standing Done bar: acceptance mapped, fresh proof, no open hard gates, scoped edits, rollback/docs where needed.


## Workflow
0. **Orient.** Read `core.md`. If a `.devrites/` workspace is active, run the preamble to
   see it; `$rite-quick` does **not** require one and does **not** create the full tree.
1. **Significance gate** (above). Fail → STOP, tell the user to run `$rite-spec <feature>`.
2. **One-line contract.** Restate in 1-3 lines: the change, its **acceptance** (how you'll
   know it works), and the **scope boundary** (what you will NOT touch). This is the entire
   "spec + plan" for a quick change: keep it in the chat; optionally drop a `brief.md` +
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
5. **Review-lite + ship.** Self-review the diff for correctness, scope, and idiom. If
   `.devrites/principles.md` exists, confirm that no declared invariant is broken. This is
   `$rite-frame`'s AUDIT pass: one pass, no subagent fan-out. Show the diff, then on the user's confirm commit it
   (Conventional Commits, atomic), or hand to `$rite-ship` if a workspace is active. **Never push
   without the user asking.**

## Escalation (mid-flight): the Spec Drift Guard still applies
If the "small" change turns out to be not small: a second slice appears, a real design
decision surfaces, scope grows past the boundary, or you hit an irreversible-risk item:
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
Evidence: tests <cmd -> pass>; assertion check <real asserts saw red | n/a>; boundary held yes
Open: <none | non-blocking follow-up | tracked workspace needs ship>
Next: <single recommended command>
Record: <commit/PR path | .devrites/work/<slug>/evidence.md | not applicable>
↻ Hygiene: /clear after commit
```

If the quick boundary does not hold or an escalation remains, use `Stopped / blocked`
and route to the full lifecycle; do not render `Done`.

**DO NOT** use this lane to dodge the gate: the express lane is for changes that are
genuinely small, not for shipping risky work faster.
