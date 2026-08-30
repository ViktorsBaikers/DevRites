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

## Boundaries and state ownership

- Give every mutable fact one authoritative owner and name how other components read,
  request change, and reconcile. Shared writable state is coupling hidden as convenience.
- At a module/service boundary, contract inputs, outputs, errors, versioning, ordering,
  idempotency, and failure ownership before choosing transport. Apply
  [`repository-topology.md`](repository-topology.md) and
  [`integration-reliability.md`](integration-reliability.md) when triggered.
- Choose synchronous work when the caller needs the result inside its latency/consistency
  contract. Choose asynchronous work only with an explicit pending state, durable handoff,
  retry/deduplication, and recovery; a queue is not a failure-handling strategy.
- Make a consistency/availability trade-off per invariant and partition behavior. Do not
  claim both without a mechanism and evidence. Security and financial/data-loss invariants
  normally fail closed; lower-risk reads may use bounded staleness when the spec permits it.
- Treat a circular dependency as evidence that ownership or layering is wrong. Break the
  cycle at the smallest existing stable contract rather than duplicating types or adding a
  service locator.

## Avoid over-engineering
- Follow [`coding-style.md`](coding-style.md#simplicity): no speculative abstraction or pattern without a current need.
- A refactor must **reduce** complexity rather than merely **relocate** it. Count the concepts a
  reader must hold; if a "cleaner" version leaves that count unchanged, it is not cleaner:
  prefer the restructuring that makes whole branches/modes/layers disappear.

## Anti-patterns to name and avoid
- God object / god function doing everything; tight coupling across layers.
- Hidden global state and singletons used as a back door.
- Two components both claiming authority over the same mutable state.
- A queue/cache/service introduced without a failure, ownership, or recovery contract.
- Copy-paste duplication instead of a shared abstraction (and its opposite: a clever
  abstraction over two things that aren't really the same).
- Speculative generality: config, hooks, and extension points with no current user.

## Symptom → suspect pattern

Route an observed code symptom to the review it should trigger; the symptom is the
evidence, not the diagnosis:

| Observable symptom | Suspect | First check |
| --- | --- | --- |
| Every change funnels through one file/module | God object / missing seam | ownership map (§ Boundaries and state ownership) |
| Tests stub half the module to exercise one function | I/O and logic entangled | separate I/O, domain logic, presentation |
| Adding one field requires editing many unrelated files | Shotgun coupling | coupling direction; [`repository-topology.md`](repository-topology.md) |
| Two components write the same mutable state | Authority conflict | one authoritative owner per fact |
| Config/flag exists with no current consumer | Speculative generality | delete or name the current user (anti-patterns above) |

**Failing case:** a review that names a pattern without pointing at the observed symptom
that motivated it is architecture preference, not finding.

## In a codebase
Match the patterns the project already uses before introducing a new one. A consistent
"good enough" pattern beats a locally-superior but foreign one. Document the *why* of any
non-obvious structural choice (see `documentation.md`).
