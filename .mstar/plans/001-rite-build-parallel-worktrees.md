# rite-build parallel worktree slices

> **For agentic workers:** REQUIRED SUB-SKILL: Use `mstar-sdd` (recommended) or inline execution. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Let `/rite-build` (via an opt-in flag/command) run up to 3 independent slices in git worktrees without changing the default one-slice contract.

**Architecture:** Opt-in `/rite-build --parallel N` (N≤3) fans out N path-disjoint slices into git worktrees from one shared control `HEAD`. The controlling root (skill) owns eligibility, leasing, host dispatch of N `devrites-slice-wright`s, per-worktree prove, and **all-or-nothing serial integrate** back onto the control line. Shared `.devrites/work/<slug>` lives only on the control tree; product writes stay wright-only inside each worktree. Default `/rite-build` remains one-slice-then-stop.

**Tech Stack:** DevRites pack skills (`rite-build`, wright-dispatch, phase-contract, candidate-integrity), `devrites-engine` (path-disjoint/lease checks in B3+; candidate digest authority unchanged), git worktrees + branches, host Task / native agent dispatch for concurrent `devrites-slice-wright`.

**Execution:** mstar-sdd

**plan_id:** `001-rite-build-parallel-worktrees`  
**workflow_id:** `wf-rite-build-parallel`  
**primary_spec:** `.mstar/specs/rite-build-parallel-worktrees.md`  
**Phase:** Execute — `plan` **LOCKED** 2026-08-23 → implement B2 MVP (T3–T10); B3/B4 deferred  
**Task category:** `logic` (product + engine/skill contract)

## Intent gate

| Item | Statement |
|------|-----------|
| Real goal | Raise multi-slice throughput on path-disjoint work without clobbering a shared working tree or changing the default one-slice safety model |
| Success | Opt-in `/rite-build --parallel N` (2≤N≤3; N=1/omitted ≡ today’s one-slice path) builds only path-disjoint pending slices in distinct git worktrees; each member passes the one-slice fail-on-red prove; all-green → serial integrate yields one coherent control candidate + shared `.devrites` updates; bare `/rite-build` remains one-slice-then-stop; one red aborts the whole batch (no partial sibling integrate) |
| Non-goals | Auto-parallel by default; Docker/VM env isolation (B4); unbounded N; skipping per-slice prove; partial sibling integrate; per-worktree `.devrites` mirrors; mid-batch `/rite-prove` |

## Locked clarify (complete — 2026-08-23)

1. **Merge model:** Fan-out worktrees → **serial integrate** into the feature line after successes. Control tree owns `.devrites` bookkeeping.
2. **UX:** Keep `/rite-build` **one-slice default**; opt-in via **`/rite-build --parallel N`** (N≤3).
3. **Isolation:** Git worktree + branch; shared host toolchains OK. Full env isolation deferred (roadmap B4).
4. **Eligibility:** Parallel only when slice path sets are **path-disjoint** (engine/plan-proven); else force serial.
5. **Failure:** One red slice → **abort integrate of the whole batch** (no partial sibling integrate).
6. **Bookkeeping:** **Shared** `.devrites/work/<slug>` on the **control** tree; leased slice cursors; no per-worktree workspace mirrors.

## Current contract blockers (evidence)

- Hard rule: **one slice at a time**; stop after one verified slice (`rite-build` SKILL + phase-contract + `one-slice-cycle.md`).
- Singular `.devrites/ACTIVE` + shared `state.md` / `evidence.md` / candidate integrity (`candidate-integrity.md`: Build maintains one mutable manifest until Polish).
- Wright-dispatch today: **at most one writer across all linked worktrees**; parallel writers forbidden until a separate design; isolated-worktree path is a **serial pilot** with host-native reconciliation.
- One wright path allowlist per dispatch; root never patches product source.
- AFK budget charges once per green built slice (`afk-discipline.md`).
- Engine boundary (docs/orchestration.md): engine owns deterministic structure/state/safety checks; skills/hosts own dispatch/scheduling. Thin-engine ADRs apply — do not grow an agent bridge into the engine for fan-out.

## Specify (product — complete 2026-08-23)

### Problem

Today `/rite-build` correctly builds **exactly one** approved slice per invocation (HITL stops; AFK may chain only **serially** under budget). Features with several **path-disjoint** pending slices therefore pay wall-clock ≈ sum(slice times) even when the host can run concurrent writers. Ad-hoc concurrent edits on one checkout are unsafe: shared index/working tree, singular ACTIVE/candidate cursors, and one wright allowlist assume a single writer. There is no first-class, gated opt-in that preserves the one-slice prove contract while isolating writers.

**Falsifier:** If users with ≥2 path-disjoint pending slices cannot get concurrent builds without either (a) changing default one-slice UX or (b) risking cross-slice working-tree clobber / incoherent `.devrites`, the problem remains.

### User value

- **Power users / AFK:** Opt-in batch of up to 3 eligible slices finishes faster without inventing a second command family for the common path.
- **Default users:** Unchanged muscle memory — bare `/rite-build` still means one verified slice then stop.
- **Reviewability:** Each slice still has its own wright contract, path allowlist, and fail-on-red prove before any integrate; batch failure does not silently land sibling diffs.

### Target users / stories

1. As a builder with 2–3 path-disjoint pending slices and AFK budget ≥K, I invoke `/rite-build --parallel K` and get K concurrent worktree builds, then one serial integrate, charging AFK once per **successfully integrated** green slice.
2. As a HITL user, I invoke bare `/rite-build` and still get exactly one slice then a stop — even if other slices are eligible for parallel.
3. As a builder with overlapping slice paths, I am forced to serial (or refused parallel) with an explicit eligibility reason — never a silent race.
4. As a builder when one parallel sibling goes red/gap, none of the batch is integrated; diagnosis artifacts remain per Architecture abort policy.

### In scope

- **UX surface:** Opt-in **`/rite-build --parallel N` only**. Parallel mode requires integer **2≤N≤3**. Omitted flag or **N=1 ≡ today’s one-slice path** (no parallel fan-out). Non-int / N>3 → hard refuse with message (no silent clamp).
- **Eligibility gate:** Parallel dispatch only when selected pending slices’ exact source/test path sets are **pairwise path-disjoint**. If fewer than 2 qualify → **force serial** with explicit reason (locked clarify #4).
- **Worktree lifecycle:** Create/cleanup of ≤3 git worktrees + branches from a shared base for the batch (mechanics → Architecture).
- **Per-slice contract:** Each worktree runs the existing one-slice write/prove responsibilities: exact `devrites-slice-wright`, path allowlist, inspect return, doubt where required, **approved fail-on-red prove**, no root product patches.
- **Serial integrate:** Only after **all** selected batch members are green; integrate into the feature line in defined order; control tree updates shared `.devrites/work/<slug>` only after successful integrate.
- **Failure (product):** Any red/gap/STOP in the batch → **do not integrate any sibling**; surface which slice failed; preserve diagnosis artifacts (Architecture 9a).
- **AFK:** Charge exactly once per **successfully integrated** green slice; zero charge on abort; stop before further dispatch at zero; malformed budget fails closed.
- **Docs/checks:** Skill/reference carve-out: today’s “parallel writers forbidden” becomes “parallel only via `--parallel` under path-disjoint + abort-batch + control lease”; pack regen/validate after Execute changes.
- **MVP boundary (product accepts architect recommendation):** B2 skill orchestration + advisory control lease is enough for MVP; **B3 engine leases not required for MVP**; B4 env isolation deferred.

### Out of scope

- Changing default `/rite-build` to auto-select parallel, or changing HITL one-slice stop.
- N>3 or unbounded fan-out; multi-feature / cross-repo batches.
- Docker/VM/container or port/cache/DB namespaces (**B4**).
- Running `/rite-prove` (whole-feature prove) inside or across the parallel batch; prove remains post-integration when all slices are built.
- Weakening per-slice red/green prove, TDD, or wright-only product writes.
- Per-worktree mirrors of `.devrites/work/<slug>` or multiple ACTIVE files as the SSOT.
- Partial integrate of green siblings when any sibling is red/gap.
- Reworking Codex/Claude wright **identity** beyond worktree cwd + allowlist + `worktree_base` (host gate stays; capability fail-closed).
- Morning Star harness multi-plan lease machinery as a substitute for DevRites slice bookkeeping (orthogonal).

### Target state (product)

1. Bare `/rite-build` ≡ today’s one-slice-then-stop contract (HITL/AFK unchanged).
2. `/rite-build --parallel N` (2≤N≤3) ≡ select up to N path-disjoint pending slices → fan-out worktrees → concurrent wright+prove → serial integrate on all-green → control `.devrites` coherent → reply names next pending or `/rite-prove` when none remain.
3. Ineligible parallel request never races writers on one tree (force serial / refuse with reason).
4. Skill text documents the official opt-in parallel path without weakening same-worktree multi-writer bans.

### Acceptance language (falsifiable)

| ID | Criterion | Passes when | Fails when |
|----|-----------|-------------|------------|
| A1 | Default UX | Bare `/rite-build` builds ≤1 slice and stops (HITL) | It starts a second slice without AFK/user re-invoke |
| A2 | Flag surface | `--parallel` with N∈{2,3} is the only parallel entry; N=1/omitted ≡ serial; N>3/non-int refused | Sibling command required, or invalid N accepted silently |
| A3 | Disjoint gate | Overlapping path sets refuse parallel / force serial with reason | Overlapping slices write concurrently in one tree |
| A4 | Isolation | Concurrent writers use distinct worktrees; no cross-write of sibling product paths | Two wrights mutate the same worktree product paths |
| A5 | Prove | Each batch member is green under the same fail-on-red rules before integrate | Integrate proceeds with a red/gap sibling |
| A6 | Atomic batch integrate | One red/gap → zero siblings integrated onto control | Any sibling lands on control without the failed one being excluded from the batch |
| A7 | Bookkeeping SSOT | Post-success, control `.devrites/work/<slug>` reflects all integrated slices; no divergent per-worktree SSOT | ACTIVE/state/evidence disagree across trees as competing SSOTs |
| A8 | AFK | After K successfully integrated green slices, budget decrements by K; abort charges 0 | Double-charge, charge on abort, or skip charge for an integrated member |

### Definition of Done (testable)

- [ ] **D1 — Default unchanged:** With ≥2 pending slices, bare `/rite-build` (HITL) completes exactly one green slice (or stops on red) and does not dispatch a second wright in that invocation.
- [ ] **D2 — Flag parse:** `--parallel` omitted → one-slice path; `--parallel 1` → one-slice path (no fan-out); `--parallel 4` / non-int → hard error with message (no silent clamp).
- [ ] **D3 — Path-disjoint:** Fixture with overlapping exact paths → parallel refused or forced serial with reason; fixture with ≥2 disjoint paths → K worktrees created (observable via `git worktree list` or lease artifact).
- [ ] **D4 — No cross-write:** During a green parallel run, `git diff --name-only` in worktree A never includes allowlisted paths of slice B (and vice versa) before integrate.
- [ ] **D5 — All-green integrate:** After K green proves, serial integrate yields one control feature-line tip; control `touched-files.md` / evidence include the **union** of batch paths; applicable `devrites-engine` candidate/readiness checks still pass on control.
- [ ] **D6 — Red aborts batch:** Inject one red/gap in a 2-slice batch → control product tree remains at base `B`; neither sibling is integrated; user-visible failure names the bad slice; lease status aborted/integrate-failed.
- [ ] **D7 — Docs:** `rite-build` SKILL + wright-dispatch/phase-contract (as needed) state default vs `--parallel`, eligibility, abort-batch, control bookkeeping; pack regen + validate green after Execute edits.
- [ ] **D8 — AFK:** With AFK budget B and K successfully integrated green parallel slices (K≤B), remaining budget is B−K; abort path leaves budget unchanged; at 0, no further dispatch.

### Open for architect / lock (do not reopen locked clarify)

Product specify is complete. Architect decisions table (below Architecture) closes plan-lock items:

1. ~~Lease filename~~ — **decided:** `.devrites/work/<slug>/parallel-lease.md` (T6).
2. ~~Path-disjoint helper in B2~~ — **decided:** script-first; engine CLI in B3 (T7/T12).
3. ~~Codex capability~~ — **decided:** fail-closed → serial with message; **Execute GO:** human sign-off on target hosts still required.
4. ~~N=1 on `--parallel`~~ — **decided:** ≡ default one-slice path.
5. ~~B3 engine leases for MVP?~~ — **decided:** not required for MVP (skill advisory lease first).

---

## Architecture (Prepare — ready for SDD)

### Architect decisions (close open-for-architect)

| Open | Decision (plan-ready) | Rationale |
|------|----------------------|-----------|
| Lease filename | **`.devrites/work/<slug>/parallel-lease.md`** | Lives under shared control workspace SSOT (locked clarify #6); markdown matches sibling artifacts (`state.md`, `evidence.md`) |
| Path-disjoint helper in B2 | **Script/pure helper first** under `scripts/` or pack skill-adjacent testable module; **engine CLI in B3** (T12) | Keeps thin-engine; ships MVP without CLI surface churn |
| Codex matrix | **Fail-closed:** if host lacks concurrent distinct-worktree writers → print `unavailable` and force serial; never root-emulate | Matches isolated-pilot host gate; preserves read-only root |
| Integrate mechanism | **Reuse isolated-pilot `transfer_commit` per sibling**; root stages on an integration branch then all-or-nothing ff to control | Same inspect keys as serial pilot; no new transfer protocol |
| Worktree layout | **`<repo>/.scratch/parallel-wt/<batch-id>/<slice-id>/`** (gitignored); branches `devrites/parallel/<slug>/<batch-id>/<slice-id>` | Outside `.devrites/` and tracked pack; lease records absolute paths |

### Roles: engine vs skill vs host

| Concern | Owner | Notes |
|---------|--------|-------|
| Parse `--parallel N`, select ≤N pending slices, HITL/AFK gates | **Skill** (`rite-build`) | Default path untouched when flag absent/N=1; AFK charge only post-successful integrate (A8/D8) |
| Derive exact source/test path sets; reject dirs/globs/`.devrites/**` | **Skill** (existing wright-dispatch prepare) | Same contract as serial |
| Path-disjoint eligibility | **Skill + script helper (MVP)**; **engine CLI in B3** | Pure function over path sets; golden tests in B2 |
| Create/remove git worktrees + branches | **Skill** (via shell) or **host** if native multi-worktree API exists | Prefer host-native when available; else skill-managed git under `.scratch/` |
| Concurrent `devrites-slice-wright` spawn/wait | **Host** | Claude Task / Codex agent threads; skill never fakes concurrency |
| Product source/test writes | **Wright only** | One allowlist per wright; cwd = that slice's worktree |
| Per-slice proof commands | **Skill** root, executed against **worktree cwd** | Still fail-on-red; still `test-plan.md` only |
| Doubt / test-analyst | **Skill** root, read-only specialists | May run after each wright return or after batch pre-integrate; must not mutate product |
| `.devrites/**` bookkeeping, ACTIVE, evidence, touched-files, AFK charge | **Skill** root on **control tree only** | Wrights forbidden from `.devrites/**` (existing); bookkeeping writes only after successful integrate |
| Parallel lease artifact + cursor lease semantics | **Skill MVP**; **engine enforce in B3** | Advisory `parallel-lease.md` under work slug |
| Candidate SHA-256 | **Engine** (`check candidate`) | Only after successful integrate on control tree |
| Structural readiness / seal / secret scan | **Engine** | Unchanged; B3 may add lease-awareness |
| Morning Star (`.mstar/**`) paths | **Control harness only** | Never rewrite via worktree-relative paths |

### Control tree vs worker worktrees

```text
CONTROL (user checkout)
  ├── .devrites/ACTIVE → <slug>
  ├── .devrites/work/<slug>/              ← sole workspace SSOT
  │     └── parallel-lease.md             ← advisory lease (MVP)
  ├── .scratch/parallel-wt/<batch>/<slice>/  ← worker worktree paths (gitignored)
  ├── product tree at base SHA B
  └── (orchestration only; no product writes)

WORKER wt-i (git worktree at B, branch devrites/parallel/<slug>/<batch>/<slice>)
  ├── product tree (writable by wright-i only)
  ├── resolves same git objects / common dir
  └── MUST NOT write .devrites/** (allowlist + inspect)
```

**Worktree placement:** `<repo>/.scratch/parallel-wt/<batch-id>/<slice-id>/`, gitignored. Never nest under `.devrites/work/<slug>/` (SSOT pollution) or tracked pack paths.

**Branch naming:** `devrites/parallel/<slug>/<batch-id>/<slice-id>` — local only; never push as part of Build.

### End-to-end batch lifecycle

1. **Orient (control).** Read ACTIVE, `state.md`, readiness (`devrites-engine check readiness <slug>`), tasks, assumptions. Refuse parallel if readiness/open human gates fail (same as serial).
2. **Parse N.** Require integer 2≤N≤3 for parallel mode (N=1 or omitted → existing one-slice path). Cap by remaining AFK budget when AFK is active (`remaining = min(N, budget)`).
3. **Select candidates.** Walk pending slices in plan order; for each, compute exact path set (from tasks/plan applicability — same as serial prepare). Build a maximal prefix/set of size ≤N that is pairwise path-disjoint. If fewer than 2 qualify → **force serial** with an explicit reason (do not error the default mental model).
4. **Lease (control).** Write lease artifact; mark leased slice IDs in `state.md` cursor notes; do not mark slices `built` yet.
5. **Fan-out.** From frozen base SHA `B = HEAD` (must be clean for product paths; same cleanliness bar as isolated-wright pilot where applicable):
   - `git worktree add -b <branch> <path> <B>` × K
   - Record paths + branches in lease
6. **Dispatch (host).** Spawn K fresh `devrites-slice-wright`s in parallel, each with:
   - `cwd` / workspace = worktree path
   - exact path allowlist for that slice only
   - `worktree_base = B` (wright's first command must verify `git rev-parse HEAD == B`)
   - verbatim acceptance, exclusions, proof commands
7. **Per-slice inspect + prove (in worktree).** For each return: path-contract check, test-diff integrity, approved proof in that cwd. Classify each sibling `green | red | gap`.
8. **Batch gate.** If **any** sibling is red/gap/malformed → **ABORT INTEGRATE** (step 9a). If all green → **SERIAL INTEGRATE** (step 9b).
9a. **Abort.** Do not merge any sibling. Leave worktrees + branches intact for diagnosis. Update lease `status: aborted`, evidence dead-end, release slice cursors back to pending. Control product tree remains at `B`. No AFK charge for aborted slices. STOP with structured report.
9b. **Serial integrate (all-or-nothing)** — extends isolated-pilot transfer, not a new protocol:
   - Each green wright returns one local unpushed `transfer_commit`, `worktree_base=B`, and exact files (same keys as wright-dispatch isolated pilot).
   - Root proves for **each** sibling before any control tip move: descendant of `B`, `git diff --name-only B..<transfer>` exact-match to allowlist, no `.devrites/**`/submodule/symlink/unrelated delta.
   - Prefer **host-native multi-worktree reconciliation** onto a staging tip when the host exposes it. Fallback (skill-managed): create local `devrites/parallel/<slug>/<batch-id>/integrate` from `B`; apply siblings **in plan order** via `git merge --ff-only <transfer_commit>` (or equivalent replay) with exact-path proof after each apply. Root still does not hand-edit product bytes.
   - On conflict, extra paths, base move, or proof regression after any apply → abort entire integrate: reset control to `B`, preserve workers, lease `status: integrate-failed`.
   - On full success: fast-forward control to integration tip; **then** (control only) upsert `touched-files.md` from the **combined** scoped diff vs `B`, update `state.md` / `evidence.md`, charge AFK once per **successfully integrated** green slice (A8), run `devrites-engine check candidate <slug>` and record digest only if the phase expects a binding refresh (Build maintains one mutable manifest; engine remains sole hash authority).
10. **Cleanup.** On success: remove worktrees + local parallel branches after evidence recorded. On abort: keep until human/root acknowledges (lease holds paths). Never delete user work.

### Path-disjoint eligibility

**Definition:** Two slices are path-disjoint iff the sets of exact project-relative file paths in their wright contracts have empty intersection.

**Rules:**

- Only exact file paths (existing wright-dispatch bans). No directory claims ⇒ no prefix heuristics required for MVP; still normalize `\`→`/`, reject `..`, symlinks, duplicates.
- Shared test helpers needed by two slices ⇒ **not eligible** together (force serial or reslice).
- Overlap discovered only at inspect (wright returned extra/shared path) ⇒ treat as **gap** for that sibling → batch abort integrate.
- Eligibility is recomputed at batch start only; mid-batch plan edits are forbidden while lease is `running`.

**Provenance (MVP):** `scripts/check-path-disjoint` (or equivalent pure helper) + golden tests; skill calls it before fan-out. **B3:** promote to `devrites-engine check path-disjoint` (same algorithm) for fail-closed preflight.

### Control-tree leasing (MVP skill artifact)

**File (locked for plan):** `.devrites/work/<slug>/parallel-lease.md`

**Minimum fields:**

- `batch_id`, `created_at`, `base_sha`, `n`, `status` (`running|aborted|integrate-failed|complete`)
- `slices[]`: `{ id, paths[], worktree_path, branch, wright_status, transfer_commit? }`
- `control_pid_or_session` (best-effort advisory)

**Semantics:**

- Lease is **advisory in MVP**: one controlling root should hold it; a second `/rite-build` (serial or parallel) seeing `status: running` **must STOP**.
- Slice cursors: leased IDs are not selectable by a concurrent serial `/rite-build` while `running` (skill rule); B3 makes this engine-visible via flock + readiness refusal.
- No per-worktree copy of `state.md` / `evidence.md`.

### Host dispatch model (N wrights)

- **Claude:** N concurrent Task agents with exact `devrites-slice-wright` profile, `acceptEdits`, cwd = worktree path.
- **Codex:** Require host-explicit concurrent writers with distinct worktree roots + native reconciliation. If absent → `--parallel` **unavailable** → message + force serial. **Never** create worktrees from the read-only root and write into them to imitate concurrency.
- Capability gate before fan-out: host must support **K concurrent writers**; if not, refuse parallel.
- Still forbidden: two writers in one worktree; mixing same-worktree writer with parallel batch; substituting generic agents.

### Failure / rollback matrix

| Failure | Product (control) | Workers | `.devrites` | AFK |
|---------|-------------------|---------|-------------|-----|
| Eligibility < 2 | unchanged | none | note serial fallback | n/a |
| Worktree create fails | unchanged | partial cleaned | lease aborted | no charge |
| Sibling red/gap | stays at `B` | preserved | lease aborted; dead-end | no charge |
| Integrate conflict | reset to `B` | preserved | lease integrate-failed | no charge |
| Proof red after merge | reset to `B` | preserved | lease integrate-failed | no charge |
| All green + integrate OK | fast-forward | removed after record | slices built; manifest upserted | +1 per slice |

**Invariant:** Never leave control with a **partial sibling integrate**. Staging/integration branch makes this mechanical.

### Interaction with existing serial / isolated pilot

- Default `/rite-build`: unchanged one-slice cycle (`one-slice-cycle.md`).
- Serial isolated-worktree pilot remains valid for single-slice isolation when host supports it (N=1 / omitted still that path, not fan-out).
- **Doc migration (Execute T5):** Replace wright-dispatch line *"Parallel writer work remains forbidden until…"* with: *Parallel writers are allowed **only** under `/rite-build --parallel N` when path-disjoint + abort-batch + control `parallel-lease.md`; same-worktree multi-writer and root-emulated worktrees remain forbidden.* Mirror the carve-out in `docs/orchestration.md` Slice-wright lifecycle and `rite-build` SKILL.

---

## MVP vs deferred

| Item | MVP (B1–B2 Execute) | Deferred |
|------|---------------------|----------|
| UX `--parallel N` | Yes | — |
| Skill orchestration + lease file | Yes | — |
| Path-disjoint check | Script/pure helper + tests | Engine CLI hard gate in B3 |
| Concurrent host wrights | Yes where capable | Codex if host lacks capability |
| Serial all-or-nothing integrate | Yes | — |
| Engine lease / ACTIVE coherence | **Not required for MVP** | **B3** |
| Env/port/cache isolation | No | **B4** |
| Parallel prove/review | No | later |

### Recommendation: B3 engine leases — **not required for MVP**

**Skill-only leases first** are enough for MVP when a **single controlling root** owns the batch end-to-end, because:

1. Product isolation is enforced by git worktrees + path-disjoint allowlists + inspect.
2. `.devrites` has a single writer (control root) by construction during the batch.
3. Thin-engine doctrine: fan-out/scheduling stays in skill/host; engine should not grow orchestration.
4. Abort-on-red + staging integrate protect candidate integrity without mid-batch digest churn.

**Escalate to B3 engine leases before** claiming multi-session safety or shipping as default-recommended for shared machines:

- Fail-closed `check readiness` / mutations while `parallel-lease` is `running`
- Refuse serial build selecting leased slice IDs
- Atomic lease acquire/release with flock (reuse `engine/internal/state` lock patterns)
- Machine-readable lease schema validated like other workspace artifacts

**Execute order:** B2 skill+host path → measure → B3 engine coherence → B4 env isolation only on demand.

---

## Global Constraints
- Preserve wright-only product writes; root orchestration-only (no source/test patches from root).
- Named workspace / `.devrites/ACTIVE` stay authoritative; one active slug; no worktree-local ACTIVE.
- No weakening of per-slice red/green prove; proof runs in the slice worktree before integrate, and control re-checks after integrate as needed.
- Candidate integrity: one control-tree manifest; no per-sibling candidate digests; engine remains sole hash authority.
- Morning Star process artifacts (`.mstar/**`) use control harness absolute paths only.
- Do not change default one-slice HITL stop; AFK still caps total green charges and charges **only after successful integrate** (A8/D8) — never on abort or pre-integrate green.
- Path contracts remain exact files only; `.devrites/**` never in wright allowlists.
- Parallelism ceiling N≤3; shared toolchains OK; B4 isolation out of scope.
- Host capability fail-closed: no fake parallel via root-written worktrees.
- Pack SSOT edits under `pack/.claude/skills/**` (and agents as needed), then regen/validate — do not hand-edit only `pack/generated/**` as source.
- Lease path is always control `.devrites/work/<slug>/parallel-lease.md` — never under worker worktrees.

---

## Risks and rollback

| Risk | Mitigation | Rollback |
|------|------------|----------|
| Host cannot run N writers | Capability gate → serial fallback | Feature flag / ignore `--parallel` |
| Path overlap missed (generated files, lockfiles) | Disjoint on declared paths + post-return `git diff` exactness; treat surprise paths as gap | Abort batch; reslice |
| Flaky shared toolchain races (DB, ports) | Document B4; prefer slices without shared daemon needs in MVP | Abort; run serial |
| Partial integrate leaves frankenstein tree | Staging branch + ff-only to control; on failure reset to `B` | `git reset --hard B` (control); keep workers |
| Lease file ignored by second session | MVP docs+STOP; B3 engine enforce | Manual clear lease; reset |
| Docs drift ("one writer forever") vs `--parallel` | Update wright-dispatch / SKILL / orchestration in same change set | Revert pack commit |
| AFK over-charge on abort | Charge only after successful integrate | Correct AFK counter in resolve |
| Worktree litter | Cleanup on success; lease lists paths on abort | `git worktree remove` + branch delete after ACK |

**Feature rollback:** Revert skill/docs changes; default path never depended on parallel. Orphan worktrees removed via lease path list. No engine migration required for MVP skill-only leases.

---

## Verification plan

Map to product A1–A8 / D1–D8:

1. **Unit / pure (D3):** path-disjoint helper — overlap, empty, case normalization, rejection of dirty paths (`scripts/` or pack tests).
2. **Skill evals (behavioral)** under `evals/behavioral/` (extend `rite-build.json` or add `rite-build-parallel.json`):
   - A1/D1: `--parallel` absent → still one-slice stop (existing BE1).
   - A2/D2: N=1 ≡ serial; N>3/non-int refuse.
   - A3: Overlapping paths → serial fallback message, no worktrees.
   - A5/A6/D6: One sibling red → no control integrate; lease aborted.
   - A7/A8/D5/D8: All green → control advances; touched-files union; AFK charged K times **only after integrate**.
3. **Harness / integration (manual or scripted fixture repo):** create 2 disjoint file slices; run parallel; assert two worktrees under `.scratch/parallel-wt/`, then one ff integrate; assert `.devrites` only on control (D4/D5).
4. **Host matrix:** Claude concurrent Tasks green path; Codex: unavailable OR supported — assert fail-closed messaging.
5. **Regression:** `devrites-engine check readiness/candidate` on control after success; pack validate / invocation integrity scripts (D7).
6. **Chaos:** kill mid-batch → lease remains `running`; restart stops until lease cleared.

---

## Tasks (locked for SDD — B2 MVP)

### Task 1: Spec stub pointer (done)

- [x] Durable pointer `.mstar/specs/rite-build-parallel-worktrees.md` exists (Prepare).

### Task 2: Plan lock (done)

- [x] User locked plan; working branch `feat/rite-build-parallel-worktrees`; `.scratch/` gitignored.

### Task 3: rite-build --parallel surface

**Files:**
- Modify: `pack/.claude/skills/rite-build/SKILL.md`
- Note: regen mirrors only via generator in Task 10

**Interfaces:**
- Consumes: locked clarify UX (`--parallel N`, 2≤N≤3; N=1/omitted ≡ serial)
- Produces: documented argument surface; default one-slice unchanged

- [x] Document `--parallel N` in SKILL frontmatter/body; invalid N refuse; keep one-slice default
- [x] Commit on `feat/rite-build-parallel-worktrees`

### Task 4: parallel-batch reference

**Files:**
- Create: `pack/.claude/skills/rite-build/reference/parallel-batch.md`
- Modify: `pack/.claude/skills/rite-build/reference/phase-contract.md`, `pack/.claude/skills/rite-build/reference/one-slice-cycle.md`

- [x] Write lifecycle, lease schema, abort/integrate, host capability, `transfer_commit` reuse
- [x] Link from phase-contract + one-slice-cycle
- [x] Commit

### Task 5: wright-dispatch carve-out

**Files:**
- Modify: `pack/.claude/skills/rite-build/reference/wright-dispatch.md`, `docs/orchestration.md`

- [x] Replace “parallel writers forbidden forever” with official `--parallel` carve-out; keep same-worktree multi-writer ban
- [x] Commit

### Task 6: parallel worktree orchestration helpers

**Files:**
- Create: `engine/internal/parallel/*` (`path-disjoint`, lease, worktree create/abort/integrate/cleanup)
- Modify: `engine/main.go` (wire `check path-disjoint` + `parallel …`)
- Modify: `pack/.claude/skills/rite-build/reference/parallel-batch.md` (engine CLI wire-up)
- Delete (do not ship): `scripts/devrites-parallel-wt.sh`, `tests/devrites-parallel-wt-test.sh`

- [x] Path-disjoint + lease + worktree create/record-green/abort/integrate/cleanup in `devrites-engine`
- [x] Go tests for overlap refuse, dirty paths, create, abort-at-base, divergent integrate, cleanup
- [x] Skill docs call engine (not bash); bash discarded
- [x] Commit

### Task 7: path-disjoint helper + tests

**Files:**
- Create: `scripts/check-path-disjoint` (or `scripts/check-path-disjoint.mjs` / `.py`)
- Create: matching golden tests under `tests/` or adjacent

**Interfaces:**
- Consumes: JSON/lines of exact path sets per slice
- Produces: exit 0 if pairwise disjoint; nonzero + reason on overlap

- [x] Pure helper: normalize `/`, reject `..`, empty intersection check
- [x] Golden tests for overlap, empty, dirty paths
- [x] Commit

### Task 8: AFK post-integrate charge

**Files:**
- Modify: `pack/.claude/skills/rite-build/reference/afk-discipline.md`
- [x] Charge only after successful integrate; abort charges 0; align A8/D8
- [x] Commit


### Task 9: behavioral evals for parallel gates

**Files:**
- Modify or Create: `evals/behavioral/rite-build.json` and/or `evals/behavioral/rite-build-parallel.json`

- [ ] Cover A1–A8 gates: overlap→serial, red→abort, success→union + AFK K
- [ ] Commit

### Task 10: pack regen + validate

**Files:**
- Touch via generator only: `pack/generated/**`

- [ ] Run project pack generate + validate targets; fix only generator inputs if needed
- [ ] Commit

### Deferred (not this Execute wave)

- Task 11–13: B3 engine leases / path-disjoint CLI / lock tests
- Task 14: B4 env isolation


---

## Plan self-review (architect + PM before locked)

1. **Spec coverage:** Architecture covers worktree lifecycle, eligibility, leasing, dispatch, `transfer_commit` integrate, abort, roles, MVP vs B3/B4; maps to A1–A8.
2. **Placeholder scan:** Clarify boxes closed; open-for-architect items 1–5 decided (see Specify § Open for architect).
3. **Type consistency:** N≤3, path-disjoint, abort-batch, shared control `.devrites`, AFK post-integrate match locked clarify + Specify.
4. **Remaining before user-lock:** Human Codex capability sign-off on target hosts (fail-closed text already written); confirm `.scratch/` gitignore entry exists or add in Execute T6.

## Status log

| When | Gate | Note |
|------|------|------|
| 2026-08-23 | specify started | User ask: parallel ≤3 slices via worktrees. Implement deferred until Prepare. |
| 2026-08-23 | clarify partial | Locked merge=serial-integrate; UX=opt-in flag/command; isolation=worktree+branch recommended. |
| 2026-08-23 | clarify complete | Path-disjoint; abort batch; shared control `.devrites`; `--parallel N`. |
| 2026-08-23 | specify complete | Product Specify A1–A8 / D1–D8 + specs stub; open-for-architect listed. |
| 2026-08-23 | plan drafted | Architecture + constraints + risks + verification + draft tasks; B3 engine leases **not** MVP; B4 deferred. Plan not user-locked. |

| 2026-08-23 | plan locked | User Lock→Execute. Branch `feat/rite-build-parallel-worktrees`. Execute B2 T3–T10; B3/B4 deferred. |
| 2026-08-24 | execute pivot | User locked: discard bash parallel-wt; ship Go `devrites-engine` path-disjoint + lease + worktree create/abort/integrate/cleanup (Task 6). |
