# Vet review: scope, four axes, and outputs

Run §0 first as a blocking gate, then review the four axes in order and produce the
required outputs. Apply the engineering lenses in [`eng-lenses.md`](eng-lenses.md)
throughout rather than as a separate checklist. Calibrate every finding through the
confidence and verification gate before presenting it.

---

## §0. Scope challenge (blocking gate)

Before reviewing implementation details, check whether the plan includes more work than
the settled spec requires.

1. **What already exists?** For each sub-problem in the plan, find the existing code/flow that
   already solves it (use a code-intelligence index if available: see `../../devrites-lib/reference/standards/tooling.md`).
   Can the plan **capture outputs from an existing
   flow** instead of building a parallel one? Reuse → extend → build new, in that order
   (`coding-style.md`). List every reuse opportunity the plan misses.
2. **Minimum diff.** Find the smallest set of changes that meets the spec's acceptance
   criteria. Flag work that can be deferred without blocking acceptance. Do not cut an
   acceptance criterion; that requires the Drift Guard.
3. **Complexity smell.** If `plan.md` touches **>8 files** or adds **>2 new services / modules /
   classes**, treat it as a smell. Check the plan's complexity gate justifies it. If it doesn't
   → harden to the smallest acceptance-preserving plan. Ask only if reduction changes
   acceptance or explicit architecture policy; never start the axes unresolved.
4. **Built-in check.** For each new pattern / infra component / concurrency approach the plan
   introduces, verify a framework/runtime built-in doesn't already do it, and that the choice is
   current best practice with no known footgun: dispatch `devrites-source-driven` to confirm at
   the source and record the citation. A custom roll where a built-in exists is a scope-reduction
   finding.
5. **Completeness check.** Identify shortcuts in edge-case handling, error paths, and
   test coverage. Prefer the complete option when AI-assisted implementation makes the
   additional work small. Flag shortcuts that save little time but leave known gaps.
6. **Distribution check.** If the plan introduces a new artifact (CLI binary, package, container,
   deployable), does it include how it gets built / published / installed? If distribution is
   deferred, say so explicitly in "NOT in scope": don't let it silently drop.

> **STOP discipline.** Fold technical reduction into the plan; ask and stop only for a
> human-owned choice.

If the smell does not trip, present the §0 findings and proceed to Axis 1.

---

## Four axes (one at a time, at most 8 findings each)

For each axis, fold verified technical findings into the plan, then walk each human-owned
decision via one coherent `AskUserQuestion` packet. Supporting findings may combine only
with one owner/trade-off. HITL pauses there; AFK follows `depth.md`. Never invent findings.

### 1. Architecture
- Component boundaries, coupling, data-flow patterns, single points of failure. Architecture records invariants, not scaffolding: each medium+ decision should state `Binds:` and `Prevents:` so the builder knows what divergence it prevents.
- Scaling characteristics; where the plan's approach breaks under real load.
- Security architecture at the seams (auth, data access, API boundaries): does the plan name
  the trust boundary for each untrusted input?
- For each new codepath / integration point: **one realistic production failure scenario** and
  whether the plan accounts for it (feeds the failure-mode table).
- Does any key flow deserve an ASCII diagram in the plan or an inline comment in the code the
  build will write? Name the files that should carry one.

### 2. Plan code-quality
- Module structure the plan implies; DRY across the slices (flag planned repetition aggressively).
- Error-handling + edge cases the plan names, and the ones it doesn't (call those out explicitly).
- Over-engineering (premature abstraction, an extension point with no second caller) vs
  under-engineering (fragile / hacky) relative to `patterns.md` + `coding-style.md`.
- Tech-debt hotspots the plan walks into; existing inline diagrams in touched files that the
  change will make stale.

### 3. Test-coverage design
Design tests before code so the build writes them alongside the implementation.
- **Framework detection:** find the project's existing test runner + conventions; match them
  (never introduce a new runner to prove one change: `testing.md`).
- **Map acceptance → tests.** Every spec acceptance criterion must map to ≥1 planned, surface-anchored test (the API response/UI state/CLI output the criterion names, not an internal proxy).
- **Tool per path:** unit (pure logic, single function, edge cases), integration/E2E (a user
  flow spanning 3+ components, an auth/payment/data-loss path, a mock-hides-failure boundary),
  eval (an LLM/prompt change that needs a quality bar).
- **Interaction inventory (UI slices): enumerate every interactive element + flow.** List each
  input field, checkbox, radio, select, toggle, button, and actionable link, plus each user
  flow; assign each ≥1 asserting test **at the right level**: elements/fields → unit/component,
  critical journeys → one E2E (never one-per-field). Every element/flow with no asserting test
  is a GAP; no element ships unverified. Write the inventory to `test-plan.md` (table in
  `artifacts.md`). This is `testing.md` "Completeness" made concrete for the plan.
- **Regression rule (mandatory, no question):** when the plan modifies existing behavior and the
  current suite doesn't cover the changed path, a regression test is added to the plan as a
  **Critical** requirement: no `AskUserQuestion`, no skipping. Regressions are the highest-priority
  test because they prove something broke. When unsure whether a change is a regression, write the test.
- Produce the **coverage diagram** (shape below) and add a specific test requirement per GAP.

#### Coverage diagram (write to `test-plan.md`)
Both code paths and user flows in one view; mark E2E-worthy `[→E2E]` and eval-worthy `[→EVAL]`:

```
CODE PATHS                                          USER FLOWS
[+] services/billing                                [+] Checkout
  ├── processPayment()                                ├── [★★★ planned] complete purchase — checkout.e2e
  │   ├── [★★★ planned] happy + declined + timeout    ├── [GAP] [→E2E] double-click submit
  │   └── [GAP]         invalid currency              └── [GAP]        navigate away mid-payment
  └── refundPayment()                               [+] Error states
      └── [★  planned] full refund only               └── [GAP]        network-timeout UX

COVERAGE: 4/9 planned (44%)  |  GAPS: 5 (2 E2E)  |  REGRESSIONS: 1 (Critical)
```
Legend: ★★★ behavior+edge+error · ★★ happy path · ★ smoke · [→E2E] integration · [→EVAL] LLM eval.
**Fast path:** every acceptance criterion already maps to a planned test → "Test coverage: all
acceptance criteria covered ✓" and continue.

### 4. Performance
- N+1 / unbounded queries and DB access patterns; memory concerns; caching opportunities;
  high-complexity hot paths.
- Per `performance.md`: name a number to measure or a budget: don't recommend speculative
  micro-tuning. A perf finding is "measure X against budget Y", not "this feels slow".

---

## Confidence and verification gate
Tag each finding `[severity] (confidence: N/10) <plan/task/spec ref> — finding`:
- **9-10** verified against a quoted line · **7-8** strong pattern match → report normally.
- **5-6** moderate → report with "verify this is real".
- **≤4** speculative → **suppress from the walk-through**, appendix only.

Before raising a finding, quote the lines that support it. If no line supports it, set
confidence to 4 or lower and suppress it. Do not inflate confidence to avoid suppression.
`devrites-plan-reviewer` follows the same rule.

---

## Present human-owned decisions
Use `AskUserQuestion` per the pack's standard. Plan-review specifics:
- **One decision = one call.** Never ask about agent work or batch unrelated choices.
- Concrete: name the plan/task section + the quoted line.
- 2-3 options, including "do nothing / proceed as-is" where reasonable.
- Per option, one line: **effort** (human ~X / with the build agent ~Y), **risk**, **maintenance**.
  If the complete option is only marginally more effort than the shortcut (AI makes it cheap),
  recommend complete.
- **Map to a rule.** One sentence tying the recommendation to a DevRites rule (reuse-first,
  fail-fast, test-behavior, measure-first, minimum diff).
- **Coverage vs kind:** if the options differ in *coverage* (more tests vs fewer, complete vs
  happy-path), add `Completeness: N/10` per option. If they differ in *kind* (two different
  architectures), skip the score and note "options differ in kind, not coverage". Never fabricate
  a score on a kind question.
- Every material call ends as a **recorded decision**: behavior-preserving technical hardening
  goes to `decisions.md`; a human-owned HITL choice gets a resolved `questions.md` qid; AFK
  records the allowed recommendation in `decisions.md`. The review must leave an auditable
  trail without turning agent work into questions.

---

## Required outputs (after the axes)
1. **"NOT in scope":** work considered and explicitly deferred, one-line rationale each. Folds
   into `plan.md` §Scope boundaries + `spec.md` Non-goals (via the Guard) so it can't silently re-enter.
2. **"What already exists":** existing code/flows that solve sub-problems, and whether the plan
   reuses or rebuilds them. Every missed reuse becomes a §0 finding.
3. **Failure-mode table:** for each new codepath: a realistic failure, and whether (a) a test
   covers it, (b) error handling exists, (c) the user sees a clear error or a silent failure. A
   failure with **no test AND no handling AND silent** is a **Critical gap**. (Shape in
   [`artifacts.md`](artifacts.md).)
4. **Worktree parallelization strategy:** analyze the slices for parallel execution. **Skip** if
   all slices touch the same module or there are <2 independent workstreams ("Sequential, no
   parallelization opportunity"). Otherwise: a dependency table (module-level, not file-level),
   parallel lanes (shared module → same lane/sequential; independent → separate lanes), execution
   order, and conflict flags where two lanes touch the same module dir. This feeds `/rite-build`'s
   isolation strategy and the autocomplete loop. (Shape in [`artifacts.md`](artifacts.md).)
5. **Build-entry preflight:** commands/cwds, tools, package state, parser/browser smoke,
   prerequisites, and provenance ([`artifacts.md`](artifacts.md)).
6. **Implementation readiness:** goal-backward coverage, wiring, dependency simulation,
   alignment, operations, and rollback verdict.
7. **Completion summary:** the compact recap defined in
   [`artifacts.md`](artifacts.md).
