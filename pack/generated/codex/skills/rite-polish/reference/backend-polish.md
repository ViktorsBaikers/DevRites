# Backend polish (when backend touched)

Runs as Phase 2 of `$rite-polish` when the feature touches server-side code: handlers /
controllers / services / routes / models / migrations / queries / jobs / workers / auth /
schemas. Polishes the **server side** to ship-quality before review, the way UI normalize
+ polish does for the client side.

## Scope detection (BE is in scope if any of these are touched)
- API/route handlers (REST/GraphQL/gRPC); controllers/services/middleware.
- DB layer: models, queries, migrations, schemas, ORM calls.
- Auth/session/permission code; trust-boundary checks.
- Background jobs, workers, schedulers, queue handlers.
- Server-side files in the project's language (`.rb`/`.py`/`.go`/`.rs`/`.java`/`.cs`/
  `.php`/`.ts`-server etc.) outside the UI layer.

## Polish checklist

### Error handling (consistent + meaningful)
- [ ] **Consistent error response shape** across endpoints (e.g. an RFC 7807 problem
      document or the project's existing convention). One shape, not three.
- [ ] **Correct HTTP / protocol status codes** (4xx for client errors, 5xx for server
      errors; not 200 with `{ error: ... }`).
- [ ] **Custom error classes** (or equivalent) so callers can distinguish error kinds.
- [ ] **Fail closed** on auth/permission/transaction errors: deny + roll back; never
      default to allow or partial commit (`error-handling.md`, `security.md`).
- [ ] **Narrow `catch`**: no blanket `catch (e) {}` swallows. If you catch, recover or
      rethrow with context.

### Logging hygiene
- [ ] **Structured logs** (key/value or JSON) with **context**: request id, user/actor
      id, operation, duration.
- [ ] Log the **events that matter**: failures, access violations, validation rejections,
      retries, auth events.
- [ ] **Never log** secrets, tokens, full credentials, PII, full request bodies for
      sensitive endpoints. Mask or omit.
- [ ] No `console.log` / debug prints left in. No noisy "got here" lines.

### Data & queries
- [ ] **No N+1**: fetch in batches / use joins or includes.
- [ ] **No unbounded result sets**: pagination/limits where data can grow.
- [ ] **Parameterized queries** only; never string-built SQL/shell/HTML.
- [ ] **Indexes** exist for new query patterns (or recorded as a follow-up if the project
      adds them by migration).
- [ ] **Transaction boundaries** are right: one logical write = one transaction; rollback
      on error; no partial commits.
- [ ] Don't return more fields than the caller needs.

### API contract
- [ ] Response shape matches the contract the spec set (and what the UI / consumers
      expect: match against `references/` if any).
- [ ] **Idempotency** where applicable (PUT/DELETE; retry-safe POSTs with idempotency
      keys).
- [ ] **Pagination, sorting, filtering** consistent with neighboring endpoints.
- [ ] **Validation at the boundary**: type/length/format/range on untrusted input;
      reject what doesn't match (see `security.md` three-tier).

### Performance (measure first)
- [ ] Hot-path work measured; obvious wins (cache, batch, hoist) applied; perf claims
      cite a number (`devrites-audit perf`).
- [ ] No accidental quadratic loops over growing collections.

### Cleanup
- [ ] **Dead routes / unused endpoints** removed if this feature created them and they're
      unused.
- [ ] Naming + comments in touched server code: clear, intent-revealing, no fake-helpful
      "// gets the user" lines.
- [ ] No leftover `TODO`s without an owner/issue.
- [ ] **Migrations** are reversible where reasonable; destructive ones have rollback notes.

### Anti-slop (code patterns: see `anti-ai-slop.md`)
- [ ] No over-defensive null/length checks layered redundantly.
- [ ] No useless wrapper functions ("`function getUser(id) { return User.find(id); }`").
- [ ] No generic AI naming (`process_data`, `handle_thing`, `do_it`).
- [ ] No "robust" code that catches everything and hides bugs.
- [ ] Didn't go **beyond the spec**: implemented what the spec asked, no extras.

## Rules
- Feature scope only. Don't refactor unrelated server code.
- Re-prove after changes: targeted tests + a real request/response observation.
- A polish change that breaks a test isn't behavior-preserving: revert and reconsider.
