---
name: rite-vet
description: Vet a defined plan before code with senior-engineer engineering review. Use when the user says "vet the plan", "engineering review", "lock in the plan", or before building. Not for code review or final seal.
argument-hint: "[slug] [--cross-model] [--full]"
user-invocable: true
required-agent-roles: devrites-plan-reviewer
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task.
- For automatic Engram calls, omit optional `project` and `session_id` unless an exact value came from Engram or repository configuration. Never derive either from `task_name`, a run ID, directory name, or normalized slug. Call `mem_session_summary` without them by default; on `unknown_session` or `unknown_project`, retry once with both optional fields omitted. If auto-detection is ambiguous, ask the user instead of guessing.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- On MultiAgent V2, call `spawn_agent` with the exact named `agent_type=devrites-<role>`, a unique `task_name`, and `fork_turns="none"`. A missing visible `agent_type` field is still V2—not capability loss, V1, or HITL—so send it anyway. If the named call rejects it, stop before any generic/default spawn. Codex loads the role TOML's `developer_instructions` natively; DevRites verifies the durable rollout, wait, completion, and delivered result.
- Only after the runtime explicitly identifies MultiAgent V1, use generic `explorer` for a read-only role with `fork_turns="none"` and name exactly one `.codex/agents/devrites-<role>.toml` contract in the message. Trusted `.codex/hooks.json` injects that contract's exact `developer_instructions` and binds the child to the fail-closed reviewer read-only guard.
- On explicitly identified MultiAgent V1, `devrites-slice-wright` uses generic `worker` with `fork_turns="none"` and the exact role TOML named in the message. Trusted `.codex/hooks.json` binds it to the active reconcile window and `.wright-allowlist`.
- The invoked skill's `required-agent-roles` frontmatter arms the fail-closed Stop receipt. Every listed role must have a confirmed start, wait, and non-empty result in this turn.
- If the required dispatch for the explicitly identified runtime is unavailable or rejected, stop for HITL. Never switch runtime lanes. Never execute a DevRites specialist role in the root context.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-vet: review the plan before build

Review a defined plan for implementation scope, architecture, code quality, test
coverage, performance, failure modes, and parallel work. Cite the source for every
finding and design the test coverage that `$rite-build` will use. Fold accepted
engineering changes into the canonical plan before code is written. Run this step on
**every** plan; depth varies with risk. `$rite-autocomplete` always includes it, and
`--cross-model` adds a second opinion from another model. **Read the active workspace
first**; if there is no `plan.md`, tell the user to run `$rite-define`.

`$rite-temper` reviews product scope and strategy in the spec. `$rite-vet` reviews how
to implement that settled scope.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
Pull the standard named by the active axis: `principles.md`, `patterns.md`,
`coding-style.md`, `testing.md`, `spec-grammar.md`, `performance.md`,
`error-handling.md`, `development-workflow.md`, `afk-hitl.md`,
`developer-experience.md`, `elicitation.md`, and `definition-of-done.md`.


## Operating rules
- **Review implementation, not product ambition.** Treat the scope established by
  `$rite-temper` as settled. Check whether the plan is simple, well tested, and low risk,
  and challenge only implementation scope creep.
- **The root updates the plan; the reviewer judges.** Vet is the plan-hardening phase, so
  behavior-preserving plan refinements (test requirements, tightened scope boundaries, ordering,
  parallel lanes, error-handling + failure-mode coverage) are written straight into
  `plan.md` / `tasks.md` / `test-plan.md`. You are the single canonical writer. A finding that
  changes **acceptance criteria or product behavior** is *not* a plan refinement: it routes
  through the **Spec Drift Guard** (record in `drift.md`, recorded decision, then `$rite-plan
  repair` for any structural reslice). Nothing that grows the build's scope lands without a
  recorded human decision.
- **Support every finding.** Every finding carries a confidence band; a finding you cannot
  back by quoting the plan/spec line (or the code it references) is forced to low confidence and
  suppressed from the main report: see the verification gate in [`reference/review-axes.md`](reference/review-axes.md).
- **Apply maximum caution to hard-to-reverse changes.** Auth, migration, public API, and
  data-model changes always pause under the irreversible-risk list.
- **Use the lowest axis band.** Never round `thin` up to `ready`; do not average the
  axes. Record the reason for every decision.
- **Search before asking.** Apply `afk-hitl.md` decision ownership: verify facts and fold
  reversible technical hardening into the plan; ask only about human-owned choices.
- **Root hardens; reviewer judges.** Dispatch follows the file-backed contract in
  [`agents.md`](../devrites-lib/reference/standards/agents.md). The root owns every
  question, decision, fold-back, readiness verdict, and workspace write.

## Workflow
0. **Read `.agents/skills/devrites-lib/reference/standards/core.md`** first.
   Then run the shared orientation preamble. It prints `state.md`, present artifacts,
   run mode, and open-question totals:
   ```bash
   devrites-engine preamble
   devrites-engine snapshot
   ```
   Then the workspace: `plan.md`, `tasks.md`, `spec.md`, `decision-coverage.md`
   (for intent + acceptance), `strategy.md` (if `$rite-temper` ran), `decisions.md`,
   `assumptions.md`, `design-brief.md` (if UI), `state.md`. Require a `plan.md` whose
   Readiness gate passes (or `Plan approved`): else STOP → `$rite-define`. Require
   `Decision coverage: CLEAR`: else STOP → `$rite-clarify`. Prefer a
   code-intelligence index if available (see
   `.agents/skills/devrites-lib/reference/standards/tooling.md`) for placement / blast-radius / reuse checks.
1. **Set review depth. Never skip this step.** Apply
   [`reference/depth.md`](reference/depth.md) exactly. Every plan leaves a recorded
   engineering verdict and `test-plan.md` coverage map.
1a. **Required initial independent pass (light and full).** Before hardening or writing,
   freeze the candidate and dispatch `devrites-plan-reviewer` with the plan/spec packet.
   Await and validate its report. Light depth reduces the inline hardening, not this
   independent gate. Dispatch the DevEx predictor alongside it when triggered (within the
   maximum-three read-only budget).
2. **Scope challenge (blocking gate):** apply §0 of
   [`reference/review-axes.md`](reference/review-axes.md). Search prior decisions first:
   `devrites-engine decisions search "<2-4 plan nouns>"`. Harden to the smallest
   behavior-preserving plan; ask only when that changes acceptance or explicit architecture policy.
2a. **Cross-artifact analyze + project gates.** Run the deterministic gate, add the
   semantic terminology/conflict pass, and score principles, the anti-slop charter, and
   conventions using their named standards. Any Critical blocks `$rite-build`; a principle
   exception is always human-owned, while an absent principles file passes. Re-check after
   hardening and write the result to `analysis.md`.
   ```bash
   devrites-engine analyze; echo "analyze rc=$?"
   ```
2b. **Build-entry preflight.** Using [`reference/artifacts.md`](reference/artifacts.md), verify
   each exact proof command/cwd/tool/version and prerequisite; package names against their
   authoritative source plus nearest manifest/lockfile; parser-sensitive planned syntax in an
   isolated fixture; and the existing UI/browser harness where applicable. Record complete
   SHA-256 provenance inputs. This is non-mutating and need not make future behavior pass.
2c. **Implementation-readiness audit.** Goal-backward map every REQ/AC/NFR, interaction,
   edge/prohibition, and decision-coverage row to a slice and executable proof. Verify
   UX/spec/architecture alignment, producer-consumer contracts, slice independence/order and
   wiring, exact prerequisites, failure paths, operations, observability, rollout, and rollback.
   Fold technical fixes into the plan. Product/risk gaps are `NEEDS CLARIFICATION` →
   `$rite-clarify`; technical plan/preflight gaps are `NEEDS REPLAN` → `$rite-plan repair`.
   Neither becomes a build qid.
3. **Review four axes:** apply [`reference/review-axes.md`](reference/review-axes.md)
   through [`reference/eng-lenses.md`](reference/eng-lenses.md). Fold verified,
   behavior-preserving technical findings into the plan. Walk only human-owned decisions with
   the human, one coherent option packet at a time. The AFK ceiling remains owned by
   [`reference/depth.md`](reference/depth.md): scope/acceptance changes and irreversible risk pause.
4. **Required outputs:** write every shape and fold-back required by
   [`reference/artifacts.md`](reference/artifacts.md), using the review rules in
   [`reference/review-axes.md`](reference/review-axes.md). Ground parallelization in:
   ```bash
   devrites-engine lanes plan "$(cat .devrites/ACTIVE 2>/dev/null)"
   ```
   Completion: every scenario and acceptance criterion maps to planned proof, every slice is
   one-pass implementable, the Build-entry preflight is green or names an owned prerequisite,
   and developer-facing plans have a predicted `devex.md` scorecard. Durable proof commands
   are portable repository commands; host-local wrappers belong only in runtime packets and
   recorded execution evidence.
4a. **Forge gate.** `$rite-define` leaves `no` / `none` / `none`; Vet alone promotes under
   [`rite-build/reference/forge.md`](../rite-build/reference/forge.md). Require a costly
   unresolved architecture fork, 2–3 distinct complete contiguous `A`–`C` strategies, every
   slice AC plus exact `test-plan.md` rows/commands, and `manifest-env-v1` as an explicit
   Build-entry prerequisite. After final fold-back the three fields must agree, else clear
   Forge before READY.
5. **Write and fold back every artifact required by
   [`reference/artifacts.md`](reference/artifacts.md).** Route every
   acceptance/behavior-changing delta through the **Spec Drift Guard** (`drift.md` +
   recorded decision + `$rite-plan repair`). After any edit to
   `brief.md`, `spec.md`, `decisions.md`, `assumptions.md`, or `questions.md`, re-scan the
   affected coverage rows, assumption audit, residual uncertainty, and closed gates.
   Partial/Missing, an unowned material assumption, or an open blocking/escalating question is
   `NEEDS CLARIFICATION` → `$rite-clarify`/HITL; never refresh past it.
   Re-run the gate after every fold-back so a task edit cannot invalidate the earlier pass:
   ```bash
   devrites-engine analyze; echo "final analyze rc=$?"
   ```
   Any non-zero result blocks the handoff. Keep `state.md` non-READY and do not
   generate the engineering digest yet; this is the frozen candidate for step 6,
   not the final handoff.
6. **One narrow recheck after accepted edits.** If steps 2 through 5 changed the frozen candidate,
   dispatch `devrites-plan-reviewer` once more with only the accepted initial findings,
   changed planning paths, affected criteria, and the new immutable identity. Do not repeat
   the full review or start a third loop. If nothing changed, the initial report is final.
   If the recheck causes an accepted edit, repeat step 5. After the recheck is
   complete and all accepted edits are folded back, re-close the matrix, rerun
   `analyze`, run `devrites-engine readiness-digest coverage <slug>`, replace the
   complete `Coverage inputs SHA-256` line, and only then run
   `devrites-engine readiness-digest engineering <slug>`.

   Now write exactly one `DevRites contract: devrites.readiness-artifacts.v2`
   field to both `test-plan.md` and `eng-review.md`, plus one typed field to
   `eng-review.md`: `Implementation readiness: READY`, `NEEDS CLARIFICATION`, or
   `NEEDS REPLAN`. Only READY sets `Phase: vet` and `Next step: $rite-build`,
   after a final sweep leaves no foreseeable human choice except a justified
   action-time checkpoint. Technical failure records its reproduction and
   `$rite-plan repair` without a qid; a human-owned contract gap routes
   `$rite-clarify` and uses the normal awaiting-human block.
   [`reference/cross-model.md`](reference/cross-model.md) owns the optional outside voice.
   Completion: the final axis floor clears, an objective technical blocker is recorded, or a
   genuine human-owned gate is recorded.
7. **STOP.** Report the scope verdict, lowest axis band, coverage gaps closed, the
   Build-entry preflight, expected action-time checkpoints, and the failure-mode criticals;
   recommend `$rite-build` only when the entry contract is ready.

> **Mid-flight discipline.** Do not replace the interactive review with
> `eng-review.md`, change acceptance through plan hardening, score without source
> evidence, or ignore unexplained complexity. See
> [`reference/anti-patterns.md`](reference/anti-patterns.md).

## Output

**Progress first**: run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: plan vetted for <slug>; depth <light|full> with axis floor <band>.
Changed: eng-review.md, test-plan.md, plan.md, decisions.md
Evidence: Implementation readiness: READY; coverage <x/y> planned; build-entry preflight <pass>; open findings Critical 0 / Important 0 / Suggestion <n>; reviewer loop <n>; outside-voice <ran|skipped-unavailable|disabled>
Open: none
Next: $rite-build
Record: .devrites/work/<slug>/eng-review.md
↻ Hygiene: /clear before $rite-build
```
If a blocker or Spec Drift Guard delta remains, use the shared `Stopped / blocked`
form and route `Fix:` to `$rite-clarify` for product/risk decisions or `$rite-plan`
for technical replanning; do not recommend `$rite-build`.
**DO NOT write code, slice, or run the build here**. That's `$rite-build`. Vet reviews and hardens the plan; it never implements.
