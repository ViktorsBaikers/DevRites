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

Ordered source aggregate SHA-256: `234697a68e6fa774e13ffcb793b4a906fd8e22649be83cefcbcfae0afcaca544`.

| File | SHA-256 |
| --- | --- |
| `CONTEXT.md` | `7b777910168e250763c38a5a6a43d43df75ef5122584c53f5ccfc4a2b9bcf7f5` |
| `docs/engine/commands.md` | `2533b28c013dbd2bed4d84860c9e454ee4151ac0f625840f1fe5eb27df51c12d` |
| `engine/internal/gate/gate.go` | `27e6fd1fd06f478fcce185da59813901693df6254e635a73943405acf51869cc` |
| `engine/internal/gate/gate_test.go` | `31418952c6bb6189cf0a643603b2176ae9a53ba820c552b5c5deb334d2890611` |
| `engine/internal/gate/readiness_binding.go` | `24f3b7c2c915a015aadc047ff7002e097241e7d0529cfd16681bfad68128d719` |
| `engine/internal/gate/readiness_binding_test.go` | `2242b5dadc21a98bad3bfb5afcb517841d143f03e8765b44897d99108bf6811b` |
| `engine/internal/state/feature.go` | `c0aacd856df6499c305eb34234ca7e080c77f3a571f170950cfe759e9a537e10` |
| `engine/internal/state/observation.go` | `2a52f991415ee1d53bf50e58d0cc0890bde40e93e7d8ac4e13c049405db82c89` |
| `engine/internal/state/observation_open_other.go` | `5054756800578cff65f9f0f7aa0bd1e36c101e92aadccdc05f7d68b11ba8543a` |
| `engine/internal/state/observation_open_unix.go` | `eb650dbce77f018ea2c4cb6b333ec33124c2d20144c34db87973722c80d47449` |
| `engine/internal/state/observation_test.go` | `6d803aa1655a3fbbbfd6a1bdaac374cef161b14f336e52b7778d8d912633d1fb` |
| `engine/internal/state/observation_unix_test.go` | `d1753787d86d5d892a295a3b97ae3b95a75295e38b8f8791fab8c2fbf782785e` |
| `engine/internal/state/schema.go` | `8bc7e8575ae94beeaa0ebd8676525a0970cb03ab9582e3bbb8e5fcaa24fced32` |
| `engine/internal/state/state_test.go` | `4275340b10bfeba2b726f8c16e06cc6240df9d2d84c01e84c5b7c2e6f2d72be6` |
| `engine/internal/state/status.go` | `dfb601037d73228ed574a584f55e9aae364bfa329c3b47a100682a25dbad81f5` |
| `engine/tests/gate_test.go` | `f9bd9ba7acab7b7dd46f45f9b33828ba31bf0e0da0e389b4e50caae204bebdaf` |
| `engine/tests/workspace_observation_migration_test.go` | `7ae184db9c3d395f2b48a5bd915fa96b72f0a44c5d319d88114fa397c99882a4` |

## Deliberately untouched

- Candidate/evidence/browser/secret-scan/close-out/product readers remain separate modules.
- `.gitignore`, `.devrites/ACTIVE`, Git history, remotes, dependencies, generated files, schemas, hooks, ADRs, release state, Workspace 3, and later lifecycle phases were not changed.
- No public injection hook was added for `concurrent_change`; deterministic acquisition stages prove it over the generic whole-failure route.
