# Parallel batch (`--parallel N`)

Opt-in only. Default `/rite-build` stays the one-slice cycle in
[`one-slice-cycle.md`](one-slice-cycle.md) and [`phase-contract.md`](phase-contract.md).
This file is the batch contract for `/rite-build --parallel N`.

## Entry rules

| Input | Behavior |
| ------- | ---------- |
| Flag omitted | One-slice path — no fan-out |
| `--parallel 1` | Same as omitted — no fan-out |
| `--parallel N` with integer **2≤N≤3** | Parallel batch when eligibility + host capability pass |
| Non-integer, `N≤0`, or `N>3` | **Hard refuse** with message; never silent clamp |

AFK active: cap `N` by remaining budget (`remaining = min(N, budget)`). Charge only
after **successful integrate** (once per integrated green slice). Abort charges zero.

While a lease exists with `status: running`, a second `/rite-build` (serial or
parallel) **must STOP**.

## Path-disjoint eligibility

Two slices are path-disjoint iff their exact project-relative source/test path sets
have empty intersection. Same prepare rules as
[`wright-dispatch.md`](wright-dispatch.md): exact files only; no directories/globs;
normalize `\`→`/`; reject `..`, duplicates, absolutes, and `.devrites/**`.

**Preflight helper (authoritative):** `devrites-engine check path-disjoint [--root <dir>] [<json-file>|-]`

- Call **only when N≥2** candidate slices are selected. The command errors on
  fewer than two slices; that is intentional — do not call it for the one-slice path.
- **Must** pass `--root <control-repo>` so symlink paths under the control tree are
  rejected. Without `--root`, symlink rejection is skipped (JSON-only dry runs).
- Exit `0` → pairwise disjoint; exit `3` → overlap or dirty paths → **force serial**
  with the stderr reason (do not fan out).
- `scripts/check-path-disjoint.py` remains for Task 7 compatibility tests; skill
  orchestration must call the engine going forward (Go is SSOT).

**Orchestration helper (authoritative):** `devrites-engine parallel …` — create/remove
`.scratch/parallel-wt/` worktrees, write/read/clear control
`parallel-lease.md`, path-disjoint gate, and staging integrate (`ff-only` when
possible, else cherry-pick replay of `B..transfer`). Skill calls the engine instead
of ad-hoc git or bash. Subcommands:

```text
devrites-engine parallel create --root <repo> --slug <slug> --batch <id> --base <sha> --json <file|->
devrites-engine parallel record-green --root <repo> --slug <slug> --slice <id> --commit <sha>
devrites-engine parallel abort --root <repo> --slug <slug> [--force]
devrites-engine parallel integrate --root <repo> --slug <slug> [--apply-to-control] [--force]
devrites-engine parallel cleanup --root <repo> --slug <slug> [--force]
devrites-engine parallel status --root <repo> --slug <slug>
devrites-engine parallel lease-write|lease-read|lease-clear --root <repo> --slug <slug> ...
```

Shared test helpers claimed by two slices ⇒ not eligible together. Overlap discovered
only at inspect (extra/shared path returned) ⇒ treat that sibling as **gap** → abort
integrate. Recompute eligibility only at batch start; no mid-batch plan edits while
the lease is `running`. If fewer than 2 slices qualify → **force serial** with an
explicit reason (do not invent fan-out of one).

## Control tree vs workers

```text
CONTROL (user checkout)
  ├── .devrites/work/<slug>/              ← sole workspace SSOT
  │     └── parallel-lease.md             ← advisory lease (MVP)
  ├── .scratch/parallel-wt/<batch>/<slice>/  ← gitignored worktrees
  └── product tree at frozen base SHA B   ← root: orchestration only

WORKER wt-i (branch devrites/parallel/<slug>/<batch>/<slice> @ B)
  ├── product tree writable by wright-i only
  └── MUST NOT write .devrites/**
```

Worktrees never nest under `.devrites/work/<slug>/` or tracked pack paths. Branches
are local only — never push as part of Build.

## Lease schema (`parallel-lease.md`)

**Path (locked):** `.devrites/work/<slug>/parallel-lease.md` on the **control** tree
only. Never mirror under worker worktrees.

**Minimum fields:**

```yaml
batch_id: <id>
created_at: <iso8601>
base_sha: <B>
n: <2|3>
status: running | aborted | integrate-failed | complete
control_pid_or_session: <advisory>
slices:
  - id: <slice-id>
    paths: [<exact files…>]
    worktree_path: <abs path under .scratch/parallel-wt/…>
    branch: devrites/parallel/<slug>/<batch>/<slice>
    wright_status: pending | green | red | gap
    transfer_commit: <sha>   # set when green wright returns
```

**Semantics (MVP advisory):** one controlling root holds the lease; `running` blocks
another Build; leased slice IDs are not selectable by a concurrent serial Build.
Engine flock/readiness enforcement is B3 — not required for MVP.

## Lifecycle

1. **Orient (control).** ACTIVE, `state.md`, readiness
   (`devrites-engine check readiness <slug>`), tasks, assumptions. Same refuse rules
   as serial on open human gates / readiness miss.
2. **Parse N.** Apply entry rules above; AFK budget cap when active.
3. **Select candidates.** Walk pending slices in plan order; build a maximal set of
   size ≤N that is pairwise path-disjoint via the helper (`--root` + N≥2).
4. **Lease.** Write `parallel-lease.md`; note leased IDs in `state.md` cursors.
   Do **not** mark slices `built` yet.
5. **Fan-out.** Freeze `B = HEAD` (product-path cleanliness bar as isolated-wright
   pilot). For each member:
   `git worktree add -b <branch> <path> <B>`; record path + branch in the lease.
6. **Dispatch (host).** Spawn K fresh `devrites-slice-wright`s **in parallel**, each
   with: `cwd` = worktree; exact path allowlist for that slice only;
   `worktree_base = B` (first wright command must prove `git rev-parse HEAD == B`);
   verbatim acceptance, exclusions, proof commands.
7. **Inspect + prove (per worktree).** Path-contract, test-diff integrity, approved
   fail-on-red proof in that cwd. Classify each sibling `green | red | gap`.
8. **Batch gate.** Any red/gap/malformed → **Abort** (9a). All green → **Serial
   integrate** (9b).
9a. **Abort.** Merge **nothing**. Leave worktrees + branches for diagnosis. Lease
    `status: aborted`; evidence dead-end; release cursors to pending. Control product
    tree stays at `B`. No AFK charge. STOP with a structured report naming the bad
    sibling(s).
9b. **Serial integrate (all-or-nothing)** — reuse isolated-pilot `transfer_commit`
    (see below). On full success: fast-forward control; **then** (control only)
    upsert `touched-files.md` from the **combined** scoped diff vs `B`, update
    `state.md` / `evidence.md`, charge AFK once per integrated green slice, run
    `devrites-engine check candidate <slug>` when the phase expects a binding refresh.
9. **Cleanup.** Success: remove worktrees + local parallel branches after evidence.
    Abort / integrate-failed: keep until human/root acknowledges (lease holds paths).
    Never delete user work.

## `transfer_commit` reuse (integrate)

Same keys and proofs as the isolated writer-worktree pilot in
[`wright-dispatch.md`](wright-dispatch.md) — **not** a new transfer protocol.

Each green wright returns one local unpushed `transfer_commit`, `worktree_base=B`,
and exact files. Before any control tip move, root proves **for each** sibling:

- commit is a descendant of `B`;
- `git diff --name-only B..<transfer>` exact-matches that slice’s allowlist;
- no `.devrites/**`, submodule, symlink, or unrelated delta.

Prefer **host-native multi-worktree reconciliation** onto a staging tip when the
host exposes it. Fallback (skill-managed): create local
`devrites/parallel/<slug>/<batch-id>/integrate` from `B`; apply siblings **in plan
order** via `git merge --ff-only <transfer_commit>` (or equivalent replay) with
exact-path proof after each apply. Root still does not hand-edit product bytes.

On conflict, extra paths, base move, or proof regression after any apply → abort
entire integrate: reset control to `B`, preserve workers, lease
`status: integrate-failed`. **Never** leave a partial sibling integrate on control.

## Host capability

- **Claude:** N concurrent Task agents with exact `devrites-slice-wright` profile,
  `acceptEdits`, cwd = worktree path.
- **Codex:** Require host-explicit concurrent writers with distinct worktree roots
  plus native reconciliation. If absent → `--parallel` **unavailable** → message +
  **force serial**. Never create worktrees from the read-only root and write into
  them to imitate concurrency.
- Capability gate **before** fan-out: host must support **K concurrent writers**.
  Still forbidden: two writers in one worktree; mixing same-worktree writer with a
  parallel batch; substituting generic agents.

## Failure matrix (summary)

| Failure | Control product | Workers | Lease / `.devrites` | AFK |
| --------- | ----------------- | --------- | --------------------- | ----- |
| <2 eligible / host unavailable | unchanged | none | note serial fallback | n/a |
| Worktree create fails | unchanged | partial cleaned | `aborted` | no charge |
| Sibling red/gap | stays at `B` | preserved | `aborted`; dead-end | no charge |
| Integrate conflict / post-merge proof red | reset to `B` | preserved | `integrate-failed` | no charge |
| All green + integrate OK | fast-forward | removed after record | `complete`; slices built; manifest upserted | +1 per slice |

## What this does not change

- Bare `/rite-build` and AFK **serial** chaining remain one-slice-at-a-time on the
  control tree.
- Per-slice fail-on-red prove, TDD, and wright-only product writes stay required.
- `/rite-prove` (whole-feature) does not run inside the batch; prove after all
  slices are built on control.
- Engine lease enforcement and env/port isolation are deferred (B3 / B4).
