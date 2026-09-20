# Engine control plane

This directory is the deterministic Go control plane. Native skills and agents
own semantic judgment; this binary owns structure, identity, and ID-reference
completeness.

## Invariants

- Build with `CGO_ENABLED=0`. The hot path is stdlib-only (no CLI framework,
  no model client). Workspace policy, state, proof, and install-application
  packages stay network-free. Direct `devrites-engine update` is the bounded,
  checksummed exception (ADR-0008, ADR-0024, ADR-0028).
- Date-derived `resolve` output reads the clock through `DEVRITES_NOW`
  (RFC-3339 or `YYYY-MM-DD`). Operational timestamps (leases, install
  manifests) use wall time by design (ADR-0006).
- `check readiness` / `check seal` do not parse reviewer prose or infer
  coverage quality. They may require that a canonical `AC-###` ID declared in
  `spec.md` also appears as a literal substring in `tasks.md` and
  `test-plan.md` when those artifacts are required.
- Local full gate: `make quality` (`gofmt`, `go vet`, `staticcheck`,
  `govulncheck`, `gosec`, `golangci-lint`, `osv-scanner`,
  `go test -race -shuffle=on -count=1`).

See [`../CONTEXT.md`](../CONTEXT.md) and [`../docs/adr/README.md`](../docs/adr/README.md).
