# `eng-review.md` + `test-plan.md` templates + fold-back rules

`eng-review.md` records review; `test-plan.md` drives Build. Fold accepted findings
into `plan.md`/`tasks.md`; prose alone never changes Build.

Authority: `.agents/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → fold technical topology; invalidate Vet/readiness; affected Vet before Build.
- `GUARD_AND_REPAIR` → no planning writes; Spec Drift Guard → Clarify → Plan repair → affected Vet.
- `BLOCKED_INPUT` → no planning writes; exact diagnostic; recover input; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## `eng-review.md`: the record

Write/update `.devrites/work/<slug>/eng-review.md`; never clobber.

```markdown
# Eng review: <slug>
Vetted: <iso>   Cross-model: ran (codex) | off
Implementation readiness: <READY | NEEDS CLARIFICATION | NEEDS REPLAN>
(`NEEDS REPLAN` is the eng-review field value; `NEEDS_REPLAN` with an underscore
is the autocomplete/reply-contract routing token — do not conflate.)
Readiness inputs SHA-256: <64 lowercase hex>

## 1. Depth
light | full (<trigger that escalated it>)     # every plan is vetted — light or full, never skipped (see depth.md)

## 2. Scope challenge
- Existing reuse: <findings adopted by the plan>
- Minimum diff: <accepted | trimmed n; list in NOT-in-scope>
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

Shared contract proof: <pass | gap: missing/one-sided/duplicated-contract/vague/non-consuming>
One-shot evidence completeness: <n/a | pass: action + retained artifact + bounds/sanitization + injective boundary map + per-seam/collision/cleanup fixtures | gap: exact missing proof>

## 3. Axis findings (floor-gated)
| Axis | Floor band | Findings (sev · confidence) |
|---|---|---|
| Architecture | strong/adequate/thin/broken | Critical (9) … |
| Plan code-quality | … | … |
| Test-coverage design | … | see test-plan.md |
| Performance | … | … |
Suppressed (confidence ≤4, unverified): <count — one line each>

## 4. Failure modes
| New codepath/boundary | Realistic failure | Partial/unknown effect | Recovery owner/path | Proof? | Silent? | Verdict |
|---|---|---|---|---|---|---|
| <path> | timeout / duplicate / race / stale | <state/effect> | <owner/action> | y/n | y/n | ok / **CRITICAL gap** |

## 5. Dependency safety
<declared order is safe | exact dependency/order correction>

## 6. Native reviewer accounts
- `devrites-plan-reviewer`: <findings + resolution | evidence-based No-findings>
- `devrites-strategy-reviewer`: <current Temper verdict | Not-applicable: low-stakes skip>
- `devrites-devex-reviewer`: <findings + resolution | No-findings | Not-applicable: no developer surface>
- Recheck: <not-needed | one narrow recheck and result>
- Cross-model (if --cross-model): <overlap / unique findings — informational until approved>
- Final: floor = <band>; unresolved → <none | blocking qid>

## 7. Completion summary
- Outcome: <one sentence; REQ/AC refs>
- Boundaries: <IN/OUT/must-NOT refs>
- Design/flow: <UI/n-a; chosen seams/why; happy/error refs>
- Delivery: <slice order + first>
- Proof/gates: <test-plan rows; justified action-time gates>
- Review: scope <accepted/reduced>; architecture <n>; quality <n>; coverage <x/y>,
  gaps <n>, Critical regressions <n>; failure modes <n>, Critical gaps <n>.
- Entry/records: preflight <pass/fail/blocked>; checkpoints <none/list>;
  NOT-in-scope + existing written; plan <hardened | n Guard deltas>.
```

## `test-plan.md`: the build-readable coverage target

Write/update `.devrites/work/<slug>/test-plan.md`. `$rite-build`'s slice-wright
writes tests from it; `$rite-prove` walks it against `evidence.md`. Specify
*what/where* to test, not implementation.

```markdown
# Test plan: <slug>
From $rite-vet on <iso>. Runner + conventions: <detected framework + command>.

## Build-entry preflight
| Gate | Command | Cwd | Expected | Prerequisites | Provenance to recapture |
|---|---|---|---|---|---|
| <test/browser/package> | <exact command> | <path> | <exit/output> | <service/browser/credential owner> | <tool version + full SHA-256s> |

Link any disposable parser fixture. Preflight values are planning evidence, not post-build proof;
Prove recaptures the observed command, candidate identity, and result.

Commands in this durable artifact are portable repository commands: no RTK or local shell
aliases, user-specific absolute paths, or temporary proof trees. Evidence records
the command actually executed.

## Consumptive action gates
| Action | Why one-shot/consumptive | Retained artifact | Bounds + sanitization | Boundary map + collision proof | Terminal-path fixtures | Cleanup survival | Retry authority |
|---|---|---|---|---|---|---|---|
| <exact approved action or n/a> | <one attempt/quota/state/evidence deletion> | <durable operator-controlled path/schema> | <size/cardinality + known/unknown/malformed handling> | <every emit seam -> stable boundary ID; per-seam injection + alias mutant rejected> | <success/known/unknown/hostile> | <assertion> | <none/fresh human authorization> |

Every behavioral row names a positive, discriminating assertion and the decisive output it
produces. A command or expected exit zero alone is not a behavioral assertion; static gates
prove only their named static criterion.

## Coverage diagram
<the ASCII code-paths + user-flows diagram from review-axes.md, with COVERAGE / GAPS / REGRESSIONS line>

## Per-gap test requirements
| ID | Path / flow | Test file (match conventions) | Asserts (input → expected) | Kind | Slice | Priority |
|---|---|---|---|---|---|---|
| T1 | <path> | <path/to.test> | <empty list → 200 + []> | unit / E2E / eval | <slice> | P1 / **Regression-Critical** |

## Interaction inventory (UI slices — every element + flow gets an asserting test)
Enumerate every interactive element and user flow the feature exposes; one row each, none
skipped. Level per `testing.md` "Completeness": elements/fields → unit/component, critical
journeys → one E2E (never one-per-field). `$rite-build` must cover every row; `$rite-prove`
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

For every row in `plan.md`'s `Shared contract proof`, the per-gap requirements name both
provider- and consumer-side asserting tests and state how each consumes the same canonical
artifact. Reference the plan row; do not reproduce its contract table here.

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

Vet hardens the plan. Use the marked action before fold-back.

- **Write directly into `plan.md` / `tasks.md`** (single canonical writer: you, not the reviewer):
  - `plan.md` §Scope boundaries ← "NOT in scope" items.
  - `plan.md` §Architecture decisions ← reuse-over-rebuild calls + named failure scenarios.
  - `plan.md` §Applicability and system ownership ← corrected topology/data/integration
    routing and each triggered standard's feature-specific output.
  - `plan.md` §Dependency graph / §Implementation order ← any dependency or ordering fix
    (e.g. refactor-before-feature split).
  - `plan.md` §Complexity & deviations gate ← any deviation the §0 challenge surfaced + its justification.
  - `tasks.md` slices ← added test requirements (point at `test-plan.md`), added error-handling /
    failure-mode coverage, tightened slice scope, a split of a refactor+behavior slice into two.
    Adjust a slice's `Gate:` upward when vet reveals higher stakes (e.g. an unflagged migration).
  - Re-run the `plan.md` Readiness gate after edits; it must still pass.
- **Reslice:** marked action only; no local predicate.
- **`decisions.md`:** one ADR per material call: `context · decision · why-not-the-alternative ·
  what-would-change-it`.
- **`assumptions.md`:** every "we'll probably need X" demoted to an explicit assumption-to-verify,
  never smuggled into scope as a fact.

## Write discipline

- **Single canonical writer.** The skill writes review/test-plan and edits plan/tasks;
  `devrites-plan-reviewer` is **read-only** and returns findings + bands.
- **No silent scope.** Growing acceptance without a decision, or dropping a criterion
  without the Guard, is a defect.
- Put `eng-review.md` + `test-plan.md` between `tasks.md` and `state.md`. Build/Prove
  read test-plan as coverage; Review/Seal may consult eng-review for scope/failures.
- Finish `test-plan.md`, fold-back, and rechecks first. Changes to `brief.md`, `spec.md`,
  `decisions.md`, `assumptions.md`, or `questions.md` require revalidating the
  affected decision-coverage rows, assumption audit, residual uncertainty, and closed gates.
  Partial/Missing, an unowned material assumption, or an open blocking/escalating question routes
  `$rite-clarify`/HITL. Re-read affected rows with current evidence; repeat the exact
  plan-reviewer and applicable strategy/DevEx after edits. Root writes READY only after
  the checklist. Normal readiness rejects missing/stale bindings; empty tables,
  placeholders, non-passing preflight rows, or an
  acceptance criterion with no meaningful test mapping remain native review blockers.
  After the final recheck, emit the binding with `devrites-engine check readiness
  --emit-binding <slug>` and replace its one exact line. Never refresh it without Vet.
