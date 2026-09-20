# Evidence: Workflow Artifact Identity

Candidate SHA-256: a6b102c354d756cb8d89bb395b54704ed9a9206ec1b652c7c24ff77426db453e

## Evidence log

Proof date: 2026-08-21. Candidate contains 38 destinations: 16 authored paths and 22 normal host derivatives. Browser proof is not applicable. Behavioral validation is provider-neutral schema and deterministic contract proof; no live model/provider behavior is claimed.

### EVID-001 — Canonical module and adapter locality (SLICE-001; AC-001)

| Proof | Observation | Result | Limitation |
| --- | --- | --- | --- |
| Canonical linkage | dedicated contract and routing proof over the canonical module, ten phase adapters, and generated hosts | one admission/identity/transaction/proof interface; stale migration and generic materializer authority absent | semantic host instructions, not live provider execution |
| Deletion test | canonical module link and adapter-thinness assertions | deleting the module would redistribute admission, source, identity, transaction, proof, resume, and product-separation rules across callers | static module-depth proof |

### EVID-002 — Sole-wright delivery transaction (SLICE-001; AC-007)

| Proof | Observation | Result | Limitation |
| --- | --- | --- | --- |
| Frozen delivery candidate | authored aggregate `fba5ca20a13743dfe305ee728d30beb7e4546f8eaab5b830ee1050890038765b` | exact 16 authored paths; driver `d31daea6970aaa3035580fd77dd1da201ee6fb425de3caeb6c2374a71cb6d6e6`; canonical module `f6f12115c3a06b40ba1f499719c10b19cc41b119ba781e40e1de082c09e98e88`; baseline `b63c058bf281be393926e2e44e4cb6b384b03fc0730ed51fcbda18c5fc4e84dc` | workspace bookkeeping excluded from destination set |
| Final delivery | `.generated-install/fba5ca20a13743dfe305ee728d30beb7e4546f8eaab5b830ee1050890038765b/journal.json` | SHA-256 `8c9e4ab00324712ed417963c17e198b524545c12acdbbc2390975c2c51918b40`; terminal `CLEANED` sequence 137 after 835.86 seconds; 16/16 exact gate records; exact 16 authored/22 generated readback; exactly the Claude/Codex module mirrors changed; transaction retains only lock, journal, sidecar | delivery `COMMITTED` is not a Git commit |
| Recursive outside evidence | immutable tuple sidecar | 94,409 rows; 16,331,309 bytes; SHA-256 `1febf51f336af78ae84475ca9b39638ea590c29ebf900e5e3536f9364f2ff4aa`; transaction container and prior failed/successful histories included; selected subtree excluded; independent replay passed | delivery-time state; later bookkeeping is reconciled separately by exact records |

### EVID-003 — Identity, source, and transaction safety (SLICE-001; AC-002–AC-004)

| Proof | Observation | Result | Limitation |
| --- | --- | --- | --- |
| Identity and limits | dedicated contract | byte golden vector; ordered path/mode/content identity; checked relational minima/overflow; exact private namespace and `fcntl.flock` ownership passed | deterministic filesystem fixtures |
| Source lifecycle | dedicated contract | authority/ready promotion, held immutable bytes, swap/forgery/loss, authenticated stale rollover suffixes, retry retention, success/exhaustion GC passed | same-user deliberate detached-session escape is outside the admitted proof interface |
| Crash recovery | dedicated contract | every canonical operation boundary, complete-write failures, journal temporaries, rollback, failure cleanup, retry epochs, exhaustion, and post-`PROVED` cleanup passed | private fixture paths omitted |
| Generator confinement | synchronized post-derivation ancestor replacement fixture | canonical generator derives `ROOT=.` under held repository descriptor; source/stage remain in held tree; replacement tree unchanged | exact current generator behavior is hash-bound by proof provenance |

### EVID-004 — Evidence and proof-process integrity (SLICE-001; AC-004, AC-005)

| Proof | Observation | Result | Limitation |
| --- | --- | --- | --- |
| Marker-owned evidence | complete prior field/target/attempt grammar and hostile mutants | outside bytes and one candidate binding preserved; malformed width/name/value/history rejected; odd escaped pipe accepted; bare/even-backslash delimiters rejected | fixture journal, not secret-bearing production data |
| Process groups | nonzero, wrong-signal, overflow, command/aggregate timeout, child survivor, TERM-resistant child, and termination mutants | no failure reaches `PROVED`; ordered TERM receipt, grace survival, KILL, group exit, and leader reap discriminated | deliberate new-session escape excluded by the stated interface |
| Finite diagnostics | exact reason/boundary rows, four-cell grammar, mutations, and unknown-value collapse | one bounded allowlisted public line; lowercase/unrecognized extra row rejected; no path/input/error reflection | deterministic diagnostics only |

### EVID-005 — Routes, adapters, and product separation (SLICE-001; AC-005, AC-006)

| Proof | Observation | Result | Limitation |
| --- | --- | --- | --- |
| Route matrix | canonical route/scenario tables, fixed independent operation facts, and ten provider-neutral corpus siblings | exact owner/action/state/cursor outcomes passed; classifier and observer fields mutate independently | provider-neutral behavior claim only |
| Product separation | actual self-built-engine fixture | workflow paths excluded; product candidate/readiness/built-slice identities remain unchanged | fixture workspace is disposable |
| Reslice composition | dedicated Acceptance-preserving Reslice regression plus frozen boundary | all 13 shared paths retain Reslice semantics; all 30 non-overlap paths and prior workspace records unchanged | previous standalone Reslice digest remains historical |

### EVID-006 — Immutable 16-command root proof (SLICE-001; AC-001–AC-007)

Historical bundle `.root-proof-3e459eae4910719a-v5` is mode `0500`; all 27 files are mode `0400`. Manifest contract `devrites.root-proof-manifest.v8` has SHA-256 `0aa8cd0e906fa0cf522ca481bd1fd1b81e40355d8a42ce9e6461aa3da9b84cc8`. Separate attestation contract `devrites.root-proof-attestation.v7` has SHA-256 `55a9858aeeff4d610c5ef4529231098415262046af2824c2202512b71cd61296`. It proves the prior candidate only; the successor proof for candidate `76700e28bc35c871b44b85bd29e020e4e812b1682112e534b4150e16a2193546` and delivery `fba5ca20a13743dfe305ee728d30beb7e4546f8eaab5b830ee1050890038765b` is pending and this historical table does not support current Seal.

| # | Approved command | Exit | Output SHA-256 | Decisive signal |
| ---: | --- | ---: | --- | --- |
| 01 | toolchain versions | 0 | `db346996121b8ce7d954174b47c261e2acfe5934d53697643d6284c0608595e8` | `go1.26.5` |
| 02 | workspace schema | 0 | `ef231fba59fb9561bdc72f7900d816113b9514817a6beb44df2130b2dd390379` | workspace schema OK |
| 03 | dedicated contract | 0 | `33e10c65e20638eabce26789ee63ad97f5db42ef218a9e25c30dd685b8531a4a` | `workflow-artifact-identity: PASS` |
| 04 | Prove walkthrough | 0 | `c41d41931c5e067350f57e1763e37fa3a849e6cb56bcaa60ff46c698779c332a` | walkthrough PASS |
| 05 | behavioral corpus | 0 | `b1140dd04d6fe1be616d560db96254aa1c06ead331ff574cc2be1d62965f1163` | zero failed |
| 06 | phase routing | 0 | `1d6b4861ca06dc7cf0e62b4ffb1f07a3a8b62ec5e9daa2de4147b404fcd302e3` | routing PASS |
| 07 | Reslice regression | 0 | `1a555552bd66d272e0c3bc9fb8cdc2e69e5a65e043c1f94354e4b6229e2801b6` | Reslice PASS |
| 08 | host parity | 0 | `a4b11af060f5915ae797d404175aca4167010bda9f25691d14ec80264be232f2` | host parity PASS |
| 09 | instruction cap | 0 | `29bf39d4dcd69f33bd724fad3ea4aecf55657767100f210bdf1a9f89ea60decd` | 217 files; 854,982 bytes |
| 10 | repository validation | 0 | `a1e86ae8b0d011491c07ddac0f2c24d016ab389370139e4e505d1899c85c5f28` | `VALIDATION PASSED` |
| 11 | cross references | 0 | `969a65e01e927ffa16aa0b50aeb9b335e8c301969f1deca90f42ee0d17800eb8` | exit 0 |
| 12 | invocation integrity | 0 | `55ef7d2c37d71e378c57ec60bc06fa18e7e0931a4022f59d18fd4d8b04647139` | exit 0 |
| 13 | pack security | 0 | `3c92250c8ce0bf46b3358ada0d740ed60bcfcb9f7961314f161dd05e36209cca` | exit 0 |
| 14 | engine race | 0 | `92a755052b263b56a103733a14d9ed75a0506df51aaa8296c124b6d2d9903630` | exit 0 |
| 15 | full repository | 0 | `c68930983e51ba16ee083df507a577c1f4bd91816d0f1a61d91954dc21a8c7c6` | exit 0 |
| 16 | protected baseline | 0 | `7ed7a0576883a00b9bbc80e9ef07c9727fa0989163e5044f24423cb94fd27ccb` | accepted `.gitignore` hash observed |

The runner recorded candidate/readiness/delivery/protected identities before and after; all 16 gate records bind corrected PATH, `PYTHONDONTWRITEBYTECODE=1`, exact private `PYTHONPYCACHEPREFIX`, removed proof cache, output bytes/hash, and elapsed monotonic time. The separate attestor recomputed all 38 no-follow records, the selected journal, exact sidecar delta, complete delivery-history tree, and repository state before sealing.

### EVID-007 — Root-caller walkthrough and measured DX (SLICE-001; AC-006)

```text
WA-PROOF-001 PASS
tthw_ms=3251
WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R004-IDENTITY-STALE boundary_id=WA-B004-SOURCE-OPEN next_route=PLAN_VET_REPAIR
cursor=prove:/rite-prove demo
product_identity=unchanged
WORKFLOW_ARTIFACT_WALKTHROUGH PASS
```

TTHW was 3,251 ms against the 90,000 ms prediction. The line is fixed and bounded; no real product, Git, release, or remote action ran.

### EVID-008 — Protected and prior-candidate boundary (SLICE-001; AC-007)

| Protected input | SHA-256 | Result |
| --- | --- | --- |
| `.gitignore` | `58c1cc88c16b9bb14b345c156703163b47c9cb6232276b50684fabae8503e8fd` | unchanged; user-owned change excluded |
| `.devrites/ACTIVE` | `9ef52ccabae0acc5264a2f1c0f404c95e8191f0261cfa525880262447c9a75dc` | unchanged; active slug remains `thin-engine-native-codex` |
| Workspace Observation manifest | `2dca74484895de119cd935db6c3692782df9173eef199c88a7d5a65898332ec9` | unchanged |
| Acceptance-preserving Reslice boundary | dedicated gate plus selected-delivery records | 13 shared destinations preserve semantics; 30 non-overlap paths and workspace records unchanged |

### EVID-009 — Adversarial proof-authority closure (SLICE-001; AC-002–AC-007)

The first 16-command bundle remained rejected after an adversarial account found six authority gaps. v2 failed safely when its external launcher inherited the known looping PATH; v3 was interrupted by a ten-minute watcher and preserved without resumption; v4 passed all commands but was rejected because prefix admission did not prove an exact sidecar delta. v5 replaced that admission with a sealed 175-record stable-extra binding and 28 exact active lifecycle paths. Stable-record mutation and unknown-active-file mutants both fail, followed by a clean preflight PASS. v1–v4 remain historical and do not support the final verdict.

A root-authored validator that imports neither runner nor attestor independently observed attestation SHA-256 `55a9858aeeff4d610c5ef4529231098415262046af2824c2202512b71cd61296`, 27 sealed files, 16 gates, 38 no-follow candidate records, 89,504 repository entries, 16,479 delivery-history entries, and all 89,287 sidecar rows; verdict PASS.

### EVID-010 — Fresh-context whole-feature Prove (SLICE-001; AC-001–AC-007)

A historical fresh read-only proof runner validated the sealed v5 manifest/attestation, its selected delivery, then-current spec/test plan/traceability/evidence, and prior 38-file candidate. It returned PASS for REQ-001–REQ-009, AC-001–AC-007, EDGE-001–EDGE-007, PROH-001–PROH-012, all ten route scenarios, and all twelve producer/consumer key links; failures none; manual steps none.

That account predates the accepted DRIFT-035 correction and does not support the current Seal. A fresh proof runner must validate the successor bundle, current delivery, exact authority exception, current workspace records, and candidate `76700e28bc35c871b44b85bd29e020e4e812b1682112e534b4150e16a2193546`.

### EVID-011 — Successor 16-command root proof after Q-011/Q-012 (SLICE-001; AC-001–AC-007)

Proof date: 2026-08-22. Frozen engine candidate `0ea164c10bcf3b271137f3c3fe51af2dffa48d9c6ebd47e788b596b3ecf1d744` (38 files). Selected delivery `ed8d3ea7d37635cd69705ff6a38065024541496a9ba11da59c9e8aeb909ab65a` reached `CLEANED` (journal SHA-256 `6724b7d04366c12d389db7bad94cee07ffe90b4cac0d5c051c3deb5b73288d26`; driver `d3800e93a2321126029fe8899f9911964b90439626a4739d4227e36dd2cc35a9`). Generated delta empty under DEC-048. Bundle `.root-proof-0ea164c10bcf3b27-v10` is sealed. Manifest `devrites.root-proof-manifest.v9` SHA-256 `31f9d6bc50ef1461b03c4297775535919e3e7ba337d05c0bcc10da4888810a19` verdict PASS. Attestation `devrites.root-proof-attestation.v9` SHA-256 `575bc212bb91cd1b9d0d0c8e89affa9bb2dc1ab4fc5f32bff3d1dcda9a34697f` verdict PASS. Fresh-context proof-runner: all 16 commands pass; REQ-001–REQ-009, AC-001–AC-007, EDGE-001–EDGE-007, PROH-001–PROH-012, ten route scenarios, and twelve wiring links pass; failures none; manual steps none. Browser proof not applicable.

| # | Approved command | Exit | Output SHA-256 | Decisive signal |
| ---: | --- | ---: | --- | --- |
| 01 | toolchain versions | 0 | `db346996121b8ce7d954174b47c261e2acfe5934d53697643d6284c0608595e8` | `go1.26.5` |
| 02 | workspace schema | 0 | `ef231fba59fb9561bdc72f7900d816113b9514817a6beb44df2130b2dd390379` | workspace schema OK |
| 03 | dedicated contract | 0 | `33e10c65e20638eabce26789ee63ad97f5db42ef218a9e25c30dd685b8531a4a` | `workflow-artifact-identity: PASS` |
| 04 | Prove walkthrough | 0 | `4acb24b6603b48f00f3bc01791e58861d979a48376f26ffa768054b57ec0d1cc` | walkthrough PASS; `tthw_ms=3343` |
| 05 | behavioral corpus | 0 | `b1140dd04d6fe1be616d560db96254aa1c06ead331ff574cc2be1d62965f1163` | 14 files; 82 scenarios; 0 failed |
| 06 | phase routing | 0 | `1d6b4861ca06dc7cf0e62b4ffb1f07a3a8b62ec5e9daa2de4147b404fcd302e3` | routing PASS |
| 07 | Reslice regression | 0 | `1a555552bd66d272e0c3bc9fb8cdc2e69e5a65e043c1f94354e4b6229e2801b6` | Reslice PASS |
| 08 | host parity | 0 | `a4b11af060f5915ae797d404175aca4167010bda9f25691d14ec80264be232f2` | host parity PASS |
| 09 | instruction cap | 0 | `ae5c9349ba3eea885b443ee9330395050497c1d166ea8bc49595c9383be6beca` | 217 files; 854965 bytes |
| 10 | repository validation | 0 | `7a86ae531bb7d32c9e462b72b925d31bf1b2847268956596e544ad23a99f17c9` | `VALIDATION PASSED` |
| 11 | cross references | 0 | `969a65e01e927ffa16aa0b50aeb9b335e8c301969f1deca90f42ee0d17800eb8` | exit 0 |
| 12 | invocation integrity | 0 | `55ef7d2c37d71e378c57ec60bc06fa18e7e0931a4022f59d18fd4d8b04647139` | exit 0 |
| 13 | pack security | 0 | `3c92250c8ce0bf46b3358ada0d740ed60bcfcb9f7961314f161dd05e36209cca` | exit 0 |
| 14 | engine race | 0 | `c533c3dcd2329481a3a760e38510f4b7b9d0bfc57ebcdc71c4781a985ff9b453` | exit 0 |
| 15 | full repository | 0 | `84d32a2e8480ae716619d86bee711951ca6d9abf3c67373876321268ae02efa8` | TESTS PASSED |
| 16 | protected baseline | 0 | `89909e9df7e56abe5e7cdabf3896181c97d0bfe0092accc663235e0964ddaf40` | gitignore `58c1cc88…`; ACTIVE `fc0dd2b2…` |

Candidate before/after equal. Protected match. Selected delivery before/after equal. Exact sidecar delta records `.code-review-graph/graph.db` as the only changed outside file versus the delivery sidecar; stable extra count 0. DEC-042 launched install of prepared `ed8d3ea7…`. Journals `570f004b…` (SNAPSHOTTING) and `59ae71d2…` (FAILED) remain historical. Readiness inputs SHA-256 observed `2e0b227bb86789ecdc595fc152dd51703b882878fbff932c048bdb8b52186c88`; `eng-review.md` still has `5ab82dcf…`, so `check readiness` is `DRV-GATE-READINESS-STALE` until `$rite-vet` records the new line. This table does not support current Seal.

### EVID-012 — Seal-correction 16-command root proof after Q-013 (SLICE-001; AC-001–AC-007)

Proof date: 2026-08-22. Frozen engine candidate `bee44b1ada3b975839e90c79d9d04a10cf7b3b48b1ec3deed8f0f99931441fd9` (38 files). Selected delivery `8fd99161cc209bc631ae9e00eed88c28bdd05604f4c94ade934f91fe480d7201` reached `CLEANED` (journal SHA-256 `2402c7e0bb014522491b5b447140c7636db62132b95dc07b637901df8de1ea11`; driver `f3d26a4cc41727dc766fcc73f6e879b94026fc55bd84e7d15c6ce0304a1620d9`). Generated delta empty under DEC-048. FAILED `aff21c0c80e683395937b81e16bd1b4c1dbd5c950ad3e9e2edf781959147f5a1` remains at its digest-keyed directory (DEC-024). Bundle `.root-proof-bee44b1ada3b9758-v11` is sealed. Manifest `devrites.root-proof-manifest.v9` SHA-256 `4be0cfa37107d9722094e624b672584e4a010c9821f7c1a8db6a4022e507cb49` verdict PASS. Attestation `devrites.root-proof-attestation.v9` SHA-256 `9fc7b703d30dd7e9194f008443b814b1f0c3a1b0d14806100b58a15cbf092756` verdict PASS. Fresh-context proof-runner: all 16 commands pass; REQ-001–REQ-009, AC-001–AC-007, EDGE-001–EDGE-007, PROH-001–PROH-012, ten route scenarios, twelve wiring links, T-001–T-017, and Q-013/DEC-050 pass; failures none; manual steps none. Browser proof not applicable. `devrites-engine snapshot` is unavailable (unknown command).

| # | Approved command | Exit | Output SHA-256 | Decisive signal |
| ---: | --- | ---: | --- | --- |
| 01 | toolchain versions | 0 | `db346996121b8ce7d954174b47c261e2acfe5934d53697643d6284c0608595e8` | `go1.26.5` |
| 02 | workspace schema | 0 | `ef231fba59fb9561bdc72f7900d816113b9514817a6beb44df2130b2dd390379` | workspace schema OK |
| 03 | dedicated contract | 0 | `33e10c65e20638eabce26789ee63ad97f5db42ef218a9e25c30dd685b8531a4a` | `workflow-artifact-identity: PASS` |
| 04 | Prove walkthrough | 0 | `91c824ddb8c9b0e92d64ec2b25d35b3239fc191039a30f37c970e53db2ef8863` | walkthrough PASS; `tthw_ms=3716` |
| 05 | behavioral corpus | 0 | `b1140dd04d6fe1be616d560db96254aa1c06ead331ff574cc2be1d62965f1163` | 14 files; 82 scenarios; 0 failed |
| 06 | phase routing | 0 | `1d6b4861ca06dc7cf0e62b4ffb1f07a3a8b62ec5e9daa2de4147b404fcd302e3` | routing PASS |
| 07 | Reslice regression | 0 | `1a555552bd66d272e0c3bc9fb8cdc2e69e5a65e043c1f94354e4b6229e2801b6` | Reslice PASS |
| 08 | host parity | 0 | `a4b11af060f5915ae797d404175aca4167010bda9f25691d14ec80264be232f2` | host parity PASS |
| 09 | instruction cap | 0 | `ae5c9349ba3eea885b443ee9330395050497c1d166ea8bc49595c9383be6beca` | 217 files; 854965 bytes |
| 10 | repository validation | 0 | `7a86ae531bb7d32c9e462b72b925d31bf1b2847268956596e544ad23a99f17c9` | `VALIDATION PASSED` |
| 11 | cross references | 0 | `969a65e01e927ffa16aa0b50aeb9b335e8c301969f1deca90f42ee0d17800eb8` | exit 0 |
| 12 | invocation integrity | 0 | `55ef7d2c37d71e378c57ec60bc06fa18e7e0931a4022f59d18fd4d8b04647139` | exit 0 |
| 13 | pack security | 0 | `3c92250c8ce0bf46b3358ada0d740ed60bcfcb9f7961314f161dd05e36209cca` | exit 0 |
| 14 | engine race | 0 | `6bc5416f1bcee23dc9529f7aaad4940195141ffca4c904ef3ee9c941442313cd` | exit 0 |
| 15 | full repository | 0 | `9d46d7577ef90d4d71f07706b75a406cdb21cc19368813620510dd94e84b3b30` | TESTS PASSED |
| 16 | protected baseline | 0 | `89909e9df7e56abe5e7cdabf3896181c97d0bfe0092accc663235e0964ddaf40` | gitignore `58c1cc88…`; ACTIVE `fc0dd2b2…` |

Candidate before/after equal. Protected match. Selected delivery before/after equal. Exact sidecar delta records `.code-review-graph/graph.db` as the only changed outside file versus the delivery sidecar; stable extra count 0. DEC-042 launched install of prepared `8fd99161…`. Dedicated `check_workspace_evidence_mapping` (SLICE-001; AC-007) went red on unmapped evidence IDs 008 and 010 and green on the live workspace. Walkthrough:

```text
WA-PROOF-001 PASS
tthw_ms=3716
WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R004-IDENTITY-STALE boundary_id=WA-B004-SOURCE-OPEN next_route=PLAN_VET_REPAIR
cursor=prove:/rite-prove demo
product_identity=unchanged
WORKFLOW_ARTIFACT_WALKTHROUGH PASS
```

Readiness inputs SHA-256 observed during the sealed bundle `2f5fef7311a7167fd130e70e5bad6cdcd85f5b224ca8cc34fb622734787907fd`. Later workspace bookkeeping after this evidence write will stale that binding until `$rite-vet` records the current line.

### EVID-013 — Seal-correction locks for Important residuals (SLICE-001; AC-007)

Proof date: 2026-08-22. `$rite-build` after seal NO-GO on `bee44b1ada3b9758…`. Exact path `tests/workflow-artifact-identity-test.sh`. Wright return: no further edit; driver SHA-256 `35b1ce7ec7cfcf3e8749216dca236fe626f14f6d0472d9868921812c86552042` (sealed prior driver `f3d26a4cc41727dc766fcc73f6e879b94026fc55bd84e7d15c6ce0304a1620d9`).

| Proof | Observation | Result | Limitation |
| --- | --- | --- | --- |
| Production fixture argv | red-mutant strips `reject_delivery_fixture_argv`; `check_production_delivery_fixture_env` | AssertionError on `--delivery-test-fast-fixture` + `--delivery-prepare`; live check PASS | fixture env reject unchanged |
| Held-out / artifacts plants | live fixtures + dead-end mutant `fchdir(held_out)+DIR=artifacts` | live PASS with hoist + outsider untouched; dead-end writes `outsider/held` | stub generator script in plant fixtures; production also uses `OUT_ROOT=.` on fd cwd |
| Dedicated suite | `bash tests/workflow-artifact-identity-test.sh` | `workflow-artifact-identity: PASS` exit 0 (~367s) | walkthrough not re-run (unchanged surface) |
| Integrity / doubt | `devrites test-integrity`; DEC-053 reconfirm; DEC-054 fresh doubt | integrity OK; both claims hold | new 38-file candidate not yet delivered; Prove owns freeze |

Browser N/A. Drift none. Next prove must deliver driver `35b1ce7e…` and re-run the full test-plan matrix.

### EVID-014 — Seal-correction retry 16-command root proof after EVID-013/DEC-055 (SLICE-001; AC-001–AC-007)

Proof date: 2026-08-22. Frozen engine candidate `8c8cb87c7f56607557892fc67374292f1b7d732b44888ea5413e2a4ace57097b` (38 files). Selected delivery `8f1e14e38e6795da42e8159a455dd11f92a57ba5b5c9a31d6ee3013b92efb16d` reached `CLEANED` (journal SHA-256 `6f83466dcc0b9214fb6340fbba8cdddc5e5b44a86236006b640748f9edd48c88`; driver `828ce648f6783d691ef67a02a4372b67253a633562fa6e9d3a8230b5b24777c2`). Generated delta empty under DEC-048. FAILED `58c32710…` and `aff21c0c…` preserved (DEC-024). DEC-055 stripped private `stage`/`backups`/`proof-cache` from 35 digests before prepare (journals/sidecars retained). Bundle `.root-proof-8c8cb87c7f566075-v12` is sealed. Manifest `devrites.root-proof-manifest.v9` SHA-256 `4f0a99fb3ea2b0368e9220928ee4c7402a6aa27a5d1ea60aa8f0a35c272ef5dc` verdict PASS. Attestation `devrites.root-proof-attestation.v9` SHA-256 `475f7be327f2974aa24652a42edfdf4adbfa2232fac0088fbbd39ccf08c00c31` verdict PASS. Fresh-context proof-runner: all 16 commands pass; REQ-001–REQ-009, AC-001–AC-007, EDGE-001–EDGE-007, PROH-001–PROH-012, ten route scenarios, twelve wiring links, T-001–T-017, EVID-013 (SLICE-001; AC-007), and DEC-055 pass; failures none; manual steps none. Browser proof not applicable. `devrites-engine snapshot` remains unavailable (unknown command). DEC-042 launched install of prepared `8f1e14e3…` with corrected PATH (nvm Node 24.18.0 and `.local/bin` before Homebrew; Feedlo looping bash excluded); PATH SHA-256 `3d078431372a5f4c3d8a98643a7db5f4dab00744b26ccf57c08d1fa38ef639f2`.

| # | Approved command | Exit | Output SHA-256 | Decisive signal |
| ---: | --- | ---: | --- | --- |
| 01 | toolchain versions | 0 | `db346996121b8ce7d954174b47c261e2acfe5934d53697643d6284c0608595e8` | `go1.26.5` |
| 02 | workspace schema | 0 | `ef231fba59fb9561bdc72f7900d816113b9514817a6beb44df2130b2dd390379` | workspace schema OK |
| 03 | dedicated contract | 0 | `33e10c65e20638eabce26789ee63ad97f5db42ef218a9e25c30dd685b8531a4a` | `workflow-artifact-identity: PASS` |
| 04 | Prove walkthrough | 0 | `24fd84657e3e37018612f1dd121a2becc937c15db215df350f82ffcea4e47a57` | walkthrough PASS; `tthw_ms=6987` |
| 05 | behavioral corpus | 0 | `b1140dd04d6fe1be616d560db96254aa1c06ead331ff574cc2be1d62965f1163` | 14 files; 82 scenarios; 0 failed |
| 06 | phase routing | 0 | `1d6b4861ca06dc7cf0e62b4ffb1f07a3a8b62ec5e9daa2de4147b404fcd302e3` | routing PASS |
| 07 | Reslice regression | 0 | `1a555552bd66d272e0c3bc9fb8cdc2e69e5a65e043c1f94354e4b6229e2801b6` | Reslice PASS |
| 08 | host parity | 0 | `a4b11af060f5915ae797d404175aca4167010bda9f25691d14ec80264be232f2` | host parity PASS |
| 09 | instruction cap | 0 | `ae5c9349ba3eea885b443ee9330395050497c1d166ea8bc49595c9383be6beca` | 217 files; 854965 bytes |
| 10 | repository validation | 0 | `7a86ae531bb7d32c9e462b72b925d31bf1b2847268956596e544ad23a99f17c9` | `VALIDATION PASSED` |
| 11 | cross references | 0 | `969a65e01e927ffa16aa0b50aeb9b335e8c301969f1deca90f42ee0d17800eb8` | exit 0 |
| 12 | invocation integrity | 0 | `55ef7d2c37d71e378c57ec60bc06fa18e7e0931a4022f59d18fd4d8b04647139` | exit 0 |
| 13 | pack security | 0 | `3c92250c8ce0bf46b3358ada0d740ed60bcfcb9f7961314f161dd05e36209cca` | exit 0 |
| 14 | engine race | 0 | `417d974995bbc2fee5367c4c0081a7b77030969d097e1aa5eb65e2f645bb9889` | exit 0 |
| 15 | full repository | 0 | `d46ef61889b7714e7eb9d3670d1599df833c8292f28e4af2f127433f0d2d07ee` | TESTS PASSED |
| 16 | protected baseline | 0 | `89909e9df7e56abe5e7cdabf3896181c97d0bfe0092accc663235e0964ddaf40` | gitignore `58c1cc88…`; ACTIVE `fc0dd2b2…` |

Candidate before/after equal. Protected match. Selected delivery before/after equal. Sidecar replay: changed `[]`; stable extra count 0. Walkthrough:

```text
WA-PROOF-001 PASS
tthw_ms=6987
WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R004-IDENTITY-STALE boundary_id=WA-B004-SOURCE-OPEN next_route=PLAN_VET_REPAIR
cursor=prove:/rite-prove demo
product_identity=unchanged
WORKFLOW_ARTIFACT_WALKTHROUGH PASS
```

Readiness inputs SHA-256 observed during the sealed bundle `59d1819ff0247cd41477d76eeed7ea9a3b228b652c6e637b3bcdc0db37c4de5a`. Later workspace bookkeeping after this evidence write will stale that binding until `$rite-vet` records the current line.

### EVID-015 — Seal-correction nested OUT_ROOT basename plant (SLICE-001; AC-007)

Proof date: 2026-08-22. `$rite-build` after seal NO-GO on `8c8cb87c…` Important (nested `claude`/`codex` under artifacts cwd). `seal-important-accept` answered N (DEC-056). Exact path `tests/workflow-artifact-identity-test.sh`. Pre-dispatch driver `828ce648…` → post-edit `bda33a29f3ba7217749e36f0c7bb2063637e91a953ceaeeb58857dfb458b0781`.

| Proof | Observation | Result | Limitation |
| --- | --- | --- | --- |
| Path contract | `git status` / SHA only on contracted untracked dedicated test | only `tests/workflow-artifact-identity-test.sh` changed | unrelated dirty worktree preserved |
| Plant fixture TDD | wright red without mkdir/cp wraps; hoist was symlink | expected RED then GREEN | fifo one-shot adversary |
| Dedicated suite (root) | `bash tests/workflow-artifact-identity-test.sh` | `workflow-artifact-identity: PASS` exit 0 (~291s) | walkthrough / full matrix not-run (Prove) |
| Doubt | independent `devrites-doubt-reviewer` on four stood decisions | all HOLD; claim HOLDS for basename Important | nested-under-head residual noted as FYI |
| Integrity | engine `test-integrity` unknown on this binary; manual no new skip/xfail/.only; fixture wired in `default_tests` | OK for build gate | Prove re-runs full matrix |

Browser N/A. Drift none. Next: `$rite-prove` must deliver driver `bda33a29…` and re-run the full test-plan matrix on a new frozen candidate.

### EVID-016 — Seal-correction retry 16-command root proof after EVID-015 (SLICE-001; AC-001–AC-007)

Proof date: 2026-08-23. Frozen engine candidate `a6b102c354d756cb8d89bb395b54704ed9a9206ec1b652c7c24ff77426db453e` (38 files). Selected delivery `71411cf0ad4f02e80f57b64e2179eb06ee02e05de4b77e1bfcd4ab76598eafca` reached `CLEANED` (journal SHA-256 `a79264c8788d99b89b0ce4733e9279d4457262cbdb6f8617ea8b09434d6faa9b`; driver `bda33a29f3ba7217749e36f0c7bb2063637e91a953ceaeeb58857dfb458b0781`). Generated delta empty under DEC-048. FAILED `58c32710…` and `aff21c0c…` preserved (DEC-024). Wright prepare → `SNAPSHOTTING` (journal then `754b00b8…`); DEC-042 coordinator launched install with exact frozen driver/argv/cwd/PATH (launch path SHA-256 `9dd19f47d6c3074d2eb53e20ad6dbbb4edd327c483296c7ab970cd5a7df1b30f`). Bundle `.root-proof-a6b102c354d756cb-v13` is sealed. Manifest `devrites.root-proof-manifest.v9` SHA-256 `6ec809df3af2de36f87b4ab7c6b0d9df8357a6e8ef26588a4358fbb300252f37` verdict PASS. Attestation `devrites.root-proof-attestation.v9` SHA-256 `f7b494f75a3b252422bf71fbbbeaff3d3feb46d661dac662766a52ea5a5414a3` verdict PASS. Fresh-context proof-runner: all 16 commands pass; REQ-001–REQ-009, AC-001–AC-007, EDGE-001–EDGE-007, PROH-001–PROH-012, ten route scenarios, twelve wiring links, T-001–T-017, prior seal-correction EVID-015 (SLICE-001; AC-007), DEC-042/048/056 pass; failures none; manual steps none. Browser proof not applicable. `devrites-engine snapshot` remains unavailable (unknown command).

| # | Approved command | Exit | Output SHA-256 | Decisive signal |
| ---: | --- | ---: | --- | --- |
| 01 | toolchain versions | 0 | `db346996121b8ce7d954174b47c261e2acfe5934d53697643d6284c0608595e8` | `go1.26.5` |
| 02 | workspace schema | 0 | `ef231fba59fb9561bdc72f7900d816113b9514817a6beb44df2130b2dd390379` | workspace schema OK |
| 03 | dedicated contract | 0 | `33e10c65e20638eabce26789ee63ad97f5db42ef218a9e25c30dd685b8531a4a` | `workflow-artifact-identity: PASS` |
| 04 | Prove walkthrough | 0 | `dcda91619fe9a1cda5e4519c0476717d7bed468323f6566ad6f2b3e719b0cd96` | walkthrough PASS; `tthw_ms=5966` |
| 05 | behavioral corpus | 0 | `b1140dd04d6fe1be616d560db96254aa1c06ead331ff574cc2be1d62965f1163` | 14 files; 82 scenarios; 0 failed |
| 06 | phase routing | 0 | `1d6b4861ca06dc7cf0e62b4ffb1f07a3a8b62ec5e9daa2de4147b404fcd302e3` | routing PASS |
| 07 | Reslice regression | 0 | `1a555552bd66d272e0c3bc9fb8cdc2e69e5a65e043c1f94354e4b6229e2801b6` | Reslice PASS |
| 08 | host parity | 0 | `a4b11af060f5915ae797d404175aca4167010bda9f25691d14ec80264be232f2` | host parity PASS |
| 09 | instruction cap | 0 | `ae5c9349ba3eea885b443ee9330395050497c1d166ea8bc49595c9383be6beca` | 217 files; 854965 bytes |
| 10 | repository validation | 0 | `7a86ae531bb7d32c9e462b72b925d31bf1b2847268956596e544ad23a99f17c9` | `VALIDATION PASSED` |
| 11 | cross references | 0 | `969a65e01e927ffa16aa0b50aeb9b335e8c301969f1deca90f42ee0d17800eb8` | exit 0 |
| 12 | invocation integrity | 0 | `55ef7d2c37d71e378c57ec60bc06fa18e7e0931a4022f59d18fd4d8b04647139` | exit 0 |
| 13 | pack security | 0 | `3c92250c8ce0bf46b3358ada0d740ed60bcfcb9f7961314f161dd05e36209cca` | exit 0 |
| 14 | engine race | 0 | `082dc06abeafea659f54541eebc0abae7d0faa0799d8c0fab0fcc5b19663a40e` | exit 0 |
| 15 | full repository | 0 | `fda52a59c8fcbbd40bba2ecb75691ce08f8c48fb7c2c6b6036b5717801eb07d1` | TESTS PASSED |
| 16 | protected baseline | 0 | `89909e9df7e56abe5e7cdabf3896181c97d0bfe0092accc663235e0964ddaf40` | gitignore `58c1cc88…`; ACTIVE `fc0dd2b2…` |

Candidate before/after equal. Protected match. Selected delivery before/after equal. Sidecar replay: changed `[]`; stable extra count 0. Walkthrough:

```text
WA-PROOF-001 PASS
tthw_ms=5966
WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R004-IDENTITY-STALE boundary_id=WA-B004-SOURCE-OPEN next_route=PLAN_VET_REPAIR
cursor=prove:/rite-prove demo
product_identity=unchanged
WORKFLOW_ARTIFACT_WALKTHROUGH PASS
```

Readiness inputs SHA-256 observed during the sealed bundle `40372e0b1eaa9ec63eeaf252ec4b2ce4574ef3f1814355725652056a63059225`. Workspace bookkeeping after this evidence write will stale that binding until `$rite-vet` records the current line.

