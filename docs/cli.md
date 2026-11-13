# `devrites-engine`: deterministic workspace core

DevRites keeps durable workflow truth in `.devrites/` Markdown. Claude Code and
Codex interpret the workflow, run exact custom agents, reconcile their results,
and make semantic judgments. The Go engine exposes only deterministic,
cross-host primitives; it never dispatches an agent or grades reviewer prose.

## Command surface

`devrites-engine help` lists the exhaustive operational surface. Standard
`help`, `-h`, `--help`, `version`, and `--version` forms also remain supported.
Each operational command and subcommand accepts `-h` / `--help` and prints that
command's usage to stdout (exit 0) without requiring a workspace. Missing
required arguments still print usage to stderr and exit 2, except `state close`,
which exits 4 (see Output and exits).

```text
devrites-engine install [flags]
devrites-engine update [flags]
devrites-engine uninstall [flags]

devrites-engine check candidate <slug>
devrites-engine check readiness <slug>
devrites-engine check readiness --emit-binding <slug>
devrites-engine check preconditions <slug> [--slice SLICE-###] [--timeout N]
devrites-engine check seal <slug>
devrites-engine check slice <slug> <SLICE-ID>
devrites-engine check diff-scope <slug> --allow <csv>|--allow-file <path> [--worktree|--staged|--base <ref>] [--snapshot|--baseline] [--verbose]
devrites-engine check perturb <slug> --patch <file> --expect <test-name> --gate <G-ID> [--timeout N] [--worktree <path> --slice <SLICE-ID>]
devrites-engine check task-graph <slug>
devrites-engine check regression <slug> [--update]
devrites-engine check continuation [slug]
devrites-engine check drift <slug> [--record]
devrites-engine check windows <slug> [--worktree|--staged|--base <ref>|--since-dispatch <wave>]
devrites-engine check dup [slug] [--all|--worktree|--staged|--base <ref>]
devrites-engine check anchors <slug> <return-file> [--skills-root <dir>]
devrites-engine check skill-trust <path>
devrites-engine check path-disjoint [<json-file>|-]
devrites-engine check indexes [--root <dir>]
devrites-engine detect commands [--root <dir>] [--json]

devrites-engine observe summary <slug>
devrites-engine orient <slug>
devrites-engine observe slice <slug> <SLICE-ID>
devrites-engine next [slug]
devrites-engine frontier [slug]
devrites-engine handoff [slug] [--resume]
devrites-engine recall --paths <p1,p2,...> [--limit n]
devrites-engine context [slug] (--phase <p>|--skill <name>) [--role <r>] [--whole]
devrites-engine metrics record <slug> --phase <p> --event <e>
devrites-engine metrics summary [slug]
devrites-engine dispatch <slug> <open|start|seal|return|status|abandon|attribute>
devrites-engine candidate freeze <slug> --wave <w> (--slice <SLICE-ID>|--paths <path>...) [--base <ref>] [--worktree <path>]
devrites-engine packet <slug> --wave <w> --phase <p> --role <r> [--slice <SLICE-ID>] [--recheck] [--checks <G-ID,...>] [--focus <file>]
devrites-engine parallel <subcommand>
devrites-engine gates <subcommand> <slug>
devrites-engine claim <add|release|list|check>
devrites-engine note <add|list|check|rm> <slug>

devrites-engine state resolve [--human] <qid> "<answer>"
devrites-engine state merge-manifest <slug> [pred...]
devrites-engine state close <slug>
devrites-engine migrate <slug> [--dry-run] [--answer id=choice]

devrites-engine secret-scan [--staged] [--stdin] [slug]
devrites-engine open-visual <path-or-name> [--slug <slug>] [--no-open]
devrites-engine overhaul <tool> ...
devrites-engine version
```

`overhaul <tool>` provides the run-record, admission, snapshot, scoring,
benchmark, and view tools used by the `/overhaul` skill; run
`devrites-engine overhaul --help` for its tools.

Commands outside this operational list and the standard help/version forms are
unsupported. There are no legacy engine aliases, tombstones, agent-protocol
versions, semantic-readiness digests, or compatibility telemetry.
`check candidate` is additive; no existing engine command or public `/rite-*`
workflow was removed or renamed.

## Updates

Run `devrites-engine update` in an installed project to resolve the latest
stable release and update the project pack plus shared engine binary. The
command downloads the release bundle and platform engine, verifies both exact
filenames against their SHA-256 sidecars, then hands the extracted candidate to
the downloaded engine. This avoids asking an old engine to validate a newer payload schema.

`devrites-engine update --check` resolves and compares release metadata without
downloading assets. `--source-dir` and `--payload-dir` remain the advanced local
update path for npm, shell, source-checkout, and test callers; a local `--check`
needs only `--source-dir`.

Release metadata requests to the exact HTTPS `api.github.com` origin use the
first available credential: `GH_TOKEN`, then `GITHUB_TOKEN`, then optional
`gh auth token --hostname github.com`. The helper is noninteractive, bounded to
three seconds and 4 KiB of output; missing, failed, timed-out, or oversized helper
output does not prevent an unauthenticated request. GitHub CLI is not required.
An explicitly supplied invalid credential fails; the updater does not silently
retry under another identity. Credentials are never sent to release assets or
off-origin redirects. Same-origin GitHub API redirects may retain them.

On a headless machine, run `gh auth login --hostname github.com --web`, then use
the displayed device code in a browser on another device. Do not put credential
values in commands, examples, logs, or support reports. `--force` permits
managed-file replacement, not authentication or rate-limit bypass. Authenticated
requests can still be rate-limited; wait before retrying when necessary.

## npm adapter

The `npx devrites` adapter accepts `add` as an alias for `install`, `upgrade` for
`update`, and `remove` for `uninstall`. These are adapter-only conveniences, not
`devrites-engine` aliases; documentation and automation should prefer the
canonical command names.

Use `npx devrites update --check` to perform the equivalent adapter-level
comparison without changing the installation. Use
`npx devrites uninstall --keep-binary` (or `--no-binary`) to remove managed host
artifacts while retaining the shared `devrites-engine` binary. `update` reuses
the install flags recorded in the manifest; host and binary flags passed to
`update` override them and are recorded for the next run.

If an older direct engine reports `missing codex/hooks.json` or asks for
`--source-dir`, run `npx devrites@latest update`, or use a release `install.sh` that is
checksum-verified (see the verified Bash bootstrap under
[README Install, update, and remove](../README.md#install-update-and-remove))
and run `bash ./install.sh update` once to install a release with the
self-contained updater.

## Checks

- `check candidate <slug>` validates the strict `touched-files.md` manifest and
  hashes its exact path/state/type/mode/content identity. A pass prints:

  ```text
  candidate-sha256: <64 lowercase hex>
  candidate-files: <manifest row count>
  ```

- `check readiness <slug>` verifies the files required to leave the workspace's
  current phase, the `tasks.md` slice graph once that artifact is required, and,
  once `eng-review.md` is required, its exact stable Build-input binding. Leaving
  Vet, it also runs the Build-entry `check slice` contract on every slice
  `state.md` does not check built, except a deferred `Satisfies: none` slice.
- `check readiness --emit-binding <slug>` renders the exact stable Build-input
  binding for Vet to record after semantic review.
- `check preconditions <slug>` runs the preflight rows whose `Run at` is `vet`
  (`--slice <SLICE>` runs that slice's `pre-dispatch` rows, keeping one that
  already passed) and records a candidate-bound transcript per row; exit 3 on
  failure.
- `check seal <slug>` checks final required files, open human gates, and the
  `tasks.md` slice graph. Once those files are complete, it verifies the stable
  readiness binding; only after that aggregate check passes does it verify that
  `evidence.md`, `review.md`, `seal.md`, and optional `browser-evidence.md`
  contain exactly one binding to the current candidate digest.
- `check regression <slug>` compares the workspace's structural progress facts —
  phase ordinal, checked `AC-###` boxes, met and abandoned ledger gates,
  done-state slices, resolved questions, and present required artifacts —
  against the durable `regression-baseline.json` high-water mark. A lost fact
  prints `REGRESSION: <detail>` and exits `3`. `--update` rewrites the baseline
  under the feature lock at a durable checkpoint and prints each regression it
  blesses as `REGRESSION-BLESSED`. A deliberate owner-authorized repair checkpoint
  may accept a backward phase along with other acknowledged facts: the current
  phase becomes the new baseline. Ordinary checks never bless losses or change
  the baseline. If the update would remove every checked acceptance criterion and
  met gate, `--allow-empty` is also required. With no baseline recorded the check
  reports `unproven` and exits `0`. Recommended placement: compare on a cold
  resume or scheduled wake before acting on durable state, ratchet after a landed
  slice checkpoint or phase advance, and treat `BLOCKED` at Seal as NO-GO.
  Never use automatic updates to hide an unreconciled regression.
- `check continuation [slug]` prints `continuation: required` and
  `invoke: <next_action>` when `.devrites/AFK` is a regular file, `status` is
  `running` (or specifically `phase=vet,status=ready`), no question cursor is
  set, no blocking or escalating question is open, and `next_action` is a valid
  internal phase command, not `/rite-ship`. Other ready statuses do not inherit
  this exception. It also reports the sequence frontier: when the local cursor
  is clear but related work needs native preparation, it can print
  `continuation: required` with `invoke: /rite-autocomplete` and the original
  source identity. Waiting, human-blocked, or completed work alone does not
  require continuation. Neither result authorizes Ship, overrides a human gate,
  or declares an unvetted workspace Build-ready. Successful verdicts exit `0`;
  malformed or ambiguous state fails visibly. `handoff` prints the same lines.
- `check drift <slug>` attributes readiness-input drift to the exact artifact.
  `--record` snapshots per-input SHA-256 digests into `readiness-inputs.json`
  under the feature lock after Vet records the aggregate binding; a later
  compare prints `DRIFT: <path> <changed|missing|added>` per differing input.
  Without a baseline it falls back to the `eng-review.md` aggregate binding and
  lists inputs modified after that file as candidates. Advisory: drift is
  reported, never blocked (a change is legal as a dated delta through the
  owning rite); a corrupt baseline exits `3`.
- `detect commands [--root <dir>] [--json]` resolves the repository's own
  command wiring — Makefile targets first, then `package.json` scripts, then
  language-manifest fallbacks — and the lockfile-derived package manager.
  Read-only: nothing is installed or executed. Skills use it to cite real
  commands in `gates.md` `CHECK` lines and proof steps instead of invented
  ones; unresolved slots print `unresolved` rather than guessing.
- `check anchors <slug> <return-file> [--wave <w> --role <r>] [--worktree <path>]`
  anchors each Critical/Important row's source quote against the native review
  account's frozen candidate, with control metadata kept separate from worker
  source. Linked workers require explicit `--worktree`; missing, ambiguous, stale,
  or mismatched bindings exit `3`. Original returns can match stored accounts by
  exact bytes; unstored returns require `--wave` and `--role`. Unverified or
  Deferred findings and Counts exceeding parsed rows also exit `3`.
  See [invocation and integrity contract](engine/commands.md#linked-worktree-review-anchors).
- `check dup [slug] [--all|--worktree|--staged|--base <ref>]` reports advisory
  near-duplicate code clusters that survive comment stripping, literal
  collapsing, and identifier renaming. Diff modes mark units overlapping
  changed hunks with `*`; stable content-derived cluster hashes let
  `.devrites/dup-ignore` dismissals survive line moves but resurface when the
  code's structure changes. Staged mode compares index blobs and excludes
  untracked files. Successful scans exit `0`; usage errors and unavailable or
  partial scans exit `2`. Empty eligible input and generated exclusions are
  explicit. This bounded heuristic produces review leads, not exhaustive proof. See `standards/duplicate-code.md` for the triage contract.
- `claim <add|release|list|check>` manages advisory session-scoped file claims
  in the append-only `.devrites/claims.jsonl` ledger. `add --session <id>
  [--ttl <min>] [--reason <s>] <path>...` records write intent for a path set
  and exits `3` naming the holder when a live claim by another session covers a
  path; `release --session <id> --id <claim>` frees it; `list [--all]` shows
  live (or all) claims; `check [--session <id>] <path>...` is the read-only
  preflight. Conflict checks cover logical paths and physical symlink aliases;
  unresolved physical paths fail closed. Same-session allocation stays exempt.
  Each claim records its host and the claiming session's long-lived
  processes; when all of them have exited on this host, `list` shows the claim
  `dead` and it no longer conflicts. Other hosts or an unavailable `ps` never
  prove exit. Claims coordinate concurrent same-tree sessions; they never
  replace the one-writer-per-worktree rule in `standards/agents.md`.
- `note <add|list|check|rm>` manages `notes.md` anchored notes: `add <slug>
  <subject> <quote> <title> [body]` binds a note to a verbatim quote in one
  repository file; `list`/`check` regrade each anchor `exact`/`moved`/`stale`/
  `ambiguous`/`lost` by relocating the quote repo-wide (dependency and workspace
  trees excluded); `check --repair` rewrites a `moved` subject; `rm` deletes by
  `NOTE-###`. `check seal` refuses a malformed file or any non-`exact` anchor.
- `recall --paths <p1,p2,...> [--limit n]` points planning and review at what
  earlier archived features decided or found about the same files. It lists
  each `.devrites/archive/<slug>/` whose candidate manifest covers a queried
  path (exact or directory prefix, case-folded), ranked by overlap, then newest
  archive-directory mtime, then slug, with up to 8 pointers per feature:
  `decisions.md` `DEC-###` entries and carry-forward constraints whose `Files:`
  overlap, and `review.md`/`seal.md`/`drift.md` lines naming a queried path.
  It is computed on every read and never writes, caches, or indexes. Output:

  ```text
  recall: 1 archived feature(s) touched these paths
  feature add-csv-export overlap=1 paths=src/routes
    archive/add-csv-export/decisions.md:5  ## DEC-001 — Reuse the route and stream CSV
    archive/add-csv-export/decisions.md:11  - Close the stream on every error path — from SLICE-002 (EVID-003); Files: src/routes/transactions/export.ts
    archive/add-csv-export/review.md:3  - `src/routes/transactions/export.ts:41` closes the stream on error
  ```

  `recall: no archive` and `recall: no archived feature touched these paths`
  exit `0`; an unreadable archive file prints `unreadable: <file> (<reason>)`
  and the scan continues; usage errors exit `2`.

Semantic readiness, traceability, acceptance interpretation, evidence quality,
doubt, test quality, reviewer reconciliation, and capability interpretation
belong to the active skill and exact native agents. Normative spec grammar is
checked by the root's explicit native re-read checklist. Repository build,
test, lint, typecheck, schema, and release commands belong to that repository
or CI.

## Sequence frontier

`devrites-engine frontier [slug]` is a read-only structural report across the
source's recorded sequence. An explicit slug may name a live or archived
workspace. Without one, the command uses the selected source; if `ACTIVE` is
absent, recovery requires a unique archived terminal sequence tip with an exact
recorded native `/rite-spec <next>` edge. Multiple possible sources fail as ambiguous;
folder names and numeric suffixes are not authority.

The report emits:

- `frontier: actionable|waiting|blocked|clear`, or `frontier: none` for a
  non-sequence source.
- `frontier-source: <slug>` and `frontier-workspace:` rows naming each related
  workspace's `location=live|archive|missing`, `action=inspect|prepare|repair|wait|complete`,
  and reason. Archived intermediaries do not hide live descendants, and archived
  workspaces never become writer candidates.
  Running or execution-unknown dispatch roles wait; completed writer returns
  without durable acceptance require native inspection, not another writer launch.
- `frontier-select-command:` with an explicit `parallel select --from-plan
  --workspaces ...` command when a live controller is available. Selection still
  enforces current readiness, recorded dependencies and independence, retained
  leases, path conflicts, and the repository-wide capacity. `frontier-picks:`
  names eligible source-qualified slices; `frontier-selection: blocked` names a
  selection refusal. These are observations, not writer launches.
  The explicit pool excludes retained dispatch and human/question/scope-held
  sources; their workspace rows remain visible. `--workspaces` is the exact
  candidate list; `--slug` identifies the controller, not an implicit candidate.
  Legacy per-source slice counters never reduce the global writer cap or exclude
  independent children; recorded workspace-opening limits still apply.
- `frontier-prepare:` for a recorded next Spec cursor or an unambiguous next entry
  in an exact position/workspace table, or `frontier-review-sequence:` artifact
  paths when remaining work or conflicting sequence maps require native review.
- `frontier-authority: <artifact-path>,...` identifies actual files supplying a
  valid nonempty scheduling map. For explicit canonical authority it names the
  owning `roadmap.md` and same-owner `decisions.md`; legacy maps list their actual
  artifacts without claiming canonical status.

Relations come from exact recorded `sequence_parent` edges, not name prefixes.
Conflicting identities, cycles, malformed related state, or invalid declared
sequence positions are refused. Missing or stale planning is a repair action,
not permission to build. Continuation flags and remaining counts constrain new
workspace preparation without silently discarding existing children.
Conflicting or amended sequence maps suppress missing-workspace preparation even
with a recorded Spec cursor until native authority normalization resolves them;
valid existing live descendants remain visible and eligible for guarded selection.

An optional `roadmap.md` becomes current scheduling authority when it contains
either `Sequence root:` or `Decision:`. Both headers are then required:

```markdown
Sequence root: actual-anchor-slug
Decision: DEC-017

| Position | Workspace |
| --- | --- |
| 1 | actual-anchor-slug |
| 2 | actual-renamed-child |
```

Exactly one related live or archived workspace may own this explicit map.
`Decision: DEC-NNN` must identify exactly one `## DEC-NNN` heading in that
owner's `decisions.md`; the root must match the actual chain anchor. Exactly
one table must have contiguous positions from 1 and unique valid workspace
slugs, matching every related recorded position and exact successor. Folder
numbering, prose amendments, and unenrolled workspaces confer no authority.
A valid explicit map supersedes historical tables in `decisions.md`, `spec.md`,
and headerless `roadmap.md` for scheduling only; preserve those historical bytes.
Without either header, `roadmap.md` retains conservative legacy parsing.
Partial/malformed headers, multiple owners, invalid root/decision/table, or a
symlink/unreadable explicit map fail closed without legacy fallback. Use native
authority normalization to record the approved actual identities while retaining
history. Structural validation establishes identity, not semantic approval,
independence, Build readiness, or Ship permission.

Under AFK, `next`, `handoff`, and `check continuation` expose the same frontier.
A source-local `parallel select --from-plan` without `--workspaces` also prints
the report to stderr, preserving its machine-readable selection on stdout and
source-local picks: reporting another workspace never silently admits it.
Native Autocomplete must inspect returned preparation/review actions and read
the exact `frontier-authority:` paths plus recorded restrictions before invoking
Spec/Plan/Vet or mixed selection. This read is required even when the owner is an
archived ancestor or another related workspace outside the local read-next bundle.
Path disjointness alone establishes neither semantic independence nor permission
to lift a planning-only restriction.

### Autocomplete controller admission

The existing `/rite-autocomplete` invocation writes or upserts `mode: autocomplete`
in `.devrites/AFK`; no extra flag is needed. It preserves other sentinel fields
unless an explicit `--parallel N` changes capacity. Mode identifies the native
controller, not authorization. Standalone AFK without mode retains its behavior;
duplicate or invalid mode requires normalization rather than silently disabling
the controller gate.

With controller mode and global capacity above one, a new Build wright
`dispatch open` checks structural global scheduling before writing claims, history,
or a round. It checks one immediate recorded successor per admitted source, not
the whole future sequence:

- Missing permitted successor: start its native Spec/Plan/Vet preparation and
  retain actual launched planning/review handles before admitting the writer.
- Existing successor: require pending preparation handles, a current native
  Build-ready binding, or its concrete human/dependency hold. A placeholder
  state/spec or a prose claim that preparation happened is insufficient.
- Opted-out or exhausted future opening and source human/question holds do not
  invent new preparation obligations. Unresolved sequence authority requires
  native normalization.

For a concrete preparation constraint, retain
`frontier-blocked: <successor-slug> — <specific constraint>` in the source's
`evidence.md`. For unresolved historical normalization on otherwise-valid local
work use `frontier-blocked: sequence — <specific constraint>`. These deferrals
remain visible for native review; they grant no approval, readiness, or claim
that normalization succeeded. Neither waives malformed canonical authority,
invalid AFK/path/readiness/security checks, or already-ready foreign Build.
A generic local `serial: SLICE-002 — <reason>` does not waive cross-workspace
preparation. A discharged deferral does not repeatedly trigger controller work.

Ready independent foreign sources must participate in explicit repository-wide
mixed selection and lease capacity/path reservations. Reserve the cohort with
`parallel create/add`; source waves may then open separately. A concrete
`serial: other-workspace/SLICE-001 — <constraint>` can retain an ordered handoff
for that exact foreign pick; an unqualified local serial reason cannot silently
omit it. This does not waive successor preparation or readiness, dependencies,
runtime/path isolation, actor claims, or reviewer/proof windows.
Dispatch `controller_batch` is an opaque cohort-instance admission identity,
recording initial whole-wave, source-qualified live-lease admission, never proof
of actor checkout, cwd, or runtime profile. Immutable `admitted_paths` snapshots
retain each pending writer lane's normalized allowlist. Missing snapshots or
changed reservations require actor reconciliation; mutable tasks cannot backfill
authority. Returned/integrated siblings may release slots for rolling refill.
Exact claim identity/session remains required; release or expiry does not stop
an actor. Retained unbound or other-cohort writers hold new actors; later lease
IDs cannot retrofit them. Native exact cwd/profile checks remain required.
Unsupported isolation permits truthful local-owner serial work and normal
ordered handoff, never an early same-tree foreign writer.

Controller continuation prioritizes actionable global normalization, preparation,
and foreign work over a direct local Build invocation while preserving the local
phase cursor. Local-only ready work does not recursively invoke Autocomplete.
After a committed dispatch return, a fresh global scheduling result resurfaces
the next obligation—even if later claim release fails; its nonzero failure and
claim exclusion remain. Rejected or unpersisted returns do not refresh.
Ordinary status reads remain read-only. Protected reviewer/proof windows defer
unsafe landing, not independent planning. No such transition invokes Ship or
launches a model through the engine.

The engine does not invoke models, create workspaces, change `ACTIVE`, resolve
semantic authority, or grant Ship approval. These outputs guide the native host;
they cannot guarantee that an arbitrary host follows the workflow.

## Acceptance ledger

`gates` operates the `gates.md` ledger inside a feature workspace: one gate per
required outcome, runnable (`CHECK`+`EXPECT`, optional `CWD`) or manual
(`EVIDENCE` only). Subcommands: `scaffold`, `status`, `run`, `reverify`,
`lint`, `precheck --worktree <path> [--slice <SLICE-ID>]` (the wright's pre-return run of `RUN: writer`
gates and `check windows`; `--slice` skips a gate only other unbuilt slices own; writes no ledger, EVIDENCE line or transcript, though `check windows` still logs its metric; never evidence), `attest <id> <note>`,
`abandon <id> <why>`.

A runnable gate executes only when its `(CHECK, CWD)` pair exactly matches a
`test-plan.md` `## Build-entry preflight` row. Passing writes
`automatic-evidence` bound to the gate definition digest; editing the gate
marks prior evidence stale. `status` prints a stable reduction ending in
`result: all-met|not-met|handoff|malformed`; exit `3` unless every gate is met
and none is abandoned. `check readiness`/`check seal` enforce the same
reduction whenever `gates.md` is a required artifact, so a partially compliant
ledger blocks the phase exit.

## Atomic state operations

`state resolve` also supports `--drop <qid> ["<reason>"]` and `--batch <file>`.
A leading `--human` (for the answer, `--drop` and `--batch` forms:
`state resolve --human --drop <qid>`, `state resolve --human --batch <file>`) also writes `answered_by: human`;
without it any `answered_by:` line in the resolved block, whatever its indentation or case, is removed.
`state close` transactionally archives a shipped workspace and clears matching
`ACTIVE`.

Mutations share root-safety checks, feature locks, and atomic per-file writes.
`state resolve` and `state apply-bundle` restore attempted files after returned
publication errors; `rollback incomplete` identifies files needing inspection
and recovery. These multi-file operations are not crash-atomic: process
termination can leave partial updates, and unlocked readers can see intermediate
state. Inspect both artifacts before resuming after an interrupted publication.

The root scans and rechecks `questions.md` to allocate the next unused qid,
manages Clarify return fields without rewriting unrelated Markdown, charges an
AFK budget once per green built slice, and counts no more than three recovery
failures per causal fingerprint from context plus Dead ends/evidence. `/rite-doctor`
is a read-only native inspection. None has an engine command or replacement
script.

## Secret scanning

Use `secret-scan --staged` for exact Git index blobs and `secret-scan --stdin`
for review text supplied through a non-logging process-stdin channel. The flags
may be combined. The scanner does not follow worktree symlinks or print matched
secret bytes. It accepts at most 4,096 entries, 64 MiB total captured input, and
4,096 findings; input, limit, and output failures close the scan.

Findings include only severity, a redacted or escaped source label, category,
and zero-based byte offset. HIGH findings exit `3`.

## Open visual

`open-visual` resolves a portable HTML visualization under the active or
`--slug` workspace `visual/` directory, or an absolute/relative `.html` path.
Unless `--no-open` is set, it launches the OS default browser for that local
file only. A missing sibling `.outline.md` prints a stderr warning and still
succeeds. When the outline exists, inventory ids missing from HTML also warn
(non-fatal; HTML-only decorative ids are ignored). Stdout prints compact agent
tips: absolute HTML path, outline path (or missing), the playbook index hint,
and `ids=ok` / `ids=mismatch` when an inventory is present. The command never
fetches remote hosts.

## Output and exits

Lifecycle checks print stable line-oriented fields. `reason: DRV-...` identifies
the deterministic gate outcome without an agent protocol or versioned wrapper.
Skills read workspace Markdown directly.

`check candidate` uses the two exact fields shown above. Invalid usage or root
selection exits `2`; a malformed/unsafe manifest or candidate mismatch prints
`candidate: BLOCKED: <reason>` to stderr and exits `3`.

- `0`: passed or completed.
- `1`: runtime or I/O failure.
- `2`: common invalid request or unreadable-state result.
- `3`: common deterministic lifecycle or safety block.
- Atomic state operations retain their documented operation-specific nonzero
  results.
- `state close` exits `4` for usage or a missing workspace.
- `state close` exits `3` for a schema refusal or a retained controller/source lease.
- `state close` exits `5` when an archive already exists.
- `state close` exits `1` for an invalid workspace, archive directory, or ACTIVE cursor.

## Root safety

`DEVRITES_ROOT` selects a project root or `.devrites/` directory.
`DEVRITES_WORKSPACE` may select one explicit contained workspace. Mutating
commands refuse ambiguous, escaped, symlinked, or otherwise unsafe roots.
Explicit producer dependencies resolve safely within the selected `DEVRITES_ROOT`
without changing or redirecting the consumer's workspace override.
Global controller queries use explicit source slugs in a per-invocation environment
without a worker's `DEVRITES_WORKSPACE`; conflicting overrides fail closed rather
than redirecting another workspace's reads. Workspace parallelism keeps one
controller lease and global writer capacity; it does not broaden Ship approval.

Install, update application, and uninstall remain manifest-owned local
operations. Direct engine update, shell, and npm may acquire a release candidate;
all remote acquisition uses exact SemVer, HTTPS-only redirect hops, mandatory
exact-filename SHA-256 sidecars, private temporary
directories, and byte/archive bounds, with no unchecked raw/source/default-branch
fallback. Update accepts
`--check` to compare the installed manifest version with the latest stable
release without downloading assets. Remote-selector flags such as `--to` and
`--pre` are unsupported.
`/rite-upgrade` is a native compatibility audit, not an engine migration. Its
read-only planner must cite a current rule and exact workspace defect; admitted
repairs run through the current Clarify, Plan, Converge, Vet, Prove, Polish,
Review, or Seal owner. Candidate repairs rerun current proof and never infer a
historical pass.

Production Go and shell Git sites remove repository/config/object/ref/pathspec
retargeting `GIT_*` variables before execution while preserving unrelated Git
environment. This targeting isolation is a caller boundary, not another engine
command.
