# Fullstack work (frontend + backend together)

Most UI features also need backend. Don't build the two sides blind to each other, and
don't build "all backend, then all frontend" — that hides integration risk until the end.

## Contract first
Define the **API / data contract** before either side codes against it — shape, field
types + units, status codes, **error bodies**, pagination, idempotency. Use
`devrites-api-interface`; doubt it with `devrites-doubt` before standing it. With the
contract fixed:
- the **backend** slice can land against it (consumer stubbed),
- the **frontend** slice can build against a mock or the real contract.
Neither side blocks the other, and the seam stays stable.

## Slice vertically through the layers
Each slice cuts **one capability** end-to-end — data/model → service → API → UI — and
leaves a **working, demoable path**. "Create item" is a slice (persist + endpoint +
minimal form); "all the models" is not. Order by dependency; the first slice is the
thinnest end-to-end path so integration surprises surface early and cheap.

## Apply the right discipline to each layer
- **Backend part** → the engineering rules: validate untrusted input at the boundary,
  authz on every sensitive action, parameterized queries, fail closed, no secrets in
  logs (`security.md`); fail-fast errors with meaningful messages (`error-handling.md`);
  measure-first performance (no N+1) (`performance.md`).
- **Frontend part** → frontend craft: shape (all states), the design system, and the
  2026 [quality-standards](quality-standards.md) (CWV, WCAG 2.2, responsive, motion).
- Map the API's error/edge responses to **real UI states** — every error the contract can
  return needs a handled, helpful UI state, not a blank screen.

## Placement spans both sides
The investigation's placement analysis covers **where the backend logic lives** *and*
**where the UI component lives**, plus the contract that joins them — so each side is
correctly placed, not bolted on.

## Prove BOTH layers
A fullstack slice is proven only when both sides have evidence:
- backend/contract: tests for the endpoint + its error/edge cases (and a real
  request/response observation);
- frontend: browser proof (states exercised, console clean, responsive, a11y).
If only one side is proven, the slice is **not** done.

## Keep them in sync
A change to the contract after both sides exist is a **Spec Drift Guard** event — it
affects FE and BE together. Stop, record it, re-plan the contract, then update both sides.
