# Spec grammar — testable requirements, parseable by tooling

Acceptance criteria are the contract the seal checks ([`testing.md`](testing.md),
[`code-review.md`](code-review.md)). Prose criteria work, but a requirement written as free
text is graded by a human reading carefully — and an ambiguous one ("handle errors
gracefully") slips past every gate because nothing can falsify it. This rule adds an
**optional, recommended structure** that makes a behavioral requirement testable by
construction and parseable by a deterministic linter, so the spec phase catches a malformed
requirement before `$rite-define` plans against it.

It is the grammar counterpart to [`testing.md`](testing.md): testing says *prove every
behavior*; this says *write each behavior so it can be proven*.

## Progressive rigor — when to use the structured form

Match the rigor to the stakes; don't pay grammar ceremony for a one-liner.

- **Simple / routine change** — the flat checklist form is correct and stays. One bullet per
  criterion, each tagged with an `AC-###` id:
  ```markdown
  ## Acceptance criteria
  - [ ] AC-001: export returns a CSV with a header row
  - [ ] AC-002: an empty dataset returns 204, not an empty 200
  ```
- **Behavioral / high-risk / cross-boundary requirement** — auth, data model, state machine,
  public API, money, a migration, anything with non-obvious edge cases — use the structured
  **Requirement / Scenario** grammar below. The act of writing the WHEN/THEN forces the edge
  cases into the open at spec time, where they're cheapest to resolve.

A spec mixes both: most criteria stay flat bullets; the two or three that carry real risk get
the structured treatment. **Absence of structured requirements is never a failure** — the
validator no-ops on a flat-bullet spec, the same discipline as the principles gate
([`principles.md`](principles.md)).

## The structured form

```markdown
### Requirement: Session tokens expire after inactivity
The system SHALL reject any session token older than 15 minutes of inactivity.

#### Scenario: token past the inactivity window
- [ ] AC-004: **WHEN** a request presents a token whose last use was > 15m ago
            **THEN** the request is rejected with 401 and the token is revoked

#### Scenario: token within the window
- [ ] AC-005: **WHEN** a request presents a token last used < 15m ago
            **THEN** the request is served and the last-use timestamp advances
```

The rules the validator enforces (`devrites-engine spec-validate`):

- **`### Requirement: <name>`** — a level-3 heading. Its block MUST carry a **SHALL** or
  **MUST** statement (in the header or the body) describing the core behavior. Keep the name
  descriptive and under ~50 characters.
- **Header identity is the key.** Requirement names are **unique** within a spec — tooling (and
  any future spec sync) matches a requirement by its header text, so two requirements can't
  share a name. Renaming a requirement is a remove + add, not an in-place edit, so the change
  is visible.
- **`#### Scenario: <name>`** — every requirement owns **at least one**. A requirement with no
  scenario is an assertion no test can target.
- **WHEN / THEN** — every scenario states a trigger (**WHEN**) and an observable outcome
  (**THEN**); chain extra conditions with **AND**. Keywords are uppercase so they parse
  unambiguously. A scenario missing either half isn't falsifiable.

## Behavior first — WHAT, not HOW

A spec requirement describes observable behavior, not the implementation that delivers it
(the spec stays technology-agnostic — see `rite-spec/reference/spec-template.md`). "**THEN**
respond 401" is behavior; "**THEN** call `AuthGuard.reject()`" leaks the design and locks the
plan before `$rite-define` has chosen it. Name the input, the observable output, and the state
change — not the function, class, or library. Implementation belongs in `plan.md`, not the
requirement.

Acceptance criteria are **surface-anchored**: they observe the outermost surface the user or
system can see. If the feature is an API contract, the THEN names the response shape/status; if it
is a UI flow, the THEN names the visible state. Internal rows, helper calls, and emitted logs can
support proof, but they are not the criterion unless the spec's surface is explicitly internal.

## How it composes (no new gate, no duplication)

- **`AC-###` ids nest inside scenarios.** Tag each scenario's checkable line with an `AC-###` id
  exactly as the flat form does. The same `devrites-engine check-acceptance` grades it against `seal.md` at
  the end of the lifecycle — this grammar checks the requirement's **shape**, that script
  checks its **proof**. The two compose; neither replaces the other.
- **`$rite-prove` and `$rite-review` get concrete hooks.** Each scenario is one behavior to
  prove and the test analyst's unit of "is this actually covered" — the WHEN/THEN maps
  straight to an arrange/assert ([`testing.md`](testing.md)).
- **The spec-quality checklists still apply.** The grammar makes a requirement *parseable*; the
  `checklists/<domain>.md` "unit tests for English" still score whether it's *complete and
  measurable* ("is 'prominent' quantified?"). Structure doesn't excuse vagueness inside a THEN.

## Outcome metrics are not acceptance criteria

A **buildable** acceptance criterion names an observable behavior a slice delivers and a test
proves — "export returns a CSV with a header row", "a token past 15m is rejected with 401". An
**outcome metric** names a post-launch business result — "reduce support tickets 50%", "lift
signup conversion 3pts", "cut p95 latency in prod under real traffic". The distinction is not
pedantic: it decides what the coverage gate is allowed to check.

Tag only buildable criteria with an `AC-###` id. An outcome metric carries no id and lives under
a separate **`## Success metrics`** heading:

```markdown
## Acceptance criteria
- [ ] AC-001: export returns a CSV with a header row      # buildable — a slice + a test

## Success metrics                                        # outcome — no AC id, not slice-mapped
- Support tickets about export drop by half within a quarter
```

Why the split earns its place: an outcome metric tagged `AC-###` poisons the coverage gate in
both directions. `devrites-engine analyze` flags it CRITICAL forever — no slice can `Satisfies:` a KPI —
so a genuinely complete plan reads as uncovered. And `devrites-engine check-acceptance` can never mark it
proven at `$rite-seal`, because no end-to-end test run inside the feature observes a quarter of
production traffic. The metric still matters — it's *why* the feature exists — but it belongs to
intent (`brief.md` / `spec.md` overview), not to the criteria the lifecycle mechanically proves.
The load-bearing test: **can one slice make this true and one test show it?** If no, it's a
success metric, not an acceptance criterion. (`analyze` scopes its coverage count to `AC-###`
ids, so an untagged `## Success metrics` line is correctly ignored by the gate.)

## The validator — `devrites-engine spec-validate`

Deterministic, zero-token. Run it on the workspace (or a spec path) at the spec readiness gate:

```bash
devrites-engine spec-validate .devrites/work/<slug>
```

Exit codes: `0` valid (or no structured requirements — nothing to lint), `1` grammar
violation(s) with `file:line` locations, `2` usage, `5` missing `spec.md`. A non-zero exit is a
blocking spec-readiness failure: fix the requirement, never soften the check. It runs as part
of `$rite-spec`'s readiness gate and is safe to re-run any time the spec changes.

## Delta form — when the capability ledger already holds this behavior

The Requirement / Scenario block is the unit the **capability ledger** stores — the living
`.devrites/specs/<capability>/spec.md` record of *what the system does now*, folded in on ship
([ledger reference](../../../rite-ship/reference/ledger.md)). When a feature changes behavior a
ledger already describes, write its spec as **deltas against the ledger** instead of a flat
snapshot, so the change — not merely the end state — is explicit and the fold is unambiguous.

Group the structured `### Requirement:` blocks under three H2 sections, each tagged with the
capability it folds into. One feature spec MAY carry deltas across several capabilities:

```markdown
## ADDED Requirements — capability: theming
### Requirement: Dark mode honors the system preference
The system SHALL default to the OS colour-scheme on first load.
#### Scenario: no stored preference
- [ ] AC-010: **WHEN** a first-time visitor loads the app **THEN** the theme matches the OS setting

## MODIFIED Requirements — capability: settings-ui
### Requirement: Settings exposes a theme control
<the full new version of the requirement — not just the diff>

## REMOVED Requirements — capability: theming
### Requirement: Theme is hard-coded to light
Removed — superseded by system-preference detection.
```

Fold semantics on `devrites-engine ledger sync` — **ADDED** appends, **MODIFIED** replaces the
same-named requirement, **REMOVED** deletes it. Matching is by **header identity** (the rule
above: names are unique and a rename is a remove + add), so the section MUST use the exact ledger
header text for MODIFIED / REMOVED.

- **The `— capability: <name>` suffix is the fold target.** Omit it and the fold defaults to the
  feature slug — correct for a single-capability feature; name it explicitly when a feature spans
  more than one, or when the capability differs from the slug.
- **Pick the kind against the current ledger, not from memory.** `devrites-engine ledger show
  <capability>` prints what it holds. A real change marked ADDED yields two competing
  requirements; new behavior marked MODIFIED has nothing to replace.
- **Greenfield stays flat.** A capability with no ledger entry is all-new — write plain
  `### Requirement:` blocks with no delta H2; the first `ledger sync` seeds the capability as if
  every block were ADDED. Never pay delta ceremony for behavior that has no prior record.

Validate the classification deterministically at the spec gate — `--against` points the linter at
the ledger root so it flags an ADDED that already exists or a MODIFIED/REMOVED that doesn't:

```bash
devrites-engine spec-validate .devrites/work/<slug> --against .devrites/specs
```

Exit `1` on a mismatch (a blocking spec-readiness failure); `0` when the deltas reconcile. The
plain-grammar checks (SHALL, WHEN/THEN, unique headers) run in the same pass — delta H2 headers
are transparent to them, so a flat spec and a delta spec lint identically.
