# Gates

- [x] G1: AC-001 admission → root authorship → retained source → identity freeze → transaction → proof → resume contract present; no stale migration or generic-materializer authority; ten adapters hold only module link plus phase-local entries
  EVIDENCE: evidenced by contract corpus and adapter rows in evidence.md; AC-001 marked complete in spec.md
- [x] G2: AC-002 one owner atomically retains source and freezes ordered identity plus Vet binding before active mutation; busy or malformed authority performs zero target writes
  EVIDENCE: evidenced by owner-acquire/source-promote traces in evidence.md; AC-002 marked complete in spec.md
- [x] G3: AC-003 partial-write reconciliation reaches exact terminal states — `CLEANED` on success, retryable `FAILED` with preimages, `EXHAUSTED` at fingerprint 3 — and post-`PROVED` recovery preserves targets
  EVIDENCE: evidenced by operation/route table traces in evidence.md; AC-003 marked complete in spec.md
- [x] G4: AC-004 every hostile input (swap/forgery/loss, contention, tear, kill) selects the exact safe route; process group reaped; unrelated files untouched
  EVIDENCE: evidenced by route-table grading in evidence.md; AC-004 marked complete in spec.md
- [x] G5: AC-005 marker-owned durable evidence preserves outside bytes and one standalone candidate binding; workflow paths never enter product candidate rows
  EVIDENCE: evidenced by evidence-update and product-separation traces; AC-005 marked complete in spec.md
- [x] G6: AC-006 dedicated test drives disposable filesystem behavior, grades fixed traces, and records root-caller verbatim output/TTHW
  EVIDENCE: `bash tests/workflow-artifact-identity-test.sh` passed in proof evidence; AC-006 marked complete in spec.md
- [x] G7: AC-007 sole-writer discipline holds — driver is the only authored/generated writer, delivery journal recovers preimages, Claude/Codex parity, instruction cap, unchanged protected identities
  EVIDENCE: evidenced by writer-accountability and parity traces in evidence.md; AC-007 marked complete in spec.md
