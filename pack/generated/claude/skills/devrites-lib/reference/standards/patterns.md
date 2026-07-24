# Patterns & architecture

Use a pattern only when it makes the design easier to understand and reduces coupling.

## Principles

- Keep cohesion high and coupling low; separate I/O, domain logic, and presentation.
- Prefer composition to deep inheritance. Introduce an abstraction only at a real seam that must vary or be tested independently.

## Choose the pattern after you understand the problem
- Identify the architectural challenge before choosing a pattern.
- Start with the **simplest structure that works**: a modular monolith beats premature
  microservices for a small team. Scale the architecture when load or team size demands
  it, not before.

## Avoid over-engineering
- Follow [`coding-style.md`](coding-style.md#simplicity): no speculative abstraction or pattern without a current need.
- A refactor must **reduce** complexity rather than merely **relocate** it. Count the concepts a
  reader must hold; if a "cleaner" version leaves that count unchanged, it is not cleaner:
  prefer the restructuring that makes whole branches/modes/layers disappear.

## Anti-patterns to name and avoid
- God object / god function doing everything; tight coupling across layers.
- Hidden global state and singletons used as a back door.
- Copy-paste duplication instead of a shared abstraction (and its opposite: a clever
  abstraction over two things that aren't really the same).
- Speculative generality: config, hooks, and extension points with no current user.

## In a codebase
Match the patterns the project already uses before introducing a new one. A consistent
"good enough" pattern beats a locally-superior but foreign one. Document the *why* of any
non-obvious structural choice (see `documentation.md`).
