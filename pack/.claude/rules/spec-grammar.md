# Spec grammar — testable requirements, parseable by tooling

Acceptance criteria are the contract the seal checks ([`testing.md`](testing.md),
[`code-review.md`](code-review.md)). Prose criteria work, but a requirement written as free
text is graded by a human reading carefully — and an ambiguous one ("handle errors
gracefully") slips past every gate because nothing can falsify it. This rule adds an
**optional, recommended structure** that makes a behavioral requirement testable by
construction and parseable by a deterministic linter, so the spec phase catches a malformed
requirement before `/rite-define` plans against it.

It is the grammar counterpart to [`testing.md`](testing.md): testing says *prove every
behavior*; this says *write each behavior so it can be proven*.

## Progressive rigor — when to use the structured form

Match the rigor to the stakes; don't pay grammar ceremony for a one-liner.

- **Simple / routine change** — the flat checklist form is correct and stays. One bullet per
  criterion, each tagged with an `[ACn]` id:
  ```markdown
  ## Acceptance criteria
  - [ ] [AC1] export returns a CSV with a header row
  - [ ] [AC2] an empty dataset returns 204, not an empty 200
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
- [ ] [AC4] **WHEN** a request presents a token whose last use was > 15m ago
            **THEN** the request is rejected with 401 and the token is revoked

#### Scenario: token within the window
- [ ] [AC5] **WHEN** a request presents a token last used < 15m ago
            **THEN** the request is served and the last-use timestamp advances
```

The rules the validator enforces (`spec-validate.sh`):

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
plan before `/rite-define` has chosen it. Name the input, the observable output, and the state
change — not the function, class, or library. Implementation belongs in `plan.md`, not the
requirement.

## How it composes (no new gate, no duplication)

- **`[ACn]` ids nest inside scenarios.** Tag each scenario's checkable line with an `[ACn]` id
  exactly as the flat form does. The same `check-acceptance.sh` grades it against `seal.md` at
  the end of the lifecycle — this grammar checks the requirement's **shape**, that script
  checks its **proof**. The two compose; neither replaces the other.
- **`/rite-prove` and `/rite-review` get concrete hooks.** Each scenario is one behavior to
  prove and the test analyst's unit of "is this actually covered" — the WHEN/THEN maps
  straight to an arrange/assert ([`testing.md`](testing.md)).
- **The spec-quality checklists still apply.** The grammar makes a requirement *parseable*; the
  `checklists/<domain>.md` "unit tests for English" still score whether it's *complete and
  measurable* ("is 'prominent' quantified?"). Structure doesn't excuse vagueness inside a THEN.

## The validator — `spec-validate.sh`

Deterministic, zero-token. Run it on the workspace (or a spec path) at the spec readiness gate:

```bash
bash .claude/skills/devrites-lib/scripts/spec-validate.sh .devrites/work/<slug>
```

Exit codes: `0` valid (or no structured requirements — nothing to lint), `1` grammar
violation(s) with `file:line` locations, `2` usage, `5` missing `spec.md`. A non-zero exit is a
blocking spec-readiness failure: fix the requirement, never soften the check. It runs as part
of `/rite-spec`'s readiness gate and is safe to re-run any time the spec changes.

## Forward note

The Requirement / Scenario block is deliberately the unit a persistent capability ledger would
store — so if DevRites later grows a living "what the system currently does" layer, today's
structured specs sync into it without a reformat. That layer is **not** shipped here; for now
the grammar earns its place purely as testable, lint-checkable acceptance criteria at the spec
phase.
