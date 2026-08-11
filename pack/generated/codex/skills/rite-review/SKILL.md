---
name: rite-review
description: Review polished feature diff for correctness, readability, architecture, security, tests proving acceptance, Critical/Important findings, and quality dimensions before seal.
argument-hint: "[scope: slice N | feature] [--full]"
user-invocable: true
---

# $rite-review: feature-scoped review

Review the **active feature scope only**. **Read the active workspace first**; if none,
tell the user to run `$rite-spec <feature>`.

> **Scope:** `/code-review` is a generic diff review with no workspace context.
> `$rite-review` reads
> `.devrites/work/<slug>/spec.md` first, runs Spec ↔ Code-review axes as
> parallel fresh-context reviewers (see [`parallel-dispatch.md`](../devrites-lib/reference/parallel-dispatch.md)), and gates feeding
> into `$rite-seal`. Use `/code-review` for a one-off diff; use
> `$rite-review` for a DevRites feature where the spec is the contract. Use the shared depth rules in
[`devrites-lib/reference/orchestration-profiles.md`](../devrites-lib/reference/orchestration-profiles.md);
all workflow-named roles remain mandatory at every depth.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
Pull these via `Read` when the diff demands them:
- `code-review.md`: small PRs, severity labels, tests-first review focus.
- `review-checklist.md`: compact pass/fail sweep before reporting the verdict.
- `principles.md`: declared project invariants (`.devrites/principles.md`); a diff that violates one with no recorded exception is a Critical, blocking finding.
- `testing.md`: confirm that passing tests actually prove the spec.
- `agents.md`: when to fan out to which review subagent.
- `security.md`: when input / auth / data / integrations / secrets are in scope.
- `security-checklist.md`: for the same security-sensitive scope, the compact trust-boundary sweep.
- `repository-topology.md`, `data-integrity.md`, `integration-reliability.md`: only
  when the spec applicability map or final diff triggers their ownership/failure/proof checks.
- `performance.md`: only when perf is relevant or a regression risk is visible.

## Operating rules
- **Feature scope only.** Review touched files + the diff. **NO whole-project refactors,
  NO drive-by cleanup.** DO NOT delete suspected dead code outside this feature without
  asking. Spec Drift Guard applies.
- **Review the finished product.** `$rite-polish` has already simplified code and
  normalized or polished UI. If review finds a remaining complexity issue, record it as
  a finding rather than rerunning a simplification pass.
- Follow the shared
  [`candidate-integrity.md`](../devrites-lib/reference/candidate-integrity.md).
  Review starts only from the digest Polish closed.
- Findings are labeled (below). Re-prove after any accepted correction.
- **Reviewers judge; root reconciles; wright fixes.** Per
  [`agents.md`](../devrites-lib/reference/standards/agents.md), root directly
  reconciles verdicts/writes artifacts; engines do not. Accepted corrections
  route to `devrites-slice-wright`.

## Workflow
0. Read `.agents/skills/devrites-lib/reference/standards/core.md` first (the always-on operating rules); pull the
   on-demand rules above as the diff demands them.
   Then read the explicit or active workspace's `state.md` directly.
1. Read `spec.md`, `tasks.md`, `state.md`, `decisions.md`, `evidence.md`,
   `touched-files.md`, `.devrites/principles.md` (if present: the binding invariants to score
   the diff against), and the `git diff`. For "what would this change break"
   questions, apply `.agents/skills/devrites-lib/reference/standards/tooling.md`: use
   the primary available index and cross-check only one named unresolved predicate before
   falling back to LSP/file search. When a finding hinges on an external library's
   current API, context7 if available can confirm the signature. Run
   `devrites-engine check candidate <slug>` and require its digest to match the
   single `evidence.md` binding and the `browser-evidence.md` binding when that
   file exists. A missing, malformed, or open candidate returns to Polish/Prove.
2. **Review tests first:** do they prove the acceptance criteria? Missing,
   weak, or wrong tests are the first findings.
   **Completion:** every acceptance criterion maps to a proven test or a labeled finding.
3. **Review spec and code separately in parallel.** A change can pass
   one axis and fail the other: code that follows every project standard but
   implements the wrong thing (Code-review pass, Spec fail), or code that does exactly
   what the spec asked but breaks project conventions (Spec pass, Code-review fail).
   Separate contexts prevent one axis from masking the other:
   - Freeze that closed digest and give the same digest to **two** read-only reviewers in
     parallel, each with its own narrow brief and no
     cross-pollination:
     - **Spec axis** → `devrites-spec-reviewer`: "Apply your documented discipline on
       the active feature workspace + diff. Report (a) criteria the spec asked for that
       are missing or partial, (b) behaviour in the diff the spec did not ask for
       (scope creep / drift), (c) criteria implemented incorrectly. Quote the spec
       line per finding. If the spec is missing or unreadable, report `Review gap:
       missing spec` and do not infer requirements from the diff."
     - **Code-review axis** → `devrites-code-reviewer`: "Apply your full documented
       discipline (tests-first, correctness, readability, architecture, maintainability,
       standards) on the active feature workspace + diff. Cite file:line per finding;
       skip what tooling already enforces. Also flag the AI-codegen smells (silent/empty
       catch, defensive try-catch bloat + redundant logging, single-use factory / needless
       indirection, dependency creep where an in-repo option exists, a 100-line function
       where 20 would do) and the silent-failure bugs (a missing value coerced to 0/''/[],
       a dropped Result/err return, off-by-one / boundary, logic that contradicts the
       comment/docstring/name). Per hunk, check whether working code was deleted that the
       task did not ask to remove. Score the diff against `.devrites/principles.md`: a change
       that breaks a declared invariant with no recorded, human-approved exception is a Critical.
       Distinguish hard documented-standard violations from baseline smells; smells are judgment
       calls unless a DevRites or project standard makes them binding."
   - **Do NOT merge or re-rank** their findings. Present them under separate
     `## Spec` and `## Code review` sub-sections in `review.md`. Surface contradictions
     between the axes explicitly (e.g. "Spec axis says complete, Code-review axis says
     untestable"): `$rite-seal` decides what blocks. Preserve each reviewer's
     `Outcome:` and admit its account through
     [`agents.md` § Result admission](../devrites-lib/reference/standards/agents.md#result-admission).
     Either `Outcome: gap` stops Review; silence, failure, or malformed output never
     becomes an empty findings list.
4. **Reconcile, don't re-review.** With the two parallel reports in hand, the inline
   lead reconciles. It does **not** re-run the code-review axes over correctness /
   readability / architecture / maintainability that `devrites-code-reviewer` already
   covered. Stay in scope ([feature-scoped-review](reference/feature-scoped-review.md)).
   Add only what the dispatched agents could not, then resolve overlaps and
   contradictions before labeling. ([five-axis-review.md](reference/five-axis-review.md)
   documents the axes the code-review agent applies.)
   - **UI feature?** Apply the **UX rubric**
     ([nielsen-heuristics](reference/nielsen-heuristics.md)) and the
     **cognitive-load lens** ([cognitive-load](reference/cognitive-load.md)) on the
     UX axis: surface heuristics scoring ≤ 2 and any cognitive-load findings
     (extraneous noise, missing progressive disclosure, vocabulary drift) at the
     appropriate severity.
5. **Security:** apply `devrites-audit security`
   ([security-review](reference/security-review.md)) when user input, auth, data
   storage, external integrations, secrets, or permissions are involved.
6. **Performance:** apply `devrites-audit perf`
   ([performance-review](reference/performance-review.md)) only when performance is
   relevant or a regression risk is visible (measure first).
7. Reconcile and accept only in-scope fixes. Consolidate them into one bounded
   wright correction; never edit source in the reviewing context. Any correction
   updates the candidate manifest, returns through affected Prove, and then starts
   a fresh Review on the new digest. Do not carry a prior reviewer account across it.
8. The root updates `review.md` and `state.md`, writing exactly one candidate
   binding in `review.md` for the digest given to every reviewer.
   **Completion:** the records name the reviewed candidate identity and every accepted
   correction has affected proof plus a fresh Review.

## Finding labels
- **Critical:** must fix before seal (correctness/security/data loss).
- **Important:** should fix before seal (likely bug, real maintainability risk).
- **Suggestion:** worth doing, not blocking.
- **Nit:** trivial/style.
- **FYI:** context, no action implied.

**Action tag (separate from severity).** Tag each finding with how to act on it:
`blocking` (fix before seal), `non-blocking` (fix when convenient), or `if-minor` (fix only if the
change is already small: a pure noise-economics lever). Only a **`blocking` Critical** gates the
seal; a `non-blocking` / `if-minor` finding is recorded, not a stop.

## Confidence and signal-to-noise
Apply [`agents.md` § Result admission](../devrites-lib/reference/standards/agents.md#result-admission).
Suppress unverifiable ≤4 hypotheses, require 7+ plus exact proof for
Critical/Important, and make every silent/unusable account a blocking gap. Roll
trivia into one line; review finds defects, not a quota.

## Severity orientation (labels, not score)

After labeling, summarize findings as `Critical / Important / Suggestion / Nit /
FYI` counts. There is no composite number. `$rite-seal` gates on
`Critical == 0` and on acceptance plus drift. Do not invent a composite score.

> **Mid-flight discipline.** When tempted to demote a Critical, hide a finding, fix without re-verification, or wander out of scope: see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output → `review.md`

Write the detailed review to `review.md`.
