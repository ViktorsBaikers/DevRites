# DevRites architecture

DevRites combines project-local Claude Code and Codex skills with one Go control
plane. It gives AI coding agents a defined engineering process: frame → spec →
mandatory adaptive clarify → optional temper → define → mandatory vet → build
bounded verified slices → prove with evidence → polish → review → seal → ship.
`converge` is the recovery state for bringing live code and recorded intent
back into agreement.
`frame` is a machine state/checkpoint and an optional, non-gating preflight
lens; Spec begins the required feature-definition path.

For the `.devrites/` load order, file budgets, artifact schema, supported
cursor readers, traceability rules, and phase-relative completeness model, see
[`engine/workspace-schema.md`](engine/workspace-schema.md).

## Layers

1. **Public lifecycle and workspace skills**: `.claude/skills/rite-*`,
   `user-invocable: true`. Each owns one bounded phase or workspace transition.
   Sequence: `rite-spec`, mandatory adaptive `rite-clarify`, optional `rite-temper`, `rite-define`, mandatory
   `rite-vet`, `rite-build`, recovery `rite-converge`, `rite-prove`,
   `rite-polish`, `rite-review`, `rite-seal`, and `rite-ship`. `rite-plan`
   repairs or reslices an active plan. The resume verb `rite-resolve` answers a
   HITL gate and clears `Awaiting human`. `rite-upgrade` is a conditional
   evidence-gated compatibility route for older active workspaces, not a phase.
   The thin `/rite` menu and read-only `rite-status` live in the public
   utility layer below.
   `/rite-seal` **decides** GO/NO-GO and writes the verdict to `seal.md`;
   `/rite-ship` is the final core lifecycle rite that **executes** the
   irreversible git ladder and **closes** the task by archiving the workspace
   and clearing `ACTIVE`. Separate steps let users audit the release decision
   before any irreversible action.
2. **Public utility and on-ramp skills**: `rite-adopt`, `rite-quick`,
   `rite-frame`, `rite-status`, `rite-doctor`, `rite-upgrade`, `rite-learn`, `rite-explain`,
   `rite-customize`, `rite-zoom-out`, `rite-prototype`, `rite-handoff`,
   `rite-pressure-test`, `rite-pov`, `rite-dogfood`, `rite-pr-feedback`, and
   `rite-autocomplete`. These are public commands. `rite-autocomplete` is the
   unattended orchestrator. By default it drives the reversible lifecycle
   through Seal GO and stops. Explicit `--ship` enters Ship preflight but still
   waits for a fresh literal `GO` and native approval before mutation. It
   chooses the recommended option at each soft gate and pauses for hard
   irreversible-risk, blocking, or escalating gates, or a NO-GO. The
   `devrites-` prefix prevents collisions with bundled Claude Code skill names
   such as `prototype`, `handoff`, `triage`, and `diagnose`; it does not mark
   visibility. `rite-pressure-test` needs no prefix because it does not collide.
3. **Internal specialist skills**: `.claude/skills/devrites-*` with
   `user-invocable: false`: `devrites-interview`, `-source-driven`,
   `-doubt`, `-ux-shape` (plans UX/UI into `design-brief.md` at `/rite-spec`),
   `-frontend-craft`, `-browser-proof`, `-debug-recovery`,
   `-api-interface`, `-audit` (dispatches the security / perf / simplify
   fresh-context reviewer on an axis argument), and `-prose-craft`. Public skills invoke these
   specialists. They do not appear in the menu. The `user-invocable:` flag,
   rather than the name prefix, determines whether a skill is public.

   Engineering rules live at `.claude/skills/devrites-lib/reference/standards/`.
   Workspace-operating lifecycle rites load `core.md` first and read
   phase-specific files on demand. Compact utilities keep a narrower local
   contract. Parallel
   reviewer fan-out at `/rite-seal` is the shared reference file
   `devrites-lib/reference/parallel-dispatch.md`, not a skill.
4. **Supporting references**: `reference/*.md` inside each skill. Long checklists,
   templates, and anti-rationalization tables loaded on demand (progressive
   disclosure) so `SKILL.md` bodies stay small.
5. **Agents**: `.claude/agents/devrites-*` contains **17 flat depth-one roles**:
   16 read-only leaves and `devrites-slice-wright`, the sole source/test writer
   role. Claude and Codex make only that role writable; Codex generates the
   other 16 profiles read-only. The read-only set includes three
   bounded work leaves (`devrites-evidence-scout`, `devrites-plan-drafter`, and
   `devrites-proof-runner`), the fresh upgrade assessor, plus the
   reviewers, auditors, and retrospector. Public rites remain
   authoritative: leaves return bounded evidence, never ask the human, change
   phase, or write canonical `.devrites/**` state. See
   [`orchestration.md`](orchestration.md) for the host boundary.
6. **Engineering rules**: DevRites' own stack-agnostic rules installed to
   `.claude/skills/devrites-lib/reference/standards/`. Workspace-operating lifecycle
   skills read `core.md` in step 0; compact utilities load only their local or conditional
   rules. On-demand files load by the phase that needs them:
   - **Craft:** `coding-style.md` · `prose-style.md` · `patterns.md` · `error-handling.md` ·
     `testing.md` · `spec-grammar.md` · `documentation.md` · `skill-authoring.md`.
   - **Quality / safety:** `code-review.md` · `principles.md` · `security.md` ·
     `performance.md` · `observability.md` · `developer-experience.md` · `deprecation.md` ·
     `anti-patterns.md`.
   - **Workflow / ops:** `development-workflow.md` · `git-workflow.md` ·
     `ci-cd.md` · `hooks.md` · `agents.md` · `context-hygiene.md` · `afk-hitl.md` ·
     `tooling.md` · `elicitation.md`.
   - **Checklists:** `definition-of-done.md` · `review-checklist.md` ·
     `test-proof-checklist.md` · `browser-proof-checklist.md` · `security-checklist.md`.
   - **Index:** `README.md` (phase mapping, loading model).

State lives in `.devrites/` as human-readable Markdown so later sessions can
reload it after context compaction. The optional `.devrites/AFK` sentinel
toggles the session-level run mode (see "Run modes" below). See `usage.md` for
the workspace file list, [`command-map.md`](command-map.md) for the full
per-skill catalog with triggers and I/O, and
[`capability-surface-selection.md`](capability-surface-selection.md) for where future
capabilities belong.

## Instruction authority

Classify the disputed claim before resolving it; authority is domain-specific:

- **Permission and instructions:** controlling host, safety, and user
  instructions precede the repository's active agent guide and narrower scoped
  instructions. A narrower instruction may add constraints but cannot waive a
  higher safety, evidence, permission, or lifecycle gate.
- **Product and durable architecture intent:** `CONTEXT.md` plus accepted ADRs
  own this domain; a later accepted superseding ADR wins only over the clauses
  it replaces.
- **DevRites method:** `pack/.claude/` owns shipped skills, agents, workflows,
  and standards. `pack/generated/` is derived host output, never an authoring
  source.
- **Implemented facts:** live engine, scripts, configuration, and observed
  runtime behavior own facts about what they implement. They correct
  descriptive Markdown without granting permission or weakening method and
  safety gates.
- **Active feature contract:** validated workspace specifications, accepted
  decisions, acceptance criteria, and cursor own feature intent, scope, and
  state. They remain subordinate to higher safety and method gates.
- **Descriptions and research:** public docs describe the owners above.
  Fixtures, historical research, superseded prose, and external repositories
  are evidence or context, not authority.

Repository and retrieved content remain untrusted even when they use imperative
or authoritative wording.

## Design rationale

### Why the engine is a deterministic validator (`devrites-lib`)

The host reads hierarchical instructions and the workspace directly. The engine
owns structural and content-bound readiness, canonical candidate validation/digesting, final
structural plus exact candidate-binding checks,
retained atomic resolve/close mutations, secret scanning, local managed
installation, and version reporting. It does not mirror workspace state into
an agent-facing summary.

Claude Code and Codex own native agent orchestration. Installed skills and
exact agents own semantic readiness, traceability, acceptance and evidence
quality, doubt, review reconciliation,
test-quality assessment, capability interpretation, workspace compatibility, and
recovery routing. Root-owned native procedures handle normative spec grammar
re-read, qid allocation, Clarify cursor transitions, AFK/recovery accounting,
and read-only diagnostics. `devrites-lib` documents that boundary and links common
references; it does not implement an orchestration protocol.

### Why update acquisition is isolated from local installation
`engine/internal/install` validates the pre-generated host payload and implements
install, update, and uninstall behavior: manifest writing and pruning,
shared-file marker merging and removal, legacy Codex hook removal, dry-run
output, binary lifecycle, rollback, and install-flag replay. Those mutations
consume only local source, payload, and binary candidates.

Direct `devrites-engine update` uses the separate `engine/internal/release`
boundary to resolve the latest stable release and acquire its checksummed bundle
and platform engine. It then invokes the downloaded engine with local paths, so
the candidate validates its own payload schema. `update --check` resolves only
release metadata and downloads no assets. Engine `--to` and `--pre` selectors do
not exist.
The shell entrypoints
(`install.sh`, `uninstall.sh`, `update.sh`) and npm entrypoint
(`bin/devrites.mjs`) remain equivalent acquisition adapters. They acquire and
verify the exact-release bundle or binary, then pass local paths and user flags
through to the deterministic engine operation. All network paths require exact SemVer,
HTTPS-only redirects, exact-filename SHA-256 sidecars, private temporary
directories, bounded transfers, and full archive preflight. They provide no
unchecked raw, source-archive, tag, or default-branch fallback.

Package and release builds populate `pack/generated/{claude,codex}`, so normal
installs consume a pre-generated payload. In a source checkout, the shell
install and update shims may regenerate missing payload components before
handing the local payload to the engine. The engine itself only validates and
copies that payload; it never generates host artifacts, and invalid payloads
fail closed.

Directly managed manifest entries carry SHA-256 ownership records. Before the
first refresh, prune, or uninstall mutation, the engine classifies every
affected path. It preserves customized or legacy entries without hashes unless
`--force` is explicit, and it checks each path again immediately before a
destructive action. Marker-owned shared files retain their block merge policy.
The engine rejects existing symlinks, junctions, and resolved target
escapes even under force.

Binary replacement is a separate transaction. Before replacement, the binary
at the exact staged path must report the requested release version. A backup in
the same directory remains until the binary at the installed path passes the
same check in a new process. On failure, the engine atomically restores the old
bytes and mode or removes a bad first install. It does not use `PATH` to
establish binary identity.

Production Git call sites use one isolation policy in Go and one parity helper
in shell. They remove environment variables that can retarget the repository,
worktree, index, objects, refs, config, or pathspec while preserving unrelated
Git variables; callers do not maintain independent deny lists.

Some duplication is intentionally preserved at host boundaries. Raw `curl | bash`
install/uninstall must be self-contained enough to fetch the bundle before any sibling
files exist. Claude assets are authored under `pack/.claude/**`; Codex assets are
generated into `pack/generated/**` and installed to `.agents/skills`, `.codex/agents`,
`.codex/config.toml`, and `AGENTS.md` because Codex and Claude use
different project-local conventions. Those generated host artifacts are delivered by the
npm installer; Claude/Codex plugin packaging is intentionally not a distribution path.

### Why semantic upgrade is evidence-gated, not a migration

The shell/npm update entrypoint acquires a candidate and uses the local engine
update operation to refresh the installed binary and pack. An active
unfinished workspace may still fail a current contract. `/rite-upgrade [slug]`
first asks the exact read-only `devrites-upgrade-planner` for a typed, cited
assessment. Age and cursor encoding do not prove staleness. A repair must name
the current rule, workspace evidence, affected gate, owner, paths, and minimum
delta; Clarify, Plan repair, Converge, Vet, Prove, Polish, Review, or Seal then
performs it under normal gates. Candidate repair runs current real proof and
never synthesizes a historical pass; ambiguous legacy candidate scope is a gap.

The engine has no structural migration command or compatibility telemetry. It
directly reads the official v1/v2 bullet and v3 table `state.md` cursors without
rewriting them. Wider pre-release compatibility experiments are not runtime
contracts. See [ADR-0025](adr/0025-evidence-gated-workspace-upgrades.md) and
[ADR-0022](adr/0022-native-orchestration-thin-engine.md).

### Why `/engine` was rejected
A single `/engine` (or `/devrites`) command would load every phase's
instructions into one context. That would increase context pressure, obscure
the purpose of each step, and make phase boundaries harder to enforce. Skill
bodies stay in context once loaded, so the recurring token cost would also
grow. Separate skills load only what the current phase needs.

### Why `rite-*` names
The team chose `rite-` because it is short, easy to remember, and matches the
product's use of "rites" for disciplined steps. The prefix also avoids
collisions. Built-in or bundled Claude Code commands include `/plan`, `/review`,
`/run`, `/verify`, `/code-review`, `/simplify`, `/security-review`, `/init`,
`/compact`, and `/debug`.

### Why a thin menu skill, not a mega-router
`/rite` is an entrypoint. It shows a compact phase-grouped menu and dispatches a
named verb to its owning skill. Menu mode does not infer workspace state;
`/rite-status` owns detailed status. `/rite` does not duplicate workflow logic,
which keeps selection small and leaves each phase in control of its context.

### Why internal skills exist
Specialist processes such as doubt, source-driven research, frontend craft,
browser proof, and audits are not direct user commands. As
`user-invocable: false` skills they:
- stay out of the command menu (less cognitive load);
- are invoked automatically by the host model or explicitly by a public skill when their
  trigger conditions hit;
- keep each public `SKILL.md` small by housing the heavy process elsewhere.

### Why spec, architecture, plan, tasks, and traceability are separate artifacts
Combining investigation, specification, planning, and slicing makes it easier
to miss a question, choose the wrong placement, or omit a requirement from a
slice. Each concern therefore has its own artifact and gate. `/rite-spec`
investigates the request and codebase in depth, then writes `spec.md` with the
product what and why, requirements, acceptance criteria, non-goals, and
measurable success. The spec
must pass native semantic review. `/rite-clarify` then audits the full actor,
journey, data, interface, and operations topology and writes
`decision-coverage.md`. A complete spec needs no questions.
`/rite-define` turns that **approved, clarified spec** into `architecture.md`
(technical map), `plan.md` (approach), `tasks.md` (vertical `SLICE-###` work), and
`traceability.md` (AC/REQ → slice → proof → evidence → files). `/rite-vet` uses
the exact plan reviewer and records `Implementation readiness: READY` before
build. The active skills and reviewers verify those claims against all relevant
artifacts; the engine checks required-file structure and binds the stable Build
inputs to that review without interpreting their prose. This checks the full
spec before Build without turning `spec.md` into a long technical document.
`/rite-plan` separately repairs, reslices, or reorders an active plan when it
goes stale or drifts.

New or materially revised specs declare one `Capability impact:`. MODIFIED
capability deltas preserve every prior scenario and normative/source-grounded
claim unless an accepted `DEC-###` explicitly authorizes replacement. A changed
API, event, schema, or other provider/consumer boundary gets one canonical
`Shared contract proof` artifact plus provider- and consumer-side asserting
tests that both consume it; no boundary change uses one justified no-impact
statement. These contracts stay in `spec.md`, `plan.md`, and the existing
traceability/proof artifacts rather than creating another registry.

### Why `/rite-polish` is one skill with two progressive-disclosure halves
Polish has two natural halves: **code** (simplify, dead code, naming, plus
backend if BE was touched) and **UI** (normalize to the design system, then
ship-quality detail). The earlier design split them into `rite-polish-code`
and `rite-polish-ui` sub-skills, but a skill-on-skill dispatch is fragile
(the caller has to re-discover the callee by description) and the two halves
share the same operating rules. They now live as `reference/code.md` and
`reference/ui.md` inside the same skill, loaded only when their phase
trigger fires. This follows the supporting-files pattern from Claude Code's current
[skills reference](https://code.claude.com/docs/en/slash-commands#add-supporting-files).
`/rite-polish` is the orchestrator: always reads `reference/code.md`
(Phase 1 + 2), detects UI scope from the diff and reads `reference/ui.md`
(Phase 3 + 4) when needed. The orchestrator accepts mode tokens (`bolder`,
`quieter`, `distill`, `harden`, `normalize-only`) that pass through to
Phase 4.

Normalization remains the entry gate for UI polish. The UI reference refuses
to run Phase 4 before Phase 3 because detail work would otherwise reinforce
patterns that do not match the project's design system.

Polish also owns every candidate-affecting capability-ledger fold, optional
`DESIGN.md` update, and durable ADR promotion. It refreshes affected proof and
closes the manifest before Review; Ship does none of these writes.

### Why seal and ship are separate phases (`/rite-seal`, `/rite-ship`)
Deciding whether a change is safe to ship has a smaller blast radius than
shipping it. `/rite-seal` asks exact fresh-context agents to judge acceptance,
evidence quality, tests, doubt, and the applicable review axes; the root
reconciles their results, requires positive discriminating proof, binds the
review and verdict to the closed candidate, runs the structural-plus-binding
engine check,
writes the GO or NO-GO verdict to `seal.md`, and runs no git commands. On GO,
it sets `state.md` to `Next step: /rite-ship` and stops.
`/rite-ship` refuses to run without a GO in `seal.md` and remains
candidate-read-only. Before type-`GO`, it performs only read-only candidate,
history, existing-index, and plan checks, then discloses the exact one-use Git
attempt. After a fresh literal `GO`, it optionally collapses eligible
checkpoints, stages exact manifest paths, validates staged scope, bytes,
bindings, and secrets, commits, and reverifies the committed candidate. It runs
push, tag, or PR actions only when the disclosed project convention requires
them and they are approved, then writes `ship.md`. It sets phase `done` and
moves
`.devrites/work/<slug>/` to `.devrites/archive/<slug>/` with every `.md` file
preserved, and clears `.devrites/ACTIVE`. The Seal GO verdict authorizes no Git
mutation; only Ship's freshly approved disclosed attempt does.

### Why `/rite-autocomplete` exists (the unattended orchestrator)
Some features are routine enough to run without human input at every phase.
`/rite-autocomplete` drives the whole lifecycle (spec → clarify → temper → define → vet →
build×N → prove → polish → review → seal → ship) by reading each phase's
`SKILL.md` and executing its
workflow. It carries state through workspace files rather than chat. A vague
prompt starts `devrites-interview`; `/rite-spec` and `/rite-clarify` then
finish the only interactive window. After decision coverage is CLEAR it runs unattended,
choosing the recommended option at each soft gate and recording the rationale
in `decisions.md`. It does **not** weaken the safety gates: genuine
product/scope/policy decisions, irreversible risk, human-only access/actions,
an open human-owned gate, a NO-GO, exhausted `max_slices`, or low confidence
still pause. Agents use bounded recovery for red tests, runtime failures, and
missing technical coverage. By default the workflow stops at Seal GO with `/rite-ship` as the next step.
The `--ship` flag (alias `--yolo`) continues through Ship preflight, discloses
the exact Git plan, and still stops for a fresh literal `GO` plus native host
approval.

### Why persistent `.devrites/` state
Long features outlive a single context window. Durable Markdown for the spec,
plan, tasks, state, evidence, drift, and decisions lets any phase reload the
current position in any session, even after compaction. Later-phase
clarification stores `return_phase` and `return_next_action` in `state.md` and
the root restores them only when the contract remains neutral. Recovery counts
the caller and recovery loop's failed attempts from current context plus
durable Dead ends/evidence, with no separate counter artifact.

### Run modes: HITL & AFK

DevRites runs the same lifecycle two ways, configured at two levels:

- **Per-slice** (planning-time): each `tasks.md` slice declares `Mode: AFK | HITL`.
  HITL slices add `Gate: advisory | validating | blocking | escalating`, `SLA`, and
  `Checkpoint` so the agent knows how disruptive the pause should be.
- **Per-session** (run-time): the presence of `.devrites/AFK` flips the session-level
  default. Empty file = AFK with safe defaults; YAML body widens behavior
  (`max_slices`, `notify`, `allow_gates`).

The pause primitive is a **pre-action interrupt**, not a post-action review queue.
It follows LangGraph's `interrupt()` / `Command(resume=)` model. `/rite-build` writes
an `Awaiting human` block to `state.md` and a question to `questions.md`, then stops.
`/rite-resolve` is the canonical resume verb. Restarting a session reads the workspace
back into a consistent state because the pause is durable Markdown, not chat memory.

Why a four-gate taxonomy (instead of a single "ask the user" pause): a single
gate becomes a queue under load. `advisory` writes only an audit log.
`validating` lets the build continue but blocks the merge. `blocking` stops
synchronously. This lets work continue when an answer can wait and pauses it
when the answer is required.
AFK always pauses for genuine product/scope/policy choices, irreversible risk,
and human-only access or actions, regardless of the sentinel. Objective test,
type, lint, runtime, and coverage failures stay inside bounded technical
recovery; exhaustion records a blocker without inventing a question. See
[`pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md`](../pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md) for the full
contract.

## Design choices at a glance

- **Surface**: 31 public `rite-*` skills (43 total), plus the thin `/rite`
  menu: 32 public and 11 internal. The lifecycle
  includes mandatory adaptive Clarify, optional Temper, mandatory Vet, and
  Converge recovery; Seal **decides** and Ship **executes + closes**. Ten
  `devrites-*` specialist skills are model-invoked; `devrites-lib` is the
  eleventh internal skill and the shared non-command library. Visibility comes
  from `user-invocable:`, not the namespace prefix.
- **Selection**: the `/rite` menu skill carries the routing table; every
  workflow skill enforces a "right skill, right time" rule in its body.
- **State**: durable `.devrites/` Markdown that survives compaction and new sessions.
  `.devrites/AFK` presence is the single source of truth for run mode (HITL/AFK);
  there is no `state.md` run-mode field to drift out of sync.
- **Run modes**: same lifecycle runs HITL (default; pause at typed gate) or AFK
  (drop `.devrites/AFK`; loop continues, discretionary pauses downgrade to
  advisory log, irreversible risk always pauses).
- **Slice rule**: in HITL, a direct `/rite-build` builds **one vertical slice,
  then stops**. An explicit `.devrites/AFK` sentinel is the bounded low-risk
  exception that may chain slices under its cap and pause rules.
  `/rite-autocomplete` owns full-lifecycle repetition.
- **Drift**: an explicit **Spec Drift Guard** in build/prove/polish/review/seal.
- **Design**: `devrites-frontend-craft` + a four-phase `/rite-polish` orchestrator (code + backend always; UI normalize + polish when UI is in scope).
- **Agents**: 17 fresh-context roles at flat depth one: 16 read-only leaves and
  one wright. Dispatch uses the exact named project role; if it cannot spawn,
  the workflow stops for HITL.
- **Review**: **feature-scoped** multi-axis review with severity labels and
  fresh-context agents at the seal.
- **Scope**: clarify → seal (decide) → ship (read-only preflight → type-GO →
  commit → optional approved push/tag/PR by project convention) → close. The CI
  pipeline stays with the project.
- **Install**: project-local, manifest-managed host artifacts; direct update,
  shell, or npm may acquire a verified release, local install application owns
  mutation, and the optional shared engine binary is the only allowed global
  artifact.

Candidate identity and bounded-input rationale are recorded in
[ADR-0026](adr/0026-content-bound-proof-and-bounded-inputs.md). The
[nine-source benchmark and adoption matrix](upstream-workflow-benchmark-2026-08-01.md)
is a dated 2026-08-01 research snapshot: it records the commits and conclusions
audited then, but it is not current authority. DevRites keeps its lifecycle and
canonical owners rather than adopting another project's runtime or terminology.

## Deviations from the original build brief (and why)

1. **Invocation semantics are explicit in frontmatter.** `user-invocable`
   controls whether a skill is a public command; `disable-model-invocation`
   independently marks explicit-only public utilities and the internal
   `devrites-lib` library. Required and conditional agent dispatch stays in the
   skill body, where the workflow step is already defined. Model-invocable
   phases can still hand off and route;
   per-phase side-effect discipline (for example, "stop after one slice") is
   enforced in the skill **body**, because invocation flags cannot express it.
2. **`context: fork` used selectively.** Verified it is a real field (isolated
   subagent, body-as-prompt, no conversation history). It is applied only to the three
   self-contained read-only audit skills (`devrites-audit simplify`,
   `devrites-audit security`, `devrites-audit perf`), which re-read
   `.devrites/` + `git diff` themselves. Interactive skills (doubt, interview, craft)
   are **not** forked because they need live context and user turns.
3. **Fresh-context adversarial review via real `.claude/agents/`** rather than relying
   on `context: fork` everywhere, because an agent can be handed the workspace path to
   read, whereas a forked skill cannot see the conversation.

Everything else follows the brief's structure, command names, state model, and
acceptance criteria.

## Repository layout

See the top-level tree in `README.md` and the installed-target tree in `usage.md`.
The canonical source pack lives under `pack/.claude/` (skills, agents, rules,
and Claude permission wiring). The build renders `pack/generated/` host artifacts;
the installer writes Claude assets under project `.claude/` and Codex assets
under `.agents/`, `.codex/`, and the managed `AGENTS.md` block.
