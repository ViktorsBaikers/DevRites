---
name: devrites-api-interface
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
user-invocable: false
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here: Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review**: an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.
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
