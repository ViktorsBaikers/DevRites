# Review: stack profiles

> Applies when: a reviewed diff crosses a runtime, version, or language line that the
> extension rows in [`README.md`](README.md) cannot see. Load with
> [`default.md`](default.md) and every matching language checklist; a card adds
> version-aware checks and replaces no checklist.

## Triggers and non-triggers

Build a stack card for each component when the diff has:

- one language running in two or more runtimes or environments (browser vs Node or
  edge worker, UI vs server components, managed vs native interop);
- a language, DSL, template, or shader with no row in `README.md`;
- an embedded-language region: SQL, GraphQL, or HTML inside a string or template;
- two majors of one framework (keyed per package, never per repository);
- a serialized value crossing between different stacks (HTTP/JSON, queue, DB column, IPC).

Skip the card when a listed checklist covers both the file's language **and** its
runtime/version, and the diff stays in one runtime. An unlisted language always
triggers: `default.md` alone is a disclosed `default-only: <reason>` gap in `Basis`,
never silent coverage. **Failing case:** a diff touching `lib/app_web/live/*.ex` gets
`default.md` only and the account reports no gap.

## Component card

```text
component: <id> · paths/ranges · lane: frontend | backend | boundary | other
runtime/environment: <browser | node<major> | edge | server component | native | ...>
language: <name + dialect + version> · evidence: <resolved lockfile entry | toolchain output>
framework: <name + major + mode (router, compiler, build mode)>
boundaries: storage · transport · trust
producers → consumers
native checks: <command> · available | unavailable | not run
confidence: high | medium | low · unknowns: <list>
```

Version evidence comes from the resolved lockfile entry or toolchain output, never
from the root manifest's name or range. A range is not a version; an unresolved
version is an unknown on the card, not a default.

## Composition

A card's checks are assembled from: shared review contract ([`code-review.md`](../code-review.md) +
`default.md`) + lane + language/version + runtime/platform + framework/version +
storage/transport/trust + project conventions + risk overlays. This is composition,
not authority: conflicts resolve under [`core.md`](../core.md#precedence) § Precedence.
A project convention selects form; it never overrides language semantics or excuses a
proven vulnerability. Add a risk overlay only when the diff shows the boundary it covers.

## Required profile fields

- **Identity/applicability:** id, component, lane, version predicates, non-triggers,
  unresolved facts.
- **Rule provenance:** source URL, version or section, retrieved date, and type
  `official-semantics` | `local-policy` | `heuristic`. A heuristic-only check stays a lead.
- **Concrete checks:** each with a failing example and a valid idiom that must stay quiet.
  A language name with no sourced check is not a profile.
- **Tool map:** installed version, edition, and processed set (what the tool analyzed).
- **Dynamic proof:** the native test or oracle that exercises each check.
- **Boundaries:** producers, consumers, embedded or generated regions.
- **Gaps/blockers, cache key** (lockfile entries + config + tool versions) **and
  invalidation** conditions.

## Six risk families

Per active profile, record each family as `rules covered` | `manual semantic review
required` | `tool coverage` | `blocked evidence`, with paths:

1. **type/value** — absence, coercion, enums, numeric precision, serialization;
2. **state/resources** — lifetimes, disposal, handles, error propagation;
3. **execution/concurrency** — async ordering, cancellation, shared state;
4. **framework** — lifecycle, rendering/server boundary, routing, auth hooks, caching;
5. **build/deploy** — codegen, targets, flags, module resolution, prod-only behavior;
6. **proof/ops** — native oracles, analyzer limits, logging, version skew.

A family with no recorded state is unreviewed, never clean. A typecheck covers
type/value at compile time only; it proves no serialized value, authorization, or race.

## Unknown-stack procedure

1. Reuse research whose version predicate still matches the cache key; never repeat
   searches per file.
2. Read the component's code, config, and tests; list the unknowns.
3. A reviewer leaf reads lockfiles and installed sources itself and never spawns
   agents or skills (a nested spawn voids its result). It returns each open version or
   semantics question as a `gap` row. The root then takes version facts from primary
   versioned docs via `devrites-source-driven` and sends a bounded area question to
   `devrites-evidence-scout`. Behavior claims cite the installed version's source or docs,
   not the latest release.
4. Write a provisional card: sourced checks, each with one failing and one passing example.
5. The root has a different fresh reviewer (`devrites-doubt`) challenge version fit and
   tool claims. Until that challenge passes, the card's probes are leads, never findings.
6. An unsupported language, unknown version, or missing compiler/toolchain is an
   assurance gap (`Outcome: gap` when the row is required): never N/A, never a
   silent `default.md`-only pass.

## Tool coverage

A tool row counts only under
[`verification-methods.md`](../verification-methods.md#tool-coverage-states): its
processed set, not its exit code, bounds what it covers. A tool that does not support
the component's language, dialect, or version covers nothing there, even on exit 0;
an embedded region the tool dropped stays unreviewed.

## Cross-layer contracts

When a card names a serialized crossing, add one contract record: producer, consumer,
serialized shape, and the round-trip proof the serialized-boundary rule in
[`testing.md`](../testing.md) requires. A type shared by both sides proves compile-time
agreement only. Lane ownership of the record follows
[`parallel-dispatch.md`](../../parallel-dispatch.md) § Engineering lanes.

## Where cards live

Record cards in the reviewer account: `Basis` names `profile:<id>` or
`default-only: <reason>`. When a later phase reuses a card, it lives in the workspace
artifact that phase already writes (e.g. `evidence.md`); there is no new artifact
type. Reuse only while the cache key is unchanged; a changed lockfile entry, config,
or tool version invalidates the affected checks and the evidence built on them.
Promotion into shared guidance happens only through a reviewed canonical change
(`$rite-learn`), never by a reviewer editing its own instructions.

## Worked examples

**(a) One language, three runtimes.** `web/Checkout.tsx` (browser, React 19),
`api/orders.ts` (Node handler), `db/orders.sql` (PostgreSQL) → cards
`ts-browser-react19`, `ts-node-handler`, `sql-postgres` plus one contract record for
the order payload: a round-trip test asserts `createdAt` arrives as a string parsed at
the boundary and `total` keeps decimal precision. Whether the driver returns `numeric`
as string or float is a version fact (verify installed version). **Failing case:** one
"TypeScript reviewed" receipt, or the green shared-type typecheck cited as contract proof.

**(b) Embedded languages.** SQL in a Go string passed to a query call, GraphQL in a TS
tagged template → each region is its own range with its own dialect on the card. A
scanner that processed the host file covers the host language only; region rows stay
`unverified` until read. **Failing case:** "scanner clean" cited for injection in the
SQL string.

**(c) Two framework majors.** `apps/admin` on Next 14 (pages router, React 18) and
`apps/web` on Next 15 (App Router, React 19) → one card per package from each
package's resolved lockfile entry. A `[react≥19]` or `[next≥15.x]` probe applies to
`apps/web` only; Next 15 default changes are version facts (verify installed version).
**Failing case:** a React-19-only finding raised against `apps/admin`.

**(d) Unlisted stack.** `lib/app_web/live/*.ex` → the trigger fires; the card takes
Elixir/Phoenix versions from `mix.lock`; the root has `devrites-evidence-scout` read the matching
versioned docs; probes stay leads until `devrites-doubt` challenges them. The native
oracle is the project's sanctioned compile/test command; an editor's `0` diagnostics
before the project compiles is `unavailable`, not clean.

**(e) Unsupported scanner.** A static analyzer exits 0 with no findings on a service
whose language version it does not support (verify the installed edition's language
table) and reports no processed set → state `unsupported`; it covers nothing and
closes no prior finding. **Failing case:** "SAST clean" recorded for that service.
