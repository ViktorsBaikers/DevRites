# Touched files: Workspace Observation

## Touched files

Completed two-slice candidate contains one concrete Workspace Observation module, retained-only Status/Gate/readiness adapters, contracted old readers, exact external diagnostics, tests, docs, and domain wording. Workspace records and user-owned `.gitignore` are excluded from candidate identity.

## Candidate manifest

| State | File | Slice | Reason |
| --- | --- | --- | --- |
| present | `CONTEXT.md` | SLICE-001 | Describe retained Workspace Observation without point-in-time overclaim. |
| present | `docs/engine/commands.md` | SLICE-002 | Document states, mappings, applicability, recovery, channels, exits, and limits. |
| present | `engine/internal/gate/gate.go` | SLICE-002 | Thread one observation through section and artifact requirements, questions, diagnostics, and readiness. |
| present | `engine/internal/gate/gate_test.go` | SLICE-002 | Prove retained Gate flow, section completeness, selected diagnostics, and seven-code recovery. |
| present | `engine/internal/gate/readiness_binding.go` | SLICE-002 | Preserve digest v1 and established fenced-block structural binding semantics from retained facts. |
| present | `engine/internal/gate/readiness_binding_test.go` | SLICE-002 | Prove digest, optional presence, retained bytes, diagnostics, and backtick/tilde structural edges. |
| present | `engine/internal/state/feature.go` | SLICE-002 | Remove superseded Feature content reader while retaining metadata-only discovery. |
| present | `engine/internal/state/observation.go` | both | Own inventory, strict confinement, bounds, classification, retained facts, and detected-change rejection. |
| present | `engine/internal/state/observation_open_other.go` | SLICE-001 | Open rooted files read-only on non-Unix targets. |
| present | `engine/internal/state/observation_open_unix.go` | SLICE-001 | Add nonblocking rooted opens on Unix before descriptor classification. |
| present | `engine/internal/state/observation_test.go` | SLICE-001 | Prove inventory, states, bytes, bounds, strict confinement, copies, mutation, and non-guarantees. |
| present | `engine/internal/state/observation_unix_test.go` | SLICE-001 | Prove successful FIFO substitution setup and bounded nonblocking completion. |
| present | `engine/internal/state/schema.go` | SLICE-001 | Define ArtifactPath as observed logical identity while preserving Phase Policy confinement. |
| present | `engine/internal/state/state_test.go` | both | Prove Status retention, rendering, selected diagnostics, errors, and contracted reader behavior. |
| present | `engine/internal/state/status.go` | both | Compute Status from one retained observation with private section assessment. |
| present | `engine/tests/gate_test.go` | SLICE-002 | Pin exact healthy, binding, missing, stale, Seal, diagnostic, and whole-failure CLI contracts. |
| present | `engine/tests/workspace_observation_migration_test.go` | SLICE-002 | Enforce package-qualified contraction, exact acquisition, receiver-aware reachability, and docs parity. |

## Source hashes

Ordered source aggregate SHA-256: `9ab6e686078d6e4612a15146d9daf1bb9c3050838964bf36dde24ca0e6881567`.

| File | SHA-256 |
| --- | --- |
| `CONTEXT.md` | `7b0a9fede9c35181030bf212847c32e28f488105a7fbb64e3feae8252a77badf` |
| `docs/engine/commands.md` | `665e8f89160d5fbf80b6e1ff639cf0e24ec6afa2d227b1498819f121a4cea325` |
| `engine/internal/gate/gate.go` | `f8b463855b7c84de0fafa91103e6689b5e6e9188eec4f12fd5811fd2e5c408dc` |
| `engine/internal/gate/gate_test.go` | `44bf4a499a7906d62ab7c532464bd6ae107d4f1837c98a917f3ece07de7123a3` |
| `engine/internal/gate/readiness_binding.go` | `38702d46fffaf636767b59613757630403279353b003ad3eb01299997928581e` |
| `engine/internal/gate/readiness_binding_test.go` | `e82cee433908c056c0dc103396ffa1978d3af67aa2abb2bc101aa74a954ce165` |
| `engine/internal/state/feature.go` | `c0aacd856df6499c305eb34234ca7e080c77f3a571f170950cfe759e9a537e10` |
| `engine/internal/state/observation.go` | `3c4bfbcfbb07a8f876b299c62f14d60e558ab8b7f4e395126bccb90645c1489a` |
| `engine/internal/state/observation_open_other.go` | `5054756800578cff65f9f0f7aa0bd1e36c101e92aadccdc05f7d68b11ba8543a` |
| `engine/internal/state/observation_open_unix.go` | `eb650dbce77f018ea2c4cb6b333ec33124c2d20144c34db87973722c80d47449` |
| `engine/internal/state/observation_test.go` | `2aa944e0f091d9cbc1ea266307e02d0ecaff51777798163c4b4024108767145d` |
| `engine/internal/state/observation_unix_test.go` | `d1753787d86d5d892a295a3b97ae3b95a75295e38b8f8791fab8c2fbf782785e` |
| `engine/internal/state/schema.go` | `1375c0aae4302b6b744680cb0a48bc96b5a7bd7c48710a95596b2652c8ad032b` |
| `engine/internal/state/state_test.go` | `20dc824c638eeda54dd79ed2ce7887de845bf3ec4335ea01e3f6965f9a6ed2c9` |
| `engine/internal/state/status.go` | `68fad1ef014af93333182fdd44197e0fb5cb5a4002dda09c3052917f6828b257` |
| `engine/tests/gate_test.go` | `81fd365288c608056e8ac43e8de52e6afd04deda6cfbf3c85ed9d5c8eeb2018a` |
| `engine/tests/workspace_observation_migration_test.go` | `7ae184db9c3d395f2b48a5bd915fa96b72f0a44c5d319d88114fa397c99882a4` |

## Deliberately untouched

- Candidate/evidence/browser/secret-scan/close-out/product readers remain separate modules.
- `.gitignore`, `.devrites/ACTIVE`, Git history, remotes, dependencies, generated files, schemas, hooks, ADRs, release state, Workspace 3, and later lifecycle phases were not changed.
- No public injection hook was added for `concurrent_change`; deterministic acquisition stages prove it over the generic whole-failure route.
