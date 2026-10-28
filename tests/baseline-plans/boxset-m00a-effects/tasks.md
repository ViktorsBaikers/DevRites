# Tasks: Boxset M00a effects
Contracts: architecture.md sections. Commands at repository root. Every slice needs plan.md preflight P1-P5 (HZN-001) and passes shared gates `scripts/dev-run.sh -- go vet ./...`, `scripts/dev-run.sh -- golangci-lint run`, `scripts/dev-run.sh -- env CGO_ENABLED=0 go build ./...`, `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 ./...`, `scripts/dev-run.sh -- env CGO_ENABLED=1 go test -race <touched pkgs>` and traceability.md Preservation. Each criterion clause gets one named test (key names: traceability.md; full list: test-plan.md at Vet).
## SLICE-001 Accepted purge work survives as a durable job with one key owner
Goal: required purge intent commits with its domain transition and keeps one live attempt per key.
Satisfies: AC-018, AC-019
Acceptance criteria: job and retire commit together (failed second statement: neither; SIGKILL around commit: both or neither); no prior active or non-staging input: no job, existing `invalid` unchanged; 32 claimers/2 runtimes: one owner per key; lost wake swept in 30 s; no event-created work; caller cancel keeps the job; overdue and killed attempts keep the key and reconcile before any claim; exhaustion -> review (key held)/failed/expired; slots <= max(1, GOMAXPROCS/2), one core progresses; `Wait` frees the slot for its park call and retakes it before returning, also on cancel; `Runtime.Slot` draws from the same ceiling, freed slots serve Slot/Wait before new claims, refused after Stop; idle commits nothing; built binary resumes killed and SIGTERM-drained purges. One production composition: `cmd/boxset serve` calls `server.Start` and `Serving.Stop`; env, usage, mount refusal, crash monitor, signals, exit codes and stderr stay in the command, and the landed serve integration cohort passes unchanged. The move also preserves the landed run-state surface unchanged: `server.WriteRunState` keeps its four states and its `starting`/`recovering`/`ready`/`draining` transitions in the same order, `server.ReadRunState` stays the only reader, and `boxset healthcheck` still exits 0 only on `ready`, so its landed cohort passes unmodified. `TestServeComposition` proves, on a real data dir: lock ownership; `Start` returning a `ready` `Serving` only after the parent ready transition's `Runtime.Start` completes; a landed `recovering` start that keeps serving and dispatches no job (pre-ready no-dispatch control); `Effects.DB`/`Effects.Jobs` identity; and stop order with lock release last.
Complexity: 5 - claim, key, recovery and drain are one ownership protocol.
Mode: AFK
Gate: none
SLA: none
Checkpoint: none
Dependencies: none
depends_on: []
Consumes / Produces: consumes parent Write, chain, drain and the landed `cmd/boxset` `serve` composition (ASM-001/002/005); produces Data model `job`, jobs.Runtime sweep and store.EnqueueJob, Runtime lanes and EnqueueJob, jobs.Attempt.Wait (park call) and Reconcile, Runtime.Slot, jobs.Config.Publish, alias.purge load, `server.Effects`, `server.Start` (`ServeConfig`, `Serving`, `Serving.Stop`), created by moving the landed command-local body into `internal/server/serve.go` or by reusing a landed importable equivalent. Serial rationale: migration models, composition and runtime are shared with SLICE-002/008; those slices land serially. It also produces the untagged slow-control seam SLICE-010 needs, in this slice's own `internal/jobs/lane.go`: a `benchSlowHooks`-shaped interface declared there and asserted where a claimed attempt starts running, so a tagged build's delay lands between `Runtime.Accept` returning and the job publishing `running` (the measured window), while an ordinary build declares no method and the site does nothing.
Known-Gotchas: EnqueueJob pure SQL (txguard); read active id before the retire UPDATE; timeouts never release keys.
Prior-slice learnings: none; fold predecessor carry-forward constraints after its Seal
Files likely touched: internal/store/migrations/00008_jobs.sql, internal/store/queries/jobs.sql, internal/store/db/jobs.sql.go, internal/store/db/models.go, internal/store/jobs.go, internal/store/jobs_test.go, internal/library/search.go, internal/library/search_test.go, internal/library/crash_test.go, internal/jobs/jobs.go, internal/jobs/lane.go, internal/jobs/purge.go, internal/jobs/jobs_test.go, internal/server/effects.go, internal/server/serve.go, internal/server/serve_test.go, cmd/boxset/main.go, cmd/boxset/jobs_integration_test.go
Writer allowlist: internal/store/migrations/00008_jobs.sql, internal/store/queries/jobs.sql, internal/store/db/jobs.sql.go, internal/store/db/models.go, internal/store/jobs.go, internal/store/jobs_test.go, internal/library/search.go, internal/library/search_test.go, internal/library/crash_test.go, internal/jobs/jobs.go, internal/jobs/lane.go, internal/jobs/purge.go, internal/jobs/jobs_test.go, internal/server/effects.go, internal/server/serve.go, internal/server/serve_test.go, cmd/boxset/main.go, cmd/boxset/jobs_integration_test.go
Tests/proof: `scripts/dev-run.sh -- sqlc diff -f internal/store/sqlc.yaml` exit 0; `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 ./internal/store/... ./internal/library/... ./internal/jobs/... ./internal/server/...` ok incl. `TestServeComposition`; `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 -tags=integration ./cmd/boxset/...` ok incl. the landed serve integration cohort unchanged, `TestPurgeResumesAfterKill` and `TestDrainLeavesRecoverablePurge`.
Browser proof required: no
Frontend craft required: no
Design brief states: none
Visual acceptance: none
Fullstack (FE+BE): no
External dependencies: none
Existing to reuse / extend: `store.DB.Write`, `PurgeGeneration`, crash_test.go SIGKILL re-exec, the landed `cmd/boxset` `serve` body (moved, never copied) and its serve integration cohort
Rollback notes: revert unmigrated trees; a migrated file never returns to an older binary (newer-schema refusal): the current or a forward-fixed binary restores the newest pre-migration snapshot (plan.md Rollback, T21)
Evidence required: test counts, sqlc diff, kill/drain transcripts with job/alias counts; P5 `parent-source-capture` record (`PARENT_COMMIT`, `PARENT_SRC_SHA256`, `PARENT-SRC-BUILD-PASS`, parent binary SHA-256, landed web build command and parent interface inventory incl. reconciled-table set) made before the first effects source edit
Edge/Prohibition coverage: EDGE-001, PROH-001, PROH-004
Done condition: every Acceptance criteria clause holds and Tests/proof plus shared gates pass.
## SLICE-002 Maintenance runs once per persisted occurrence
Goal: scheduled work runs once per persisted occurrence.
Satisfies: AC-020
Acceptance criteria: oracle literals match; rollback runs nothing early, jump once; downtime coalesces; racing sweeps insert one occurrence; open job coalesces, full queue defers; no overlap; `jobs.prune` seeded once, reseeded only on zone change, deletes only terminal rows >7 d in <=500-row Writes; no scheduler module; built binary keeps `next_due_at` across restart and loads `TZ=Europe/Berlin` with `ZONEINFO=/nonexistent`.
Complexity: 3 - one rule set, one occurrence identity.
Mode: AFK
Gate: none
SLA: none
Checkpoint: none
Dependencies: SLICE-001 (via jobs.Runtime sweep and store.EnqueueJob)
depends_on: [SLICE-001]
Consumes / Produces: consumes Runtime, EnqueueJob; produces Schedules contract, RunNow kind jobs.prune. Serial rationale: migrations/models and server.Effects compose the same live runtime as SLICE-001/008.
Known-Gotchas: `time.Date` in gap/repeat picks an offset silently; oracle literals never from `Next`.
Prior-slice learnings: fold Build carry-forward constraints
Files likely touched: internal/store/migrations/00009_schedules.sql, internal/store/queries/schedules.sql, internal/store/db/schedules.sql.go, internal/store/db/models.go, internal/jobs/schedule.go, internal/jobs/schedule_test.go, internal/jobs/prune.go, internal/jobs/testdata/occurrences.json, internal/server/effects.go, cmd/boxset/main.go, cmd/boxset/jobs_integration_test.go, cmd/boxset/rollback_integration_test.go, cmd/boxset/testdata/parentseed/main.go
Writer allowlist: internal/store/migrations/00009_schedules.sql, internal/store/queries/schedules.sql, internal/store/db/schedules.sql.go, internal/store/db/models.go, internal/jobs/schedule.go, internal/jobs/schedule_test.go, internal/jobs/prune.go, internal/jobs/testdata/occurrences.json, internal/server/effects.go, cmd/boxset/main.go, cmd/boxset/jobs_integration_test.go, cmd/boxset/rollback_integration_test.go, cmd/boxset/testdata/parentseed/main.go
Tests/proof: `scripts/dev-run.sh -- sqlc diff -f internal/store/sqlc.yaml` exit 0; `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 ./internal/jobs/... ./internal/store/...` ok; `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 -tags=integration ./cmd/boxset/... -run 'TestScheduleSurvivesRestart|TestTZDataEmbedded'` ok; T21 rollback rehearsal (gate REG-ROLLBACK; byte-identical to its gates.md CHECK and test-plan.md Build-entry preflight command; the only command selecting cmd/boxset/rollback_integration_test.go), with `PARENT_COMMIT` and `PARENT_SRC_SHA256` taken verbatim from the P5 `parent-source-capture` record; its `sh -euc` script is the capture script byte-for-byte followed by the `exec ... go test` step: `scripts/dev-run.sh --timeout 600 -- sh -euc 'want=$1; shift; [ "$#" -ge 1 ] && [ "$#" -le 12 ] || { echo PARENT-SRC-CHUNKS-REFUSED; exit 1; }; W=$(mktemp -d); printf %s "$@" | base64 -d | gzip -dc | head -c 33554433 >"$W/parent.tar"; [ "$(sha256sum <"$W/parent.tar" | cut -c1-64)" = "$want" ] || { echo PARENT-SRC-DIGEST-REFUSED; exit 1; }; mkdir "$W/p"; tar -x -C "$W/p" -f "$W/parent.tar"; { cmp -s "$W/p/web/package.json" web/package.json && cmp -s "$W/p/web/bun.lock" web/bun.lock; } || { echo PARENT-WEB-LOCK-DRIFT; exit 1; }; cp -R /src/web/node_modules "$W/p/web/node_modules"; (cd "$W/p" && bun --cwd web run build && env CGO_ENABLED=0 go build -trimpath -o "$W/boxset-parent" ./cmd/boxset); go version -m "$W/boxset-parent"; sha256sum "$W/boxset-parent"; echo PARENT-SRC-BUILD-PASS; exec env CGO_ENABLED=0 go test -count=1 -v -timeout 570s -tags=integration,effectsrollback ./cmd/boxset -run "^TestEffectsRollbackFromParentSnapshot\$" -args -parent-bin "$W/boxset-parent" -parent-tree "$W/p"' sh "$PARENT_SRC_SHA256" $(git archive --format=tar "$PARENT_COMMIT" -- go.mod go.sum cmd internal web ':(exclude,glob)**/*_test.go' ':(exclude,glob)**/*.test.ts' ':(exclude,glob)**/testdata/**' ':(exclude)web/e2e' ':(exclude)internal/server/dist' | gzip -n -9 | base64 -w0 | fold -w 98304)` prints `PARENT-SRC-BUILD-PASS` then `--- PASS: TestEffectsRollbackFromParentSnapshot`.
Browser proof required: no
Frontend craft required: no
Design brief states: none
Visual acceptance: none
Fullstack (FE+BE): no
External dependencies: none (stdlib time/tzdata)
Existing to reuse / extend: SLICE-001 sweep and dedupe
Rollback notes: revert unmigrated trees; migrated files recover by current-binary snapshot restore (plan.md Rollback, T21), never by an older binary
Evidence required: oracle results, restart/TZ transcript; T21 rehearsal transcript: argv, runner image id and inputs SHA-256, `PARENT_COMMIT`, `PARENT_SRC_SHA256`, parent/current/seed binary SHA-256 and build settings, fixture scale (titles, aliases per generation), O/P/Q chain, hash, object and domain-set digests and alias counts, reconciled-set results (per-user username/`password_phc`/role/disabled vs pre-restore, per-user revision pre-restore/S/after, `users` collection revision pre-restore/S/after, setup consumed side, install id, epoch values, revoked-key/pre-restore-session/fresh-control auth), `lock_held` exit, parent refusal exit codes and reasons with unchanged DB and snapshot SHA-256, recovery-copy path with its P-equality digests, restored-file SHA-256 (recorded only)
Edge/Prohibition coverage: EDGE-002
Done condition: every Acceptance criteria clause holds and Tests/proof plus shared gates pass.
## SLICE-003 Interactive work is admitted or refused before any effect and every job is visible
Goal: interactive work is admitted or refused with 429 before any row; jobs are visible under existing rights.
Satisfies: AC-019, AC-024
Acceptance criteria: 5th per caller and 17th global -> `interactive_cap`, 1,025th optional -> `queue_full`, row count unchanged, across restart; coalescing and `ErrKindNotRunnable`; interactive claim with background full; controlled HTTP consumer (test-registered operation behind the parent guard, real principal/runtime/file) returns 429 with `code` and `Retry-After` 2/30, and a client disconnecting after 202 leaves a completing job; jobs GET pages (501 -> 422), binds cursor (other epoch -> 409), shows states/reasons, matches openapi.json and parent route suites; no other route, right or policy row.
Complexity: 3 - one admission decision and its read model.
Mode: AFK
Gate: none
SLA: none
Checkpoint: none
Dependencies: SLICE-001 (via Runtime lanes and EnqueueJob), SLICE-002 (via RunNow kind jobs.prune)
depends_on: [SLICE-001, SLICE-002]
Consumes / Produces: consumes parent error model, cursor, route table, principal (ASM-003/006); produces Admission service, Runtime.Accept, Job schema in openapi.json, jobs GET.
Known-Gotchas: caller only from the principal; consumer only in _test.go; openapi.json via `boxset openapi`.
Prior-slice learnings: fold Build carry-forward constraints
Files likely touched: internal/jobs/accept.go, internal/jobs/accept_test.go, internal/store/queries/jobs.sql, internal/store/db/jobs.sql.go, internal/api/jobs.go, internal/api/jobs_test.go, internal/api/errors.go, internal/api/admission_test.go, internal/api/openapi.json, internal/server/guard.go, web/src/lib/api.gen.ts, internal/api/api.go, internal/api/openapi_test.go, internal/server/server.go
Writer allowlist: internal/jobs/accept.go, internal/jobs/accept_test.go, internal/store/queries/jobs.sql, internal/store/db/jobs.sql.go, internal/api/jobs.go, internal/api/jobs_test.go, internal/api/errors.go, internal/api/admission_test.go, internal/api/openapi.json, internal/server/guard.go, web/src/lib/api.gen.ts, internal/api/api.go, internal/api/openapi_test.go, internal/server/server.go
Tests/proof: `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 ./internal/jobs/... ./internal/api/... ./internal/server/...` ok; `scripts/dev-run.sh -- bun --cwd web run check:api` ok.
Browser proof required: no
Frontend craft required: no
Design brief states: none
Visual acceptance: none
Fullstack (FE+BE): no
External dependencies: none
Existing to reuse / extend: parent problem model, cursor, route table
Rollback notes: revert; no migration or right
Evidence required: boundary transcripts with row counts and 429 headers, openapi diff
Edge/Prohibition coverage: EDGE-003, PROH-004
Done condition: every Acceptance criteria clause holds and Tests/proof plus shared gates pass.
## SLICE-004 Decoders reject oversize and hostile input without truncation
Goal: every ADR 0004 §6 input class is capped incrementally with structured reasons.
Satisfies: AC-023
Acceptance criteria: per Ingestion limits, cap passes and max+1 fails for every class incl. gzip bomb, lying Content-Length and image header; malformed inputs fail with reasons; callback error never success; reject heap growth < cap + 1 MiB; built binary validates 1 MiB native JSON, 413 at +1 byte.
Complexity: 3 - one limit discipline, five classes.
Mode: AFK
Gate: none
SLA: none
Checkpoint: none
Dependencies: none
depends_on: []
Consumes / Produces: consumes stdlib decoders; produces Ingestion API including ingest.Body.
Known-Gotchas: json replaces invalid UTF-8 silently; Decoder has no depth limit; heap tests `!race`.
Prior-slice learnings: fold Build carry-forward constraints
Files likely touched: internal/ingest/ingest.go, internal/ingest/body.go, internal/ingest/json.go, internal/ingest/feed.go, internal/ingest/dataset.go, internal/ingest/image.go, internal/ingest/ingest_test.go, cmd/boxset/ingest_integration_test.go
Writer allowlist: internal/ingest/ingest.go, internal/ingest/body.go, internal/ingest/json.go, internal/ingest/feed.go, internal/ingest/dataset.go, internal/ingest/image.go, internal/ingest/ingest_test.go, cmd/boxset/ingest_integration_test.go
Tests/proof: `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 ./internal/ingest/...` ok; `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 -tags=integration ./cmd/boxset/... -run TestNativeJSONRequestCap` ok.
Browser proof required: no
Frontend craft required: no
Design brief states: none
Visual acceptance: none
Fullstack (FE+BE): no
External dependencies: none
Existing to reuse / extend: stdlib decoders, parent body cap
Rollback notes: revert; no state
Evidence required: per-cap table, heap numbers
Edge/Prohibition coverage: EDGE-003, PROH-004
Done condition: every Acceptance criteria clause holds and Tests/proof plus shared gates pass.
## SLICE-005 Outbound requests reach only validated destinations with scoped credentials
Goal: every wire request reaches only validated destinations with origin-scoped credentials.
Satisfies: AC-031
Acceptance criteria: denied addresses incl. IPv4-mapped metadata fail before connect; DNS flip denied; only the exact approved LAN origin dials private; IP-dialed TLS verifies hostname; redirects revalidated, downgrade denied, cross-origin hop strips credential/cookie/Authorization, credential query refused, mutation 3xx refused after one request; env proxies get 0 requests; forbidden digest fails before resolve; sentinels redacted in errors.
Complexity: 4 - one trust decision shared by dial, redirect, credentials.
Mode: AFK
Gate: none
SLA: none
Checkpoint: none
Dependencies: SLICE-004 (via ingest.Body)
depends_on: [SLICE-004]
Consumes / Produces: consumes parent redactor (ASM-004), `ingest.Body`; produces outbound.Client and Op, Outbound types, `Do`.
Known-Gotchas: Proxy nil explicitly; stdlib redirect stripping is hostname-only; no network, use resolver seam.
Prior-slice learnings: fold Build carry-forward constraints
Files likely touched: internal/outbound/client.go, internal/outbound/dial.go, internal/outbound/redirect.go, internal/outbound/client_test.go, internal/outbound/dial_test.go, internal/outbound/redirect_test.go
Writer allowlist: internal/outbound/client.go, internal/outbound/dial.go, internal/outbound/redirect.go, internal/outbound/client_test.go, internal/outbound/dial_test.go, internal/outbound/redirect_test.go
Tests/proof: `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 ./internal/outbound/...` ok.
Browser proof required: no
Frontend craft required: no
Design brief states: none
Visual acceptance: none
Fullstack (FE+BE): no
External dependencies: none
Existing to reuse / extend: parent redactor, httptest
Rollback notes: revert; no state
Evidence required: denial matrix with connect counts
Edge/Prohibition coverage: EDGE-004, PROH-002
Done condition: every Acceptance criteria clause holds and Tests/proof plus shared gates pass.
## SLICE-006 Retries spend one budget and unknown mutations reconcile before any resend
Goal: safe reads spend one bounded budget; lost-ack mutations reconcile before any resend.
Satisfies: AC-032, AC-033, AC-024, AC-019
Acceptance criteria: 503 x5 -> 3 requests then `budget_exhausted` <=15 s; over-budget Retry-After -> 1 request; waits and hops charge; one request per attempt; equal reads merge, any scope change does not; `TestSingleFlightOriginScope`: concurrent safe reads with the same Target.ID, ConfigRevision, CredentialIdentity, Method, path, query, Body, Language and Representation, each Op's Target.Origin the approved origin of its own URL, differing only in scheme, hostname or port on unprivileged loopback fixtures (injected resolver, test TLS) -> counting transport sees one physical request per origin and each caller gets its own fixture's distinct result; canonical key invariant (no listener): omitted vs explicit 80 (http) / 443 (https) and host case variants give equal keys, while any other scheme, hostname or port gives a different key; `TestSingleFlightSafePOSTBodyScope`: identical safe POST bodies -> 1 request, body-only change -> 2 requests and each caller its own correct result, 1 MiB+1 safe Body -> `body_too_large`, 0 requests; cancelled caller leaves the flight result to others; caller mutating its URL/Target after a cancelled Do does not change the in-flight request; 32 distinct-body calls held at the server then cancelled -> 33rd call `queue_full` retry_after_s=2, 0 requests, and after those flights settle the next call is sent; 2,048 approved synthetic host aliases on the loopback resolver (host + ceiling key each) fill 4,096 non-idle budget keys -> next new host `budget_exhausted` limit=budget_keys max=4096, 0 requests, existing allowances unchanged; host keys follow the actual request/hop hostname; every physical body closed; Close stops admission (client_closed, 0 requests), cancels and awaits owned flights without waiting on callers; Effects.Outbound uses Runtime.Slot and closes before Runtime.Stop; no key or log holds the Body or credential sentinel; writes never merge; indexer Retry-After spares the sibling, ScopeHost delays both, ceiling bounds the sum; exact UA; limiter wait frees the only slot; one background slot: A's flight in a rate delay, B joins -> 1 request, both succeed, no deadlock; A cancelled first -> flight completes on its own timer and B gets the result; `New` with nil Slot rejected; mutation classes per contract; Locate decides present/absent only after the complete bounded walk (duplicate on a later page -> ambiguous; cap, repeated cursor or error -> unknown); loopback client-shaped service: lost ack -> succeeded with 1 add, failed lookup -> review with 1 add across restart, kill after write never resends before lookup; manifests equal EVID-003, admission passes.
Complexity: 5 - one physical-attempt budget across limiter, retry, hops, merge and runtime; split layers pass while it fails.
Mode: AFK
Gate: none
SLA: none
Checkpoint: none
Dependencies: SLICE-005 (via outbound.Client and Op), SLICE-001 (via jobs.Attempt.Wait and Reconcile)
depends_on: [SLICE-005, SLICE-001]
Consumes / Produces: consumes Client, lanes, Attempt.Wait, Runtime.Slot, `server.Start`, `server.NewEffects` (SLICE-001/002); produces outbound.Client, Op and Locate with Page/Presence, budgets, retry, merge, mutation classes, slot-releasing Wait, manifests, `(*outbound.Client).Close`, `server.ServeConfig.Outbound`, `server.Effects.Outbound` (the one composition client: outbound client permits/limiter cache/shared Slot, built by `NewEffects` from `ServeConfig.Outbound` with `Slot` = `Jobs.Slot`, closed in drain before `Runtime.Stop`). Serial rationale: `internal/server/effects.go`, `internal/server/serve.go` and `cmd/boxset/main.go` extend SLICE-001/002 and precede SLICE-008; `lane.go` extends SLICE-001; `outbound/client.go` extends SLICE-005.
Known-Gotchas: never WaitN; one monotonic clock; remote kind/service test-only.
Prior-slice learnings: fold Build carry-forward constraints
Files likely touched: go.mod, go.sum, docs/spikes/s5-version-admission.md, internal/outbound/budget.go, internal/outbound/retry.go, internal/outbound/mutation.go, internal/outbound/locate.go, internal/outbound/client.go, internal/outbound/budget_test.go, internal/outbound/retry_test.go, internal/jobs/lane.go, internal/jobs/remote_integration_test.go, internal/outbound/close_test.go, internal/server/effects.go, internal/server/effects_test.go, internal/server/serve.go, internal/server/serve_test.go, cmd/boxset/main.go
Writer allowlist: go.mod, go.sum, docs/spikes/s5-version-admission.md, internal/outbound/budget.go, internal/outbound/retry.go, internal/outbound/mutation.go, internal/outbound/locate.go, internal/outbound/client.go, internal/outbound/budget_test.go, internal/outbound/retry_test.go, internal/jobs/lane.go, internal/jobs/remote_integration_test.go, internal/outbound/close_test.go, internal/server/effects.go, internal/server/effects_test.go, internal/server/serve.go, internal/server/serve_test.go, cmd/boxset/main.go
Tests/proof: `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 ./internal/outbound/... ./internal/jobs/... ./internal/server/...` ok incl. `TestClientCloseDrain` and `TestEffectsOutboundComposition` (both T08); `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 -tags=integration ./internal/jobs/... -run 'TestLostAck|TestUncertainAbsence|TestKilledMutation'` ok; `scripts/dev-run.sh -- go mod verify` ok; `scripts/dev-run.sh --write docs/spikes/issue-drafts -- scripts/s5-toolchain.sh --admission` prints S5-ADMISSION-PASS.
Browser proof required: no
Frontend craft required: no
Design brief states: none
Visual acceptance: none
Fullstack (FE+BE): no
External dependencies: golang.org/x/time v0.16.0 (EVID-003); golang.org/x/sync v0.23.0 direct
Existing to reuse / extend: SLICE-001 reconciling, crash_test.go re-exec, SLICE-001 `server.Start`, `server.NewEffects` and drain
Rollback notes: revert incl. go.mod/go.sum/ADM row; S5-ADMISSION-PASS on reverted tree
Evidence required: wire counts, add counts, manifest diff, admission line, composition slot-occupancy counts, Close drain counts (live flights, free permits, open bodies, post-Close requests) and drain order
Edge/Prohibition coverage: EDGE-001, EDGE-004, PROH-001, PROH-002
Done condition: every Acceptance criteria clause holds and Tests/proof plus shared gates pass.
## SLICE-007 Recordings are scrubbed before saving and tests replay offline by default
Goal: only scrubbed synthetic recordings are saved; tests replay offline by default.
Satisfies: AC-040, AC-041
Acceptance criteria: two host recordings of `outbound-contract` byte-identical, sentinel-free; `scrub_incomplete` publishes nothing, committed file kept; live targets GET only (HEAD and every other method refused with target request count 0); unclassified and provenance-less refused; deterministic replay through the real client over valid, invalid shape, partial, 401, 429-then-success, duplicate identity, repeated cursor, timeout-after-write; unrecorded request and network escape fail; runner root accepts only `[a-z0-9-]{1,32}`, publishes on exit 0 only.
Complexity: 4 - one format and refusal rule from capture to replay.
Mode: AFK
Gate: none
SLA: none
Checkpoint: none
Dependencies: SLICE-006 (via outbound.Client, Op and Locate)
depends_on: [SLICE-006]
Consumes / Produces: consumes `Config.Transport`, runner export; produces Fixture harness, set, recipe, runner root.
Known-Gotchas: child has no network: throwaway services on loopback; Makefile calls only scripts/dev-run.sh.
Prior-slice learnings: fold Build carry-forward constraints
Files likely touched: internal/fixture/fixture.go, internal/fixture/record.go, internal/fixture/replay.go, internal/fixture/scrub.go, internal/fixture/fixture_test.go, internal/fixture/record/main.go, internal/fixture/synthetic/synthetic.go, internal/outbound/contract_test.go, internal/outbound/main_test.go, internal/outbound/testdata/fixtures/outbound-contract/exchanges.json, Makefile, scripts/dev-run.sh, scripts/dev-run-test.sh, README.md, docs/spikes/s5-version-admission.md
Writer allowlist: internal/fixture/fixture.go, internal/fixture/record.go, internal/fixture/replay.go, internal/fixture/scrub.go, internal/fixture/fixture_test.go, internal/fixture/record/main.go, internal/fixture/synthetic/synthetic.go, internal/outbound/contract_test.go, internal/outbound/main_test.go, internal/outbound/testdata/fixtures/outbound-contract/exchanges.json, Makefile, scripts/dev-run.sh, scripts/dev-run-test.sh, README.md, docs/spikes/s5-version-admission.md
Tests/proof: `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 ./internal/fixture/... ./internal/outbound/...` ok; host `make fixtures-record SET=outbound-contract` exit 0 twice, then `git diff --exit-code -- internal/outbound/testdata/fixtures` exit 0; host `sh scripts/dev-run-test.sh` and `sh scripts/dev-run-smoke.sh` PASS; README consumer smoke (no text-pinning test). From a clean checkout, the README record command, run as copied, exits 0 twice and its replay command passes. `make fixtures-record SET=Outbound-Contract` is refused before Docker (`write_path_invalid`) and leaves the tree hash unchanged. Each recorder reason asserted by `TestFixtureRecordSafety` (`scrub_incomplete`, `target_unclassified`, `provenance_required`) has its README recovery entry, and its observed output holds no sentinel value.
Browser proof required: no
Frontend craft required: no
Design brief states: none
Visual acceptance: none
Fullstack (FE+BE): no
External dependencies: none
Existing to reuse / extend: runner export, `deploy/dev/redact`
Rollback notes: revert; a refused recording publishes nothing
Evidence required: two recording transcripts, git diff, refusal counts; README quickstart transcript (copied commands, exit codes, published path, invalid-SET refusal line, tree hash); TTHW measured for the documented flow from a clean checkout on the already-provisioned (warm) `work` image, compared with the DevEx source estimate (~10 min), repeated at Prove; cold-image TTHW is NOT-MEASURED unless its existing bootstrap owner already has evidence, and its network/bootstrap prerequisite stays a linked README caveat
Edge/Prohibition coverage: EDGE-006, PROH-002
Done condition: every Acceptance criteria clause holds and Tests/proof plus shared gates pass.
## SLICE-008 Event stream delivers ordered state only under current authority
Goal: `/api/v1/events` delivers ordered state under current authority with bounded queues.
Satisfies: AC-034, AC-035, AC-036, AC-037
Acceptance criteria: watermark race yields complete ordered state; in-ring replay gapless; other epoch/boot (restart, restore), evicted and malformed IDs resync with reasons; 50/s x 10 s to 10 healthy -> 500 each in order while 2 blocked overflow-resync or close, Publish never blocks; downgrade, disable, logout, idle expiry close before next frame, another admin still receives; replay rights-filtered; key stream closes on downgrade, revoked key 401 after restart; `auth.Revalidate` is the only revalidator: provider `TestRevalidateCurrentAuthority` over persisted principals and guard-minted refs and consumer `TestEventCurrentAuthority` bind the same function (no stub or permissive callback); a direct local-bypass peer keeps its stream while a trusted-proxy or forwarded-header bypass attempt gets none; the built binary's composition binds it (a real PATCH downgrade closes that viewer's stream while another admin's continues); caps 429; draining -> system, close, 503; uncompressed; payloads validate against openapi.json; a maximum valid `EventJob` (store-bound reason_params, escaping-heavy fields) is delivered whole when its real encoding is <=8,192 B, else announced resync (encoded length recorded); forced oversize and injected encode failure each consume the seq, raise the replay lower bound, send resync{overflow} with the recovery watermark, then close, with none of the payload's bytes in ring, queue or wire; old cursors -> gap resync; queue 256 frames/256 KiB, ring 1 MiB/5 min, caps 4/64, per-frame revalidation and seq order unchanged; built binary over HTTP. Authenticated `GET /api/v1/session` (existing route and right) also returns `installId`: 12 lowercase hex from the ready-time `installation.install_id` read that the hub's `hello`, the cookie name and the cursor use. The `Session` component is regenerated through `boxset openapi` into openapi.json and api.gen.ts. `TestSessionConfirmsHello` runs two built-binary installations that both have user id 1 and restoreEpoch 1: each session triple equals its own hello and installation row, and the two differ only in `installId`.
Complexity: 4 - one ordered authorized delivery path.
Mode: AFK
Gate: none
SLA: none
Checkpoint: none
Dependencies: SLICE-003 (via Job schema in openapi.json), SLICE-001 (via jobs.Config.Publish)
depends_on: [SLICE-003, SLICE-001]
Consumes / Produces: consumes parent principal and its provenance decisions (`auth.Sessions`, `auth.Keys`, `auth.Policy`, local bypass/trusted-proxy rule), lifecycle, ring, users PATCH, session DELETE, the existing `GET /api/v1/session` operation and the ready-time installation identity read (ASM-002/003/005/006), `server.Start`, `server.NewEffects`; produces /api/v1/events and Event* components, event frames, Events contract, `auth.Revalidate` and opaque `auth.CredentialRef`, `Session.installId` (additive field on the existing response; openapi.json and api.gen.ts regenerated), `server.Effects.Hub` (`Hub.Publish` wired into `jobs.Config`; streams closed first in drain). Serial rationale: composition (after SLICE-001/002/006), runtime Publish, route table and OpenAPI extend SLICE-001/003.
Known-Gotchas: subscribe and read sequence under one lock; heartbeat re-read unconditional.
Prior-slice learnings: fold Build carry-forward constraints
Files likely touched: internal/events/hub.go, internal/events/stream.go, internal/events/events_test.go, internal/auth/revalidate.go, internal/auth/revalidate_test.go, internal/api/events.go, internal/api/users.go, internal/api/session.go, internal/api/openapi.json, internal/server/lifecycle.go, internal/server/guard.go, internal/server/effects.go, internal/server/serve.go, internal/server/serve_test.go, internal/jobs/jobs.go, cmd/boxset/events_integration_test.go, web/src/lib/api.gen.ts, internal/api/api.go, internal/api/openapi_test.go, internal/server/server.go
Writer allowlist: internal/events/hub.go, internal/events/stream.go, internal/events/events_test.go, internal/auth/revalidate.go, internal/auth/revalidate_test.go, internal/api/events.go, internal/api/users.go, internal/api/session.go, internal/api/openapi.json, internal/server/lifecycle.go, internal/server/guard.go, internal/server/effects.go, internal/server/serve.go, internal/server/serve_test.go, internal/jobs/jobs.go, cmd/boxset/events_integration_test.go, web/src/lib/api.gen.ts, internal/api/api.go, internal/api/openapi_test.go, internal/server/server.go
Tests/proof: `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 ./internal/auth/... ./internal/events/... ./internal/api/... ./internal/server/... ./internal/jobs/...` ok incl. `TestRevalidateCurrentAuthority` and `TestEventCurrentAuthority`; `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 -tags=integration ./cmd/boxset/... -run 'TestEventsOverHTTP|TestSessionConfirmsHello'` ok; `scripts/dev-run.sh -- bun --cwd web run check:api` ok.
Browser proof required: no
Frontend craft required: no
Design brief states: none
Visual acceptance: none
Fullstack (FE+BE): no
External dependencies: none (Huma v2.39.1 pinned)
Existing to reuse / extend: parent ring, lifecycle hooks, guard and its provenance decisions (credential current-row and local-bypass rules, shared with `auth.Revalidate`; moved in place if needed, never copied)
Rollback notes: revert; no migration; SPA keeps refetch
Evidence required: per-subscriber counts, close reasons, openapi diff
Edge/Prohibition coverage: EDGE-005, PROH-003
Done condition: every Acceptance criteria clause holds and Tests/proof plus shared gates pass.
## SLICE-009 Tabs share one authorized stream per installation
Goal: confirmed tabs share one authorized stream per installation; visible states unchanged.
Satisfies: AC-037, AC-034, AC-036
Acceptance criteria: first-touch integration proof before client code at mapped `http://boxset-lan.test:<port>` (`isSecureContext === false`): built worker loads under production CSP, 0 violations, both bases; `eventsStream` carries the cookie and yields `hello` before stream end; two tabs share one worker; failure = same-scope /rite-plan repair, not a technology branch. Then at `/` and `/media`: 3 tabs -> one server stream; owner exit -> no gap or one announced resync; restart -> resync + refetch; downgrade/sign-out -> existing no-permission/sign-in in every tab; each tab confirms only with the `{installId, user.id, restoreEpoch}` triple from its own api.ts `GET /api/v1/session` response (no injected triple; the worker's `hello` never confirms a port); other identity and unconfirmed tab get no frame; two same-origin installations with equal user id and restoreEpoch and distinct `installId` stay isolated (an A-confirmed tab receives no B frame); invalidations flush after settle incl. failure.
Complexity: 4 - one identity-scoped relay.
Mode: AFK
Gate: none
SLA: none
Checkpoint: none
Dependencies: SLICE-008 (via /api/v1/events and Event* components, Session.installId)
depends_on: [SLICE-008]
Consumes / Produces: consumes parent api.ts, session, Query, e2e harness (ASM-006) and `Session.installId` in regenerated api.gen.ts (SLICE-008); produces Browser stream contract.
Known-Gotchas: localhost is a secure context, use a Host-allowed mapped name; worker graph has no Kit/DOM imports.
Prior-slice learnings: fold Build carry-forward constraints
Files likely touched: web/src/lib/api.ts, web/src/lib/session.ts, web/src/lib/events/stream-worker.ts, web/src/lib/events/client.ts, web/src/lib/events/client.test.ts, web/e2e/events.spec.ts, internal/server/e2e_test.go
Writer allowlist: web/src/lib/api.ts, web/src/lib/session.ts, web/src/lib/events/stream-worker.ts, web/src/lib/events/client.ts, web/src/lib/events/client.test.ts, web/e2e/events.spec.ts, internal/server/e2e_test.go
Tests/proof: `scripts/dev-run.sh --capture $CAP -- env CGO_ENABLED=0 go test -tags=e2e ./internal/server -run TestBrowserStreamIntegration` ok first; `scripts/dev-run.sh -- bun --cwd web run test` ok; `scripts/dev-run.sh -- bun --cwd web run check` and `run check:api` ok; `scripts/dev-run.sh --capture $CAP -- env CGO_ENABLED=0 go test -tags=e2e ./internal/server -run TestBrowserEvents` ok with capture PASS line.
Browser proof required: yes
Frontend craft required: yes
Design brief states: confirmed identity live; empty stream keeps page; disconnect with existing reconnect cue; resync refresh; expiry/revocation/identity switch -> sign-in or no-permission; restore/recovery public state; long reasons never raw
Visual acceptance: each state x 320/768/1024/1440 CSS px x keyboard/touch at `/` and `/media`, 200% text, 400% zoom, light/dark, reduced motion: matches parent captures, 0 CSP, 0 axe, no new horizontal scroll, layout shift or focus loss
Fullstack (FE+BE): yes
External dependencies: none
Existing to reuse / extend: parent api.ts, Query client, Playwright harness
Rollback notes: revert; SPA returns to refetch
Evidence required: per-check integration line, stream counts, captures, CSP/axe counts
Edge/Prohibition coverage: EDGE-005, PROH-003
Done condition: every Acceptance criteria clause holds and Tests/proof plus shared gates pass.
## SLICE-010 Admitted job start stays within T08 under effects load and container limits
Goal: admitted job start meets T08 under effects load within container limits, invalid evidence refused.
Satisfies: AC-024, AC-035
Acceptance criteria: EFFECTS-BENCH-PASS only for an eligible 5-invocation series meeting <=100 ms with noise and passing all correctness checks; GOMAXPROCS=1/64 invocations pass; noise >= budget, missing sample, mixed binary/protocol or ineligible run -> INVALID; benchslow -> FAIL; V1-only/missing/partial/ineligible V2 data, no `--bench-data`/`data_dir=/bench-data`, bad RUNS, position mismatch or missing position-7 retention/child observation -> INVALID; position-7 memory.peak with outbound saturation and resident child < 4g; non-production never passes, class from executed build info + digest; invocations <=600 s incl. untimed build/seed; every execution records complete attributed writer commit/wait probe samples started within the active 50/s producer interval (>=200 in-load, none open at end), and the judge prints the per-arm descriptive EFFECTS-WRITER-BASELINE lines, never a writer budget verdict; missing, short, out-of-interval, unattributed or recomputation-mismatched writer series -> INVALID.
Complexity: 4 - harness, workload and judge must change together.
Mode: HITL
Gate: validating
SLA: 4h
Checkpoint: owner confirms reserved CPUSET (same-NUMA, no SMT siblings), no swap, empty `BOXSET_BENCH_CONFIG` and checkout `docs/spikes/data` (holds `"$RUN/work"`) on NVMe ext4 just before measuring; exists only at action time (HZN-002).
Dependencies: SLICE-008 (via event frames), SLICE-003 (via Runtime.Accept), SLICE-002 (via jobs.prune), SLICE-001 (via alias.purge load and server.Start), SLICE-006 (via outbound client permits/limiter cache/shared Slot)
depends_on: [SLICE-008, SLICE-006, SLICE-003, SLICE-002, SLICE-001]
Consumes / Produces: consumes the SLICE-001 untagged slow-control seam (`internal/jobs/lane.go` interface asserted at the attempt start, `internal/jobs/benchslow.go` tagged implementation) so `-tags benchslow` delays the attempt inside the measured window; consumes `server.Effects.DB`, `server.Effects.Jobs` and `server.Effects.Outbound` from `server.Start`'s `Serving.Effects`, the composition `cmd/boxset serve` calls (same store and client; synthetic Resolver passed only through `ServeConfig.Outbound`; stopped through `Serving.Stop`; bench/effects calls no `store.Open`, `jobs.New`, `outbound.New` or `events.New`), parent `internal/child`, runner `--bench --bench-data` V2 mode, `benchprofile` `DecodeRecordSet`/`CheckDataEligibility` (ASM-007; no copy); produces `bench/effects`, `internal/jobs/benchslow.go`, exported `storebench.Probe`/`storebench.CreateProbe` in bench/store/probe.go (moved from bench/store/foreground_test.go; callers migrated, no alias or old method).
Known-Gotchas: timing only from the untagged -trimpath harness built by explicit `go build` before timed regions; DBs only below `/bench-data`, never `/tmp`; events from real PATCH; no threshold beyond T08; writer samples are a descriptive first baseline, never a budget; probe only through the composition's `*store.DB`.
Prior-slice learnings: fold Build carry-forward constraints
Files likely touched: bench/effects/main.go, bench/effects/protocol.go, bench/effects/workload.go, bench/effects/protocol_test.go, internal/jobs/benchslow.go, bench/store/probe.go, bench/store/foreground_test.go
Writer allowlist: bench/effects/main.go, bench/effects/protocol.go, bench/effects/workload.go, bench/effects/protocol_test.go, internal/jobs/benchslow.go, bench/store/probe.go, bench/store/foreground_test.go
Tests/proof: `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 ./bench/effects/...` ok (judge PASS/FAIL/INVALID controls); judge controls also: V1-only/missing/partial/ineligible V2, absent `data_dir`, non-production or label-only class, mixed digests, malformed/duplicate/unordered/wrong-count RUNS, missing retention or child observation, absent/short/open-probe writer series, samples started outside the active producer interval counted in-load, writer samples in a wrong arm/position/ORDER/GOMAXPROCS/digest, summaries without raw or unequal to `storebench.Summarize` of raw, or a self-reported writer verdict/label -> INVALID; `TestHarnessServeComposition`: the harness composition step (what `main` runs after its existing data-dir device and eligibility gate) calls `server.Start` on a temp data dir and requires `Serving.State` = `ready`. Its CreateProbe/Probe loop through `Serving.Effects.DB` records exactly the committed `bench_probe` increments, `Accept(jobs.prune)` for a harness-seeded admin through `Serving.Effects.Jobs` reaches `succeeded`, and `Serving.Stop` returns nil with the data-dir lock reacquirable; unchanged store cohort `scripts/dev-run.sh -- env CGO_ENABLED=0 go test -count=1 ./bench/...` passes with bench/store tests unmodified except Probe/CreateProbe call sites; measured: plan.md § Timing protocol command, positions 1-7, `TAGS=''`, fresh RUN each, judge on `"$RUNS"` prints EFFECTS-BENCH-PASS (test-plan row measured AC-024/AC-035); `TAGS=benchslow`, judge on `"$SLOW_RUNS"` prints EFFECTS-BENCH-FAIL; bench cohorts unchanged.
Browser proof required: no
Frontend craft required: no
Design brief states: none
Visual acceptance: none
Fullstack (FE+BE): no
External dependencies: none
Existing to reuse / extend: parent eligibility, runner bench mode, store foreground probe and nearest-rank Summarize (bench/store)
Rollback notes: revert; no product state; published runs untouched
Evidence required: RUN dirs with V2 records, RUNS/SLOW_RUNS, per-run digest/build settings, position-7 service/child counters and memory.peak, judge lines, medians/noise, correctness counters, per-execution writer scheduled/skipped/in-load counts and producer interval bounds, EFFECTS-WRITER-BASELINE lines, slow-control FAIL
Edge/Prohibition coverage: EDGE-003, EDGE-005, PROH-004
Done condition: every Acceptance criteria clause holds and Tests/proof plus shared gates pass.
