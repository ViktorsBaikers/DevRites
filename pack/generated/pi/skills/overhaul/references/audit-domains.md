# Audit domains: the floor, not the ceiling

## Contents

- [Applicability matrix](#applicability-matrix)
- [Correctness and data integrity](#correctness-and-data-integrity)
- [Database, queries, migrations and storage](#database-queries-migrations-and-storage)
- [Concurrency, caching, retries and distributed reliability](#concurrency-caching-retries-and-distributed-reliability)
- [Security, privacy and trust boundaries](#security-privacy-and-trust-boundaries)
- [Architecture, maintainability and anti-slop](#architecture-maintainability-and-anti-slop)
- [Tests and verification quality](#tests-and-verification-quality)
- [Dependencies and supply chain](#dependencies-and-supply-chain)
- [Backend performance](#backend-performance)
- [Frontend engineering](#frontend-engineering)
- [Frontend craft, interaction and accessibility](#frontend-craft-interaction-and-accessibility)
- [Classifying frontend guidance](#classifying-frontend-guidance)
- [Cross-layer contracts](#cross-layer-contracts)
- [Operations and project-specific extensions](#operations-and-project-specific-extensions)

## Applicability matrix

Before a review shard starts, build the matrix domain × component × lane × active
profile. Each cell is `applicable` (reviewed under the correct semantics),
`not applicable` (with evidence), or `blocked` (with the missing capability). Missing
expertise, tools or browser access is `blocked`, never `not applicable`. Extend the
matrix for discovered stacks, risks and requirements. Every item below is a question
to answer with evidence, not a finding to assume; a clean domain has checks and
inspected ranges recorded.

## Correctness and data integrity

Check: business invariants across API, service, job, database and UI; validation
versus normalization; null, empty and missing as distinct; boundaries and
off-by-one; precision, overflow, currency, units and rounding; locale, time zones,
DST and clocks; exhaustive enum, union and state-machine handling;
authorization-sensitive defaults; partial implementations, stubbed success and fake
integrations; unreachable flows; TODOs in required behavior; silently ignored
errors; dead code; schema/application disagreement; serialization and version drift;
undocumented behavior changes.

Trace reads and writes through the whole use case. Prove uniqueness, referential
integrity, tenant scoping, ordering, pagination, deletion and retention, and
invariants under failure and concurrency, not only the happy path. Identify the
authoritative contract; when code, docs and tests disagree, ask.

## Database, queries, migrations and storage

Check: N+1 and repeated queries; over-fetching; unbounded results; joins and
cardinality; fan-out; pagination strategy; aggregates, sorting, filtering;
parameterization; query shape and planner estimates; indexes and selectivity; lock
duration; connection pools; transaction scope and isolation; deadlocks; stale
replicas and read-after-write needs; backfills; large-table and mixed-version
migrations; constraints; rollback.

Judge indexes by cost and benefit: never recommend one from a filter alone, never
treat every sequential scan as a bug, and weigh realistic cardinality and skew, the
engine and version, read/write mix, write amplification, storage, maintenance and
deployment locks (for example PostgreSQL `CREATE INDEX CONCURRENTLY` cannot run in a
transaction and needs a failed-build cleanup). Measure app-visible latency and query
count as well as plans. `EXPLAIN ANALYZE` executes the statement: use it only in an
authorized disposable environment.

## Concurrency, caching, retries and distributed reliability

Check: check-then-act races; lost updates; write skew; stale reads; ABA and TOCTOU
where relevant; lock ordering; atomicity; cancellation; leaked tasks, goroutines or
threads; shared mutable state; async context; transaction retry boundaries; cache
keys and invalidation; tenant-aware caches; stampedes; stale authorization
decisions; duplicate and out-of-order messages; idempotency key scope and expiry;
atomic side-effect recording; poison jobs; queue starvation and backpressure; retry
amplification; timeout budgets; jitter; circuit breaking; recovery after partial
completion.

Prove interleavings deterministically with barriers or a controllable scheduler,
never arbitrary sleeps. Distinguish data races from logical races; a race detector
covers only the first. Retries must preserve intent and handle outcomes unknown to
the caller; never retry non-idempotent work indiscriminately.

## Security, privacy and trust boundaries

Map assets, entry points, principals, privileges, tenants, trust transitions and
sensitive outputs first. Then check as applicable: authentication, authorization and
ownership checks; session and token lifetimes; credentials; injection; path
traversal; uploads; deserialization; SSRF; XSS; CSRF; CORS; browser messaging; cache
poisoning; request framing; cryptographic misuse; unsafe defaults; resource
exhaustion; sensitive logs; privacy and data lifecycle; local IPC; CI/build
permissions; IaC and container configuration; supply-chain entry points.

For AI-facing software add: untrusted retrieved, repository or tool content; prompt
injection; model-output validation; tool authorization and approval binding (tool,
full arguments, requester, target, expiry — a retry or resumed session must not
authorize a changed or duplicate side effect); RAG poisoning; excessive privileges;
agent state poisoning; secret exfiltration paths. A guardrail prompt is not a
security boundary.

A confirmed finding shows the lower-trust principal, the input, the boundary crossed,
the affected resource and the concrete effect, traced entry point → propagation →
sink. A missing defense-in-depth layer is hardening, not an exploit. Validate only in
authorized local or test scope. Map to CWE (base or variant level, pinned list
version) or an ASVS requirement (`v5.0.0-x.y.z` style) only after looking the ID up;
`unmapped` is an honest outcome, and no CWE↔ASVS link may be inferred.

## Architecture, maintainability and anti-slop

Check: real dependency direction and cycles; responsibility boundaries; hidden
globals; duplicated sources of truth; over-centralized modules; abstraction leakage;
expensive coupling; error semantics; resource ownership; unsafe casts and type
escapes; broad catches; shotgun changes; superficial wrappers; cargo-cult patterns;
speculative infrastructure; copy-paste divergence; placeholder code;
performance-obscuring indirection.

A refactor needs a concrete defect, inconsistency, maintenance cost or measurable
risk. Never rewrite working idiomatic code toward a personal style. Similar-looking
code is a lead, not proof of one abstraction: verify behavior before consolidating.
Never remove defensive checks, audit logs, comments explaining invariants,
intentional compatibility layers or legal notices as "slop". "Slop" means an
observable defect or unhelpful prose, never a claim about who wrote the code.
Prose clean-up applies only to approved docs, comments or report text and must keep
facts, terminology, identifiers and meaning. No UI or copy change rides on a backend
approval.

## Tests and verification quality

Check assertion strength, observable behavior, unhappy paths, boundaries, state
transitions, retries, races, idempotency, time dependence, migrations, rollback,
compatibility, tenant and security isolation, service contracts, flakiness and test
isolation. Mocks must not remove the database, network or concurrency behavior the
defect lives in. Confirm tests are discovered and actually run.

Line and branch coverage locate gaps; they never prove correctness. Use focused
mutation testing, property-based tests, fuzzing, differential checks or fault
injection where they fit; keep counterexamples and seeds; inspect surviving mutants
and account for equivalent or invalid mutants and tool limits. Never write tests
that merely restate the implementation.

## Dependencies and supply chain

Inventory direct and transitive **resolved** versions, their runtime/build/test
roles, origins, licenses, support and end-of-life status, advisories, deprecated
APIs, provenance, integrity and install scripts. Verify the latest compatible and
the latest upstream versions separately against official registries, releases and
advisories at run time; "newer" alone justifies nothing.

For serious candidates compare keep, minimal patch, compatible upgrade, major
upgrade, replacement and removal on behavior and API differences, maturity, platform
support, maintenance, migration cost, transitive risk, bundle/startup/runtime
effect, licensing, rollback and tests. Consider "no change" seriously; stars and
novelty are not evidence. Use OSV or the ecosystem's advisory tooling; an advisory
match is not proven reachability, and a reachability tool that cannot prove use does
not prove safety. Unavailable advisory access is `UNKNOWN`. OpenSSF Scorecard
informs supply-chain practice only. Any dependency change needs plan approval; no
blind force-upgrades or blanket churn.

## Backend performance

Profile algorithms and hot paths, database, IO and network behavior, allocation and
GC, CPU contention, event-loop blocking, thread and connection pools, fan-out,
serialization, compression, cold start, batch behavior, memory leaks, cache hit and
miss behavior, throughput at fixed latency and error SLOs, and tail latency under
expected and overload traffic. Measure real work with representative data and
concurrency; a microbenchmark win is never a proportional application claim. Track
CPU, memory, IO, cost and correctness trade-offs next to latency, and keep
degradation and recovery behavior intact. Methods: [`repair-and-measurement.md`](repair-and-measurement.md#performance-measurement).

## Frontend engineering

Review real source semantics under the active client profiles:

- **Components and state:** ownership, state placement, derived versus duplicated
  state, render purity, lifecycle cleanup, stable identities, effect or reactivity
  misuse, subscription/timer/resource leaks, stale closures, navigation and URL
  state, deep links, back/forward, and the line between reusable UI and business
  logic. Async UI state should be a discriminated union or state machine, not
  independent booleans.
- **Data flow:** request ownership, waterfalls, duplicate fetches, cache keys and
  invalidation, stale results arriving out of order (a cancellation signal alone is
  not a stale-response guard), cancellation, optimistic updates and rollback, double
  submission, offline and reconnect, auth and tenant transitions, error propagation,
  retries, client/server disagreement.
- **Trust:** untrusted HTML, URLs and messages; sanitization versus output encoding;
  secrets or server-only modules reachable from client bundles; sensitive state in
  local persistence; CSP and integrity; client-only authorization illusions;
  server-to-client serialization carrying unused or sensitive fields; mutable
  module-level state shared across server requests.
- **Performance:** bundle and chunk cost, critical rendering path, hydration and
  streaming, long tasks, scheduling, unnecessary work, large tables and lists,
  layout thrash, memory/CPU/frame budgets. Check the real build mode and compiler or
  framework optimizations before proposing memoization, virtualization or caching;
  a memoization change needs profiler evidence, and removing existing memoization
  under an optimizing compiler needs tests first.
- **Tests:** component, integration and end-to-end tests that observe behavior, cover
  asynchronous ordering and error states, assert something real and are discovered
  and run. Native clients use their platform harness, not a forced DOM test.

## Frontend craft, interaction and accessibility

The craft role ([`overhaul-craft`](.pi/agents/overhaul-craft.md)) examines the existing brief and
design system, hierarchy, typography, spacing, density, tokens, theming, semantics,
focus, keyboard operation, touch and pointer, zoom and reflow, responsive or native
layout, localization and long or RTL content; every UI state (loading, empty, error,
partial, stale, offline, disabled, pending, permission-denied, success, conflict,
destructive) including transitions and interruption; actionable feedback and
recovery without inventing product copy or business decisions; motion purpose,
interruptibility and reduced-motion behavior.

Two frontend verdicts stay separate: engineering/behavior/performance assurance, and
design/interaction/accessibility evidence. A beautiful screenshot cannot pass a
broken state machine; passing API tests cannot clear an inaccessible UI. Subjective
visual direction is a human decision, never a number.

For web targets report LCP, INP, CLS, TTFB and critical-journey timings with
field/lab labels; Lighthouse TBT is not INP and CLS is not latency. Missing field
data is "unavailable", never "passing". Native, desktop and game targets use
platform frame and interaction measures. Automated accessibility checks cover a
subset of criteria, not conformance; prefer native semantics and verify keyboard,
focus and relevant assistive-technology behavior. ARIA alone does not implement a
widget. The WCAG 2.2 AA target-size floor is 24 CSS px (SC 2.5.8); larger values are
advisory. A visual snapshot update is a proposed changed expectation, never automatic
approval of a regression.

## Classifying frontend guidance

Upstream design skills conflict with each other and with product brands. Record every
frontend check as exactly one of: engineering requirement; accessibility or platform
requirement; approved design-system constraint; optional aesthetic proposal. Only the
first three can fail a control; aesthetic proposals earn no security, correctness or
score credit. A generic visual cue (a font, a palette, a card grid, a plain table,
an eyebrow label) is a lead that needs a concrete mismatch with the product brief,
usability, accessibility or the accepted design system before it is a defect. Never
import palette, font or radius bans, marketing layouts, compulsory animation
libraries, default React or Tailwind assumptions, automatic redesigns, foreign
scoring formulas, or another skill's stop and permission rules. Prefer current
official semantics and accessibility evidence plus accepted product constraints when
sources disagree; ask the user about material design choices, new UI libraries or
major restructuring.

## Cross-layer contracts

The boundary role ([`overhaul-boundary`](.pi/agents/overhaul-boundary.md)) owns the live graph of
API operations, events and messages, server actions, IPC/FFI, shared models,
generated clients and the critical journeys consuming them, and checks both
producer and consumer: schema, version, enum and nullability; numeric, ID and date
precision; pagination, order and filter semantics; auth and error behavior; retries
and idempotency; cache freshness; tenant scoping; timeouts and cancellation;
deployment compatibility. Compile-time shared types do not prove runtime conformance;
a mock written from the frontend's expectations does not prove compatibility; a
schema-generated test does not prove domain correctness or the final UI journey.
Contract and fuzz tests may send mutating requests: run them only against isolated
systems with authorization. When a producer or consumer is out of reach, separate
documented or observed contract evidence from unverifiable internals. A coordinated
interface change needs one reconciled contract revision, explicit write owners,
ordered tasks and approval.

## Operations and project-specific extensions

Check observability, missing or unsafe logs, metric cardinality, tracing boundaries,
health and readiness, feature flags, deployment order, graceful shutdown, rollback,
backup and restore, version skew, configuration drift, resilience and diagnosability.
Add language- or domain-specific extensions when applicable: memory ownership and
FFI, native sanitizers, embedded constraints, GPU and frame budgets, numerical, data
and ML reproducibility, regulated or safety requirements. An unfamiliar domain is
`blocked` until researched ([`stack-profiles.md`](stack-profiles.md#unknown-stack-procedure)),
never silently labeled safe.
