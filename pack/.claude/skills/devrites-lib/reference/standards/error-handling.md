# Error handling

Errors are part of the contract, not an afterthought. Make failure loud, local, and
recoverable.

## Fail fast
- Validate preconditions at the top; stop the moment state is invalid, close to the root
  cause. Don't let bad data propagate and surface as a confusing failure three layers
  later.

## No silent catches
- Never swallow an error to make a problem "go away", that hides bugs and corrupts
  state silently.
- Catch the **narrowest** error you can handle, not a blanket catch-all. A
  bare/broad catch masks unrelated failures.
- If you catch, either recover meaningfully, or rethrow/wrap with added context. Don't
  log-and-continue past an error you didn't handle.

## Classify the outcome before retrying

Never retry blind — match the outcome first:

- **Rejected** (refused: validation/authz/conflict): fix input; unchanged retry fails again.
- **Unknown** (timed out mid-call): check state at the source before any retry.
- **Partial** (half-committed): [`data-integrity.md`](data-integrity.md) § partial failure — reconcile or roll back, never resume blind.
- **Clean failure** (not started / fully rolled back): safe to retry after fixing the cause.

**Failing case:** an **Unknown** outcome retried unchanged double-applies (duplicate charge). Idempotency: [`data-integrity.md`](data-integrity.md); outcome taxonomies: [`integration-reliability.md`](integration-reliability.md). Not provable → `cannot_verify` and stop.

## Meaningful messages
- Error messages state what failed, the relevant context (ids, inputs, not secrets),
  and ideally how to recover. Cryptic messages cost hours.
- Distinguish *expected* failures (validation, not-found) from *unexpected* (bugs);
  handle the first as flow, surface the second.

## Fail closed (security-relevant paths)
- On error in an auth/permission/transaction path, **deny and roll back**: never
  default to granting access or committing partial state.

## Logging
- Log key events (failures, access violations, validation errors) with enough context to
  debug. Prefer structured logs (key/value or JSON) over string soup.
- **Never log secrets, credentials, tokens, or personal data.** Show users a clear,
  non-revealing message; keep the detail in the logs for the team.
