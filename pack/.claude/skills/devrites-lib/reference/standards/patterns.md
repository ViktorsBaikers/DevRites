# Patterns & architecture

Patterns are tools for **loose coupling and clarity** — not a goal. Use one only when it
makes the design simpler to reason about, not to show it's there.

## Principles
- **SOLID, applied with judgment** — single responsibility, open/closed, Liskov,
  interface segregation, dependency inversion. They're guidance toward low coupling and
  high cohesion, not boxes to tick.
- **Composition over inheritance** — prefer assembling behavior from small pieces over
  deep class hierarchies; inheritance couples tightly and resists change.
- **Depend on abstractions at real seams** — inject dependencies where you genuinely
  need to swap or test them; don't wrap everything in an interface "just in case".
- **Separation of concerns** — keep I/O, business logic, and presentation in distinct
  layers so each can change and be tested on its own.

## Choose the pattern after you understand the problem
- Identify the actual architectural challenge first; pick the pattern that fits it.
  A pattern applied to the wrong problem adds indirection and cost.
- Start with the **simplest structure that works** — a modular monolith beats premature
  microservices for a small team. Scale the architecture when load or team size demands
  it, not before.

## Avoid over-engineering (the common failure)
- Don't add abstraction before two real callers exist. Premature generalization is a
  cost you pay forever for a benefit you may never get.
- Don't force a pattern everywhere; not every problem needs one.
- Watch the cost: misused patterns add memory/indirection/overhead and obscure intent.
- A refactor must **reduce** complexity, not just **relocate** it. Count the concepts a
  reader must hold; if a "cleaner" version leaves that count unchanged it isn't cleaner —
  prefer the restructuring that makes whole branches/modes/layers disappear.

## Anti-patterns to name and avoid
- God object / god function doing everything; tight coupling across layers.
- Hidden global state and singletons used as a back door.
- Copy-paste duplication instead of a shared abstraction (and its opposite — a clever
  abstraction over two things that aren't really the same).
- Speculative generality: config, hooks, and extension points with no current user.

## In a codebase
Match the patterns the project already uses before introducing a new one. A consistent
"good enough" pattern beats a locally-superior but foreign one. Document the *why* of any
non-obvious structural choice (see `documentation.md`).

## Reuse first
Before adding a new util / component / helper / type, **search** for an existing one and
prefer **reuse → extend → build new**, applying the AHA caveat (duplication beats the
wrong abstraction). Canonical rule in [`coding-style.md`](coding-style.md#reuse-before-you-write).
