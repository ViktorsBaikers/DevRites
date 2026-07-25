---
name: rite-prove
description: Prove a completed feature with tests and the full test suite, build/typecheck/lint, end-to-end/browser evidence, screenshots, commands, and outputs for seal. Not for single-slice proof.
argument-hint: "[feature-slug]"
user-invocable: true
required-agent-roles: devrites-proof-runner
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- On MultiAgent V2, call `spawn_agent` with the exact named `agent_type=devrites-<role>`, a unique `task_name`, and `fork_turns="none"`. Codex loads that role TOML's `developer_instructions` natively. Because V2 collaboration lifecycle calls bypass hooks, DevRites verifies the current durable parent/child rollout for the exact role, wait, completion, and non-empty delivered result.
- On MultiAgent V1, when the named role is not exposed, use generic `explorer` for a read-only role with `fork_turns="none"` and name exactly one `.codex/agents/devrites-<role>.toml` contract in the message. Trusted `.codex/hooks.json` injects that contract's exact `developer_instructions` and binds the child to the fail-closed reviewer read-only guard.
- On MultiAgent V1, `devrites-slice-wright` uses generic `worker` with `fork_turns="none"` and the exact role TOML named in the message. Trusted `.codex/hooks.json` binds it to the active reconcile window and `.wright-allowlist`; do not substitute `worker` for an exposed V2 named role.
- The invoked skill's `required-agent-roles` frontmatter arms the fail-closed Stop receipt. Every listed role must have a confirmed start, wait, and non-empty result in this turn.
- If any required named or generic agent dispatch is unavailable or rejected, stop for HITL. Never execute a DevRites specialist role in the root context.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-prove: prove the completed feature

Record evidence for the **whole feature**. Read the active workspace first; if none,
run `$rite-spec <feature>`.

> **Scope:** built-in `/verify` proves one change and `/run` launches the app.
> `$rite-prove` covers a feature. It walks `spec.md` acceptance
> criteria one-by-one, runs the full relevant test suite + build/typecheck/lint,
> ascends the browser-proof ladder (step 4),
> and writes `evidence.md` + `browser-evidence.md` keyed to the active
> `.devrites/work/<slug>/`. No DevRites workspace → use `/verify` or `/run` alone.

## Gate: all slices must be built first
Read `tasks.md` + `state.md`. **If ANY slice is still pending/unbuilt, STOP** and tell the
user to finish it with `$rite-build`: `$rite-prove` runs once, when the full task is
complete, not after each slice. (Each slice already got its own targeted tests during
`$rite-build`; this phase proves the assembled feature as a whole.)

**Never report an unobserved pass.** If a command could not run, report that and give
exact manual steps.

**Scoped reruns are allowed.** `$rite-prove` runs once when the full feature is assembled.
After `$rite-polish` or `$rite-review` edits code,
the existing `evidence.md` no longer post-dates the change, so re-run `$rite-prove` over
the affected criteria/routes to refresh proof before `$rite-seal`.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
Pull these via `Read` when relevant:
- `testing.md`: pyramid, determinism, no-flake discipline.
- `test-proof-checklist.md`: compact proof-quality gate for tests and recorded evidence.
- `browser-proof-checklist.md`: for UI scope, the required browser states and Visual Verdict evidence.
- `spec-grammar.md`: when the spec uses structured `### Requirement:` / `#### Scenario:`
  blocks, each scenario (WHEN/THEN) is one observable behavior to walk and prove.
- `performance.md`: measure first when perf is in scope.
- `observability.md`: when the change has a runtime surface (endpoint, job, integration,
  user flow): telemetry must be present **and observed to emit**, not assumed.
- `developer-experience.md`: when the change ships a developer-facing surface (API / CLI / SDK /
  webhook / config / error messages / getting-started): **measure** the DX scorecard (run the flow,
  measure time-to-hello-world, and capture verbatim error text), rather than asserting it.
- `definition-of-done.md`: standing Done bar: acceptance mapped, fresh proof, no open hard gates, scoped edits, rollback/docs where needed.


## Operating rules
- Evidence over confidence. Feature scope only: fix within the feature or record a
  blocker; don't refactor unrelated code.
- Spec Drift Guard applies: if tests/evidence reveal the spec is wrong, stop and handle
  drift (`rite-build/reference/spec-drift-guard.md`).
- **Runner observes; root records; wright fixes.** Use the file-backed fresh-context
  contract in [`agents.md`](../devrites-lib/reference/standards/agents.md). The root owns
  the evidence verdict and canonical writes. Every accepted source/test correction is one
  bounded `devrites-slice-wright` packet, never an inline edit.

## Workflow
0. Read `.agents/skills/devrites-lib/reference/standards/core.md` first (the always-on operating rules); pull the
   on-demand rules above when relevant.
   Then run `devrites-engine preamble` for deterministic workspace orientation.
1. **Confirm the gate** (all slices built). Read `spec.md` (acceptance criteria +
   "Commands discovered"), `tasks.md`, `state.md`, `test-plan.md` if present (the vetted
   coverage target from `$rite-vet`), and the full `git diff`.
2. **Discover commands** if not recorded:
   [test-command-discovery](reference/test-command-discovery.md): README, package
   scripts, Makefile, CI configs, Gemfile/Rakefile, pyproject, go.mod, Cargo.toml.
   **Completion:** exact runnable test/build/typecheck/lint commands are recorded or explicitly unavailable.
3. **Run proof in fresh context.** Freeze the candidate and dispatch
   `devrites-proof-runner` with the exact commands, cwd, prerequisites, acceptance map,
   and scratch boundary. Await and validate its observed full relevant test suite plus
   **build / typecheck / lint** report. The runner writes no canonical evidence.
4. **UI feature?** Include `design-brief.md`, `references.md`, routes, browser harness, and
   allowed scratch path in the proof packet, then have the runner apply the browser proof ladder:
   [proof-ladder](reference/proof-ladder.md) + [browser-proof](reference/browser-proof.md)
   (`devrites-browser-proof`): routes, viewports, screenshots (opened + described),
   console, network, interaction paths, and the brief's proof targets. Compare screenshots
   with target references and record deltas. An unresolved material mismatch is a failed
   result; the root handles any accepted correction at step 6 before re-rendering.
5. **Map proof completely.** Follow
   [`reference/acceptance-proof.md`](reference/acceptance-proof.md) for acceptance/scenario
   coverage and the conditional critical-path, observability, developer-surface, and wiring
   branches. Completion: every criterion, planned interaction, and declared key link has a
   proof class plus passing evidence, or is recorded as a blocker.
6. **On failure** → [failure-triage](reference/failure-triage.md) +
   `devrites-debug-recovery`. The root reconciles the reproduction. Send an accepted,
   in-scope correction to the sole writer, `devrites-slice-wright`; then freeze the new
   candidate and dispatch a fresh proof runner for affected checks. If a fix would exceed
   scope, record a blocker.
7. The root updates `evidence.md`, `browser-evidence.md` (if UI), `traceability.md`, and
   `state.md`. New proof goes to `evidence.md` (`proof.md` is a read-only alias:
   see `devrites-lib/reference/workspace-artifact-schema.md`).

> **Mid-flight discipline.** When tempted to claim an un-observed pass, skip a rung of the browser-proof ladder, or proceed with slices pending: see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output

**Progress first**: run `devrites-engine progress`, then use the shared completion reply contract
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
