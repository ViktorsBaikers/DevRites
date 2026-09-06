# Test plan: add CSV export

## Build-entry preflight
| Gate | Command | Cwd | Expected | Prerequisites | Provenance to recapture |
| --- | --- | --- | --- | --- | --- |
| focused tests | npm test -- transactions/export | project | exit 0 | synthetic rows | Node and package-lock digests |

## Per-gap test requirements
| ID | Path / flow | Test file | Asserts | Kind | Slice | Priority |
| --- | --- | --- | --- | --- | --- | --- |
| T1 | authenticated export | export.test.ts | CSV response and caller scope | integration | SLICE-001, SLICE-002 | P0 |
| T2 | large export | export.test.ts | bounded memory use | performance | SLICE-001 | P1 |

## Acceptance → test map
- AC-001 → T1
- AC-002 → T1
- AC-003 → T2
