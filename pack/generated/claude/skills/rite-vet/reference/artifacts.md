# `eng-review.md` + `test-plan.md` templates + fold-back rules

`eng-review.md` records the review; `test-plan.md` is the build-readable coverage target.
Fold findings into `plan.md`/`tasks.md`: review prose alone does not change the build.

## `eng-review.md`: the record
Write to `.devrites/work/<slug>/eng-review.md`. If one exists for the slug, **update** it, don't
clobber.

```markdown
# Eng review: <slug>
Vetted: <iso>   Cross-model: ran (codex) | off
DevRites contract: devrites.readiness-artifacts.v2
Implementation readiness: <READY | NEEDS CLARIFICATION | NEEDS REPLAN>
Readiness inputs SHA-256: <exact value from `devrites-engine readiness-digest engineering <slug>`>

## 1. Depth
light | full (<trigger that escalated it>)     # every plan is vetted — light or full, never skipped (see depth.md)

## 2. Scope challenge
- What already exists (reuse vs rebuild): <findings — each reuse the plan should adopt>
- Minimum diff: scope accepted as-is | trimmed <n items, listed in NOT-in-scope>
- Complexity: ok | smell (<n files, m new services) → asked → <reduce|proceed, why>
- Built-in / completeness / distribution: <flags, with source citations>

## 2a. Build-entry preflight
| Gate | Command + cwd | Tool/version | Prerequisite owner | Full provenance inputs | Fixture/smoke | Verdict |
|---|---|---|---|---|---|---|
| <test/browser/package> | <command · cwd> | <version> | <owner + item> | <tool/manifest/lockfile/config SHA-256> | <what ran> | pass/fail/blocked |

## 2b. Implementation readiness
| Surface | Requirement/decision | Boundary/wiring | Slice | Proof | Verdict |
|---|---|---|---|---|---|
| <journey/data/interface/ops> | <REQ/AC + coverage row> | <contract + key link> | <SLICE-###> | <test-plan row> | ready/gap |

Inventory/currentness: <pass/gaps> · slice order/independence: <pass/gaps> ·
UX/spec/architecture: <pass/n-a/gaps> · operations/rollout/rollback: <pass/n-a/gaps>

## 3. Axis findings (floor-gated)
| Axis | Floor band | Findings (sev · confidence) |
|---|---|---|
| Architecture | strong/adequate/thin/broken | [Critical](9) … |
| Plan code-quality | … | … |
| Test-coverage design | … | see test-plan.md |
| Performance | … | … |
Suppressed (confidence ≤4, unverified): <count — one line each>

## 4. Failure modes
| New codepath | Realistic failure | Test? | Handling? | Silent? | Verdict |
|---|---|---|---|---|---|
| <path> | timeout / nil / race / stale | y/n | y/n | y/n | ok / **CRITICAL gap** |

## 5. Parallelization
Sequential — no opportunity   # OR the dependency table + lanes + order + conflict flags

## 6. Reviewer loop
- iter 1: <findings → resolution> · iter 2: … (≤3)
- Cross-model (if --cross-model): <overlap / unique findings — informational until approved>
- Final: floor = <band>; unresolved → <none | blocking qid>

## 7. Completion summary
- Scope: accepted | reduced   · Architecture: <n>   · Code-quality: <n>
- Coverage: <x/y> planned, <n> gaps, <n> regressions (Critical)
- Build entry: preflight <pass/fail/blocked>; action-time checkpoints <none/list>
- Failure modes: <n> mapped, <n> critical gaps
- NOT in scope: written   · What already exists: written
- Plan: hardened in place | <n> deltas via Spec Drift Guard
```

## `test-plan.md`: the build-readable coverage target
Write to `.devrites/work/<slug>/test-plan.md`. `/rite-build` (the slice-wright) reads it to write
tests alongside the code; `/rite-prove` walks it against `evidence.md`. Keep it about *what to
test and where*, not implementation detail.

```markdown
# Test plan: <slug>
From /rite-vet on <iso>. Runner + conventions: <detected framework + command>.
DevRites contract: devrites.readiness-artifacts.v2

## Build-entry preflight
| Gate | Command | Cwd | Expected | Prerequisites | Provenance to recapture |
|---|---|---|---|---|---|
| <test/browser/package> | <exact command> | <path> | <exit/output> | <service/browser/credential owner> | <tool version + full SHA-256s> |

Link any disposable parser fixture. Build evidence recomputes every digest; preflight values
are not post-build proof.

Commands in this durable artifact are portable repository commands: no RTK or local shell
aliases, user-specific absolute paths, or temporary proof trees. A runtime packet may name
an execution adapter, and evidence may record the portable and executed forms.

## Coverage diagram
<the ASCII code-paths + user-flows diagram from review-axes.md, with COVERAGE / GAPS / REGRESSIONS line>

## Per-gap test requirements
| ID | Path / flow | Test file (match conventions) | Asserts (input → expected) | Kind | Slice | Priority |
|---|---|---|---|---|---|---|
| T1 | <path> | <path/to.test> | <empty list → 200 + []> | unit / E2E / eval | <slice> | P1 / **Regression-Critical** |

## Interaction inventory (UI slices — every element + flow gets an asserting test)
Enumerate every interactive element and user flow the feature exposes; one row each, none
skipped. Level per `testing.md` "Completeness": elements/fields → unit/component, critical
journeys → one E2E (never one-per-field). `/rite-build` must cover every row; `/rite-prove`
fails any row with no passing result — an unverified element is a NO-GO, like an unproven criterion.
```markdown
| Element / flow | Kind (field/checkbox/select/radio/toggle/button/link/flow) | Level | Test file | Asserts (action → expected) |
|---|---|---|---|---|
| email field | field | unit | <form.test> | invalid → error; valid → accepted |
| login | flow ★★★ | E2E | <login.e2e> | submit → dashboard |
```
Omit the whole section only for a slice with **no** interactive surface (pure backend/logic).

## Acceptance → test map
- <spec acceptance criterion> → <T1, T3>   # every criterion maps to ≥1 test
```

## Parallelization table (in `eng-review.md` §5 when there's an opportunity)
```markdown
| Step / workstream | Modules touched (dir-level) | Depends on |
|---|---|---|
| API endpoints | controllers/, serializers/ | — |
| Worker | jobs/, lib/queue/ | API endpoints |

Lanes:  A: API → Worker (sequential, shared lib/)   ·   B: migration (independent)
Order:  launch A + B in parallel worktrees → merge → C
Conflicts: Lanes A and B both touch models/ — sequential or coordinate.
```

## Fold-back: the part the build follows
Vet *is* the plan-hardening phase, so behavior-preserving refinements are written **directly**;
acceptance/behavior changes route through the **Spec Drift Guard**.

- **Write directly into `plan.md` / `tasks.md`** (single canonical writer: you, not the reviewer):
  - `plan.md` §Scope boundaries ← "NOT in scope" items.
  - `plan.md` §Architecture decisions ← reuse-over-rebuild calls + named failure scenarios.
  - `plan.md` §Dependency graph / §Implementation order ← the parallel lanes + any ordering fix
    (e.g. refactor-before-feature split).
  - `plan.md` §Complexity & deviations gate ← any deviation the §0 challenge surfaced + its justification.
  - `tasks.md` slices ← added test requirements (point at `test-plan.md`), added error-handling /
    failure-mode coverage, tightened slice scope, a split of a refactor+behavior slice into two.
    Adjust a slice's `Gate:` upward when vet reveals higher stakes (e.g. an unflagged migration).
    For an eligible architecture competition, set the canonical `Forge`, `Forge strategies`,
    and `Forge scorecard` fields only after the acceptance→test map is final; otherwise keep
    `no` / `none` / `none`.
  - Re-run the `plan.md` Readiness gate after edits; it must still pass.
- **Route through the Spec Drift Guard** (record `drift.md` + a recorded decision, then `/rite-plan
  repair` for any structural reslice) for anything that **changes an acceptance criterion, product
  behavior, or the spec**, including a scope reduction that drops a criterion. A folded change with
  no recorded decision is the batch-dump failure.
- **`decisions.md`:** one ADR per material call: `context · decision · why-not-the-alternative ·
  what-would-change-it`.
- **`assumptions.md`:** every "we'll probably need X" demoted to an explicit assumption-to-verify,
  never smuggled into scope as a fact.

## Write discipline
- **Single canonical writer.** The skill writes `eng-review.md` / `test-plan.md` and edits
  `plan.md` / `tasks.md`; `devrites-plan-reviewer` is **read-only** and only returns findings + bands.
- **No silent scope.** A plan refinement that grows acceptance without a recorded decision, or a
  deferral that drops a criterion without the Guard, is a defect, not a convenience.
- Add `eng-review.md` + `test-plan.md` to the workspace between `tasks.md` and `state.md`.
  `/rite-build` (the slice-wright) and `/rite-prove` **read `test-plan.md`** as the coverage
  target; `/rite-review` and `/rite-seal` may consult `eng-review.md` for the failure-mode +
  scope record.
- Write `test-plan.md` and finish all fold-back edits and rechecks first. If this pass changed
  `brief.md`, `spec.md`, `decisions.md`, `assumptions.md`, or `questions.md`, revalidate the
  affected decision-coverage rows, assumption audit, residual uncertainty, and closed gates.
  Partial/Missing, an unowned material assumption, or an open blocking/escalating question routes
  `/rite-clarify`/HITL; never hide it with a new digest. Once re-closed, run
  `devrites-engine readiness-digest coverage <slug>` and replace the complete
  `Coverage inputs SHA-256` line in `decision-coverage.md`. Only then run
  `devrites-engine readiness-digest engineering <slug>` and copy its complete
  field line into `eng-review.md`. Write
  `DevRites contract: devrites.readiness-artifacts.v2` exactly once in both artifacts
  only after their current-contract review is complete. The build gate rejects a stale digest,
  placeholders, non-passing preflight/readiness rows, empty tables, or an
  acceptance criterion with no test mapping.
