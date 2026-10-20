## SLICE-001: per-client request quota

Goal: reject a client's request once it has used its whole quota.
Write paths: `internal/quota/quota.go`

### Acceptance criteria (excerpt)

- AC-001: With a limit of N, the first N requests are allowed and request N+1 is rejected.
- AC-002: A limit of zero or less rejects every request.
