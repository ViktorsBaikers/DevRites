# Security

Assume inputs are hostile and trust is earned. Security is a property of every change
that touches input, auth, data, or external systems — not a separate phase.

## Treat all external input as untrusted
- Validate on the **server side** before use: type, length, format, range. Reject what
  doesn't match the expected pattern; don't try to "sanitize" your way around bad input.
- Prevent injection: use parameterized queries, never string-built SQL/shell/HTML;
  encode/escape at output boundaries.
- Don't trust client-supplied trust signals — IDs, roles, prices. Re-check server-side.

## Least privilege
- Code, service accounts, DB connections, API tokens, and file access run with the
  **minimum** permissions needed. Scope tightly so a breach is contained.
- Check authorization on every sensitive action, server-side. Guard against IDOR (acting
  on another user's object by changing an id).

## Secrets
- Never hard-code secrets or commit them. Use the project's secret mechanism / env /
  vault. Never log secrets, tokens, or personal data.
- Deliver secrets just-in-time and scope them; rotate on exposure.

## Fail closed
On any security-relevant error, deny access and roll back — never default to allow or to
a half-committed state.

## Dependencies & data
- Audit new/updated dependencies; don't add known-vulnerable versions.
- Expose the least data necessary; encrypt sensitive data where required; don't return or
  log more than the caller needs.

## Trust boundary (three tiers)
untrusted (user/external input) → boundary (explicit validation + authz) → trusted (core
logic on known-good data). Every value must cross the boundary tier deliberately; one
that skips it is a finding.
