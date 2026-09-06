# Parallel batch (`--parallel N`)

Opt-in only. Default `/rite-build` stays [`one-slice-cycle.md`](one-slice-cycle.md).

## Entry

| Input | Behavior |
| --- | --- |
| omitted / `--parallel 1` | Serial one-slice |
| `--parallel N` (**2≤N≤3**) | Parallel when eligible + host pass |
| non-integer / `N≤0` / `N>3` | Hard refuse (no silent clamp) |

AFK caps `N` by remaining budget. Charge **only after successful integrate**
(once per integrated green sibling). Abort / integrate-failed → **0**. Running
lease blocks another `/rite-build`.

## Path-disjoint

Exact project-relative source/test paths only. Normalize `\`→`/`. Reject `..`,
duplicates, absolutes, `.devrites/**`. Empty pairwise intersection required.

**SSOT:** `devrites-engine check path-disjoint [--root <dir>] [<json-file>|-]`
(N≥2 only; pass `--root`). Exit `0` → fan-out; else force serial. Inspect-time
overlap → sibling **gap** → abort. **Failing case:** two parallel wrights share a
path and fan-out proceeds without that check.

## Control vs workers

Control owns `.devrites/work/<slug>/` (lease + bookkeeping).
Workers: `.scratch/parallel-wt/<batch>/<slice>/` on
`devrites/parallel/<slug>/<batch>/<slice>` @ base `B`. Wrights never write
`.devrites/**`.

Lease: `batch_id`, `created_at`, `base_sha`, `n`,
`status` (`running|aborted|integrate-failed|complete`), `control_pid_or_session`,
`slices[]` (`id`, `paths`, `worktree_path`, `branch`, `wright_status`,
`transfer_commit`).

## Lifecycle

1. Orient/gate; parse N; select ≤N path-disjoint pending slices.
2. Write lease; freeze `B=HEAD`; `parallel create` worktrees.
3. Dispatch ≤3 wrights in parallel (cwd=worktree; allowlist; prove `HEAD==B`).
4. Inspect + fail-on-red → `green|red|gap`.
5. Any red/gap → **abort** (no partial integrate). All green → serial integrate.
6. Integrate: `transfer_commit` descends from `B`; `` `<base>..<transfer>` ``
   path-exact; apply in plan order. Conflict → reset to `B`, `integrate-failed`.
7. Success: FF control; union `touched-files.md`; update state/evidence; AFK +1
   per integrated sibling; optional `check candidate`.
8. Cleanup: success removes worktrees/branches; abort keeps until acknowledged.

## Engine verbs

```text
devrites-engine parallel create|record-green|abort|integrate|cleanup|status
devrites-engine parallel lease-write|lease-read|lease-clear
```

Go is SSOT. Skill calls the engine — never ad-hoc git/bash orchestration.

## Runtime isolation

Each parallel worktree must own isolated runtime resources. Before fan-out,
assign per-slice env (document in the lease or worktree README):

| Resource | Pattern |
| --- | --- |
| HTTP port | `PORT=<base+N>` or `DEVRITES_PARALLEL_PORT_<slice>` |
| Compose | `COMPOSE_PROJECT_NAME=devrites-<slug>-<batch>-<slice>` |
| SQLite/DB file | separate path under the worktree or `/tmp` |
| Dev server | one process per worktree; never share a listener |

Reusing the control worktree's running server, DB, or compose stack across
siblings causes cross-slice pollution and flaky proof. Abort if isolation cannot
be established.

## Host

omp: N concurrent `task` tool / `tasks[]` batch wrights (`acceptEdits`, cwd=worktree). Codex: require
host-explicit concurrent worktree writers + native reconcile; else force serial.
Never two writers in one worktree; never root-emulated concurrency.
