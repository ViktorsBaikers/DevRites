# Vet review: scope, four axes, and outputs

Run blocking §0, four axes, and outputs with [`eng-lenses.md`](eng-lenses.md);
calibrate findings before presentation.

Authority: `.pi/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → fold technical topology; invalidate Vet/readiness; affected Vet before Build.
- `GUARD_AND_REPAIR` → no planning writes; Spec Drift Guard → Clarify → Plan repair → affected Vet.
- `BLOCKED_INPUT` → no planning writes; exact diagnostic; recover input; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

---

## §0. Scope challenge (blocking gate)

Before implementation review, check for work beyond the settled spec.

1. **What exists?** Find reusable code/flows via
   [`tooling.md`](../../devrites-lib/reference/standards/tooling.md). Reuse → extend →
   build new; list missed reuse instead of planning parallel work.
2. **Minimum diff.** Find the smallest contract-complete change. Flag non-blocking work;
   use the marked action for topology cuts.
3. **Complexity smell.** **>8 files** or **>2 new services/modules/classes** needs a justified
   complexity gate; otherwise harden to the smallest contract-complete plan. Use the
   marked action for topology reduction.
   **Feature ceiling.** A plan over the
   [feature ceiling](../../rite-plan/reference/slicing.md#feature-ceiling-split-an-epic-never-override-the-budget)
   (over-budget `tasks.md`/`plan.md` from slice or AC count) is a scope finding, not a
   `Budget override`: route to `/rite-plan course-correct` (`MVP cut`) with the
   continuation sequence named; do not vet the whole epic.
4. **Built-in check.** For each new pattern/infra/concurrency approach, invoke
   `devrites-source-driven` inline to verify current framework/runtime support and cite it. Custom
   work where a built-in suffices is a scope-reduction finding.
5. **Completeness.** Find edge/error/test shortcuts; prefer the complete option when the
   extra work is small, and flag small savings that leave known gaps.
6. **Distribution check.** New artifacts (binary/package/container/deployable) need
   build/publish/install plans or explicit deferral in "NOT in scope".
7. **Applicability check.** Compare `spec.md`'s topology/data/integration/security/delivery
   decisions with live seams. A false `not applicable` or an `applies` row without the
   focused standard's owner, failure/recovery, deployment order, and proof output is `broken`.
8. **Decision horizons.** Independently apply
   [`plan-template.md`](../../rite-define/reference/plan-template.md#decision-horizons) to the
   plan, questions, assumptions, decisions, and checkpoints; no known item may disappear.
   Reject `local` for blockers or public contracts, security/data invariants, acceptance,
   migration/rollback, dependencies, or cross-slice interfaces. Local/checkpoint deferral
   needs bounded owner, evidence trigger, fallback, and resolution proof; a risk spike needs
   necessary executable evidence, discriminating criteria, and fallback branches. Any defect is
   `broken`; unresolved human blockers return to Clarify.

> **STOP discipline.** Fold technical reduction into the plan; ask and stop only for a
> human-owned choice.

After justifying complexity or folding the contract-preserving reduction, present
the §0 findings and proceed to Axis 1. A human-owned unresolved choice still pauses.

---

## Four axes (complete inventory, bounded presentation)

Fold verified technical findings into the plan; present each human-owned decision in one
`AskUserQuestion` packet. Combine support only for one owner/trade-off. HITL pauses; AFK
uses `depth.md`. Never invent findings.
Report every supported Critical/Important before repair. Group duplicates; compact
presentation never caps the inventory or hides the ninth substantive defect.

### 1. Architecture

- Boundaries, coupling, data flow, single points of failure. Record invariants, not
  scaffolding; medium+ decisions state `Binds:`/`Prevents:`.
- Repository/deployable roots, canonical contract and mutable-state ownership, shared
  resources, dependency cycles, and old/new deployment combinations when applicable.
- Scaling ceiling under real load.
- Security architecture at the seams (auth, data access, API boundaries): does the plan name
  the trust boundary for each untrusted input?
- One realistic production failure per new codepath/integration (feeds failure-mode table).
- Does any key flow deserve an ASCII diagram in the plan or an inline comment in the code the
  build will write? Name the files that should carry one.

### 2. Plan code-quality

- Implied modules and planned cross-slice repetition.
- Error-handling + edge cases the plan names, and the ones it doesn't (call those out explicitly).
- Over-engineering (premature abstraction, an extension point with no second caller) vs
  under-engineering (fragile / hacky) relative to [`patterns.md`](../../devrites-lib/reference/standards/patterns.md) + [`coding-style.md`](../../devrites-lib/reference/standards/coding-style.md).
- Tech-debt hotspots and diagrams the change makes stale.

### 3. Test-coverage design

Design tests before code.

- **Framework:** match the existing runner/conventions; never add a runner for one change.
- **Acceptance → tests.** Every AC maps to a planned surface assertion, not an internal proxy.
- **Map applicable risk → tests.** Data and integration rows cover their relevant
  duplicate/retry/concurrency/interruption/tenant/timeout/partial/outage/order/rollback cases,
  or record an evidence-backed dismissal. A mock that cannot exhibit the named risk is a GAP.
- **Tool per path:** unit for pure logic; integration/E2E for 3+ components,
  auth/payment/data-loss, or mock-hidden failure; eval for LLM/prompt quality.
- **UI inventory:** list every interactive element/flow in `test-plan.md`; assign
  elements/fields a unit/component assertion and critical journeys an E2E. Untested = GAP;
  no one-E2E-per-field (`artifacts.md`).
- **Regression (mandatory):** changed behavior without path coverage adds a **Critical**
  regression test—no question or skip. If uncertain, write it.
- Produce the **coverage diagram** (shape below) and add a specific test requirement per GAP.

#### Coverage diagram (write to `test-plan.md`)

Both code paths and user flows in one view; mark E2E-worthy `[→E2E]` and eval-worthy `[→EVAL]`:

```text
CODE PATHS                                          USER FLOWS
[+] services/billing                                [+] Checkout
  ├── processPayment()                                ├── [★★★ planned] complete purchase — checkout.e2e
  │   ├── [★★★ planned] happy + declined + timeout    ├── [GAP] [→E2E] double-click submit
  │   └── [GAP]         invalid currency              └── [GAP]        navigate away mid-payment
  └── refundPayment()                               [+] Error states
      └── [★  planned] full refund only               └── [GAP]        network-timeout UX

COVERAGE: 3/7 planned (43%)  |  GAPS: 4 (1 E2E)  |  REGRESSIONS: 0
```

Legend: ★★★ behavior+edge+error · ★★ happy path · ★ smoke · [→E2E] integration · [→EVAL] LLM eval.
**Fast path:** every acceptance criterion already maps to a planned test → "Test coverage: all
acceptance criteria covered ✓" and continue.

### 4. Performance

- N+1 / unbounded queries and DB access patterns; memory concerns; caching opportunities;
  high-complexity hot paths.
- Per [`performance.md`](../../devrites-lib/reference/standards/performance.md): name a number to measure or a budget: don't recommend speculative
  micro-tuning. A perf finding is "measure X against budget Y", not "this feels slow".

---

## Confidence and verification gate

Tag each finding `[severity] (confidence: N/10) <plan/task/spec ref> — finding`:

- **9-10** verified against a quoted line · **7-8** strong pattern match → report normally.
- **5-6** moderate → report with "verify this is real".
- **≤4** speculative → **suppress from the walk-through**, appendix only.

Quote supporting lines. Without them confidence is ≤4 and suppressed; never inflate it.
`devrites-plan-reviewer` follows this rule.

## Finding kind: contract or mechanism

Tag every Critical/Important `kind: contract` or `kind: mechanism`; root confirms the tag
at reconciliation.

- **contract** — a *guarantee* (acceptance, interface, boundary, ownership, data shape,
  ordering/durability, failure outcome, dependency choice) an implementer could build
  differently and still pass the planned tests → plan text → `NEEDS REPLAN` + Plan repair.
- **mechanism** — *how* a pinned clause is achieved (syscall order, fsync/rename shape,
  lock scope, retry shape, parser detail): a test discriminates the wrong implementation
  → root adds one named case row to `test-plan.md` (input → observable outcome + the
  mutant that must fail) plus its `traceability.md` ref, re-emits the readiness binding,
  rechecks mechanically (anchor, ID parity). No Plan repair, no reviewer redispatch;
  Build's RED test proves it. Prove in code, not in prose.

Mechanism becomes `contract` only when the plan asserts the opposite mechanism or the
clause is a public/shared seam. **Failing case:** round after round of prose pins on
already-pinned clauses.

---

## Present human-owned decisions

Use standard `AskUserQuestion`:

- **One decision = one call.** Never ask about agent work or batch unrelated choices.
- Concrete: name the plan/task section + the quoted line.
- 2-3 options, including "do nothing / proceed as-is" where reasonable.
- Per option: **effort** (human/build agent), **risk**, **maintenance**; recommend complete
  when marginally costlier than a shortcut.
- **Map to a rule.** One sentence tying the recommendation to a DevRites rule (reuse-first,
  fail-fast, test-behavior, measure-first, minimum diff).
- **Coverage vs kind:** coverage options get `Completeness: N/10`; architecture/kind options
  state “differ in kind” and get no fabricated score.
- Record every material call: technical hardening/AFK in `decisions.md`; HITL in a resolved
  `questions.md` qid. Do not turn agent work into questions.

---

## Required outputs (after the axes)

1. **"NOT in scope":** considered/deferred work + rationale, folded into plan boundaries and
   spec Non-goals via the Guard.
2. **"What exists":** solving code/flows and reuse/rebuild disposition; missed reuse is §0.
3. **Failure-mode table:** per new codepath: realistic failure, partial/unknown effect,
   recovery owner, test, handling, and user-visible/silent result. No test + no handling +
   silent = **Critical** (`artifacts.md`).
4. **Dependency safety:** verify graph/order and shared file/state/contract/lock/port/queue/env
   conflicts. Correct the plan; native host scheduling needs no lane artifact.
5. **Build-entry preflight:** commands/cwds, tools, package state, parser/browser smoke,
   prerequisites, and provenance ([`artifacts.md`](artifacts.md)).
6. **Implementation readiness:** goal-backward coverage, wiring, dependency simulation,
   alignment, operations, and rollback verdict.
7. **Completion summary:** the compact recap defined in
   [`artifacts.md`](artifacts.md).
