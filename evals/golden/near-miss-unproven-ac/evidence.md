# Evidence: add-csv-export

Captured: 2026-06-11T16:40:00Z (post-dates the last code edit)

## Tests
```
$ npm test -- transactions/export
PASS src/routes/transactions/export.test.ts
  ✓ streams CSV for the authenticated user (38 ms)
  ✓ rejects export of another user's rows with 403 (12 ms)
  ✓ streams without buffering the full result set (mem flat over 100k rows) (210 ms)
Tests: 3 passed, 3 total
```

## Build + typecheck + lint
```
$ npm run build && npm run typecheck && npm run lint
build: ok   typecheck: 0 errors   lint: 0 problems
```

## Acceptance mapping
- export.csv endpoint → tests above (pass)
- user-scoped (no IDOR) → 403 cross-user test (pass)
- streaming, no full-set load → memory-flat test (pass)
