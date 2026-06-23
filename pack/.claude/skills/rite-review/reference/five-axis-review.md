# Five-axis review

The axes the dispatched `devrites-code-reviewer` applies to the diff (tests first
— always), under one severity scale (Critical / Important / Suggestion / Nit / FYI).
The `/rite-review` inline lead **reconciles** the returned report against the Spec
axis — it does not re-run these axes itself. This file is the shared definition of
what "full code-review discipline" covers; use it to judge whether the agent's report
is complete, and to scope anything the agent could not (e.g. UI-only lenses below).

## 0. Tests (first)
- Do tests exist for the changed behavior, and do they prove the acceptance criteria?
- Would they fail if the code were wrong? (No assertion-free or tautological tests.)
- Edge cases: empty, boundary, error, permission-denied, concurrency.

## 1. Correctness
- Does it do what the spec says? Off-by-one, null/undefined, error paths, race
  conditions, incorrect assumptions about inputs.
- Does it handle the states the slice promised (loading/empty/error for UI)?

## 2. Readability
- Can the next engineer understand it without the author? Naming, function length,
  nesting depth, comments that explain *why* not *what*.
- Structural smells: a conditional **bolted onto an unrelated flow** (wants its own
  helper/state/policy — a design smell, not a nit); **repeated conditionals on the same
  shape** (a missing model or dispatcher).

## 3. Architecture
- Right seam/boundary? Coupling and cohesion. Does it fit existing patterns or
  introduce a competing one? Is the abstraction earned (not premature)?
- Does a refactor **reduce** complexity or just **relocate** it? Count the concepts a
  reader must hold; a "cleaner" version that leaves that count unchanged isn't cleaner.
- Is feature-specific logic **leaking into a shared module** instead of its owning layer?
  Is a **type boundary** left implicit by a gratuitous `any`/cast or a silent fallback?
- **Name the remedy, not just the smell** — replace a conditional chain with a typed
  dispatcher, separate orchestration from business logic, move feature logic to its owning
  package, delete a pass-through wrapper, split a large file. Prefer the move that removes
  moving pieces over one that re-centralizes the same complexity.

## 4. Security
- Trust boundaries, input validation, authz checks, secrets handling. Hand off to
  `devrites-audit security` when input/auth/data/integration is in scope.

## 5. Performance
- Obvious N+1s, unnecessary work in hot paths, payload sizes. Hand off to
  `devrites-audit perf` when perf is relevant — **measure before claiming**.

## 6. Maintainability
- Tests, docs/comments where needed, no dead code added, no TODOs left, consistent
  with project conventions.

## Frontend axes (if UI)
- UX flow matches neighbors; all interaction states; a11y (focus, labels, contrast,
  keyboard); responsive; design-system alignment (no drift, no anti-AI-slop).

## Sizing & speed
Prefer reviewing roughly one slice / ~100 lines of meaningful change at a time. Larger
diffs hide defects — recommend splitting rather than rubber-stamping. Watch **file size,
not just diff size**: a small diff that pushes an already-large file further past a healthy
boundary wants decomposition (extract helpers / split modules) *first* — decompose, then
add.
