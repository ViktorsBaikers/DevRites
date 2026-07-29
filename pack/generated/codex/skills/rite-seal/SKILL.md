---
name: rite-seal
description: Decide GO / NO-GO readiness on the active feature. Use when the user asks "seal this", "GO / NO-GO", "decide if we can ship", "can we ship", or "safe to merge". Not for commit/push/tag/close; use $rite-ship.
argument-hint: "[feature-slug] [--full]"
user-invocable: true
required-agent-roles: devrites-test-analyst
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


# $rite-seal: GO / NO-GO

The decision gate before shipping. **Read the active workspace first**; if none, tell
the user to run `$rite-spec <feature>`. Produces `seal.md` with a clear verdict.
`$rite-seal` **decides**; the irreversible git commit/push/tag and the task close-out
live in `$rite-ship`, which refuses to run without a GO recorded here.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.agents/skills/devrites-lib/reference/standards/core.md` first. The other rule files load on demand:
pull these via `Read` before sealing:
- `agents.md`: review-subagent fan-out at seal.
- `code-review.md`: severity labels (Critical / Important / Suggestion / Nit / FYI).
- `review-checklist.md`: compact final review pass/fail sweep.
- `test-proof-checklist.md`: compact proof-quality gate for tests and evidence.
- `browser-proof-checklist.md`: for UI scope, the browser-state and Visual Verdict gate.
- `security-checklist.md`: when input / auth / data / integrations / secrets are in scope.
- `principles.md`: declared project invariants (`.devrites/principles.md`) are a pass/fail gate; a diff that violates one with no recorded, human-approved exception is a NO-GO.
- `documentation.md`: record decisions in `decisions.md` before sealing.
- `observability.md`: a runtime surface that ships blind is an Important finding.
- `deprecation.md`: when the diff removes / migrates code, API, or data (read with the
  risk-and-rollback step below).
- `definition-of-done.md`: standing Done bar: acceptance mapped, fresh proof, no open hard gates, scoped edits, rollback/docs where needed.


## Operating rules
- Evidence over confidence: a criterion is met only if evidence proves it.
- Spec Drift Guard applies: unresolved drift is a NO-GO.
- **Honest verdict.** DO NOT round a NO-GO up to GO to be agreeable.

## Severity gate (the ship/no-ship rule)

Read `review.md` and the latest reviewer outputs.

| State | Gate result |
|---|---|
| `Critical == 0` and `Important == 0` and acceptance proven and drift resolved | **GO** (proceed) |
| `Critical == 0` and `Important > 0` and acceptance proven and drift resolved | Render interactive prompt: *"`Important > 0` open. Proceed to seal? [y/N]"*. Default **N**. If the user types `y`, GO; otherwise NO-GO with the open Important findings listed as blockers-by-policy. |
| `Critical > 0` | **NO-GO**, no exceptions. List every Critical with `file:line` and fix direction. |
| Any acceptance criterion unproven, any bespoke `Prohibitions (must-NOT)` `resolved/test` row lacking linked proof, or any `plan.md` Key links row with no covering `EVID-###` wiring check (built-but-not-wired; `none declared` → n/a) | **NO-GO**, list the unproven criteria/prohibitions/links. |
| Any criterion tagged `proof: judgment` in `evidence.md` (untagged reads `judgment`) | Judgment proof needs a human eye. HITL: enumerate them in the verdict digest: the human's GO covers them. AFK: **no auto-GO**: append a `gate: validating` question listing them and stop. |
| Fan-out roster incomplete: `devrites-engine footprint roster` rc=3 (a reviewer neither dispatched nor skip-recorded) | **NO-GO** on the review's *completeness*, not a verdict on the diff: dispatch the missing reviewer or record a justified skip, then re-seal. Same standing as an unproven criterion. |
| Silent review axis (`devrites-engine review-integrity` rc=1 (an adversarial axis in `review.md` reported nothing and justified nothing) | **Important**) a suspected rubber-stamp, on the review's *completeness* not the diff. Re-run that axis or record its `No-findings:` justification (`code-review.md` § Zero findings is suspicious), then re-seal. |
| A declared extension check unaddressed (a valid `.devrites/extensions/*/component.yaml` declares `checks: at: seal` and `seal.md` records neither the check's outcome nor a skip justification | **Important**) same standing as a silent axis, on the review's *completeness*. Address the check's instruction or record why it doesn't apply, then re-seal. Checks are additive-only (an extension raises the bar, never lowers it); an extension that fails `extensions validate` has inactive checks, not a blocker. |
| Visual Verdict `FAIL` on an acceptance-mapped UI criterion (`browser-evidence.md`) | **NO-GO**: an unmet acceptance criterion. A declared-state `FAIL` is Important (the `Important > 0` row). UI build with a `design-brief.md` but no Visual Verdict → Important evidence gap. No brief → not applicable. |
| Diff violates a declared project principle (`.devrites/principles.md`) with no recorded, human-approved exception | **NO-GO**, list each violated principle with `file:line`. Same standing as an unproven criterion (absent / empty file → none declared → not a blocker). |
| Unresolved drift in `drift.md` | **NO-GO**, route through `$rite-plan` first. |
| Any `questions.md` entry with `gate: validating` and `status: open` | **NO-GO** regardless of behavior impact: an open validating gate is merge-blocking by definition. A slice marked `built (pending review)` is not done. |
| A stood decision (boundary / data-model / auth / public-API / migration / branching) in the `decisions.md` `## Decisions stood` ledger with no recorded `devrites-doubt` verdict: `doubt: MISSING` / `doubt-coverage` rc=3 | **Important**. **NO-GO** when the undoubted decision is irreversible-risk (auth / public-API / migration), the same standing as an unproven acceptance criterion. Severity rides the unverified **decision**, not the exit code: `doubt-coverage` rc=1 (zero doubt dispatched) is a **prompt to verify**, not itself a finding: confirm against the ledger, where every-slice-trivial (`- none`) passes and a skipped triggering decision is the finding. |
| `devrites-engine docs-stale` warns that user-facing CLI/API/install/docs-referenced code changed without README/docs updates | **FYI** unless the spec promised docs or users need docs to satisfy acceptance; then escalate to Important. |
| Outside voice (`devrites-engine outside-voice`) is unavailable in `auto` mode | Advisory only; record `outside-voice: skipped-unavailable`. Disabled mode records `outside-voice: disabled`. |

## Workflow

Before the verdict, run the deterministic advisory checks and record their outcomes in `seal.md`:
```bash
devrites-engine docs-stale; echo "docs-stale rc=$?"
devrites-engine outside-voice; echo "outside-voice rc=$?"
```
If outside voice is `available`, ask the same artifacts/diff second opinion;
findings stay advisory until verified with line quotes or accepted into the normal review pipeline.
If `.devrites/extensions/` exists, read each valid extension's `component.yaml` for `checks:
at: seal` entries and record every declared check's outcome (or skip justification) in `seal.md`.
The severity gate treats an unaddressed one as Important.
For developer-facing surfaces, compare `devex.md` predicted TTHW against the measured proof path and
record the boomerang result.

Run the full execution contract in
[`reference/phase-contract.md`](reference/phase-contract.md). It is not optional:
it contains the orientation, evidence, risk, reviewer fan-out, roster,
review-integrity, verdict, convention, and learning steps for this gate.

## On GO → hand off to $rite-ship

`$rite-seal` makes the **decision** and stops. It does **not** run `git commit`,
`git push`, `git tag`, publish, or deploy. Those moved to `$rite-ship`, which renders
the type-GO prompt and refuses to run without a GO recorded here. Keeping the decision
and the irreversible action as two separately-auditable steps is the point: a GO seal
is a verdict, not an authorization to push.

On **GO**: write `seal.md`, set `state.md` `Next step: $rite-ship`, and tell the user
the feature is cleared to ship. Do **not** set phase `done`: `$rite-ship` marks done
after the task is shipped and archived. The `Important > 0` interactive y/N earlier in
the gate is the one off-ramp seal still owns; the type-GO off-ramp now lives in ship.
For a **UI feature**, note in the hand-off that `$rite-ship` offers an optional
**design-memory** rollup: persist this feature's proven design language into a project
`DESIGN.md` so later features inherit it (`../rite-ship/reference/design-memory.md`).

## Output

Use the full output contract in [`reference/output.md`](reference/output.md).
It preserves the progress-first GO / NO-GO shape, uses the shared completion
reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)),
and keeps the boundary that `$rite-seal` decides only; `$rite-ship` executes.
