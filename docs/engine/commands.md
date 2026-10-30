# `devrites-engine` commands

The engine is a deterministic, stdlib-only control plane. It has no model or
provider dependency and does not dispatch agents, interpret reviews, or choose
workflow strategy.

## Complete operational command inventory

| Command | Deterministic responsibility |
| --- | --- |
| `install [flags]` | Install manifest-owned host artifacts and the optional shared binary. |
| `update [flags]` | Refresh an existing managed installation. |
| `uninstall [flags]` | Remove managed artifacts while preserving runtime workspace state. |
| `check candidate <slug>` | Validate the strict manifest and compute the content-bound project-candidate identity. Exits `3` when the manifest omits a path the feature's `WIP(<slug>):` checkpoint commits changed (net from the feature base to HEAD). Also exits `3` with `unattributed root edit: <paths>` while the feature's `tree-baseline.json` tripwire flags a product path (ADR-0038; see `dispatch`). |
| `check readiness <slug>` | Check target-Phase files, open human gates from Clarify onward, the `tasks.md` slice graph when that artifact is required, canonical `AC-###` presence in `tasks.md`/`test-plan.md` when those artifacts are required, and the current stable Build-input binding when applicable. Leaving Vet with a `Run at` column, a slice whose `Files likely touched`/`Writer allowlist` names a dependency manifest or lockfile needs a `dependency-dry-run SLICE-###` row at `vet` with a passing transcript. Leaving Vet, every slice `state.md` does not check built (a deferred `Satisfies: none` slice excepted) must also pass the Build-entry `check slice` contract; each failure prints one `invariant: slice-check:` line naming the slice and its exact `check slice` reason, to fix in `/rite-plan` before READY. |
| `check readiness --emit-binding <slug>` | Render the exact stable Build-input binding for Vet to record after review. |
| `check preconditions <slug> [--slice SLICE-###] [--timeout N]` | Run the `test-plan.md` Build-entry preflight rows whose optional `Run at` cell is `vet` (with `--slice`, `pre-dispatch <SLICE>`); `action` rows and rows without the cell never run. Writes one candidate-bound transcript per row under `references/gates/preflight-*/`; a row passes on exit 0. A `pre-dispatch` row whose newest transcript passed is kept, not re-run: it is a pre-edit baseline, so a re-dispatch never runs it against the slice's own edits; changing its command or cwd retires the pass. Exit `3` on any failure or an invalid `Run at` value. |
| `check seal <slug>` | Check files required by target Phase `seal`, open human gates, the `tasks.md` slice graph, canonical `AC-###` presence when `tasks.md`/`test-plan.md` are required, the stable Build-input binding, and exact candidate bindings. `check readiness` and `check seal` also block a `scripts/` test driver that a tasks.md Writer allowlist plans or touched-files.md lists when no gates.md `CHECK` runs it and no `Driver waiver: <path> — DEC-### <reason>` line cites a recorded decision; waived drivers print as `note: unexecuted driver (waived by DEC-###): <path>` lines, on pass or block. |
| `check path-disjoint [--root <dir>] [<json-file> | -]` | Verify slice path sets are pairwise disjoint. |
| `parallel select --cap <1-10> [--root <dir>] [--slug <slug>] [--from-plan \| <json-file> \| -]` | Choose the greedy path-disjoint subset ≤ cap from supplied ready slices, or with `--from-plan` (needs `--slug`) from the plan: tasks.md slices whose dependencies state.md checks `built`, with Writer allowlist (else Files likely touched) paths plus the nearest manifest/lockfile when `External dependencies:` names a package (`none*`, existing, or no-new prose without a coordinate). With `--slug` and a running lease, the lease's not-yet-landed slices hold cap slots and paths, slices it names are never picked, and only new picks print. |
| `parallel <create\|add\|integrate\|cleanup\|status\|abort\|lease-read\|lease-write\|lease-clear\|record-green>` | Deterministic parallel worktree leases, integration, and cleanup for parallel writer batches. A lease is a pipeline: `integrate --slice <id> --apply-to-control` lands one green sibling onto control (refusing if anything but this lease moved control), and `add` starts new path-disjoint siblings from the current control tip while others still run. |
| `check task-graph <slug>` | Validate `tasks.md` slice dependency graph for cycles, unknown deps, malformed tokens, duplicate IDs, missing `Dependencies`, and `depends_on` mismatch. Prints a non-blocking `ADVISORY size:` line (and records a `size-advisory` metric) for each slice with 12+ Writer allowlist files and no `Size rationale:`. A `Dependencies` ID may carry a `(via <interface>)` suffix (empty or unclosed is a malformed token); prints a non-blocking `ADVISORY edge:` line (and records an `edge-advisory` metric) for each edge with no via, or whose via text is not a case-insensitive substring of the dependency's `Consumes / Produces:`. Prints `schedule: <n> slices in <r> build rounds at cap 10` (the `/rite-build` batch selection replayed over each slice's Writer allowlist, else Files likely touched; omitted when a slice has no path) and a non-blocking `ADVISORY hub:` line (top five; records a `hub-advisory` metric) for each path, compared case-folded, that three or more slices without `Serial rationale:` write. Advisories print only when the graph is valid. |
| `check regression <slug> [--update]` | Compare the workspace's structural progress facts (phase ordinal, checked ACs, met/abandoned gates, done slices, resolved questions, present artifacts) against the recorded `regression-baseline.json` high-water mark; `--update` ratchets the baseline forward under the feature lock and prints each blessed regression. |
| `check drift <slug> [--record]` | Attribute readiness-input changes to the exact artifact. `--record` writes the per-input `readiness-inputs.json` digest baseline under the feature lock; without it the workspace diffs current inputs against that baseline (or falls back to the aggregate binding recorded in `eng-review.md`). Advisory: exit `0` on drift; a corrupt baseline exits `3`. A `test-plan.md` change limited to exact `Workflow Artifact reviewed identity: <64hex>` lines is reported as `reviewed-identity-only` (ADR-0034). `--record` also stores each slice's proof bytes (its `tasks.md` block plus the `test-plan.md` lines naming it); both modes print `ADVISORY proportion:` for a slice whose proof grew over 2x since that baseline or carries over 3x the median proof per AC. |
| `check windows <slug> [--worktree\|--staged\|--base <ref>\|--since-dispatch <wave>]` | Fail on deferral and bar-lowering markers the change introduces without a waiver row in the workspace's `windows.md`: added TODO/FIXME/XXX/HACK, skipped/focused-test and stub idioms, checker suppressions, one-line empty catch; removed `deleted-test` files and net `removed-assertion` lines in remaining test files. `--since-dispatch <wave>` reports only `deleted-test`/`removed-assertion` against the work-tree snapshot (`tree` in `dispatch.json`) recorded when that build wright wave opened, untracked files included; it takes no `--cwd` and records the `windows-since` metric. |
| `check dup [slug] [--all\|--worktree\|--staged\|--base <ref>] [--min-lines n] [--threshold f] [--ignore-file <path>] [--limit n] [--exclude <csv>]` | Report near-duplicate code clusters that survive comment stripping, literal collapsing, and identifier renaming — token-shingle matches chained into segments, clustered transitively, ranked with bounded path/line-distance boosts. Diff modes mark units overlapping changed hunks with `*`; stable content-derived cluster hashes let `.devrites/dup-ignore` dismissals survive line moves but resurface on structural edits. Staged mode reads index blobs and excludes untracked files. Bounded heuristic: successful scans/leads exit `0`; usage, unavailable input, and partial scans exit `2`. |
| `check anchors <slug> <return-file> [--skills-root <dir>]` | Deterministic quote anchoring for one reviewer return (agents.md § Result admission). For each Critical/Important `Finding:` row (list markers and bold tolerated), it anchors the backticked observed quote at the cited `path:line[-line]`, looking in the workspace first, then the project, never outside either. Exact match → `anchored`. Exactly one other line → `relocated <cited> -> path:N`. Otherwise → `Unverified: <row> — anchor lost\|ambiguous`; this also covers a missing file:line or quote, or a quote under 8 characters. The `breaks "<quote>"` must appear in spec.md/decisions.md (spec ID) or in the named shipped standard under the skills root (`--skills-root` for host installs). No breaks quote → `Deferred: <row> — no spec anchor`. Prints the candidate digest and records an `anchors` metrics event. Exit `0` all anchored, `3` any Unverified/Deferred row (never drives a repair round) or a `Counts:` head declaring more Critical/Important findings than `Finding:` rows, `2` usage or unreadable input. |
| `check slice <slug> <SLICE-ID>` | Pre-dispatch lint of one slice's wright contract: contract present, exact Writer allowlist paths, resolvable `Satisfies` AC ids, bounded scope. Same non-blocking size advisory as `check task-graph`, plus the `check drift` proportion advisory for that slice. |
| `check diff-scope <slug> --allow <csv>\|--allow-file <path> [--worktree\|--staged\|--base <ref>] [--baseline] [--verbose]`, `check diff-scope <slug> --snapshot` | Verify the changed-path set stays inside the declared allowlist; the mechanical gate run before reviewer dispatch. `--snapshot` records the mode and SHA-256 of every currently changed path in `<feature>/diff-scope-baseline.json` (an engine machine file, never reported); run it before writer dispatch. `--baseline` drops a changed path whose fingerprint still matches, printing `preexisting (unchanged)`; a baselined path whose bytes or mode changed, or a new path, still blocks; a run with only preexisting changes is an empty diff; a missing baseline exits `2`. More than 20 violations print grouped by the first two path segments (count plus 3 examples, at most 20 groups); `--verbose` lists every path. Exit codes are unchanged. |
| `check perturb <slug> --patch <file> --expect <test-name> --gate <G-ID> [--timeout N] [--worktree <path> --slice <SLICE-ID>]` | Safe-perturbation probe (testing.md § Safe perturbation). Copies the candidate (HEAD plus every touched-files.md manifest row, at the project's prefix below the git toplevel) into a detached temporary worktree, requires it to hash to the `check candidate` digest, applies the wright-supplied mutant patch (toplevel-relative paths), and runs the gate's CHECK only when it is an approved test-plan.md preflight row. RED counts only when a runner FAIL line (go test, pytest, cargo test, jest/vitest) names `--expect`; build/setup failure, timeout, exit 0 (survivor), or a RED naming only other tests is `cannot_verify`. Reverts the patch, requires the copy back at the candidate digest and the gate GREEN (exit 0 plus EXPECT), and requires the checkout candidate unchanged. Writes `<feature>/evidence/perturb/<gate>-<digest12>-aN.log`. Never mutates the checkout. `--worktree <path> --slice <SLICE-ID>` (both or neither) probes a linked worktree of the same repository instead (a parallel sibling, checked as for `candidate freeze --worktree`): its HEAD plus the slice's tasks.md write set, hashed as `candidate freeze` does, with `--patch` resolved against the worktree; the transcript stays under the control workspace. Exit `0` verified, `3` cannot_verify or refused, `2` usage. |
| `check skill-trust <path>` | Scan one skill/agent Markdown file for structural trust violations. |
| `check indexes [--root <dir>]` | Report install manifest and in-repo code-index artifact presence (`.codebase-memory/`) as JSON. codebase-memory-mcp normally indexes into its central store, so its `index_status` is authoritative. |
| `observe summary <slug>` | Emit a sanitized workspace summary, as one compact JSON line, from one retained observation. `task_graph.ok` is true iff `task_graph.problems` is empty; `problems` lists cycle, unknown-dep, malformed-token, duplicate-id, and missing-`Dependencies` blockers. `inflight_waves` lists dispatch waves still `open` or `sealed`, with their pending roles. |
| `orient <slug>` | Alias for `observe summary`. |
| `observe slice <slug> <SLICE-ID>` | Print one `SLICE-###` section of `tasks.md` so Build reads a slice, not the file. |
| `next [slug]` | Print the minimal remaining lifecycle path plus advisory skips; in Build also `eligible_now:`, `batch:` and the `select-command:` (`parallel select --root <project> --slug <slug> --cap N --from-plan`, N = AFK max_parallel else 10) that reproduces the batch. Prints `blocked: unattributed root edit: ...` after `open-questions:` and exits `3` while the ADR-0038 tripwire flags a product path; the rest of the output still prints. |
| `handoff [slug] [--resume]` | Emit the deterministic resume record: cursor fields, `awaiting_human`, open `questions.md` gate kinds, the `gates.md` reduction with unmet/stale/abandoned ids, `decisions.md` dead ends, in-flight dispatch waves (`inflight_wave:` lines: phase, state, pending roles, slices), missing required files, and the canonical read-next order. Read-only unless `--resume`, which appends one `resume` event (role root) to `metrics.jsonl`; `metrics summary` then reports minutes from each resume to the next dispatch. The `/rite-handoff` spine and the compaction-fallback entry point. |
| `recall --paths <p1,p2,...> [--limit n]` | Read-time query over `.devrites/archive/`: list archived features whose `touched-files.md` candidate manifest covers a queried project-relative path (exact or directory prefix, case-folded), ranked by overlap count, then newest archive-directory mtime, then slug (default 10 features). Each feature prints up to 8 `archive/<slug>/<file>:<line>` pointers, then `+N more`: `decisions.md` `DEC-###` entries and `## Carry-forward constraints` bullets whose `Files:` overlap (or a constraint naming no files), plus `review.md`/`seal.md`/`drift.md` lines that mention a queried path; fenced examples are ignored. Never writes, caches, or indexes (no metric either); symlinked archive entries are refused. An unreadable archive file prints `unreadable:` and the scan continues. Exit `0` (including `recall: no archive`), `1` invalid archive directory, `2` usage. |
| `context [slug] (--phase <p>\|--skill <name>) [--role <r>] [--slice <SLICE-ID>] [--recheck <inventory>] [--trigger a,b] [--inline-contract] [--whole]` | Emit one deduplicated read-set bundle per phase/role/skill from each skill's `loads:` manifest; `--skill` works without a workspace when `--out` is given. A root `--phase build\|autocomplete` bundle (no `--role`, no `--slice`) defaults to `--slice <active_slice>` while the state.md cursor phase is `build` and active_slice names a slice present in tasks.md (prints `scope: slice SLICE-### (auto from state.md) ...`); `--whole` opts out. Every bundle opens, after its header comments, with a table of contents: one `L<start>-<end> <section label>` line per `===== label =====` section, giving its exact line range in the bundle file. A bundle without `--role` (the root's own read) gives over-budget artifacts as by-reference indexes; a `--role` bundle leads with the role's agent contract and names the manifest's `rootOnly` paths instead of inlining them. When the contract resolves under `.claude/agents`, `.omp/agents`, or `.codex/agents` (the host loads it as the agent's system prompt), the bundle writes one header line naming its path, size, and SHA-256 instead of the body; other layouts (pi, devin, unknown) and `--inline-contract` (only with `--role`) inline the body, and a missing contract still exits `3`. Inside inlined files, a `--role` bundle replaces each `<!-- root-only -->`…`<!-- /root-only -->` block with a one-line stub naming the omitted heading, and does the same for each `<!-- model-surface -->` block unless the role is `security-auditor`, the feature has an `ai-spec.md`, or `spec.md`/`plan.md`/`tasks.md`/`design-brief.md` are absent or (outside the spec template's `AI-SPEC annex`) name a model, agent, RAG, or tool surface; an unbalanced or headingless marker exits `2`. `--slice` inlines that slice's `tasks.md` block and turns each over-budget artifact, and each artifact over 16 KiB except `spec.md`, into a by-reference index (path, SHA-256, headings, line hits for every ID the block cites); repeat `--slice` for several slices; an unknown slice exits `3`. In phase `build`, a `--role` bundle other than `slice-wright` without `--slice` exits `3` (`unscoped build reviewer bundle`). `--recheck <inventory>` (phase `build`, with `--role` and `--slice`; path relative to the workspace) writes `<phase>-<role>.<slice>.recheck.bundle.md`: the inventory file and `core.md` inline whole, every other non-`rootOnly` pack file becomes a by-reference index (path, SHA-256, headings with line numbers) followed by each section the inventory cites as `<file>.md § <heading>`; workspace handling matches `--slice`. A missing or empty inventory exits `2`. First reviews, full-inventory rechecks, Vet, `/rite-review`, and Seal do not use it. |
| `dispatch <slug> <open\|start\|seal\|return\|status\|abandon\|attribute>` | Launch-wave barrier for parallel dispatch: seal needs a handle per role, return needs seal; the start that gives the last role its handle seals the wave, and an explicit seal on a sealed wave is a no-op; `open --claim-session <id> --claim <path>... [--claim-ttl <minutes>]` records one file claim for the wave in claims.jsonl and exits `3`, creating no wave and no claim, while a live foreign session holds an overlapping path; the return that completes the wave releases that claim; `abandon --release` releases it on any other host-confirmed terminal path (a stop or a refused launch); re-opening an abandoned wave inherits its kept claim or, with new claim flags, replaces and releases it; claim flags apply to open only and `--release` to abandon only; start/return auto-record metrics, `return --outcome findings|no-findings|gap` records the role's Outcome line, and `abandon` records its reason, so `metrics summary` shows per-role yield and lost waves. Bare `status` prints every open, sealed, or abandoned wave and folds complete waves into one `N complete wave(s) (use --all)` line; `--all` lists every wave; it still exits nonzero while any wave is open or sealed. For a reviewer, auditor, or test-analyst role, `return --result <file>` stores the return verbatim as `packets/<wave>-<role>.return.md` (exit `3` if that file exists), derives the outcome from its `Outcome:`/`Counts:` head (a malformed or inconsistent head records `gap` with a warning, never `no-findings`), exits `3` when an explicit `--outcome` disagrees, and records each `Finding:` row's `role|slice|file|section|severity` key, so `metrics summary` adds per-role `resolution: resolved re-raised open late` counts. A `--result` file that already is that `packets/` path is kept as the stored return. A build wave with a wright role needs `--slice <SLICE-ID>` per slice, and a slice-bound vet wave takes it too; a reviewer-only wave (for example a Build slice review) may take it, uncapped, so its `Finding:` keys name that slice; a reviewer, auditor, or test-analyst wave needs `--packet <file>` and `open` exits `3` on a packet that seeds a conclusion (gate `*-PASS` token, `no-findings`, an `Outcome:`/`Counts:` line, an "already passed" claim, or zero Critical/Important counts) or tells the agent to read an installed `agents/devrites-*.md`/`.toml` role contract (on the line or under a "Read …" heading or label; canonical `pack/` paths excepted), and a wright wave's `--contract` gets the same role-contract check; a Review or Seal reviewer wave also exits `3` unless the packet names an existing non-empty `.diff`/`.patch` file and the manifest covers every checkpointed path; every 5 rounds of either kind on one slice, `open` exits `3` until `--reslice-review <DRIFT-ID>` names an unused `drift.md` `## <DRIFT-ID>` heading for that slice. `open` also warns when an earlier wave for the same slice never returned. Under `.devrites/AFK` max_parallel >= 2, a first-round build wright wave narrower than the `parallel select --from-plan` batch exits `3` (printing the batch, the command and the parallel-batch lifecycle) until evidence.md records `serial: <skipped ids> — <reason>`; repair rounds, lease siblings and HITL runs are never refused. A build wright `--contract` must name `gates precheck` under `Writer-safe checks:` when gates.md has a `RUN: writer` gate. `open` exits `3` when a `--role` names no pack agent in a project-level install (a `.<lane>` suffix is allowed), and lints a reviewer packet against the wave's roles, so a recheck may name its own role's fingerprint. Root-edit tripwire (ADR-0038): the first `open` records `<feature>/tree-baseline.json`, which maps dirty product paths outside `.devrites/` and `.scratch/` to their mode plus SHA-256, accepted as user work. The return that completes a wright wave (any phase) adds its slice write sets and claim paths. `open` exits `3` with `unattributed root edit: <paths>` while a dirty product path differs from that baseline and no live claim or in-flight wright wave covers it, unless the new wright wave covers the path through `--claim` or a slice write set. `attribute --reason <evidence> --path <path>...` records owner work, gate output, a standalone wright, or a sanctioned root write for flagged paths only. Each finding writes an `unattributed-edit` metrics row. The engine never reverts a file, and a workspace without the baseline is never blocked. |
| `candidate freeze <slug> --wave <w> (--slice <SLICE-ID> \| --paths <path>...) [--base <ref>] [--worktree <path>]` | Writes packets/<w>.diff (tracked changes vs base, default HEAD, plus untracked files as /dev/null diffs) and packets/<w>.sha256 (base, diff digest, per-file sha256, aggregate check-candidate digest); never replaces a wave's frozen candidate with a different one. `--worktree <path>` (requires `--base`) reads the same path set from a parallel sibling's linked worktree before it lands: the path must be a symlink-free worktree top level that `git worktree list` names, shares the control repository, and is neither a submodule nor the control checkout. It exits `3` on uncommitted path-set changes there and adds a `head <sha>` row naming the worktree HEAD (the transfer commit); packets stay in the control workspace. |
| `packet <slug> --wave <w> --phase <p> --role <r> [--slice <SLICE-ID>] [--recheck] [--checks <G-ID,...>] [--focus <file>]` | Renders one reviewer/auditor/test-analyst or same-worktree slice-wright by-reference packet (packets/<w>.<role>.md, or packets/<w>.contract.md for a wright) naming the context bundle (the .recheck bundle with `--recheck`) and frozen diff. It copies packets/<w>.sha256 (header rows only when large), lists a wright slice's tasks.md `Tests/proof` as its first writer-safe check, resolves `--checks` through gates.md and approved preflight rows (adding `gates precheck` for `RUN: writer` gates), and accepts only anchored `--focus` rows (a doubt-reviewer's 1-3 row claim is required). It runs the dispatch open lints on a draft and never replaces a prior packet with a refused one (exit `3`). |
| `metrics record <slug> --phase <p> --event <e> [--role r] [--bytes n] [--note s]` | Append an event to the workspace's `metrics.jsonl` ledger. |
| `metrics summary [slug] [--by-slice]` | Reduce `metrics.jsonl` into a per-phase roll-up. `--by-slice` reads `dispatch.json` instead: per slice, build wright rounds, slice-bound Vet rechecks, and agent-minutes (role start to `returned_at`; abandoned waves excluded). When the ledger has `resume` events it adds `resumes count=N minutes_to_dispatch=...` (each resume timed to the next `dispatch` event; `none` when another resume came first, `open` when none followed). |
| `claim <add\|release\|list\|check>` | Advisory session-scoped file claims in `.devrites/claims.jsonl` (append-only). `add` records intent to write a path set with a TTL (1–240 min, default 30); a live claim by another session exits `3` and names the holder. `release` appends a release event owned by the claiming session. `list` shows every unreleased claim (live, expired, dead) plus one `N released (use --all)` count; `--all` includes released rows; a claim whose holding session has provably exited on this host shows `dead` and no longer conflicts. `check` is the read-only preflight for a path set. Coordination aid only — never a substitute for one-writer-per-worktree. |
| `note <add\|list\|check\|rm> <slug> ...` | Anchored workspace notes in `notes.md`: `add <slug> <subject> <quote> <title> [body]` binds a note to a verbatim quote in one repository file and assigns `NOTE-###`; `list` and `check` regrade each anchor `exact`/`moved`/`stale`/`ambiguous`/`lost` by searching the repo for the quote (dependency and workspace trees excluded); `check --repair` rewrites a `moved` subject to its new file; `rm` deletes by ID. `check seal` refuses a malformed `notes.md` and any non-`exact` anchor. |
| `gates <scaffold\|status\|run\|reverify\|lint\|precheck\|attest\|abandon> <slug>` | Operate the `gates.md` acceptance ledger: seed per-AC gates, execute runnable gates bound to `test-plan.md` preflight rows, attest manual gates, and reduce to `all-met`/`not-met`/`handoff`/`malformed`. A gate with `INPUTS:` records its files' content digest and is reused by `run` and `reverify` while they are unchanged. `run`/`reverify --transcripts` writes a redacted, candidate-bound transcript per executed or reused gate under `references/gates/<gate-id>/`. `precheck <slug> --worktree <path> [--slice <SLICE-ID>]` runs every `RUN: writer` gate and `check windows` against the wright's worktree (with `--slice`, it prints `SKIP <gate-id>` for a gate whose id only other unbuilt slices name in `tasks.md`, since gate ids reuse `AC-###` ids; a gate no slice names still runs), prints PASS/FAIL per gate id, exits `3` on any FAIL or on a malformed ledger, and writes no ledger, EVIDENCE line or transcript (`check windows` still appends its metric event); its output is never evidence. |
| `state resolve <qid> "<answer>"` | Resolve an open question and update `questions.md` plus `state.md` atomically. |
| `state merge-manifest <slug> [pred...]` | Fold the recorded predecessor chain's manifests into the release candidate manifest. |
| `state set <slug> <key>=<value>... [--unset <key>]...` | Write `state.md` cursor fields all-or-nothing under the feature lock. Refuses an unknown `phase`, the engine-owned `schema` row, a pipe or line break in a value, and an overwrite of a saved `return_phase`/`return_next_action`. Setting `status=awaiting_human` under `.devrites/AFK` prints each open question the sentinel's `allow_gates` auto-picks. |
| `state append <slug> <ledger.md> --heading "<text>" --body-file <path\|->` | Append one `## <heading>` section to `assumptions.md`, `decisions.md`, `drift.md`, `evidence.md`, or `questions.md` atomically. Refuses a heading that already exists outside fenced code and a body line that opens another `#`/`##` section; for a new `questions.md` `q-` entry, also refuses a non-open status, a missing or unknown gate, a raised_at that is not a full RFC 3339 timestamp, options outside 2-4 concrete choices plus the closing "Something else" escape hatch, `(Recommended)` on any option but 1, and `proposed:` without a `(Recommended)` option 1 (or option 1 without `proposed:`); question or entry length is advisory only; reports an advisory when the ledger passes its line budget, and blocks (exit 3, nothing written) when a ledger other than evidence.md would land at or above 1.5x its line or byte budget without a covering `Budget override: <=N lines; <reason>` (N at most 2x the line budget). |
| `state close <slug>` | Archive a shipped workspace and clear matching `ACTIVE`. |
| `migrate <slug> [--dry-run] [--answer id=choice]` | Normalize a pre-v5 workspace to the current schema; fail-closed, one-shot. |
| `secret-scan [--staged] [--stdin] [slug]` | Scan exact staged blobs, stdin, or touched regular files for credential material. |
| `open-visual <path-or-name> [--slug <slug>] [--no-open]` | Resolve a local visual HTML file, optionally open it in the OS browser, warn if the sibling outline is missing or inventory ids are absent from HTML, and print agent path tips. No network. |
| `detect commands [--root <dir>] [--json]` | Resolve the repository's own test/lint/vet/build commands: explicit Makefile targets first, then `package.json` scripts, then language-manifest fallbacks, plus the lockfile-derived package manager. Read-only, no installs, no execution. |
| `overhaul <records\|admit\|snapshot\|score\|bench\|render> ...` | Deterministic tools behind the standalone `/overhaul` skill: run-record generations and cross-record validation, worker-receipt admission and quote anchoring, index-safe baseline snapshots and per-attempt deltas, exact-rational readiness scoring, benchmark ratio intervals, and escaped offline review/report views. Operates only on explicit run-area and repository paths. See [Overhaul tools](#overhaul-tools). |
| `version` | Print the engine version. |

`help`, `-h`, and `--help` print this operational inventory. Each operational
command and subcommand also accepts `-h` / `--help` and prints that command's
usage to stdout without requiring a workspace. `version` and `--version` print
the binary version. Other unlisted command forms are rejected as unknown; the
engine has no compatibility aliases or tombstones. The `add`/`upgrade`/`remove`
conveniences belong only to the `npx devrites` adapter, not to the engine
command namespace.

## Check boundary

The candidate gate validates and hashes path/state/type/mode/content identity;
it does not infer scope from Git. The readiness gate checks target-Phase
structure and applies open-question blocking only when that target is Clarify
or later, plus the exact `tasks.md` slice graph when `tasks.md` is required,
plus canonical `AC-###` ID presence in `tasks.md` and `test-plan.md` when those
artifacts are required, plus the exact stable Build-input binding after Vet. The
seal gate always targets Phase `seal`, repeats that graph, ID map, and binding,
and checks exact candidate bindings in evidence, optional browser evidence,
review, and seal. None judges the meaning of `CLEAR`/`READY` prose,
parses reviewer narratives, infers semantic acceptance coverage, counts
assertions, interprets capability deltas, or decides whether a technical plan is
sound.

Those judgments are made by the current skill and exact native roles, including
`devrites-plan-reviewer`, `devrites-proof-runner`, `devrites-spec-reviewer`,
`devrites-test-analyst`, and `devrites-doubt-reviewer`. The root reconciles their
reports against live artifacts and observed repository proof.

## State

State mutations use the shared physical-root checks, feature lock, and atomic
write path. `state resolve` additionally supports `--drop` and `--batch`;
`state close` owns transactional archive plus `ACTIVE` clearing.

`migrate` normalizes a pre-v5 workspace: legacy bullet cursor fields become
canonical table rows, the `schema` row is recorded, and missing required
artifacts are created as empty stubs (content is never synthesized, and bound
proof files stay byte-exact). It is one-shot and fail-closed: on ambiguity it
writes nothing, prints its questions, and exits `3`; answers arrive on rerun
via `--answer id=choice`. `--dry-run` prints the plan and always writes
nothing. See [ADR-0029](../adr/0029-v5-workspace-schema-and-native-migration.md).

Normative spec grammar checks, qid allocation, Clarify cursor transitions, AFK
slice accounting, recovery attempt accounting, and installation diagnostics are
explicit root-owned native procedures. The workflow owns reproduction,
hypothesis ranking, tool selection, and routing. No replacement scripts or
counter artifacts are introduced.

## Install and update boundary

Install application and uninstall accept local source, pre-generated host
payload, and optional staged binary inputs. Direct `devrites-engine update`
selects the latest stable release, downloads its bundle and platform engine,
then invokes that downloaded engine with local candidate paths. Shell and npm
may instead acquire and pass the same local inputs. `update --check` compares
installed and latest release metadata without downloading assets. `--to` and
`--pre` are not supported engine flags.

Remote acquisition is isolated to the release boundary, exact-SemVer,
HTTPS-only at every redirect, bounded, and requires exact-filename SHA-256
sidecars. Archive validation completes during bounded extraction and unchecked
raw/source/default-branch fallbacks are absent.

## Secret scanning

`secret-scan --staged` enumerates changed index entries and reads their exact
blob object IDs with replacement objects disabled. It does not substitute
working-tree bytes or follow a worktree symlink. `--stdin` reads supplied text
from process stdin; callers must not put that text in argv, environment, command
logs, here-documents, or temporary files.

Each invocation accepts at most 4,096 entries, 64 MiB total captured input, and
4,096 findings. Findings never include matched bytes, excerpts, or value hashes.
Input, limit, and output errors exit `2`; HIGH findings exit `3`.

## Open visual

`open-visual` resolves `<path-or-name>` to a local `.html` file under the
active/`DEVRITES_WORKSPACE`/`--slug` workspace `visual/` directory, or via an
absolute/relative path. Missing sibling `.outline.md` warns on stderr but does
not hard-fail. When the outline exists, the engine compares `## ID inventory`
ids to HTML `id="..."` attributes and warns (non-fatal) for inventory ids
missing from HTML; HTML-only decorative ids are ignored. Unless `--no-open`,
the engine starts the OS opener (`open`, `xdg-open`, or Windows `start`) for
the local file only — never a network fetch. Stdout prints the absolute HTML
path, outline path tip, playbook index hint, and an `ids=ok` / `ids=mismatch`
summary when an inventory is present.

## Overhaul tools

`overhaul` serves the explicit-only `/overhaul` skill. Each tool takes explicit
paths and never resolves a `.devrites` workspace, calls a model or uses the
network. The skill documents the record schemas and when to call each tool.

| Tool | Forms | Exit codes |
| --- | --- | --- |
| `records` | `init <repo> <run-id>`, `digest <file>`, `stage <run>`, `publish <run>`, `validate <run>`, `prune <run> [--keep K]` | 0 ok; 1 violations (listed as `VIOLATION:` lines); 2 usage or I/O |
| `admit` | `receipt <run> <receipt.json> [--observed <file>]`, `anchor <tree> <proposals.json>` | 0 admissible; 1 rejected; 2 usage or I/O; 3 duplicate or late receipt |
| `snapshot` | `capture <repo> <out>`, `fingerprint <repo>`, `verify <repo> <out> [--agent-paths <file>]`, `state <repo> <out.json>`, `delta <before.json> <repo>` | 0 ok; 1 index changed or changes outside agent paths; 2 refusal, usage or I/O |
| `score` | `--rubric <r> --results <s> --gates <g> [--out <file>]`, `compare --rubric <r> --baseline <a> --candidate <b>` | 0 computed; 2 invalid input (nothing counts as scored) |
| `bench` | `<result.json>` | 0 computed; 2 invalid input |
| `render` | `<run> <staged-generation> <review\|report>` | 0 written; 2 usage, invalid records or I/O |

`snapshot` runs git read-only with `GIT_OPTIONAL_LOCKS=0`,
`diff.autoRefreshIndex=false` and no external diff drivers, so `.git/index`
stays byte-identical. Secret-pattern files and files holding private key material
are hashed only: never copied into the snapshot tree and never included in its patches.
Clean tracked files are hashed but not copied, because git already holds them;
only dirty and untracked files are copied. A snapshot may live outside the
repository or inside its `.devrites/overhaul/` run area, which every snapshot
command leaves out of the repository state. `records init` creates
`<repo>/.devrites/overhaul/<run-id>/` (mode 0700) and, when git does not already
ignore it, adds `/.devrites/overhaul/` to the local `.git/info/exclude`, never to
the shared `.gitignore`.
`admit receipt` checks the attempt in the open staged generation (`g<N>.tmp` whose
`.base` is `CURRENT`) when one exists, otherwise in `CURRENT`, so a phase can record
many dispatches and admissions before one publish. The receipt must be the file at
`<run>/receipts/<run_id>__<task_id>__<attempt_id>__<role>.json` (and match the attempt's
recorded `receipt` path); extra or misspelled arguments exit `2`.

## Output and exit contracts

`check candidate` passes with exactly:

```text
candidate-sha256: <64 lowercase hex>
candidate-files: <manifest row count>
```

Invalid usage/root selection exits `2`; a candidate validation block prints
`candidate: BLOCKED: <reason>` and exits `3`.

`check readiness --emit-binding <slug>` passes with exactly:

```text
Readiness inputs SHA-256: <64 lowercase hex>
```

It binds the fixed records documented in the
[workspace schema](workspace-schema.md#build-readiness-binding), not mtimes or
ambient Git state. Ordinary readiness and Seal require that exact standalone
line in `eng-review.md`; stale input returns
`reason: DRV-GATE-READINESS-STALE` and routes through `/rite-vet`.

### Workspace observation diagnostics

Lifecycle checks acquire the fixed workspace Markdown inventory once. Each
artifact is classified as `absent`, `empty`, `malformed`, `unsafe`,
`unreadable`, or `present`. Retained content is limited to 1 MiB per file and
8 MiB aggregate. Diagnostic lines use this exact shape:
`artifact: <logical-path>: <state> (<code>)`.

The closed diagnostic codes and recoveries are:

| Code | Exact Gate recovery | Exact standalone readiness-binding payload |
| --- | --- | --- |
| `malformed_markdown` | `next: repair <logical-path>: replace invalid Markdown with valid Markdown; required artifacts need substantive content` | `readiness input <logical-path> is malformed (malformed_markdown); replace invalid Markdown with valid Markdown` |
| `parent_symlink` | `next: repair <logical-path>: replace the symlinked parent with a real directory` | `readiness input <logical-path> is unsafe (parent_symlink); replace the symlinked parent with a real directory` |
| `final_symlink` | `next: repair <logical-path>: replace the symlink with a regular file` | `readiness input <logical-path> is unsafe (final_symlink); replace the symlink with a regular file` |
| `non_regular` | `next: repair <logical-path>: replace the non-regular entry with a regular file` | `readiness input <logical-path> is unsafe (non_regular); replace the non-regular entry with a regular file` |
| `file_too_large` | `next: repair <logical-path>: reduce the file to at most 1 MiB` | `readiness input <logical-path> is unsafe (file_too_large); reduce the file to at most 1 MiB` |
| `permission_denied` | `next: repair <logical-path>: grant read permission` | `readiness input <logical-path> is unreadable (permission_denied); grant read permission` |
| `read_failure` | `next: repair <logical-path>: restore a readable regular file` | `readiness input <logical-path> is unreadable (read_failure); restore a readable regular file` |

The Gate recovery column remains exact for target-policy-required artifacts. For
a selected optional readiness input, the same code-specific repair appends
`; optional readiness input may instead be removed` and does not call the input
required.

These seven codes are the closed Workspace Observation classification and
recovery mapping outcomes. A selected public consumer emits only a code
reachable for its consumed fixed logical path. Invalid workspace ancestry is
`workspace_invalid`, not an artifact `parent_symlink` diagnostic.

Status emits diagnostics without recovery or `next:` lines, after section rows
and before `result`. Gate emits diagnostics after `reason` and before recovery,
`invariant`, and `retry` lines. Generic add-content recovery applies only to
absent or empty target-required artifacts. Standalone readiness-binding
failures use the existing `readiness-binding: BLOCKED:` prefix and the logical
readiness-input state/code plus recovery; they never disclose physical paths or
content.

Whole observation failures are `workspace_invalid`, `aggregate_too_large`, and
`concurrent_change`. Their disclosure-safe payloads are exact:

- `workspace observation: workspace_invalid: workspace is unavailable; verify the selected logical workspace and canonical workspace override, then retry`
- `workspace observation: aggregate_too_large: retained content exceeds the 8 MiB aggregate limit; reduce retained Markdown below 8 MiB, then retry`
- `workspace observation: concurrent_change: workspace changed during acquisition; retry`

An absent or empty `state.md` appends `add real content to state.md and retry` to
the existing logical error. A malformed, unsafe, or unreadable `state.md`
appends `repair state.md and retry`. A ledger without a phase appends `record
phase in state.md and retry`; an unknown phase appends `record a known phase in
state.md and retry`.

Whole observation failures use stderr, exit `2`, and no lifecycle result or
reason on stdout. Standalone readiness-binding failures use one stderr line,
exit `3`, and empty stdout. Per-artifact lifecycle blocks keep existing reason
IDs and stdout exit `3`; successful checks keep stdout exit `0`. Seal evidence
freshness still runs separately after a successful Seal gate.

- `0`: passed or completed.
- `2`: common invalid request or unreadable-state result.
- `3`: common deterministic lifecycle or safety block.
- Atomic state operations retain their documented operation-specific nonzero
  results.

Lifecycle checks emit stable line-oriented fields, including a `reason: DRV-...`
identifier for the deterministic outcome. The native `/rite-doctor` workflow
emits its own human-readable OK/WARN/FAIL report. Neither surface introduces an
agent API or versioned wrapper.

Strict mutators resolve the physical root once and refuse unsafe symlinks,
nested-repository inheritance, ambiguity, or escapes. Repository source
validation belongs to `scripts/validate.sh` and CI, not an installed engine.

Production Go and shell Git callers remove environment variables that can
retarget the repository, worktree, index, objects, refs, config, or pathspec,
while retaining unrelated Git variables. This isolation is shared caller
policy, not another public command.
