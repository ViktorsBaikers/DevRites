---
name: devrites-api-interface
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
user-invocable: false
required-agent-roles: none
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task.
- For automatic Engram calls, omit optional `project` and `session_id` unless an exact value came from Engram or repository configuration. Never derive either from `task_name`, a run ID, directory name, or normalized slug. Call `mem_session_summary` without them by default; on `unknown_session` or `unknown_project`, retry once with both optional fields omitted. If auto-detection is ambiguous, ask the user instead of guessing.
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


# devrites-api-interface: contract before implementation

When a slice crosses a boundary (FE/BE, service/service, module/module) or exposes a
public interface, define the contract first so both sides can proceed and the interface
stays stable.

## Define the contract first
- **Shape:** request/response or function signature; field names, types, optionality,
  units. Follow the project's existing naming and conventions.
- **Status & errors:** success codes, error codes, error body shape, validation
  messages. Errors are part of the contract, not an afterthought.
- **Semantics:** idempotency, pagination, ordering, nullability, side effects.
- **Versioning/compat:** is this new or a change to an existing contract? A breaking
  change to an existing consumer is a user decision (and a drift event if unplanned).

## Stability principles
- Design for the caller. The interface should make the common case easy and the wrong
  call hard.
- **Prefer addition over modification.** A new field is additive and optional; changing a
  field's type or removing one is a breaking change. You can add later. You can't un-ship a
  shape consumers already read (observable behavior is the contract: [`deprecation.md`](../devrites-lib/reference/standards/deprecation.md) Hyrum's law).
- **One-Version Rule.** Design as if only one version of this interface will ever exist:
  extend the single contract rather than fork a v2 you then maintain in parallel. Forking
  multiplies the surface and breeds diamond-dependency conflicts; bump a version only when an
  addition genuinely can't stay backward-compatible.
- Match existing endpoints/modules in style: don't introduce a competing convention.
- **Validate at the boundary, and only there** (untrusted → trusted); don't trust
  caller-supplied trust signals (IDs, roles). Validation does *not* belong between two internal
  typed functions, on your own database's data, or in a utility already called by validated code.
  A check inside the trusted core hides the bug in the boundary that should have caught it. A
  third-party API response is external input: always untrusted. (Three-tier boundary:
  [`security.md`](../devrites-lib/reference/standards/security.md); see `rite-review/reference/security-review.md`.)

## Type craft: make the wrong call unrepresentable
- **Brand your ids.** A bare `string`/`number` id is assignable to any other id, so the compiler
  won't stop you passing a `userId` where a `taskId` is due. Give each a nominal brand
  (`type TaskId = string & { readonly __brand: 'TaskId' }`) and the mix-up becomes a type error,
  not a production incident.
- **Model variants as discriminated unions**, each state carrying only its own fields, so an
  impossible combination can't be constructed in the first place.

## Enables the split
A clear contract lets `$rite-plan split` proceed: the backend slice can land against the
contract with a stub consumer; the frontend slice can build against a mock or the real
contract. Neither side blocks on the other.

## Doubt the contract
Before standing the interface, run `devrites-doubt`: boundary decisions are exactly the
non-trivial kind worth an adversarial check.

## Done when
The contract is complete only when **every** field carries a type + optionality + unit,
**every** success and error status code is enumerated with its error-body shape, the
`devrites-doubt` verdict is accept, and the contract + rationale are recorded in
`decisions.md`. A contract that pins only the happy-path shape is not done.
