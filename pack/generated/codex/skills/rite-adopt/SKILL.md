---
name: rite-adopt
description: Adopt an existing or legacy codebase into DevRites by reverse-engineering current behavior, seeding conventions, and establishing the workflow. Use for inherited or live applications.
argument-hint: "[path or area to adopt] [+ what you want to build next]"
user-invocable: true
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


# $rite-adopt: onboard existing code

`$rite-adopt` derives a spec and initial conventions from existing code. It produces the
same `spec.md` used by the rest of the lifecycle and seeds the conventions ledger with
observed project idioms.

Use it once when onboarding a repository or one of its sub-areas. Continue with
`$rite-clarify`, `$rite-temper`, `$rite-define`, and `$rite-build`.

> **Need only a code map?** `$rite-zoom-out` maps unfamiliar code without creating a
> workspace or ledger. Use `$rite-adopt` when the project should enter the lifecycle.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
Pull `documentation.md` when recording the adoption decisions (why-not-what) in
`decisions.md`; pull `principles.md` when the code
upholds invariants worth proposing as project principles (step 4a).

## Operating rules (DevRites core)
- No silent assumptions · prefer the project's existing conventions (you are *documenting*
  them, not imposing new ones) · ask the human when the adoption scope or the next-build
  objective is unclear.

## Workflow
0. **Read `.agents/skills/devrites-lib/reference/standards/core.md`**, then run
   `devrites-engine preamble` for deterministic workspace orientation.
1. **Scope the adoption** (`$ARGUMENTS`). Which repo or sub-area is being onboarded, and:
   if stated: what the user wants to build *next* on top of it. If the next-build objective
   is missing, ask once (it shapes the spec's acceptance); if the area is ambiguous, confirm
   before investigating the whole tree.
2. **Inspect the existing code** to establish the project's current structure. Use a
   code-intelligence index if available (codebase-memory-mcp first (its `get_architecture`
   gives a fast overview), cross-checked with codegraph + graphify, else standard methods
   (LSP / Read/Grep/Glob); see `.agents/skills/devrites-lib/reference/standards/tooling.md`) for
   structure, callers, and impact. Capture, per [adoption](reference/adoption.md): **current
   behavior**, **architecture + placement** (layers, seams, where each kind of thing lives),
   the **commands** (test / build / typecheck / lint), and the **idioms** (naming, layering,
   error model, test style) + recurring **gotchas**. Read `PRODUCT.md` / `DESIGN.md` /
   `CLAUDE.md` / `AGENTS.md` if present.
3. **Write `spec.md`** via [rite-spec's spec template](../rite-spec/reference/spec-template.md)
   and create the workspace + set `.devrites/ACTIVE`
   ([state-workspace](../rite-spec/reference/state-workspace.md)). The spec records the
   **current behavior as the baseline** and the **next objective** (what adoption is for) with
   measurable acceptance. Also write `decisions.md`, `assumptions.md`, `questions.md`, and
   `state.md` (phase: spec).
3a. **Seed the capability ledger** from the baseline. If the derived `spec.md` carries
   structured `### Requirement:` blocks, fold them into the living
   `.devrites/specs/<capability>/spec.md` ledger so the project's current proven behavior is
   recorded before the first new feature. The next `$rite-spec` writes deltas against this ledger
   ([ledger.md](../rite-ship/reference/ledger.md)). A flat baseline folds as all-ADDED into the
   feature slug's capability; tag capabilities in the spec first if you want finer granularity.
   ```bash
   devrites-engine ledger diff .devrites/work/<slug>   # preview
   devrites-engine ledger sync .devrites/work/<slug>   # seed
   ```
   Skip when the baseline records no structured requirements (nothing to seed).
4. **Seed the conventions ledger** from observed behavior:
   [adoption § seeding](reference/adoption.md). This is the bootstrap exception to
   evidence-gated promotion. Seeds start at the base band with onboarding provenance;
   later sealed slices may confirm or contradict them, and fresh evidence wins.
   **Completion:** every seed names observed evidence, provenance, and the base band.
4a. **Propose candidate principles** (optional and human-ratified). When the code
   consistently enforces an invariant, such as integer cents for money, redacted PII in
   logs, or preserved v1 endpoints, propose it as a **candidate principle** rather than a
   convention. Principles are prescriptive gates, so **never seed them automatically**.
   Present each candidate through `AskUserQuestion` with evidence, and write human-ratified
   candidates to `.devrites/principles.md` with a dated Governance entry
   ([`principles.md`](../devrites-lib/reference/standards/principles.md)). An unratified
   candidate remains a convention, not a gate. If no invariant qualifies, declare no
   principles.
5. **Hand off.** Continue with `$rite-clarify`. Its topology scan asks no questions when
   the derived contract is already clear. Every plan then runs `$rite-vet` before build.
   Do not plan or build here.
   **Completion:** one next rite is reported and no plan or application code was written.

> **Mid-flight discipline.** Do not invent conventions, seed an assumed idiom, or turn
> adoption into a rewrite. Adoption records existing behavior; a later feature changes it.
> See [`anti-patterns`](reference/anti-patterns.md).

## Output

**Progress first**: run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: adopted existing behavior into <slug>; baseline spec and placement recorded.
Changed: spec.md, decisions.md, conventions ledger, principles proposals <updated|none>
Evidence: not applicable; reverse-derived behavior is recorded for review
Open: <none | adoption questions>
Next: $rite-clarify
Record: .devrites/work/<slug>/spec.md
↻ Hygiene: /clear before the next phase
```
