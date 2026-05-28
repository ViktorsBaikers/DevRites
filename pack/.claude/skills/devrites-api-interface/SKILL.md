---
name: devrites-api-interface
description: Design stable APIs and interface contracts before implementation — REST/GraphQL endpoints, module boundaries, type contracts, FE/BE splits. Use when the user says "design the API", "contract this out", "split frontend and backend", or a slice crosses a boundary. Not for internal helpers or post-ship interfaces (use `/rite-review`).
user-invocable: false
---

# devrites-api-interface — contract before implementation

When a slice crosses a boundary (FE/BE, service/service, module/module) or exposes a
public interface, define the contract first so both sides can proceed and the interface
stays stable.

## Define the contract first
- **Shape** — request/response or function signature; field names, types, optionality,
  units. Follow the project's existing naming and conventions.
- **Status & errors** — success codes, error codes, error body shape, validation
  messages. Errors are part of the contract, not an afterthought.
- **Semantics** — idempotency, pagination, ordering, nullability, side effects.
- **Versioning/compat** — is this new or a change to an existing contract? A breaking
  change to an existing consumer is a user decision (and a drift event if unplanned).

## Stability principles
- Design for the caller. The interface should make the common case easy and the wrong
  call hard.
- Be conservative in what you expose; you can add later, but removing/changing breaks
  consumers.
- Match existing endpoints/modules in style — don't introduce a competing convention.
- Validate at the boundary (untrusted → trusted); don't trust caller-supplied trust
  signals (IDs, roles). (See `rite-review/reference/security-review.md` three-tier.)

## Enables the split
A clear contract lets `/rite-plan split` proceed: the backend slice can land against the
contract with a stub consumer; the frontend slice can build against a mock or the real
contract. Neither side blocks on the other.

## Doubt the contract
Before standing the interface, run `devrites-doubt` — boundary decisions are exactly the
non-trivial kind worth an adversarial check. Record the contract + rationale in
`decisions.md`.
