# ADR-0040: Early preparation admission without recursive roadmap blocking

- **Status:** Accepted
- **Date:** 2026-11-06

## Context

ADR-0039 guarded Build-wright admission. A subsequent consumer trace entered
Temper and launched native review without starting its recorded successor;
it consumed a context bundle before any dispatch account. Writer-only checks
therefore left the earlier preparation path unguarded.

Requiring every source to prepare its successor immediately would also recurse
through the entire roadmap: preparing B for A would demand C before A's own
work. Sequence normalization needs a read-only bootstrap that cannot itself
be blocked by the authority it investigates.

## Decision

Extend ADR-0039's deterministic admission boundary to ordinary local
Temper/Define/Plan/Vet context generation and dispatch opening in Autocomplete
at effective capacity greater than one. Both use one validation owner, before
workflow writes. Context does not require an open account: reviewer context
and packet construction precede wave opening. Build keeps its full admission.

- Internal `--prepare-for <parent-source>` identifies existing immediate-successor
  preparation. Canonical identity, parent binding, adjacent positions and both
  chains must agree. Only supported preparation phases/roles qualify; root
  context may omit a role. Dispatch records optional `preparation_for`; context
  records no account. Neither permits product writes or Build.
- Internal `--sequence-research` permits only exact-source, read-only
  `evidence-scout` investigation in preparation phases before the authority gate.
  It is mutually exclusive with `--prepare-for`. Dispatch records optional
  `sequence_research`. The exception grants no scope, approval, phase advancement,
  successor creation, writer admission, readiness or upstream preparation credit.
- Derive preparatory-only B from its validated immediate nonterminal parent A,
  absence of ordinary forward/Build-wright admission and absence of normal handoff to B.
  This applies before the first marked wave. Observe B's continuation without
  making C an actionable prerequisite for A. Ordinary B admission still checks
  C before publication; that admission, terminal A or normal ordered handoff
  releases the routing boundary. ACTIVE selection represents native handoff,
  not proof of implementation or approval.
  Read-only Build accounts do not release it. Explicit descendant `next` and
  `check continuation` honor inherited waiting instead of mandating a local cursor;
  other actionable global work and independently admitted proof remain visible.
- Preserve completed account history and existing pending/abandoned actor and
  unfinished writer integration protection. Settled purpose bindings remain
  historical after authorized replanning, not current-parent authority or new
  preparation credit; unsettled mismatches still fail closed.
- A resolved finite tail needs no successor. New-opening opt-out or budget
  exhaustion does not erase an already-created child. Existing narrow deferrals,
  human gates, claims, capacity, isolation and protected proof remain authoritative.

No new user Autocomplete flag, model client, launch bridge, receipt or queue.
Standalone/cap1 and initial Spec/Clarify/bootstrap behavior remain unchanged.
Actual native launches still require registered host handles; declared purpose
metadata does not prove profile, checkout or semantic progress.

## Alternatives considered

| Option | Why rejected |
| --- | --- |
| Keep the gate only on Build waves | Earlier context and native planning can bypass it. |
| Require an open account before producing context | Reviewer opening requires a packet built from that context. |
| Require B's successor before admitting B's preparation for A | Recurses through the roadmap instead of pipelining one successor. |
| Use a preparation marker as the only routing boundary | The creation/context interval precedes the first marked wave. |
| Treat normalization research as ordinary planning admission | Conflicting authority blocks its own investigation. |

## Consequences

The engine rejects the observed early CLI path, not merely its eventual writer.
Native hosts retain semantic scope, ordered handoff, exact profile/model/cwd,
result reconciliation and publication authority. Arbitrary host compliance is
not an engine guarantee.

Guard regressions: `engine/internal/lib/frontier_preparation_test.go`.
