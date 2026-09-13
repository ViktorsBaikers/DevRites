# Integration reliability

Load this for third-party APIs, webhooks, queues, background jobs, caches, or
cross-service calls. The boundary contract includes failure, timing, duplication, and
recovery—not only the success payload.

## Contract the boundary

Name the provider and consumer, version, authentication, request/response or event
schema, optional and unknown fields, ordering guarantee, rate limit, timeout budget,
retry responsibility, idempotency key, and user-visible degradation. Validate an
external response as hostile input before trusted code consumes it.

**Uploads and paginated collections are boundary contracts, not transport details.** An
upload names body and expanded-size caps, chunk/resume behavior, partial-upload
reconciliation, and the rejection path for oversized or malformed payloads. A paginated
collection names cursor/offset semantics, page-size cap, the terminal “no more pages”
signal, and how rows written during paging surface — skipped or duplicated rows are
contract data, not surprises. **Failing case:** a client pages by offset while rows
insert ahead of it and silently misses records; a boundary contract that cannot name
this behavior fails Vet.

For every call or delivery, classify the observed outcome:

| Outcome | Required behavior |
| --- | --- |
| Success | Validate the complete required shape; tolerate documented additive fields. |
| Invalid or partial response | Reject or use an explicitly safe degraded path; never manufacture required data. |
| Authentication/expired credential | Stop retries that cannot succeed, surface ownership, and reveal no credential. |
| Rate limit/overload | Honor provider guidance when trustworthy, bound backoff, and expose delayed/degraded status. |
| Timeout/network partition | Treat effect as unknown unless the protocol proves otherwise; reconcile before retrying a non-idempotent action. |
| Provider outage/version change | Open the circuit or shed load when the project supports it; retain a bounded recovery path and compatibility signal. |

## Retry and delivery rules

- Retry only a named transient failure and only when the operation is idempotent or has
  a durable deduplication key. Bound attempts, elapsed time, and exponential backoff;
  add jitter when many workers could synchronize.
- Assign one retry owner across SDK, worker and queue, or explicitly share one
  end-to-end attempt/deadline budget that redelivery cannot reset. Prove exhaustion
  by counting physical downstream calls, including inner retries, and advancing a
  controlled/injected clock to the deadline. Assert exhaustion, not wall-clock
  duration ([testing.md](testing.md#determinism-no-flaky-tests)); elapsed-time logs
  are observations, never timing assertions in tests.
  **Failing case:** each of three layers permits four attempts, producing 64 calls;
  three passing local retry tests do not prove the operation's budget. Budget
  overrun blocks proof and Seal until retry ownership or shared accounting is fixed.
- A timeout is not proof the provider did nothing. Query by idempotency key/status or
  reconcile before creating a second effect.
- Webhook/queue consumers acknowledge only after durable success or durable handoff.
  Duplicate delivery, duplicate jobs, and out-of-order delivery are normal inputs:
  deduplicate durably and reject, buffer, or reconcile stale sequence/version values by contract.
  **Failing case:** the handler acks before the durable write; a duplicate delivery
  creates a second effect.
- A poison message must not block the partition forever. Bound redelivery, retain the
  failure reason without secrets, move to the project's quarantine/dead-letter path,
  and define replay after correction.
- A queue emits observability signals — backlog depth, poison/quarantine count,
  reconciliation lag, saturation, ownership — with [`observability.md`](observability.md)
  as the signal-taxonomy owner. A queue backlog still needs an accepted
  capacity/drain/recovery action; auto-scaling without downstream capacity protection
  only moves the outage.

## Partial failure and recovery

Map each multi-step effect as `not started | committed | unknown | compensating |
reconciled`. If one system commits and another fails, name the durable record that
drives retry or compensation. Do not catch/log/continue into a false success.

For synchronous versus asynchronous design, apply [`patterns.md`](patterns.md); async
processing changes the contract to accepted/pending/failed/retryable; it does not make the
failure disappear.

## Cache and partition behavior

- Define source of truth, key scope (including tenant), invalidation trigger, TTL, and
  acceptable staleness. Cache deletion failure and stale reads need an observed path.
- Never use cache presence as authorization. On partition or cache outage, choose an
  explicit fail-open or fail-closed behavior based on the protected invariant.
- After reconnect, reconcile version/order rather than assuming arrival order equals
  commit order.

## Required plan and proof

For each boundary, `plan.md` records:

| Boundary | Timeout/retry/idempotency | Duplicate/order/partial handling | Degradation/recovery | Observability | Proof |
| --- | --- | --- | --- | --- | --- |
| `<provider → consumer>` | `<budgets/key>` | `<rules>` | `<user/system path>` | `<signals/owner>` | `<test/rehearsal>` |

Proof drives success, invalid shape, partial response, auth failure, rate limit, timeout,
duplicate, out-of-order delivery, and outage when relevant. Deterministic cases use a
contract-capable fake or sandbox per [`testing.md`](testing.md) — never a mock that
simply returns the expected payload (that tests the stub, not the boundary) — plus at
least one real boundary check when authorized and safe.

## Stop conditions

Stop planning or Seal when a non-idempotent unknown outcome can be blindly retried, a
consumer can acknowledge before durable handling, a poison/backlog path has no owner, a
partial response can become success silently, or outage recovery and monitoring are
missing. Unavailable provider evidence is `cannot_verify`, not a pass.
