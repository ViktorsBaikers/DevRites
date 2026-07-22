# Go authoritative workflow schema and quality research

Date: 2026-07-20

## Question

How should a Go CLI keep lifecycle and schema values authoritative, avoid
shotgun changes, decide between code and configuration, safely generate
derivatives, validate the model, and find dead or defective code?

This note uses primary or first-party technical sources. Recommendations marked
**Inference for DevRites** apply those sources to this repository.

## Executive recommendation

Use one typed, ordered Go registry as the authority for machine-readable
workflow facts. A phase definition should own its stable identifier, order,
required sections, aliases, and other phase-level policy. Consumers should ask
the registry instead of maintaining their own phase slices, maps, or switches.

Keep the registry in code, not runtime configuration: the lifecycle is versioned
application behavior and should change with tests and a release. Generate files
only where a second representation is genuinely required. Validate registry
invariants in one test and make regeneration drift a CI failure.

## 1. Model stable domain values as typed code

Go supports defined string/numeric types with methods, typed constants, and
related constant sets. The language specification explicitly shows defined
types combined with constants and methods; `iota` is available for sequential
integer sets ([Go specification: type definitions and constants](https://go.dev/ref/spec#Type_definitions),
[Go specification: `iota`](https://go.dev/ref/spec#Iota)). Protocol Buffers uses
the same broad shape when generating Go enums: a defined type, typed constants,
a `String` method, and lookup metadata derived from the schema
([Go generated code guide: enumerations](https://protobuf.dev/reference/go/go-generated/#enumerations)).

**Inference for DevRites:** retain `type Phase string` because serialized state
needs stable readable values, but make one ordered `[]PhaseDefinition` the
machine authority. Keep named `Phase...` constants so callers remain type-safe;
declare each serialized string only once. Derive or query all of the following
from the registry:

- known-phase validation;
- lifecycle order and progress position;
- required sections and proof/status requirements;
- legacy aliases used by migration;
- phase groups such as pre-build, proof-required, and terminal;
- user-facing labels where labels are truly metadata.

Do not replace several duplicated maps with one giant untyped map. A small
struct gives each fact a name and lets the compiler check field types. Do not
put unrelated constants such as filenames, parser field aliases, UI glyphs, and
timeouts into a global `constants.go`; centralize a fact at the narrowest package
that owns it.

## 2. Code versus configuration

The Twelve-Factor definition draws a useful boundary: configuration is what
varies between deploys, while internal application configuration that does not
vary between deploys is best kept in code
([Twelve-Factor App: Config](https://www.12factor.net/config)).

**Inference for DevRites:**

| Value | Authority | Reason |
| --- | --- | --- |
| Phase IDs, order, required artifacts, legal aliases | Typed Go registry | Versioned product semantics; must change with tests and release |
| Workspace state such as current phase and next action | Validated workspace data | Per-workspace mutable state |
| Credentials, external service endpoints, machine-specific paths | Environment/CLI config | Deployment or machine variation |
| User policy that is intentionally supported as an override | Explicit config with defaults and validation | Product feature, not an accidental escape hatch |

Making the phase arc runtime-configurable would weaken compatibility guarantees
and move failures from compile/test time to user runtime. Add configurability
only when users have a real supported need to define a different lifecycle.

## 3. Generate derivatives, never another authority

The Go tool documents that `go generate` runs explicit author-defined commands,
is **never** run automatically by `go build` or `go test`, and generated source
should carry the standard `// Code generated ... DO NOT EDIT.` marker
([`go generate` command documentation](https://pkg.go.dev/cmd/go#hdr-Generate_Go_files_by_processing_source)).
The Go team also states that generated, tested source needed by clients must be
checked into the repository
([Go blog: Generating code](https://go.dev/blog/generate)).

**Inference for DevRites:** start without a generator when all engine consumers
can import/query the Go registry. Add a minimal stdlib generator only for
representations that cannot consume Go directly, such as a compact machine
manifest shipped in the skills pack or a lifecycle table in documentation.

If generation is added:

1. The generator reads/imports the authoritative registry; it must not parse a
   copied list.
2. Output is deterministic, formatted, marked generated, and checked in when
   release consumers need it.
3. CI runs generation and fails when `git diff --exit-code` is non-zero. This is
   necessary because Go deliberately does not run generators during tests or
   builds.
4. Humans edit only the registry. Generated files say how to regenerate.

Protobuf demonstrates when a separate schema language is justified: one schema
generates typed code and lookup maps for multiple representations/languages
([Protocol Buffers Go generated-code guide](https://protobuf.dev/reference/go/go-generated/)).
DevRites does not need Protobuf, CUE, or JSON Schema merely to centralize a Go
enum. Consider one only if the lifecycle becomes a cross-language public data
contract. JSON Schema is specifically a language for annotating and validating
JSON structure and constraints
([JSON Schema specification](https://json-schema.org/specification)).

## 4. Validate the authority, not every consumer independently

The official Go testing guidance recommends table-driven tests when many cases
share the same logic, avoiding copy-pasted tests
([Go Wiki: Table-driven tests](https://go.dev/wiki/TableDrivenTests)). Go fuzzing
is built into the toolchain and is intended to find edge cases humans miss
([Go fuzzing documentation](https://go.dev/doc/security/fuzz/)).

**Inference for DevRites:** add one registry-invariant test that iterates every
definition and fails on:

- an empty or duplicate phase ID;
- an empty or duplicate alias;
- duplicate or non-monotonic order;
- an unknown required section;
- a transition/group reference to an unknown phase;
- a terminal phase with a successor;
- a phase that should require proof/status but whose metadata says otherwise.

Then keep small behavior tests at public boundaries: readiness, migration,
progress, stop gates, and snapshots. These should consume the registry rather
than restating the whole lifecycle as expected data. A single explicit expected
phase list is still useful as a contract test; duplication in a test oracle is
intentional because deriving both input and expected output from the same table
cannot detect an accidentally missing phase.

Fuzz the untrusted text boundary (cursor/state parsing), not the static registry.
Useful properties are: parsing never panics, unknown phases remain errors,
canonical writes round-trip, and legacy aliases normalize predictably.

## 5. Prevent shotgun changes

Centralization works only if consumers cannot quietly recreate the same facts.
The minimum enforcement is architectural rather than magical:

1. Put the registry and queries in the owning `state` package.
2. Export query functions or read-only copies, not mutable global maps/slices.
3. Replace lifecycle switches and local `[]Phase` values with registry queries.
4. Allow consumer switches only for genuinely phase-specific behavior that
   cannot be represented as metadata; require a default that rejects unknown
   phases.
5. Search CI or a focused test for duplicated full lifecycle literals in code
   and machine templates. Do not ban phase words in prose; documentation must be
   readable and is not executable policy.

This is deliberately smaller than building a generic workflow framework. One
data table plus a few queries solves the observed change-amplification problem.

## 6. Static analysis and dead-code audit stack

No single tool proves correctness. The Go `vet` documentation says it finds
suspicious constructs missed by the compiler but uses heuristics and is not a
firm correctness indicator ([`go vet` documentation](https://pkg.go.dev/cmd/vet)).
The Go team's `deadcode` command builds a whole-program call graph and reports
unreachable functions. Its `-test` flag includes test executables, and its docs
warn that results are specific to one `GOOS`/`GOARCH`/build-tag configuration
([`deadcode` documentation](https://pkg.go.dev/golang.org/x/tools/cmd/deadcode),
[Go blog: Finding unreachable functions with deadcode](https://go.dev/blog/deadcode)).
Staticcheck provides additional correctness and unused-code checks, including
the `U1000` family ([Staticcheck checks](https://staticcheck.dev/docs/checks/)).
`govulncheck` reports known vulnerabilities that are actually reachable through
the program's call graph ([Go vulnerability management](https://go.dev/doc/security/vuln/)).

**Recommended audit order:**

```text
go test ./...
go vet ./...
staticcheck ./...
deadcode -test ./...
govulncheck ./...
```

Run race-enabled tests for concurrency changes and a bounded fuzz job for state
parsers. Run `deadcode` for every supported build configuration, including the
supported Unix and Windows targets, before deleting reported code. Review each
finding; generated files, reflection, build tags, assembly, and external entry
points can change reachability assumptions.

## 7. Concrete DevRites adoption sequence

1. Characterize current behavior with the existing test suite.
2. Introduce `PhaseDefinition` and the single ordered registry in
   `engine/internal/state`; keep serialized phase strings stable.
3. Move `KnownPhase`, order, requirements, aliases, groups, and labels behind
   registry queries.
4. Migrate consumers subsystem by subsystem and delete their local lifecycle
   tables in the same change.
5. Add the invariant/contract tests and a generated-drift check only if a
   generator is actually introduced.
6. Run the audit stack, delete only confirmed dead code, and rerun all supported
   platform tests.

## Decision summary

- **Choose:** one typed Go metadata registry, narrow query API, validation test.
- **Do not choose yet:** runtime-configurable lifecycle, a general workflow DSL,
  or a new schema/code-generation dependency.
- **Generate only:** unavoidable cross-format derivatives, with deterministic
  output and CI drift detection.
- **Audit continuously:** compiler/tests, vet, Staticcheck, deadcode by supported
  build configuration, govulncheck, and focused fuzzing.
