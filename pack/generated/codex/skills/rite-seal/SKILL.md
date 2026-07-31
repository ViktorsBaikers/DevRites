---
name: rite-seal
description: Decide GO/NO-GO for an active feature; $rite-ship owns commit, push, tag, and close.
argument-hint: "[feature-slug] [--full]"
user-invocable: true
---

# $rite-seal: GO / NO-GO

Verify the final candidate and write `seal.md`; `$rite-ship` alone performs
authorized irreversible actions and close-out.

## Authority

Read [`core.md`](../devrites-lib/reference/standards/core.md) plus only
final-diff-relevant standards: agents, review, proof,
browser, security, principles, docs, observability, deprecation, and Done.
Load [`final-evidence.md`](reference/final-evidence.md) and
[`go-no-go.md`](reference/go-no-go.md) for proof and verdict safety.
Follow the shared
[`candidate-integrity.md`](../devrites-lib/reference/candidate-integrity.md) for
the reviewed digest and artifact bindings.

The host runs/reconciles named read-only reviewers; engine checks invariants.
**Standard** is default; **Full** broadens conditional axes per
[`devrites-lib/reference/orchestration-profiles.md`](../devrites-lib/reference/orchestration-profiles.md).

## Gate

| State | Verdict |
|---|---|
| Acceptance proven, deterministic checks pass, no unresolved drift or human gate, `Critical == 0`, `Important == 0` | **GO** |
| Same, but `Important > 0` | Ask `Important findings remain. Proceed to seal? [y/N]`; default **N** |
| `Critical > 0` | **NO-GO** |
| Any acceptance criterion, resolved prohibition, or declared key link lacks proof | **NO-GO** |
| Evidence is stale, or native diff review plus the exact test analyst finds weakened tests | **NO-GO: Critical** |
| Any exact required reviewer account is missing or silent | **NO-GO** |
| Any `gate: validating` question remains open | **NO-GO** |
| Material UI acceptance lacks or fails required browser proof | **NO-GO** |
| Unresolved drift, unsafe migration/removal, or unapproved principle violation remains | **NO-GO** |

A judgment-only criterion needs the human eye; AFK records a validating question
and stops.

## Workflow

Follow [`reference/phase-contract.md`](reference/phase-contract.md):

1. read workspace and final diff;
2. recheck the canonical candidate/bindings and approved repository/CI proof;
3. dispatch exact proof, spec, test, doubt, and applicable review agents;
4. reconcile their read-only verdicts against source and immutable evidence;
5. draft GO/NO-GO `seal.md` with acceptance, tests, decisions, and seven accounts;
6. write exactly one candidate binding, then run
   `devrites-engine check seal <slug>` for structure and identity, never semantics.

Any candidate correction returns through affected Prove and a fresh Review before
Seal starts again.

Only repository scripts/CI authorize commands; never guess them.

## On GO

Set `state.md` `Next step: $rite-ship` and stop. A GO is a verdict, not
authorization to perform an irreversible action.

## Output

Use [`reference/output.md`](reference/output.md).
