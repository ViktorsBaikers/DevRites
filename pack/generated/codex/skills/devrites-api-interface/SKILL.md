---
name: devrites-api-interface
description: Shape stable API, type, module, or frontend/backend contracts before implementation. Use when a slice crosses a boundary; not for internal helpers.
user-invocable: false
---

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

## Doubt the contract
Before standing the interface, run `devrites-doubt`.

## Done when
The contract is complete only when **every** field carries a type + optionality + unit,
**every** success and error status code is enumerated with its error-body shape, the
`devrites-doubt` verdict is accept (on reject: revise the contract and re-doubt), and the
contract + rationale are recorded in `decisions.md`. A contract that pins only the
happy-path shape is not done.

## Enables the split
A clear contract lets `$rite-plan split` proceed: the backend slice can land against the
contract with a stub consumer; the frontend slice can build against a mock or the real
contract. Neither side blocks on the other.
