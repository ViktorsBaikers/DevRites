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

## 3. Architecture
- Right seam/boundary? Coupling and cohesion. Does it fit existing patterns or
  introduce a competing one? Is the abstraction earned (not premature)?

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
diffs hide defects — recommend splitting rather than rubber-stamping.
