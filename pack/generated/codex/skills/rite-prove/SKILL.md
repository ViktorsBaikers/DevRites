---
name: rite-prove
description: Prove a completed feature with tests and the full test suite, build/typecheck/lint, end-to-end/browser evidence, screenshots, commands, and outputs for seal. Not for single-slice proof.
argument-hint: "[feature-slug]"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers — NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-prove — prove the completed feature

Turn "I think it works" into recorded evidence for the **whole feature**. Read the active
workspace first; if none, run `$rite-spec <feature>`.

> **Differs from built-in `/verify` and `/run`:** those prove a single change /
> launch the app. `$rite-prove` is feature-scoped — it walks `spec.md` acceptance
> criteria one-by-one, runs the full relevant test suite + build/typecheck/lint,
> ascends the browser-proof ladder (step 4),
> and writes `evidence.md` + `browser-evidence.md` keyed to the active
> `.devrites/work/<slug>/`. No DevRites workspace → use `/verify` or `/run` alone.

## Gate: all slices must be built first
Read `tasks.md` + `state.md`. **If ANY slice is still pending/unbuilt, STOP** and tell the
user to finish it with `$rite-build` — `$rite-prove` runs once, when the full task is
complete, not after each slice. (Each slice already got its own targeted tests during
`$rite-build`; this phase is the comprehensive proof of the assembled feature.)

**Never report a pass you didn't observe.** If a command couldn't run, say so and give exact manual steps.

**Re-runnable, scoped.** `$rite-prove` runs once when the full feature is assembled, but
it can be **re-run scoped** afterwards: when `$rite-polish` or `$rite-review` edit code,
the existing `evidence.md` no longer post-dates the change, so re-run `$rite-prove` over
the affected criteria/routes to refresh proof before `$rite-seal`.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
Pull these via `Read` when relevant:
- `testing.md` — pyramid, determinism, no-flake discipline.
- `test-proof-checklist.md` — compact proof-quality gate for tests and recorded evidence.
- `browser-proof-checklist.md` — for UI scope, the required browser states and Visual Verdict evidence.
- `spec-grammar.md` — when the spec uses structured `### Requirement:` / `#### Scenario:`
  blocks, each scenario (WHEN/THEN) is one observable behavior to walk and prove.
- `performance.md` — measure first when perf is in scope.
- `observability.md` — when the change has a runtime surface (endpoint, job, integration,
  user flow): telemetry must be present **and observed to emit**, not assumed.
- `developer-experience.md` — when the change ships a developer-facing surface (API / CLI / SDK /
  webhook / config / error messages / getting-started): **measure** the DX scorecard (run the flow,
  time time-to-hello-world, capture the verbatim error text), don't assert it.
- `definition-of-done.md` — standing Done bar: acceptance mapped, fresh proof, no open hard gates, scoped edits, rollback/docs where needed.


## Operating rules
- Evidence over confidence. Feature scope only — fix within the feature or record a
  blocker; don't refactor unrelated code.
- Spec Drift Guard applies: if tests/evidence reveal the spec is wrong, stop and handle
  drift (`rite-build/reference/spec-drift-guard.md`).

## Workflow
0. Read `.agents/skills/devrites-lib/reference/standards/core.md` first (the always-on operating rules); pull the
   on-demand rules above when relevant.
   Then run `devrites-engine preamble` for deterministic workspace orientation.
1. **Confirm the gate** (all slices built). Read `spec.md` (acceptance criteria +
   "Commands discovered"), `tasks.md`, `state.md`, `test-plan.md` if present (the vetted
   coverage target from `$rite-vet`), and the full `git diff`.
2. **Discover commands** if not recorded —
   [test-command-discovery](reference/test-command-discovery.md): README, package
   scripts, Makefile, CI configs, Gemfile/Rakefile, pyproject, go.mod, Cargo.toml.
   **Completion:** exact runnable test/build/typecheck/lint commands are recorded or explicitly unavailable.
3. **Run the full relevant test suite** for the feature (not a single slice), then the
   relevant **build / typecheck / lint**.
4. **UI feature?** Read `design-brief.md` + `references.md`, then run the browser proof ladder over the feature's routes —
   [proof-ladder](reference/proof-ladder.md) + [browser-proof](reference/browser-proof.md)
   (`devrites-browser-proof`): routes, viewports, screenshots (opened + described),
   console, network, interaction paths, and the brief's proof targets. Compare screenshots
   with target references, record deltas, fix/re-render, and do not pass with an unresolved
   material mismatch.
5. **Map proof completely.** Follow
   [`reference/acceptance-proof.md`](reference/acceptance-proof.md) for acceptance/scenario
   coverage and the conditional critical-path, observability, developer-surface, and wiring
   branches. Completion: every criterion, planned interaction, and declared key link has a
   proof class plus passing evidence, or is recorded as a blocker.
6. **On failure** → [failure-triage](reference/failure-triage.md) +
   `devrites-debug-recovery`. Reproduce → isolate → fix within scope → re-run; if a fix
   would exceed scope, record a blocker.
7. Update `evidence.md`, `browser-evidence.md` (if UI), `traceability.md`, and
   `state.md`. New proof goes to `evidence.md` (`proof.md` is a read-only alias —
   see `devrites-lib/reference/workspace-artifact-schema.md`).

> **Mid-flight discipline.** When tempted to claim an un-observed pass, skip a rung of the browser-proof ladder, or proceed with slices pending — see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output

**Progress first** — run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: feature proof complete for <slug>.
Changed: evidence.md, browser-evidence.md <updated|n/a>, devex.md <updated|n/a>, state.md
Evidence: acceptance <total>/<total> (judgment-only <n>); scenarios <total>/<total|n/a>; key links <n>/<n|none>; tests/build/lint/browser <pass summary>
Open: none
Next: $rite-polish
Record: .devrites/work/<slug>/evidence.md
↻ Hygiene: /clear before $rite-polish
```
If any check fails, a blocker remains, or a criterion is unproven, use the shared
`Stopped / blocked` form and route `Fix:` to the failing check or `$rite-build`; do
not recommend `$rite-polish`.
