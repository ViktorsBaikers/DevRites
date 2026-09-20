# Assumptions: Workflow Artifact Identity

## Assumption register

| ID | Assumption | Evidence | Confidence | Owner | Validation | Consequence |
| --- | --- | --- | --- | --- | --- | --- |
| ASM-001 | `DEVRITES_ROOT` permits disposable engine separation and module-selected Go 1.26.5 is already available for self-build. | `engine/main.go:108-123`; rooted tests; Vet observed `go -C engine env GOVERSION` = `go1.26.5` | high | Build | recapture module toolchain before mutation; env-unset dedicated mode builds actual CLI and invokes before/after evidence | If unavailable, block before fixture; no mock or network fallback. |
| ASM-002 | Normal host generation honors private same-filesystem `DEVRITES_HOST_ARTIFACT_DIR` and mirrors canonical Markdown to complete Claude/Codex tree. | existing generator contract and `tests/host-artifacts-test.sh` | high | bounded wright + hash-bound driver | driver-owned 16/22 delivery journal, full stage/install/restart/rollback, parity, sibling equality; root verify-only; DEC-042 launcher does not write or recover | If false, bounded recovery wright restores and stops; never target default tree or hand-edit derivatives. |
| ASM-003 | Removing stale migration prose yields enough instruction-budget headroom for deeper canonical contract. | stale sections exceed planned additions; current total 854,997/855,000 | medium | Build | instruction baseline after exact edit | If false, compact duplicate adapter prose; do not raise limits without Vet. |

No human-owned assumption remains.
