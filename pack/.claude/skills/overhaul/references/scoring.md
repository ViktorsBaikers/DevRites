# Scoring: Evidence-Backed Readiness Index

## Contents

- [What the index is and is not](#what-the-index-is-and-is-not)
- [Freeze the control catalog before repairs](#freeze-the-control-catalog-before-repairs)
- [Audit sections and score domains](#audit-sections-and-score-domains)
- [Credit and arithmetic](#credit-and-arithmetic)
- [Lane scores](#lane-scores)
- [Thresholds and hard gates](#thresholds-and-hard-gates)
- [Attribution, monotonicity and adequacy](#attribution-monotonicity-and-adequacy)
- [Worked examples](#worked-examples)
- [Calculator tool](#calculator-tool)

## What the index is and is not

The index is this skill's local 0–10 policy score over one approved scope, rubric
revision and candidate. The 9.7 target, domain weights, lane thresholds and 9.0
floors are the user's requested policy, disclosed and approved with the plan. It is
not an ISO/IEC 25010 or OWASP ASVS certification, a probability of correctness, a
"97% secure" claim, or comparable across projects, scopes or rubric revisions.
ISO/IEC 25010 names quality characteristics; ASVS supplies security requirements;
coverage, OpenSSF Scorecard and Lighthouse each measure something narrower. None of
them defines this formula. A 9.7 or 10 never proves the absence of undiscovered bugs.

Compute every number with the calculator below. Never estimate a score by
impression; if the calculator cannot run, every score and gate is `UNKNOWN`.

## Freeze the control catalog before repairs

A control is a meaningful requirement or risk check for the approved scope — not a
finding count, a test case, or a lint rule. Group micro-checks into one control;
splitting an easy check into many must never add weight. Freeze the catalog in an
immutable `rubric-r<N>.json` before any repair; the plan pins its digest.

Each control definition carries: stable `id`; exactly one `domain`; `lanes` it
belongs to; components, requirement, profile rule and contract IDs; `failure_case`;
risk `rationale`; `weight` (1 modest, 3 material, 5 high-consequence);
`hard_gate`; `evaluation` (`executed`, `static`, `source-proof`, `human`; a
`performance` control passes only on `executed`, `source-proof` or `human` evidence,
because static inspection establishes no metric);
`evidence_required`; `independent_verification`; and `na` (null, or an approved
exclusion with reason and evidence). Candidate status, verifier receipts and
evidence live in separate results files that reference the rubric digest.

The calculator accepts control weights 1, 3 or 5 only. Default domain weights
(tailor only with a stated project-risk reason and user approval before remediation;
each changed domain needs a `weight_deviations` entry with `domain`, `reason` and the
`approval` reference, or the calculator rejects the rubric):

| Domain key | Meaning | Weight |
| --- | --- | ---: |
| `correctness` | Functional correctness and data integrity | 20 |
| `security` | Security and privacy | 20 |
| `reliability` | Reliability, concurrency and recovery | 15 |
| `performance` | Performance and scalability | 15 |
| `tests` | Test effectiveness and verification | 10 |
| `architecture` | Architecture, maintainability and code quality | 10 |
| `ux` | User experience, accessibility and compatibility | 5 |
| `operations` | Dependencies, build integrity and operational readiness | 5 |

A backend without a GUI still owns compatibility and operator/developer experience
controls under `ux`; never mark a whole domain N/A because one sub-feature is absent.
Exclude a domain only with a recorded reason and evidence. The rubric lists every
domain key in `domains` (weight) or in `domain_exclusions` (`domain`, `reason`,
`evidence`); lane-level exclusions go in `lane_domain_exclusions` (`lane`, `domain`,
`reason`, `evidence`). The calculator rejects a missing or unknown domain. A newly confirmed
high-impact finding creates or updates a control and a hard gate; it never becomes
an opportunity to add easy passing controls that dilute the denominator.

Mark hard-gate controls for every mandatory safety, security, data-integrity,
compatibility and critical-user-journey requirement.

## Audit sections and score domains

Every finding, receipt `domains_checked` entry and applicability cell uses one of the
eight score keys. The [audit floor](audit-domains.md) sections map to them:

| Audit section | Score domain |
| --- | --- |
| Correctness and data integrity; Database, queries, migrations and storage (integrity, migrations) | `correctness` |
| Security, privacy and trust boundaries; Dependencies (advisories, provenance, unresolvable names) | `security` |
| Concurrency, caching, retries and distributed reliability | `reliability` |
| Backend performance; Database (query cost); Frontend performance | `performance` |
| Tests and verification quality | `tests` |
| Architecture, maintainability and anti-slop; Over-engineering and dead weight; Frontend engineering (structure) | `architecture` |
| Frontend craft, interaction and accessibility; Public API and developer experience; Cross-layer contracts (compatibility) | `ux` |
| Operations and project-specific extensions; Documentation; Dependencies (support, end of life, build integrity) | `operations` |

A finding that spans two domains takes the domain of its consequence and names the
other's control in `controls`.

## Credit and arithmetic

Binary credit only, from current evidence meeting the predeclared acceptance
condition (including required independent verification):

```text
PASS -> p_i = 1
FAIL / UNKNOWN / BLOCKED / NOT_RUN / DEFERRED / ACCEPTED_RISK -> p_i = 0
N/A  -> removed only by an approved exclusion with evidence of true inapplicability
```

No partial credit. A genuinely separable requirement may have predeclared
sub-controls; never split a control after it fails. Risk acceptance is a labeled
user decision, not a PASS. A compensating control passes only if it satisfies the
original requirement and its acceptance evidence. The calculator counts a PASS with
no evidence, or without required independent verification, as `UNKNOWN` and lists
the downgrade.

```text
D_d = 10 * SUM(w_i * p_i) / SUM(w_i)          over applicable controls in domain d
Q   = SUM(W_d * D_d) / SUM(W_d)                over applicable domains
```

Weights must be finite and strictly positive; control IDs unique; exclusions carry
reason and evidence. Remove an inapplicable control by exclusion, never by a zero
weight. A zero denominator is `NOT_ASSESSABLE`, never 10. The calculator uses exact
rationals and compares thresholds at full precision: 9.696 fails 9.7 even though it
displays as 9.70.

Score baseline and candidate on the same rubric revision. When new scope or risk
legitimately changes the rubric, version it, get approval for material changes, and
recalculate both; historical proof that was never collected is `UNKNOWN`, not a
fabricated baseline failure. Report coverage separately from the index: inventoried
files, semantically reviewed ranges, cross-boundary scenarios, executed controls and
independently verified controls.

## Lane scores

Lane views come from the same catalog; membership is fixed with the plan.

```text
D_(l,d) = 10 * SUM_(i in C(l,d))(w_i * p_i) / SUM_(i in C(l,d))(w_i)
Q_l     = SUM_(d applicable to l)(W_d * D_(l,d)) / SUM_(d applicable to l)(W_d)
```

`C(l,d)` holds the unique controls of domain `d` whose `lanes` include `l`. Every
non-excluded domain applies to every applicable lane unless the rubric records a
lane-domain exclusion with reason and evidence; an applicable lane-domain with no
controls is `NOT_ASSESSABLE` and blocks readiness. The calculator discloses each
lane's renormalized weight denominator. A shared control appears in each lane view
it belongs to, counts once in the global arithmetic, and passes only when every
required side is evidenced; report distinct executed/verified controls separately
from lane appearances. Never replace `Q` with an average of lane scores. A
backend-only project has no invented frontend score, but its contract and
compatibility controls remain.

## Thresholds and hard gates

Policy: `Q >= 9.7`, every applicable `Q_l >= 9.7`, every applicable `D_d >= 9.0`,
every applicable `D_(l,d) >= 9.0`, all at full precision, plus every hard gate. A
rubric `policy` object (`global_min`, `lane_min`, `domain_min`, `lane_domain_min`)
may raise these thresholds, never lower them; the calculator rejects a lower value.
A user-approved weaker target is reported beside the verdict and never turns it into
`MEETS_TARGETS`.
High numbers never compensate for a failed gate.

| Gate | PASS requires | Mode |
| --- | --- | --- |
| `G-THRESHOLDS` | All four threshold families pass (computed) | all |
| `G-MANDATORY` | Every hard-gate control PASS (computed) | all |
| `G-NO-CRITICAL-HIGH` | No confirmed critical/high defect in scope, whatever its disposition | all |
| `G-NO-OPEN-SERIOUS-LEAD` | No unresolved credible potentially critical/high lead | all |
| `G-COVERAGE` | Every required range, lane/profile rule and boundary check complete with current evidence; no unvalidated high-risk profile | all |
| `G-CATALOG-ADEQUACY` | Independent adequacy review of this rubric passed | all |
| `G-LANE-RECEIPTS` | Separate qualified frontend and backend engineering receipts where both apply | all |
| `G-CONCURRENCY` | Overlapping frontend/backend dispatch evidence where both apply | all |
| `G-FRESHNESS` | Every PASS rests on evidence valid for the scored fingerprint | all |
| `G-REPORT-CONSISTENCY` | Records validate; views carry the current stamp | all |
| `G-APPROVAL` | Authentic, current approval of the exact plan revision and tasks | repair |
| `G-TASKS-COMPLETE` | Every approved task met its acceptance evidence | repair |
| `G-ORACLES` | Each task's oracle red on baseline and green on candidate as [its kind requires](repair-and-measurement.md#oracles-by-finding-kind), independently verified | repair |
| `G-NO-REGRESSION` | No new regression, skipped required check, or unapproved behavior change | repair |
| `G-PERF-TARGETS` | Approved targets and guardrails demonstrated (N/A only when none approved) | repair |
| `G-PRESERVATION` | Behavior, UI/UX and public-contract preservation evidence | repair |
| `G-FINGERPRINTS` | Source/evidence fingerprints match; no writes during decisive proof | repair |
| `G-OWNERSHIP` | Changed paths within approved contracts; generated files via generators | repair |
| `G-FINAL-CHALLENGE` | Independent final challenge passed on the assembled candidate | repair |

Gate states: `PASS`, `FAIL`, `UNKNOWN`, `NOT_APPLICABLE` (only `G-PERF-TARGETS`,
`G-LANE-RECEIPTS`, `G-CONCURRENCY`, with a reason), and `WAIVED_OPERATIONAL` (only
`G-CONCURRENCY`, with the approval event that accepted degraded sequential review).
A gate PASS without evidence references counts as `UNKNOWN`.

Readiness verdicts: `NOT_ASSESSABLE` (a required score cannot be computed);
`MEETS_TARGETS` (all thresholds and gates pass); `MEETS_TARGETS_WITH_APPROVED_DEGRADATION`
(the only open gate is `G-CONCURRENCY = WAIVED_OPERATIONAL`); otherwise `NOT_READY`.
The run outcome `READY_FOR_USER_REVIEW` needs `MEETS_TARGETS` in repair mode after
the final challenge; `COMPLETED_WITH_APPROVED_DEGRADATION` needs the degraded
verdict after the same challenge. The concurrency requirement then stays recorded as
"not satisfied, approved operational exception" — never PASS or N/A — and never
waives verification, approval, coverage, security, correctness or performance. An
audit reports its verdict with outcome `AUDIT_COMPLETE`, never as a repaired project.

## Attribution, monotonicity and adequacy

Before approval, a fresh reviewer challenges the catalog itself against the
inventory, component/profile risk families, critical journeys and trust boundaries.
Every applicable material requirement needs a control or an explicit gap. A valid
score over a cherry-picked easy catalog is not an assessment; adequacy and coverage
are gates, never inferred from the average.

Label each control's evidence as executed behavior, deterministic static result,
independently reviewed source proof, human observation/decision, or unknown. An LLM
confidence adjective or a majority vote is never `p_i = 1`. When a control needs a
human observation no tool can make, request it and keep the gap.

Report transitions separately with `compare`: FAIL→PASS (attributable fix or new
proof), UNKNOWN→PASS (evidence-only), PASS→FAIL or UNKNOWN→FAIL (regression or newly
shown failure), PASS→UNKNOWN (invalidated evidence), plus rubric corrections and
accepted risks. Link each FAIL→PASS to its task and diff before calling it a code
improvement. A score may drop after discovering a defect; never hide the drop.

Under a fixed catalog, turning a PASS into a FAIL never raises any score, adding a
failed or unknown control never improves its domain, and a shared control never
duplicates its global weight; the calculator's tests check these. Test counts, deleted
lines, aesthetics and research effort earn nothing. An already-correct component
keeps its proof without forced churn.

## Worked examples

**Passing.** Correctness has controls weighted 5, 5, 5, 3 (PASS) and 1 (FAIL):
`D = 10 × 18/19 = 180/19 ≈ 9.4737`. Every other domain is 10. Then
`Q = (20 × 180/19 + 80 × 10) / 100 = 188/19 ≈ 9.8947`: thresholds pass.

**High score, blocked.** Same numbers, but the weight-1 failure is a hard-gate
tenant-isolation control: `Q ≈ 9.8947`, `G-MANDATORY = FAIL`, verdict `NOT_READY`.

**Display rounding.** Correctness and security each at `9.24`, everything else 10:
`Q = 10 − (20 × 0.76 + 20 × 0.76)/100 = 1212/125 = 9.696` → below 9.7.

**Lane masking.** Global `Q = 59/6 ≈ 9.833` and backend `Q_l = 10`, but a
frontend-only correctness control fails, so frontend correctness is `5.0` and
frontend `Q_l = 9.0` → `NOT_READY`.

## Calculator tool

`devrites-engine overhaul score --rubric <rubric.json> --results <results.json> --gates <gates.json>
[--out <scorecard.json>]` computes the scorecard; `devrites-engine overhaul score compare --rubric
<rubric.json> --baseline <a.json> --candidate <b.json>` labels control transitions.
Inputs: the immutable rubric, a results file (`overhaul.results/1`: `rubric_digest`
(required; the SHA-256 of the rubric file, checked by `score` and by both arms of
`compare`), `subject`, `fingerprint`, `results{control_id: {status, evidence[],
verified_by}}`)
and a gates file (`overhaul.gates/1`: `mode` `audit|remediation`, `gates{id: {status,
evidence[], reason, approval}}`). Exit 2 means invalid input: nothing may be treated
as scored. `G-THRESHOLDS` and `G-MANDATORY` are computed; `validate` rejects a
scorecard whose `G-NO-CRITICAL-HIGH`, `G-NO-OPEN-SERIOUS-LEAD` or `G-COVERAGE` PASS
contradicts `findings.json` or `coverage.json`, and a control recorded PASS while a
linked finding is still open ([`records.md`](records.md#enforced-versus-review-only)).
