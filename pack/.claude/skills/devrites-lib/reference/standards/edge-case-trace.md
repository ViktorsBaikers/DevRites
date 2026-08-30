# Edge-case trace

Use this when requirements, plans, diffs, or proof change a boundary, branch,
validation rule, deletion contract, retryable action, or claim that a path is safe.
The trace finds relevant cases, records why irrelevant ones were dismissed, and prevents
confidence from turning an untested negative claim into a pass.

## Scope before enumeration

Name the observable surface, caller/actor, state or data it owns, external boundaries,
and the must-NOT outcome the author would reject even if the happy path worked. Do not
expand into a whole-system checklist: a class applies only when the changed surface can
reach it.

## Closed probe classes

Probe each relevant class once:

| Class | Questions |
| --- | --- |
| Boundary | Empty/missing, minimum/maximum, off-by-one, oversized, invalid encoding/shape. |
| State | Initial, repeated, stale, terminal, illegal transition, interruption and resume. |
| Ordering | Duplicate, out-of-order, retry, partial completion, clock/time-zone boundary. |
| Concurrency | Competing writer, lost update, cancellation, race, deadlock or resource exhaustion. |
| Authority | Unauthenticated, unauthorized, wrong tenant, forged identity, privilege increase. |
| Dependency | Timeout, partial/invalid response, rate limit, outage, version/config mismatch. |
| Persistence | Transaction split, crash, migration/backfill restart, rollback, retention/deletion. |
| Compatibility | Old/new reader or writer, caller not updated, feature-flag off/on, environment difference. |
| Wiring | Code exists but is not registered, called, awaited, persisted, emitted, or consumed with real data. |
| Removal | Deleted behavior, caller, data, telemetry, docs, or fallback has no surviving owner. |

Route detailed applicable cases to
[`repository-topology.md`](repository-topology.md),
[`data-integrity.md`](data-integrity.md),
[`integration-reliability.md`](integration-reliability.md), or
[`security.md`](security.md); do not repeat those standards here.

## Trace procedure

1. **Walk explicit paths.** Follow every changed condition, loop exit, error, and
   boundary value to the nearest observable outcome.
2. **Walk fixed-set siblings.** A special case for one enum/status/role/mode implies
   every untouched sibling is a path to check.
3. **Follow real wiring.** Verify existence, substance, registration/call path, and
   real data flow. A complete-looking implementation can still be hollow, orphaned,
   or a stub.
4. **Check negative intent.** Ask what silently permitted outcome would violate a
   requirement, invariant, non-goal, or security boundary. Add a prohibition only
   when bespoke intent is not already owned by a standard.
5. **Check removal.** Name the contract removed code carried and its surviving owner,
   or cite the accepted decision that retires it.

## Disposition and evidence

Every applicable case receives one status:

- `covered`: mapped to a REQ/AC and positive discriminating test or observed runtime proof;
- `backstop`: an independent held-out, property/metamorphic, or direct behavioral check
  names the wrong outcome it would detect — and is **exogenous**: not produced or
  executed by the same code path it validates (a check the changed code also controls is
  `covered` evidence, not a backstop);
- `dismissed`: unreachable or irrelevant with a concrete reason and supporting evidence;
- `unresolved`: a material case lacks a fact or proof surface and blocks the owning gate.

Judgment may dismiss a demonstrably irrelevant case; it cannot prove behavior. When a
case is not inferable from available evidence, say `unresolved`/`cannot_verify` rather
than estimating confidence upward.

## Backstop honesty (fail-closed)

A row marked `covered` or `backstop` **must** name an evidence class: test path,
command output, observed runtime, or an independent held-out/property check. A row
with disposition but **no** evidence class is **`cannot_verify`** at Prove/Seal — not
a pass.

**Failing case:** the happy-path suite is green, the trace lists "error path handled"
with no test or runtime proof → Prove blocks until the row gains a discriminating
surface or moves to `unresolved`.

## Outputs

Spec records relevant cases in **Edge Coverage** and bespoke negative intent in
**Prohibitions**. Plan/Vet maps applicable cases to a slice, recovery, and proof. Review
reports only reachable gaps:

```md
[Important] path:line — <trigger> reaches <unhandled outcome>; consequence:
<observable harm>. Required correction: <minimal handling>. Missing proof: <test/signal>.
```

Use the caller's severity scale. Do not create a separate edge score, pad rows with
irrelevant classes, or report a case already handled and proven.
