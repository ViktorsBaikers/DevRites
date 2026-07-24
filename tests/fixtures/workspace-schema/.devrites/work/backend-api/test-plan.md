# Test plan

## Build-entry preflight
| Gate | Command | Cwd | Expected | Prerequisites | Provenance to recapture |
| --- | --- | --- | --- | --- | --- |
| focused | fixture validation | repository | exit 0 | none | fixture checksum |

## Per-gap test requirements
| ID | Path / flow | Test file | Asserts | Kind | Slice | Priority |
| --- | --- | --- | --- | --- | --- | --- |
| T1 | primary flow | fixture proof | input → expected result | integration | SLICE-001 | P1 |

## Acceptance → test map
- AC-001 → T1
- AC-002 → T1
