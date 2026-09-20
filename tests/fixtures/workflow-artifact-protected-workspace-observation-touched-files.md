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

Ordered source aggregate SHA-256: `a07e836d3fe33b550b2a85809a991f406b026907f0cadf11f0b1a8796890020b`.

| File | SHA-256 |
| --- | --- |
| `CONTEXT.md` | `7b777910168e250763c38a5a6a43d43df75ef5122584c53f5ccfc4a2b9bcf7f5` |
| `docs/engine/commands.md` | `5c8795d655dbff2b42828a79de89caf68c0b1f732ec7f6caaeddacb0a86a67c5` |
| `engine/internal/gate/gate.go` | `142f3efde932c6790046c09a4c20da724580fe0b4d2fd217f45c19aa08eba97e` |
| `engine/internal/gate/gate_test.go` | `bbc3b63f7778d3651a119344987c7c1fcbb6611443072dae00c62b901508ef9f` |
| `engine/internal/gate/readiness_binding.go` | `24f3b7c2c915a015aadc047ff7002e097241e7d0529cfd16681bfad68128d719` |
| `engine/internal/gate/readiness_binding_test.go` | `7977fbd79d8d753be7a7329f21b5452551adb977024b3b1525464771dcdb1514` |
| `engine/internal/state/feature.go` | `c0aacd856df6499c305eb34234ca7e080c77f3a571f170950cfe759e9a537e10` |
| `engine/internal/state/observation.go` | `e24e48f0926b2b45612a41bd688e0d294ef774ebb4fa617bb11168e2c9962934` |
| `engine/internal/state/observation_open_other.go` | `5054756800578cff65f9f0f7aa0bd1e36c101e92aadccdc05f7d68b11ba8543a` |
| `engine/internal/state/observation_open_unix.go` | `eb650dbce77f018ea2c4cb6b333ec33124c2d20144c34db87973722c80d47449` |
| `engine/internal/state/observation_test.go` | `73536ae9f0521e6d5be54982dfafb8dd07b97e2f1b09a6d31d611a123f2da59d` |
| `engine/internal/state/observation_unix_test.go` | `d1753787d86d5d892a295a3b97ae3b95a75295e38b8f8791fab8c2fbf782785e` |
| `engine/internal/state/schema.go` | `f3c45b4f3f6fc05210db4c595f2ded8d3874895df99d0529eed245bf956c87d2` |
| `engine/internal/state/state_test.go` | `c2149f3eb03bde2afac8fe081e18d08e4c9ae7c3e83e9f1817d208093880e0aa` |
| `engine/internal/state/status.go` | `5bbdf3c4319944dcc1ca0264c7e11a89b821e5e046883954d447a7167363515d` |
| `engine/tests/gate_test.go` | `2a83630c7faecab632a62d8d0a1ca53d199ef7e56b84bdec5ad801526dbb7ec5` |
| `engine/tests/workspace_observation_migration_test.go` | `96878fa73f228a4edb067fa64ba95265fc6a18d6fd05196fb4894a8c4485b52b` |

## Deliberately untouched

- Candidate/evidence/browser/secret-scan/close-out/product readers remain separate modules.
- `.gitignore`, `.devrites/ACTIVE`, Git history, remotes, dependencies, generated files, schemas, hooks, ADRs, release state, Workspace 3, and later lifecycle phases were not changed.
- No public injection hook was added for `concurrent_change`; deterministic acquisition stages prove it over the generic whole-failure route.
