---
name: rite-converge
description: Converge intent and live code. Use when resuming a half-built feature, after `$rite-adopt` drift, or the user asks "what's left to build". Not for initial planning.
argument-hint: "[feature-slug]"
user-invocable: true
required-agent-roles: devrites-evidence-scout
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


# $rite-converge: compare live code with intent

Read `spec.md`, `plan.md`, and `tasks.md` as the **sole source of intent** (with
`.devrites/principles.md` as governing constraints), assess what the **live codebase**
implements, and **append every unmet piece as a new traceable `SLICE-###`** at the
bottom of `tasks.md` so `$rite-build` can finish it. Use this for a resumed half-built
feature, an adopted codebase that drifted from its derived spec, or a build that stalled
mid-slice. **Read the active
workspace first**; if there's no `spec.md`/`plan.md`/`tasks.md`, tell the user which
prerequisite skill to run.

> **This is not a diff tool.** `$rite-converge` compares current code with intent. It
> does not use git history or compare branches. For a change-scoped
> review use `$rite-review`; to prove a finished feature use `$rite-prove`.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
Pull on demand:
- `principles.md`: the project invariants (`.devrites/principles.md`); code that violates a
  MUST principle is the highest-severity gap and produces a remediation slice.
- `spec-grammar.md`: buildable acceptance criteria vs `## Success metrics` (outcome KPIs the
  code can't make true and this pass never enqueues); structured `### Requirement:` /
  `#### Scenario:` blocks, each scenario one behavior to check as built / partial / absent.
- `tooling.md`: prefer a code-intelligence index (codebase-memory-mcp → codegraph → graphify,
  else LSP / Read/Grep/Glob) to read the live code, not assumptions.
- `testing.md`: a criterion with code but no covering test is *partial*, not done.

## Operating rules
- **APPEND-ONLY, never rewrite.** The only write to `tasks.md` is **appending** new
  `SLICE-###` entries. Never rewrite, renumber, reorder, or delete an existing slice
  (including slices a prior convergence appended). Never edit `spec.md` or `plan.md`. Never
  touch application code: completing the appended slices is `$rite-build`'s job, not this
  skill's.
- **Clean means byte-for-byte unchanged.** When the code already satisfies everything, leave
  `tasks.md` untouched (no empty convergence header) and report a clean result. Recommend
  `$rite-prove`.
- **Use artifacts as intent.** The spec, plan, tasks, and principles are the
  contract. If assessing reveals the *spec* is wrong (the code is right and the requirement is
  stale), that's **Spec Drift**: stop and route it through the Spec Drift Guard
  (`rite-build/reference/spec-drift-guard.md`) + a recorded decision; never paper over a spec
  bug by appending a task that changes correct code to match a wrong requirement.
- **Partial is not done.** Code that exists but is untested, half-wired, or covers only the
  happy path is an unmet gap: enqueue the remainder, don't round it up.
- **Principles are non-negotiable.** A live violation of a declared invariant with no recorded
  exception is the top-severity gap, walked first. Absent/empty principles file → none declared
  → skip the check gracefully, never block for its absence.
- **Scout observes; root classifies and writes.** Use
  [`agents.md`](../devrites-lib/reference/standards/agents.md). The evidence scout returns
  live-code citations only; the controlling chat owns built/partial/absent calls and append-only
  workspace changes.

## Workflow
0. **Read `.agents/skills/devrites-lib/reference/standards/core.md`** first (the always-on
   operating rules), then run `devrites-engine preamble` for deterministic workspace orientation.
1. **Confirm the gate.** Require `spec.md` + `plan.md` + `tasks.md` in the active workspace. If
   any is missing, **STOP** and name the prerequisite (`$rite-spec` for a missing spec,
   `$rite-define` for a missing plan/tasks, `$rite-adopt` to onboard existing code). Do not
   produce partial output.
   Require `decision-coverage.md` with `Decision coverage: CLEAR`; otherwise STOP →
   `$rite-clarify`.
2. **Load intent:** [`reference/convergence-assessment.md`](reference/convergence-assessment.md).
   From `spec.md`: buildable `AC-###` / `### Requirement:` scenarios (skip `## Success metrics`);
   from `plan.md`: architecture decisions + named touch-points (files/components the plan says
   get built); from `tasks.md`: existing slices + their `Satisfies:`; from
   `.devrites/principles.md`: the invariants.
   **Completion:** every buildable criterion, touch-point, slice output, and principle is in the assessment inventory.
3. **Run the mechanical checks**, then read the code. `devrites-engine analyze` gives coverage
   and consistency; `devrites-engine coverage` gives the AC→slice→proven matrix. They catch *unmapped*
   criteria; they do **not** see whether mapped code is built and correct, for that,
   read the live code (code-intelligence index per `tooling.md`).
   ```bash
   S="$(cat .devrites/ACTIVE 2>/dev/null)"
   devrites-engine analyze "$S"; echo "analyze rc=$?"
   devrites-engine coverage "$S" > /dev/null; echo "coverage rc=$?"
   ```
4. **Assess each unit as built / partial / absent** against the live code (the rubric is in
   [`reference/convergence-assessment.md`](reference/convergence-assessment.md)): every
   acceptance criterion / scenario, every plan touch-point, and every existing slice's stated
   Produces. A principle violated in the current code is its own top-severity gap.
   Dispatch up to three independent inventory partitions to `devrites-evidence-scout` on the
   same frozen candidate, await their dossiers, then reconcile the cited facts in the root
   context. **Completion:** every inventory unit is classified once with live-code evidence.
5. **Enqueue the remainder as new slices.** For each *partial* or *absent* unit, append a
   `## SLICE-###` (continue the numbering after the highest existing id) in the `rite-define`
   slice grammar, each with a `Satisfies:` line tracing to the AC/REQ it closes and a
   `Convergence: <iso>` marker line. Dependency-order them after the existing slices; a
   principle-remediation slice sorts first. **If every unit is built → append nothing.**
   **Completion:** every partial/absent unit has one traceable appended slice, or the file is byte-for-byte unchanged.
6. **Write append-only + bookkeeping.** Append the slice batch to `tasks.md` (nothing else in
   that file changes); refresh `traceability.md` (`devrites-engine coverage` → new rows for the
   appended slices). Appending a slice changes the plan input, so invalidate the prior vet:
   update `state.md` to `Phase: plan`, `Next step: $rite-vet`, and set an existing
   `eng-review.md` field to `Implementation readiness: NEEDS REPLAN`. When nothing was unmet,
   leave the plan/vet verdict untouched and set `Next step: $rite-prove`. Append
   `decisions.md` for any material call.
7. **STOP.** Report units assessed, built / partial / absent counts, slices appended, and any
   principle violation found; recommend `$rite-vet` when slices were appended (or
   `$rite-prove` if the code already
   converged).

## Appended slice format
Use the complete
[`canonical slice grammar`](../devrites-lib/reference/workspace-artifact-schema.md#canonical-slice-grammar)
with one added `Convergence:` field after `Satisfies:`:
```markdown
<!-- Convergence 2026-07-07: slices below appended by $rite-converge — live code assessed against intent. -->
## SLICE-014 <name of the unmet capability>
Satisfies: AC-007            # the criterion / scenario this closes
Convergence: 2026-07-07      # marks this as a convergence-appended slice, not an original
<all remaining canonical slice fields>
```

> **Mid-flight discipline.** Do not rewrite an existing slice, edit source, mark a
> happy-path-only implementation as built, or add work for a spec you suspect is wrong.
> See [`reference/anti-patterns.md`](reference/anti-patterns.md).

## Output

**Progress first**: run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: convergence assessed for <slug>; <n> slices appended.
Changed: tasks.md <appended|unchanged>, traceability.md, state.md, eng-review.md <invalidated|unchanged>
Evidence: units <built>/<total> built · <partial> partial · <absent> absent · principle violations 0
Open: none
Next: $rite-vet
Record: .devrites/work/<slug>/tasks.md
↻ Hygiene: /clear before $rite-vet
```
When nothing was unmet, render the same green form with `Next: $rite-prove`.
If spec drift or another blocker remains, use the shared `Stopped / blocked` form
and route `Fix:` to `$rite-plan`; do not recommend `$rite-build`.
**DO NOT write application code, rewrite existing slices, or edit spec.md/plan.md here.**
This phase assesses and enqueues; `$rite-build` implements.
