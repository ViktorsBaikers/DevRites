---
name: rite-upgrade
description: User-invoked semantic upgrade for an active legacy DevRites workspace. Reconciles unfinished planning with the installed workflow contract while preserving completed work and history.
argument-hint: "[feature-slug]"
user-invocable: true
disable-model-invocation: true
required-agent-roles: none
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


# $rite-upgrade: bring an active workspace onto the current contract

Use this when a workspace created under older DevRites rules cannot safely continue with
the installed pack. It reconciles the workspace to the current desired state; it does not
replay a chain of release-specific scripts. Structural `devrites-engine migrate` remains a
separate first step.

Current semantic contract: `devrites.readiness-artifacts.v2`.

## Rules consulted

Read `devrites-lib/reference/standards/core.md`,
`devrites-lib/reference/standards/agents.md`,
`devrites-lib/reference/workspace-artifact-schema.md`, and the current Clarify, Define,
Plan, Vet, Converge, and Build phase contracts needed by the assessment.

## Invariants

- Never edit application source, tests, dependencies, Git state, or historical evidence.
- Preserve completed-slice identity, status, acceptance, dependencies, historical proof,
  answered questions, and `touched-files.md`.
- Reassess only unfinished planning. Archived or `done` workspaces are a no-op.
- Never downgrade a workspace that names a future semantic contract.
- This is a recovery command, not a lifecycle phase; never write `Phase: upgrade`.
- Durable plans contain portable repository commands, not RTK/local aliases, user-specific
  absolute paths, or temporary proof trees. Runtime packets may name an execution adapter;
  evidence may record both logical and executed forms.
- Ask at most one coherent question, and only for product, policy, irreversible risk, or
  human-only access/action. Retry and repair authorization is agent-owned.
- The explicit `$rite-upgrade` invocation authorizes the bounded workspace reconciliation.
  The root remains the only canonical writer.

## Workflow

0. **Orient and normalize structure.** Read the active slug or `$ARGUMENTS`. Run:
   ```bash
   devrites-engine doctor; echo "doctor rc=$?"
   ```
   A binary/pack integrity mismatch stops at `$rite-doctor`; do not run migration or
   mutate semantic artifacts. Once healthy, orient without writing:
   ```bash
   devrites-engine preamble [feature-slug]
   devrites-engine snapshot [feature-slug]
   ```
   A missing workspace stops at `$rite-spec`; an archived or `Status: done` workspace is
   a no-op. For an active healthy workspace, run:
   ```bash
   devrites-engine migrate
   devrites-engine preamble [feature-slug]
   devrites-engine snapshot [feature-slug]
   ```
   If every semantic artifact names the current contract and `build-readiness` passes,
   mark the candidate as a likely no-op; the fresh upgrade assessment in step 2 must
   confirm that before reporting it.
1. **Freeze the preservation baseline.** Create the standard orchestrator-controlled
   `agent-packet/v1` baseline. Include exact paths for current phase/artifact contracts,
   `state.md`, `spec.md`, `decision-coverage.md`, `architecture.md`, `plan.md`, `tasks.md`,
   `traceability.md`, `test-plan.md`, `eng-review.md`, `questions.md`, `decisions.md`,
   `assumptions.md`, `drift.md`, `evidence.md`, and `touched-files.md` when present. Record
   the source diff identity plus the protected identity/status/acceptance/dependency
   fields for every completed slice and each preserved artifact identity. Freeze the
   workspace while it is assessed.
2. **Classify from fresh context.** Dispatch `devrites-upgrade-planner` with objective
   `reconcile this workspace with devrites.readiness-artifacts.v2`, the exact baseline,
   and budgets `25 files / 2,000 loaded lines / 180 result lines`. Require one validated
   `upgrade-assessment`. The agent writes nothing and asks nothing.
3. **Reconcile ownership.** Reject stale, malformed, or preservation-breaking advice.
   Dispatch `devrites-evidence-scout` only for a bounded missing fact. Drop an open
   retry/tool-repair question with `devrites-engine resolve --drop <qid>
   "superseded by explicit semantic upgrade; agent-owned recovery"` and continue. Keep a
   genuine existing human gate intact; if a newly discovered human-owned choice is
   unavoidable, persist one option packet and stop.
4. **Apply current desired state in the root.**
   - Remove obsolete old-engine reconstruction, old workflow hashes, temporary proof
     recipes, and host-local wrappers from active `plan.md`, `tasks.md`, and
     `test-plan.md`. Keep historical evidence unchanged.
   - Re-run Clarify against unfinished intent until `Decision coverage: CLEAR`; it writes
     exactly one `DevRites contract: devrites.readiness-artifacts.v2` field. Do not stamp a
     marker over stale content.
   - Re-run Temper only where unfinished scope still needs strategic review.
   - For unstarted planning, use the current Define contract. When active planning needs
     normalization, dispatch `devrites-plan-drafter` in repair mode and reconcile only
     pending slices. Preserve all completed slices without changing their protected fields.
     For all-built active work, normalize only gate-required planning fields. Use Converge
     only when live code and recorded intent genuinely differ.
   - Invalidate stale engineering readiness after any planning edit. Run the full current
     Vet contract, including its fresh plan reviewer, portable proof commands, preflight,
     and digest refresh. Vet writes exactly one current contract field to `test-plan.md`
     and `eng-review.md`.
5. **Verify preservation and readiness.** Recompute the frozen source diff, every protected
   completed-slice field, `evidence.md`, answered-question records, and `touched-files.md`;
   any mismatch is a hard stop and must be restored before proceeding. Then run:
   ```bash
   devrites-engine build-readiness [feature-slug]; echo "readiness rc=$?"
   ```
   Follow the gate's current route for `2` through `7`; `8` means the semantic
   reconciliation is still incomplete. Finish only at `0`. Preserve the original active
   build cursor when it still names the next pending slice; otherwise use the current
   deterministic next action.
6. **STOP.** Do not start Build. A second `$rite-upgrade` must take the current-contract
   no-op path.

## Output

Run `devrites-engine progress`, then use the shared completion reply contract in
`devrites-lib/reference/reply-contract.md`:

```text
Done: workspace <slug> upgraded to devrites.readiness-artifacts.v2.
Changed: <active planning artifacts only | none; already current>
Evidence: build-readiness rc=0; source/completed slices/history preserved
Open: <none | one genuine human gate>
Next: <snapshot next command>
Record: .devrites/work/<slug>/state.md
↻ Hygiene: /clear before the next lifecycle step
```

On a no-op, say so and keep the snapshot's current next command.
