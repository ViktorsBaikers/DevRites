# Developer experience: Workflow Artifact Identity

## Scope

Prediction mode at Vet. Developer-facing surfaces are canonical Markdown interface, copy-pasteable Vet admission/journal grammar, finite route/action/cursor contract, diagnostic line, and dedicated test entry point. No public CLI command, SDK, network interface, config installation, or end-user UI is added.

## Predicted scorecard

| Dimension | Prediction | Planned proof |
| --- | --- | --- |
| Discoverability | strong — sole module path is linked from all ten adapters; Vet owns one named admission block | structural adapter rows and cross-reference checks |
| Time-to-hello-world (TTHW) | at most 90 seconds from exact walkthrough command entry through first one-target `CLEANED`/cursor return; recovery cases occur after interval | Prove runs `bash tests/workflow-artifact-identity-test.sh --prove-walkthrough` and records monotonic `tthw_ms`; no seal claim before measurement |
| Getting-started friction | low — one repository-root flag; no credential/service/feature network; already-available module-selected Go 1.26.5 required; actual engine self-builds when env binary absent | preflight `go -C engine env GOVERSION GOTOOLCHAIN`, then clean disposable invocation |
| Error-message quality | actionable and safe — exact reason ID, boundary ID, and next route in one bounded line; no hostile/secret/path/raw-error content | every diagnostic row, stale-authority walkthrough, collision/leak mutants; quote verbatim output |
| Ergonomics and consistency | adequate — existing phase verbs/cursors preserved; modes/index/order/limits and retry semantics explicit; common path is classifier → root transaction → return | ten adapter rows, precedence overlaps, idempotent rerun/no-budget fixture |
| Docs accuracy | strong if implementation matches tables — admission, journal, encodings, operation, route, and diagnostic examples are canonical test inputs | parser consumes canonical tables; golden vector and row-deletion mutants |

## Predicted root-caller flow

1. Root reaches current admitted set and saved caller cursor.
2. Canonical classifier returns one route; adapter does not restate implementation.
3. Owner lock and operation table drive source, install, proof, evidence, cleanup, and return.
4. Success restores exact cursor and stops before consumptive action.
5. Failure emits one fixed line whose `next_route` names Plan/Vet, offline recovery, cleanup resume, owner wait, or existing gate.

Exact repository-root command:

```sh
bash tests/workflow-artifact-identity-test.sh --prove-walkthrough
```

TTHW starts at command entry before toolchain/self-build and ends at first successful `CLEANED` plus cursor return. Interrupted resume, stale authority, and idempotent rerun continue afterward outside interval. Exact output shape is owned by `test-plan.md`; stale line is:

`WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R004-IDENTITY-STALE boundary_id=WA-B004-SOURCE-OPEN next_route=PLAN_VET_REPAIR`

Prove must capture actual output; prediction is not evidence.

## Initial blocker resolution

| Blocker | Resolution | Verification |
| --- | --- | --- |
| no serializable admission/journal grammar | exact versioned blocks, field order, limits, target/history rows | parser + malformed/duplicate fixtures |
| ambiguous byte encoding | NUL, uint32 big-endian, base-8 mode, UTF-8 byte order, zero-based index, golden hashes | golden-vector test |
| route labels lacked exact caller behavior | ordered precedence and route owner/action/state/cursor/output table; ten adapter rows | overlap and adapter assertions |
| no finite diagnostic taxonomy | 22 reason rows, 16 boundary IDs, exact line, safe unrecognized-input collapse | every-row/collision/leak mutants |
| command required undeclared engine binary | observed module-selected Go 1.26.5 prerequisite; test self-builds actual engine when env absent; no mock/network fallback | toolchain preflight then env-unset invocation |
| no root-caller walkthrough | exact `--prove-walkthrough`; TTHW ends at first success; recovery cases follow outside interval; exact ordered output | Prove journey T-015 |

## Measurement contract

Prove records exact command/cwd, module toolchain, command-entry/first-success monotonic duration, ordered output, post-interval recovery traces, cursor, and product equality. Seal compares measured TTHW/output against 90000 ms prediction. Material gap or unactionable output becomes finding; missing measurement makes no DX claim.

## Measured scorecard

| Dimension | Observation | Verdict |
| --- | --- | --- |
| Discoverability | canonical module is linked by all ten adapters; cross-reference and invocation-integrity gates passed | pass |
| TTHW | exact walkthrough reached first `CLEANED` plus cursor return in 5,966 ms against the 90,000 ms prediction (EVID-016/v13; prior 6,987 ms v12 and 3,716 ms v11 are historical) | pass |
| Getting-started friction | the repository command ran with the declared Bash/Python/Node/module-Go toolchain and no credential, network, service, or mock fallback | pass |
| Error-message quality | observed `WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R004-IDENTITY-STALE boundary_id=WA-B004-SOURCE-OPEN next_route=PLAN_VET_REPAIR` with no path, hostile input, or raw error | pass |
| Ergonomics and consistency | observed `cursor=prove:/rite-prove demo`, `product_identity=unchanged`, and final walkthrough PASS in the declared order | pass |
| Docs accuracy | canonical tables, dedicated parser/mutants, route proof, host parity, and full repository validation passed on the final candidate | pass |

Measured readiness: PASS. Browser/UI and live-provider experience remain not applicable.
