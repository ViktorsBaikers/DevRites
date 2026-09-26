# Audit domains: the floor, not the ceiling

## Contents

- [Applicability matrix](#applicability-matrix)
- [Correctness and data integrity](#correctness-and-data-integrity)
- [Database, queries, migrations and storage](#database-queries-migrations-and-storage)
- [Concurrency, caching, retries and distributed reliability](#concurrency-caching-retries-and-distributed-reliability)
- [Security, privacy and trust boundaries](#security-privacy-and-trust-boundaries)
- [Architecture, maintainability and anti-slop](#architecture-maintainability-and-anti-slop)
- [Over-engineering and dead weight](#over-engineering-and-dead-weight)
- [Tests and verification quality](#tests-and-verification-quality)
- [Dependencies and supply chain](#dependencies-and-supply-chain)
- [Backend performance](#backend-performance)
- [Frontend engineering](#frontend-engineering)
- [Frontend craft, interaction and accessibility](#frontend-craft-interaction-and-accessibility)
- [Classifying frontend guidance](#classifying-frontend-guidance)
- [Cross-layer contracts](#cross-layer-contracts)
- [Public API and developer experience](#public-api-and-developer-experience)
- [Documentation](#documentation)
- [Operations and project-specific extensions](#operations-and-project-specific-extensions)

## Applicability matrix

Before a review shard starts, build the matrix domain × component × lane × active
profile and record it in `coverage.json` `applicability[]` (`domain` as one of the
eight [score keys](scoring.md#audit-sections-and-score-domains), `component`, `lane`,
`state`, `reason`, `receipts[]`). Each cell is `applicable` (reviewed under the
correct semantics), `not-applicable` (with a reason and evidence), or `blocked` (with
the missing capability). Missing expertise, tools or browser access is `blocked`,
never `not-applicable`. Extend the matrix for discovered stacks, risks and
requirements. Every item below is a question to answer with evidence, not a finding
to assume. A cell is closed only by an admitted receipt that lists its domain in
`domains_checked`; a cell with no such receipt is not checked, never clean, and
`G-COVERAGE` cannot pass over it.

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
performance-obscuring indirection. Complexity without a current need has its own
floor in [Over-engineering and dead weight](#over-engineering-and-dead-weight).

A refactor needs a concrete defect, inconsistency, maintenance cost or measurable
risk. Never rewrite working idiomatic code toward a personal style. Similar-looking
code is a lead, not proof of one abstraction: verify behavior before consolidating.
Never remove defensive checks, audit logs, comments explaining invariants,
intentional compatibility layers or legal notices as "slop". "Slop" means an
observable defect or unhelpful prose, never a claim about who wrote the code.
Prose clean-up applies only to approved docs, comments or report text and must keep
facts, terminology, identifiers and meaning. No UI or copy change rides on a backend
approval.

## Over-engineering and dead weight

Code that serves no current requirement still costs reading, testing, securing and
migrating. Agent-written code shows these patterns more often — larger patches, more
duplicated blocks, fewer reuses of existing helpers — but authorship is never
evidence: check every codebase the same way. Tag each candidate with its cheapest cut:

- `delete`: unreferenced code, exports, routes, flags, config keys and environment
  variables; scratch and debug files, commented-out blocks, leftover debug output;
  parameters and options every caller leaves at the default; compatibility shims,
  aliases and "legacy" branches for forms that never shipped; features and edge-case
  paths no requirement, caller or test needs.
- `yagni`: an interface, abstract base or protocol with one implementation (fakes
  count); a factory, builder, registry or plugin system with one product; a type
  parameter used with one type; a strategy, hook or options bag nobody varies; a
  layer that only passes data along; configuration or a flag whose value never
  differs across real deployments.
- `reuse`: a near-duplicate of a helper, type or pattern that already exists under
  another name. Search by what the code does, not by its name.
- `stdlib` / `native`: hand-written code that the standard library, the platform
  (HTML, CSS, database constraints, the runtime) or an installed dependency already
  provides. Name the replacement and the version that has it.
- `dependency`: a package used for a few lines, a manifest entry nothing imports, two
  packages doing one job, or a name that does not resolve on its registry
  (hallucinated or typosquat risk: route to security).
- `shrink`: wrappers that only delegate; single-use helpers whose name says no more
  than their body; checks for states the types or an internal caller already exclude;
  the same validation repeated across internal layers; broad catches and
  success-shaped fallbacks that hide failure; retries, timeouts or circuit breakers
  around local work, or nested retry layers; caching without a measurement; one error
  logged at every layer; comments and docstrings that restate the code.

Evidence for each candidate: the population count (implementations, callers,
non-default arguments, importers, values across deploy configuration) with tool,
scope and file count reconciled with `git ls-files`, covering tests, reflection and
string dispatch, dependency injection, generated code, entry points (CLI, handlers,
plugins, serverless), public exports and external consumers; history (`git log -S`)
showing whether the form shipped or the value ever varied; the lines and dependencies
removable; and the concrete cost. Dead-code, unused-dependency and clone tools yield
candidates, not findings; token clone detectors miss renamed copies, so a low
duplication figure proves nothing.

Never flag: validation at trust boundaries (user input, network, files,
deserialization, environment, IPC) — generated code misses these more often than it
repeats them; exported, published or persisted contracts and their documented
migration windows; test seams that tests use; real plural implementations, including
fakes and plugins; audit, security and compliance logging; caching, retries or
low-level code backed by a linked measurement, incident or SLO; decisions recorded in
an ADR (report a mismatch with the ADR instead); refactoring and tests that make
change easier; a single smoke test or assertion self-check. When nobody can say why
code exists, check its history and owners before proposing removal.

Severity follows consequence: most cuts are low or medium `maintainability`.
Duplicated logic that can diverge on money, authorization, tenancy or time is a
correctness finding; a swallowed error is correctness or reliability; an unresolvable
dependency is security.

## Tests and verification quality

Check assertion strength, observable behavior, unhappy paths, boundaries, state
transitions, retries, races, idempotency, time dependence, migrations, rollback,
compatibility, tenant and security isolation, service contracts, flakiness and test
isolation. Mocks must not remove the database, network or concurrency behavior the
defect lives in. Confirm tests are discovered and actually run. Detect flakiness by rerunning, in
random order with a recorded seed (`go test -shuffle=on -count=N`, `pytest -p
randomly`, Jest `--randomize`): a test that fails then passes on unchanged code is
flaky and cannot serve as an oracle; order dependence is its most common cause.
Flag tautological
tests (asserting what a mock returns, or only "not null"), mocks of internal
collaborators or of the unit under test, near-duplicate tests that differ by one
literal, and production code special-cased to pass a test.

Line and branch coverage locate gaps; they never prove correctness. Use focused
mutation testing, property-based tests, fuzzing, differential checks or fault
injection where they fit; keep counterexamples and seeds; inspect surviving mutants
and account for equivalent or invalid mutants and tool limits. Never write tests
that merely restate the implementation.

## Dependencies and supply chain

Inventory direct and transitive **resolved** versions, their runtime/build/test
roles, origins, licenses, support and end-of-life status, advisories, deprecated
APIs, provenance, integrity and install scripts, plus the unused, redundant or
unresolvable entries named under [Over-engineering and dead weight](#over-engineering-and-dead-weight). Verify the latest compatible and
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

The craft role ([`overhaul-craft`](.devin/agents/overhaul-craft.md)) examines the existing brief and
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

The boundary role ([`overhaul-boundary`](.devin/agents/overhaul-boundary.md)) owns the live graph of
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

## Public API and developer experience

For libraries, SDKs, CLIs, plugins and configuration consumed outside the repository:

- **Compatibility:** diff the exported surface against the last release with the
  ecosystem's checker (`gorelease` or `apidiff`, `cargo-semver-checks`, `griffe
  check`, api-extractor with publint and `attw`, japicmp); a breaking change needs a
  major version or a documented migration. `gorelease` does not fail on `v0`
  modules; report their breaking changes anyway.
- **CLI:** data on stdout and diagnostics on stderr; exit 0 only on success, with
  distinct non-zero codes for usage and runtime errors; `--help` and `--version`;
  machine-readable output where scripts consume it; no color when not a TTY or when
  `NO_COLOR` is set; never prompt without a TTY, and a flag for every prompt.
- **Errors:** name the cause and the offending value, the constraint, and how to fix
  it; no stack trace as the only message; never rely on color alone.
- **Configuration:** documented precedence (flags, environment, project, user,
  system); unknown keys rejected with a suggestion; a validate command or dry run.

Each claim gets an executed check: the compatibility diff, `cmd --json` parsing,
`NO_COLOR=1 cmd` containing no escape codes, a bad argument giving a non-zero exit
with empty stdout, golden tests on error text.

## Documentation

Inventoried docs are reviewed, not skipped. Check that install, setup and quick-start
commands run on a clean checkout; examples compile or run (Go `Example` tests,
doctests, `cargo test --doc`); generated references (CLI help, OpenAPI, config
schemas) match a fresh regeneration with no diff (`cog --check`, `mdox fmt --check`,
`git diff --exit-code`); documented flags, endpoints and settings exist in code and
the reverse; local links and anchors resolve (`lychee --offline`). Wrong docs that
send a user down a broken path are findings with the same severity rules as code.

## Operations and project-specific extensions

For every long-running service, check with a fault test where possible:

- **Signals:** structured logs carrying the trace or correlation ID from the inbound
  `traceparent` to outbound calls and jobs; RED metrics per endpoint and USE metrics
  per resource under stable OpenTelemetry semantic-convention names; no secrets,
  tokens or personal data in logs, spans or metrics; bounded metric cardinality.
- **Health:** liveness checks only the process, never a dependency; readiness checks
  what serving needs; a startup probe covers slow boot. Stop the database: readiness
  fails, liveness stays up.
- **Shutdown:** on SIGTERM readiness fails first, in-flight work drains and consumers
  stop fetching, all within the platform's grace period.
- **Timeouts and retries:** every outbound call has connect and total timeouts from
  the caller's deadline; retries are bounded, jittered, at one layer, only for
  idempotent work; queues and concurrency are bounded.
- **Configuration:** from environment or mounted files, never secrets in the
  repository or image; a missing required setting fails startup and names it.
- **Change safety:** rollback or roll-forward path, staged rollout, feature-flag
  kill switches with owners and removal dates, expand-and-contract migrations with
  lock and statement timeouts (squawk, `atlas migrate lint`), version skew between
  old and new instances.
- **Recovery:** backups proven by a restore that queries data against RTO and RPO;
  alerts on symptoms and SLO burn, each with a runbook; SLOs on user-visible
  behavior; CPU and memory requests and a memory limit.

Also check deployment order, configuration drift, resilience and diagnosability.
Add language- or domain-specific extensions when applicable: memory ownership and
FFI, native sanitizers, embedded constraints, GPU and frame budgets, numerical, data
and ML reproducibility, regulated or safety requirements. An unfamiliar domain is
`blocked` until researched ([`stack-profiles.md`](stack-profiles.md#unknown-stack-procedure)),
never silently labeled safe.
