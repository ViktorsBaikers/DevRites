# Error handling

Errors are part of the contract, not an afterthought. Make failure loud, local, and
recoverable.

## Fail fast
- Validate preconditions at the top; stop the moment state is invalid, close to the root
  cause. Don't let bad data propagate and surface as a confusing failure three layers
  later.

## No silent catches
- Never swallow an error to make a problem "go away" — that hides bugs and corrupts
  state silently.
- Catch the **narrowest** error you can actually handle, not a blanket catch-all. A
  bare/broad catch masks unrelated failures.
- If you catch, either recover meaningfully, or rethrow/wrap with added context. Don't
  log-and-continue past an error you didn't handle.

## Meaningful messages
- Error messages state what failed, the relevant context (ids, inputs — not secrets),
  and ideally how to recover. Cryptic messages cost hours.
- Distinguish *expected* failures (validation, not-found) from *unexpected* (bugs);
  handle the first as flow, surface the second.

## Fail closed (security-relevant paths)
- On error in an auth/permission/transaction path, **deny and roll back** — never
  default to granting access or committing partial state.

## Logging
- Log key events (failures, access violations, validation errors) with enough context to
  debug. Prefer structured logs (key/value or JSON) over string soup.
- **Never log secrets, credentials, tokens, or personal data.** Show users a clear,
  non-revealing message; keep the detail in the logs for the team.
