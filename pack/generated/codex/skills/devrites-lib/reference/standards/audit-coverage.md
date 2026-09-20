# Coverage-led auditing

> Applies when: auditing a surface larger than one feature diff — a repo, subsystem,
> or trust boundary — where "I looked at it" must become countable coverage.

A surface audit must answer *what the run covered and what it never examined* — with
a coverage ledger, not a summary paragraph.

## Coverage ledger

The coordinator alone maintains the ledger; a unit is one auditable cell — surface
(path/endpoint/command) × trust boundary × check class.

States: `planned` → `assigned` → `candidate` (produced a finding candidate) →
`covered` (evidence recorded); plus `deferred` (postponed, reason recorded),
`blocked` (exact blocker recorded), and `out_of_scope` (declared-scope exclusion —
never written as `covered`). `deferred` and `blocked` are honest states; a unit
silently dropped is a lie.

Instruments obey the same honesty rule: a scanner or metric that could not read
its target reports `unmeasured`/`no_data`, never a placeholder presented as a
measurement. A health score computed over zero parseable files is not a 100.
When the audit covers whole-repo structure, score it per
[`architecture-health.md`](architecture-health.md) and record before/after
deltas as ledger evidence.

## Findings carry three verdicts, not two

- **`confirmed`** — complete source trace plus a bounded observed or owner-observable
  result; names the lower-trust principal, crossed boundary, affected
  principal/resource, and concrete effect.
- **`needs_validation`** — a real lead blocked on one exact unresolved fact, which is
  named. Carries **no severity**: severity without established impact is theater.
- **`rejected`** — a disproved candidate kept with its disproof; it suppresses
  re-raising the same unchanged claim, never coverage of its unit.

Hardening notes are not findings: if one layer already prevents the attack, the
absent second layer is a depth observation, not a vulnerability.

## Finder never verifies its own find

Every surviving candidate goes to a fresh verifier that did not produce it and tries
to **disprove** it, per [`agents.md`](agents.md) § Independence — a seeded packet
voids the verification. Promotions to `confirmed` and material changes to root
cause, trace, reproduction, impact, or severity require fresh independent
verification. Without that independent evidence, keep the candidate unresolved;
a rewritten explanation is not verification.

## Critics between waves

After each hunting wave, a **coverage critic** — a fresh agent comparing the ledger
with targeted source and entry-point evidence — looks for holes: unseeded surfaces,
boundaries with no check class,
units stuck `assigned`, `rejected` patterns suggesting a hunter fires blanks.
Accepted discoveries become `planned` units. Reserve critic and verifier capacity
**before** spending on hunters; a wave launched with no verification reserve hunts
for findings nobody may claim.

## Re-runs are additive, never assumed

A prior ledger must be re-checked against current source — a prior commit ref is not
evidence a path is unchanged. Prior `confirmed` carries forward only when its source
is unchanged **and** it re-passes current verification; prior `needs_validation` /
`blocked` / `deferred` become current work and never suppress a unit. A partial prior
run contributes recorded evidence and gaps, never an implied "the rest is fine"; no
prior ledger → the report says so.

## Profiles scale breadth, never the bar

`quick` (one hunter wave + one critic pass, coarser units), `standard` (waves to
critic-clean, per-candidate fresh verifier), `deep` (units split per
subsystem/lifecycle; carried records re-verified). Profiles change granularity and
redundancy, never the evidence bar: boundary-plus-result, independent verification,
three verdicts, and honest `deferred`/`blocked` survive every profile. A scoped run
presents itself as partial coverage.

## Budget discipline

A budget counts agent invocations, not effort. Reserve critic and verifier capacity
before spending on hunters; a budget that cannot fund reconnaissance plus required
reserves funds nothing — narrow scope or profile instead of silently thinning
evidence. A run exhausted mid-verification ends `incomplete` with unvalidated
candidates still linked to `candidate` units — never relabeled, never reported
complete.

## Target execution is sandboxed

When a unit requires *running* the target — a build, a test, a probe — execute
it under containment, never in the operator's ambient environment:

- No external network and no live-service probing from the audit run.
- No credentials, tokens, or secrets visible to target-controlled processes.
- Only commands the repository or CI itself sanctions run verbatim; invented
  commands are findings to propose, not actions to take.
- Output captured through the harness, not through target-written files the
  run then trusts.

An audit that executed unaudited code with ambient privileges has crossed the
boundary it was measuring; that run's `confirmed` rows name the execution
context so reviewers can judge contamination.

**Failing case:** "no critical findings" reported while 40% of units sit `planned` —
the honest sentence was "60% of the surface was examined."
