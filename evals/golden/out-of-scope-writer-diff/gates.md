# Gates

- [x] G1: `GET /transactions/export.csv` streams the caller's transactions as CSV (AC-001)
  CHECK: npm test -- transactions/export
  EXPECT: exit 0
  CWD: project
  EVIDENCE: automatic-evidence=v1; def=75936df508d830c915ac0e4afaaaac22e6a99781ab775ac499d51e388d1d04f8; exit=0; expect=matched; output-sha256=28d3b9e880a77975493dc7e359144c0295a4f694cfe0af4f928c22307bc5c320; output-bytes=7; at=2025-01-01T00:00:00Z
- [x] G2: the export rejects access to another user's rows (AC-002)
  EVIDENCE: T1 cross-user case returns 403; EVID-002 in evidence.md
- [x] G3: a 100k-row export does not load the full result set into memory (AC-003)
  EVIDENCE: T2 memory-flat assertion over 100k synthetic rows; EVID-003 in evidence.md
