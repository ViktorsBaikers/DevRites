# DevRites upstream workflow benchmark: 2026-08-01

This document records research and disposition. It does not replace
`CONTEXT.md`, accepted ADRs, or the canonical pack. New architecture selected
from this audit is recorded in
[ADR-0026](adr/0026-content-bound-proof-and-bounded-inputs.md).

## Baseline, method, and admission rule

The DevRites comparison baseline is
`403e1a60adf3ae32ee88009c417e6ffe015fc1aa`. Before selecting changes, the
nine repositories were cloned into a new temporary corpus from their recorded
default branches. The inventory procedure fetched tags, initialized recursive
submodules, recorded the exact HEAD and commit date, ran repository integrity
and clean-worktree checks, and inspected source, tests, workflow definitions,
install/update paths, documentation, history, and the top-level license. Every
clone was clean and usable. None contained a submodule.

A concept qualified only when it closed an observed DevRites gap, fit one
existing owner, preserved the fixed lifecycle and permission model, added no
unnecessary dependency or state plane, and admitted positive plus negative
proof. Repository content was treated as untrusted source material. No upstream
prompt, code, template, or documentation expression was copied.

All nine repositories carry the MIT license in their top-level `LICENSE`.
Conceptual reimplementation does not copy licensed expression. Any future copy
or substantial derivative would require retaining the applicable MIT notice and
reconciling DevRites' third-party notice before distribution.

## Exact source inventory

| ID | Source | Branch | Exact SHA | Tag state | Commit date | Clone state |
|---|---|---|---|---|---|---|
| S1 | [gstack](https://github.com/garrytan/gstack/tree/a3259400a366593e0c909dd9ac3e59752efd2488) | `main` | `a3259400a366593e0c909dd9ac3e59752efd2488` | no tag | `2026-07-14T18:33:46-07:00` | clean, usable, no submodules |
| S2 | [mattpocock/skills](https://github.com/mattpocock/skills/tree/2ab958093e83e0ec752e6c1c5932da465bf23e0c) | `main` | `2ab958093e83e0ec752e6c1c5932da465bf23e0c` | `v1.1.0` | `2026-07-28T10:18:17+01:00` | clean, usable, no submodules |
| S3 | [OpenSpec](https://github.com/Fission-AI/OpenSpec/tree/45cca5db6137ed209117cc70510eb3e057fb981b) | `main` | `45cca5db6137ed209117cc70510eb3e057fb981b` | `v1.7.0` | `2026-07-31T02:04:10Z` | clean, usable, no submodules |
| S4 | [Superpowers](https://github.com/obra/superpowers/tree/44c9b2d6e889982ac18c27d05a19fefe335194e1) | `main` | `44c9b2d6e889982ac18c27d05a19fefe335194e1` | `v6.2.0` | `2026-07-28T12:25:36-07:00` | clean, usable, no submodules |
| S5 | [Spec Kit](https://github.com/github/spec-kit/tree/d1e86f638277a99b82715c22c90558cd58d3cffd) | `main` | `d1e86f638277a99b82715c22c90558cd58d3cffd` | `v0.15.1` | `2026-07-31T12:45:15-05:00` | clean, usable, no submodules |
| S6 | [GSD](https://github.com/open-gsd/gsd-core/tree/640eaee16e08bc5426cec570871cc0503a6c7d4c) | `next` | `640eaee16e08bc5426cec570871cc0503a6c7d4c` | `v1.9.1` | `2026-08-01T12:12:20-04:00` | clean, usable, no submodules |
| S7 | [BMAD Method](https://github.com/bmad-code-org/BMAD-METHOD/tree/a35e4c30d59a8edd808c834610428714785d5847) | `main` | `a35e4c30d59a8edd808c834610428714785d5847` | `v6.10.0` metadata; HEAD is 44 commits past the tag | `2026-08-01T08:11:07-07:00` | clean, usable, no submodules |
| S8 | [oh-my-claudecode](https://github.com/Yeachan-Heo/oh-my-claudecode/tree/41a4c0f77144c5beb5f5f000a89cff379c680606) | `main` | `41a4c0f77144c5beb5f5f000a89cff379c680606` | `v4.15.7` | `2026-07-23T04:44:59Z` | clean, usable, no submodules |
| S9 | [ECC](https://github.com/affaan-m/ECC/tree/e4e4163101f162881e628f300a9ca4e6a940bcea) | `main` | `e4e4163101f162881e628f300a9ca4e6a940bcea` | `v2.1.0` | `2026-07-29T14:11:01-04:00` | clean, usable, no submodules |

This inventory corrects the earlier report's stale GSD `d3305fc3` and BMAD
`e39cbbba` pins. GSD is intentionally pinned to its default `next` branch. BMAD's
package metadata still says 6.10.0, but the audited tree is not the tagged
v6.10.0 tree.

## DevRites baseline retained from the earlier comparison

Useful findings already present at the comparison baseline remain part of this
record:

- Review roles use one admitted outcome envelope: `findings`, `no-findings`, or
  blocking `gap`. Confidence, exact evidence, reachable impact, and root
  reconciliation remain the shared reviewer contract.
- Source intake requires provenance, one native owner, costs, parity impact,
  positive and negative proof, and an explicit disposition. Another catalog or
  runtime remains out of scope.
- The agent validator checks reviewer-contract loading and output shape;
  frontmatter parsing rejects duplicate top-level keys before YAML parsing.
- npm package tests prove the exact runtime-script set against a real packed
  tarball, and source installation repairs a partially present generated host
  payload.
- The stronger existing owners remain unchanged: the 15-phase lifecycle, exact
  fresh-context roles, sole exact-path writer, Markdown workspace, immutable
  proof/review separation, fail-closed secret and path checks, and the offline
  stdlib-only Go engine.

## Standardized repository dossiers

### S1: gstack

- **Purpose/philosophy:** gstack packages planning, implementation support,
  review, browser QA, shipping, and retrospective work into an opinionated
  high-velocity assistant environment.
- **Architecture/components:** A Bun/TypeScript runtime, generated role skills,
  a Chromium/browser service, setup and update hooks, a ten-host adapter
  registry, evaluation storage, and Git-backed long-lived context form one
  integrated system (`ARCHITECTURE.md`, `hosts/`, `test/helpers/eval-store.ts`).
- **Strongest ideas:** Its live multi-host scenarios persist partial and final
  results, compare historical cost and turn budgets, and expose coverage debt.
  Review findings also carry confidence and concrete evidence.
- **End-to-end workflow:** A user can tune a plan, implement, run specialist
  review, exercise browser QA, prepare a ship action, and write retrospective
  context without leaving the suite.
- **Skill/agent patterns:** Host-specific roles are generated from shared
  material and can fan work out, but reviewer, QA, and ship roles are allowed to
  mutate more state than DevRites reviewers.
- **Spec/planning:** Product and executive plan lenses improve challenge quality;
  they do not provide DevRites' durable requirement, traceability, and decision
  contract.
- **Execution controls:** Browser automation, worktree use, and eval budgets are
  concrete controls. Several fallback paths continue in the current worktree or
  classify errors heuristically.
- **Proof/review:** Browser QA, self-verification regressions, and live model
  comparisons are material strengths. Scheduled coverage is incomplete: the
  audited TODO records only 9 of roughly 66 E2E files in periodic CI.
- **State/context:** Partial eval artifacts and Git-backed memory improve
  continuity, at the cost of another mutable context authority.
- **Safety/failure handling:** Content, DNS, classifier, and parser paths can fail
  open; shell assignments are later evaluated; setup may update in the
  background; upgrade can hard-reset; worktree failure may fall back to the
  current tree.
- **DX/docs:** The first-use path and task-oriented documentation are strong,
  while the breadth of browser, tunnel, runtime, and memory services raises the
  operational floor.
- **Extension model:** The typed host registry and specialist integrations are
  effective for a many-host product but premature for DevRites' two maintained
  hosts.
- **Install/update:** Setup installs Bun/browser dependencies and wires hooks;
  session-time update behavior and destructive upgrade recovery are unsuitable
  defaults.
- **Weaknesses/tradeoffs:** Integrated convenience comes with a daemon/browser
  stack, telemetry and update coupling, mutable review paths, and large live-eval
  maintenance.
- **Already present in DevRites:** Result admission, independent review, browser
  proof, exact-role dispatch, evidence artifacts, runtime smokes, and bounded
  recovery already cover the core safety intent.
- **Weaker/missing in DevRites:** DevRites lacks systematic budgeted live-host
  scenario execution with coverage accounting and historical comparison.
- **Unsuitable concepts:** Browser/tunnel daemons, reviewer mutation, finding
  quotas, background auto-update, telemetry, hard-reset upgrade, and a second
  memory system are rejected.
- **License implication:** MIT permits adaptation with notice; this report copies
  no implementation or prose. Direct reuse would require attribution review.
- **Native adaptation/disposition:** Defer a bounded live-host eval runner until
  its recurring cost and security model have an owner. Retain the existing
  reviewer-admission and browser-proof mechanisms; add no gstack runtime.

### S2: mattpocock/skills

- **Purpose/philosophy:** This repository is a curated skill library centered on
  choosing the right skill, doing the necessary source work, loading references
  only when needed, and defining completion.
- **Architecture/components:** Forty-one Markdown skills, a router, authoring
  guidance and glossary, templates/scripts, and plugin metadata form a catalog
  rather than a stateful workflow engine.
- **Strongest ideas:** Its authoring model asks authors to reduce routing
  ambiguity, order instructions by need, state how the model corrects course,
  define observable completion, and delete guidance that no longer changes
  behavior.
- **End-to-end workflow:** A user selects a focused skill for specification,
  tracer-bullet delivery, TDD, debugging, review, or handoff; the skill performs
  its task and yields to another skill rather than advancing canonical state.
- **Skill/agent patterns:** Skills prefer references and concrete legwork, but
  generic agents and catalog routing do not enforce DevRites' exact role and
  permission boundaries.
- **Spec/planning:** Ticket, spec, and tracer-bullet guidance favors actionable
  plans. It does not own a persistent acceptance/traceability schema.
- **Execution controls:** TDD and debugging procedures are instruction-enforced;
  commits and external issue writes may occur without DevRites' fresh approval
  boundary.
- **Proof/review:** Checkable completion and review guidance are useful, but the
  audited repository has no validating test suite or CI for skill semantics.
- **State/context:** Handoff artifacts carry local continuity; there is no
  canonical lifecycle cursor or evidence ledger.
- **Safety/failure handling:** Some flows assume authority to commit or write to
  external trackers. An unfinished wizard handles environment and GitHub secrets
  without a sufficient promotion boundary.
- **DX/docs:** The authoring glossary is compact and teachable. Router, docs,
  skills, and version metadata contain observable drift.
- **Extension model:** Contributors add another skill or plugin entry. Importing
  that open-ended catalog would duplicate DevRites' routed public rites.
- **Install/update:** Plugin distribution is simple, but package 1.1.0 and Claude
  plugin 1.2.0 metadata disagree at the audited revision.
- **Weaknesses/tradeoffs:** Low ceremony aids contribution but leaves behavioral
  regressions, permissions, and cross-surface drift largely unchecked.
- **Already present in DevRites:** Progressive references, routing budgets,
  authoring validation, outcome evals, exact agents, TDD, debug recovery, review,
  and handoff already implement the useful method.
- **Weaker/missing in DevRites:** The glossary is clearer than some DevRites
  authoring prose, and semantic-drift negative fixtures could be stronger.
- **Unsuitable concepts:** A sibling skill catalog, generic agents, external issue
  authority, unconditional commits, and secret-handling wizard defaults are
  rejected.
- **License implication:** MIT permits reuse with notice. Only general method was
  considered; no text or template was copied.
- **Native adaptation/disposition:** Retain the baseline skill-authoring owner and
  its semantic validation. Merge terminology only when it shortens that owner;
  do not add skills, agents, or distribution paths from this source.

### S3: OpenSpec

- **Purpose/philosophy:** OpenSpec treats a proposed change as a graph of durable
  specification artifacts and folds accepted deltas back into a main spec set.
- **Architecture/components:** A TypeScript CLI, YAML artifact schemas, a
  validated artifact DAG, generated assistant skills, delta parsing/folding,
  change/spec stores, archive logic, and beta references/worksets make up the
  system (`src/core/artifact-graph/`, `src/core/specs-apply.ts`).
- **Strongest ideas:** MODIFIED folding detects scenario loss, preserves multiple
  scenarios, handles header conflicts, and supports idempotent early sync.
  Archive pre-validates the resulting specs before writes.
- **End-to-end workflow:** Propose creates artifacts; apply drives implementation;
  update or sync folds deltas; verify inspects the result; archive closes the
  change.
- **Skill/agent patterns:** Generated skills adapt the artifact graph to multiple
  assistants. The graph, rather than exact reviewer/writer roles, is the main
  coordinator.
- **Spec/planning:** Main specifications plus ADDED, MODIFIED, and REMOVED deltas
  are explicit and inspectable. Custom schemas allow a project to redefine the
  artifact workflow.
- **Execution controls:** Artifact readiness is structural and can become DONE
  when a generated file merely exists. Archive offers bypass flags and performs
  sequential writes without repository-wide rollback.
- **Proof/review:** Verify is optional in the expanded profile and relies partly
  on keyword/path inference instead of immutable executed evidence.
- **State/context:** Change directories, main specs, references, stores, and
  worksets retain context. Beta cross-repository references can remain stale
  until a user synchronizes them.
- **Safety/failure handling:** Prevalidation is strong, but user/project schemas
  and followed schema-directory symlinks widen the prompt and supply-chain
  boundary; archive bypasses weaken fail-closed behavior.
- **DX/docs:** A worked beginner transcript and inspectable change files make the
  model approachable.
- **Extension model:** Custom schemas, adapters, stores, and worksets are broad by
  design and would create a second workflow language in DevRites.
- **Install/update:** The CLI carries telemetry and self-update concerns that do
  not belong in DevRites' offline engine boundary.
- **Weaknesses/tradeoffs:** Flexibility weakens uniform proof, permits
  existence-only completion, and multiplies schema/runtime compatibility work.
- **Already present in DevRites:** Durable Markdown workspaces, Requirement and
  Scenario grammar, capability deltas, native phase owners, Prove, Seal, and
  reversible archive already cover the main lifecycle.
- **Weaker/missing in DevRites:** The current capability fold can lose prior
  MODIFIED scenarios or source claims and lacks an explicit justified no-impact
  form.
- **Unsuitable concepts:** A configurable artifact DAG, custom prompt schemas,
  telemetry/self-update, heuristic optional verify, and broad adapter/store
  registries are rejected.
- **License implication:** MIT allows reuse with notice. The selected behavior is
  independently specified; no parser, schema, or prompt was copied.
- **Native adaptation/disposition:** Adopt loss-aware MODIFIED capability/source
  preservation and an explicit no-impact declaration in the existing ledger
  owner. Move the fold before Review/Seal; add no OpenSpec runtime.

### S4: Superpowers

- **Purpose/philosophy:** Superpowers is a skill-centered development method that
  prescribes discovery, planning, isolated execution, TDD, debugging, review,
  and completion checks.
- **Architecture/components:** Harness-neutral Markdown skills, adapters and
  hooks, plan-scoped file workspaces, fresh subagent packets, shell tests, and a
  visual brainstorming companion form the distribution.
- **Strongest ideas:** Plan-scoped file briefs, hashable review packets, narrow
  re-review after a fix, honest null-result eval reporting, condition-based waits,
  and backward root-cause tracing are practical mechanisms.
- **End-to-end workflow:** Route through the skill system, brainstorm and approve
  a design, create an isolated worktree, write a full task plan, execute each task
  with TDD and review/fix, run broad review, then integrate the branch.
- **Skill/agent patterns:** Fresh subagents receive bounded task and review
  packets. The current task reviewer combines concerns despite older docs still
  describing a two-stage review.
- **Spec/planning:** Plans can include full implementation detail and code, which
  improves determinism but increases context and early commitment.
- **Execution controls:** Worktrees, per-task commits, test-first work, and
  condition waits are explicit. Mandatory 1 percent routing and repeated
  park-and-complete loops add fixed cost.
- **Proof/review:** Verification-before-completion and scoped re-review are sound;
  several integration assertions do not prove all guarantees claimed by their
  prompts.
- **State/context:** Plan-scoped files reduce cross-plan leakage and make review
  packets reproducible without another database.
- **Safety/failure handling:** A raw-HTML visual server, weak content policy,
  page-visible key, remote branding, and spoofable user-role markers make the
  visual/bootstrap surfaces unsuitable as-is.
- **DX/docs:** Concrete TDD, async, and debugging recipes are useful, although
  documentation and current combined-review behavior drift.
- **Extension model:** Harness ports and adapter-specific bootstraps widen reach
  but increase parity and trust-boundary work.
- **Install/update:** Some install guidance asks an agent to mutate user or main
  configuration; that conflicts with DevRites' managed local installer.
- **Weaknesses/tradeoffs:** The mandatory workflow is expensive for small changes,
  overcommits plans to code, and couples method correctness to prompts and hooks.
- **Already present in DevRites:** Exact slice packets, one writer, test-first
  behavior, causal debugging, independent review, immutable proof, bounded
  correction, and host generation already cover its core strengths.
- **Weaker/missing in DevRites:** The baseline candidate packet is not bound to
  content; evidence freshness uses modification times rather than file identity.
- **Unsuitable concepts:** Mandatory worktrees and task commits, global bootstrap,
  five-round parking, raw-HTML visual tooling, and mutable-main installation are
  rejected.
- **License implication:** MIT permits reuse with notice. The candidate identity
  design is a clean DevRites format, not copied Superpowers code or prose.
- **Native adaptation/disposition:** Adopt a canonical content-bound candidate
  manifest and scoped re-proof after candidate changes. Keep all enforcement in
  existing Build, Prove, Polish, Review, Seal, and engine owners.

### S5: Spec Kit

- **Purpose/philosophy:** Spec Kit supplies a specification-first command set and
  CLI that can project templates and workflows into many assistant hosts.
- **Architecture/components:** A Python CLI, command templates, constitution and
  persistence guidance, presets, extensions, role bundles, a persisted YAML
  workflow engine, and multi-host generators form the product.
- **Strongest ideas:** Its persistence taxonomy is easy to explain. Download and
  ZIP handling enforce byte and path limits, check entry types, and have rollback
  coverage.
- **End-to-end workflow:** Establish principles, specify, clarify, plan, derive
  tasks, analyze consistency, implement, and converge; alternative YAML workflows
  can select a different command graph.
- **Skill/agent patterns:** Commands are generated for multiple integrations and
  may be grouped into roles. Permissions are described rather than enforced by
  exact host profiles.
- **Spec/planning:** Constitution, structured feature specs, clarification,
  checklists, plans, and tasks provide a familiar specification path.
- **Execution controls:** The generic engine stores workflow progress and exact
  top-level resume state, but requirements are advisory and shell fields can be
  interpolated without a capability boundary.
- **Proof/review:** Generated story tests can be optional, checklist gates can be
  bypassed, and the built-in full workflow omits several clarify, convergence,
  proof, and review stages.
- **State/context:** Feature directories, active context, workflow YAML, presets,
  and persistence categories keep artifacts visible but introduce multiple ways
  to define lifecycle truth.
- **Safety/failure handling:** Archive acquisition and extraction are strong.
  Project hooks, raw shell interpolation, auto-commit behavior, and optional gates
  are weaker trust boundaries.
- **DX/docs:** Multi-integration onboarding and persistence explanations are good;
  some release/citation metadata trails the implementation.
- **Extension model:** Stackable presets, extensions, hooks, workflows, and role
  bundles are useful for a framework product but duplicate DevRites' fixed rites.
- **Install/update:** The Python CLI bounds remote data and validates ZIPs before
  extraction, with rollback tests for failed generation and updates.
- **Weaknesses/tradeoffs:** Broad configurability improves reach while making
  permission, proof, and workflow guarantees conditional on project content.
- **Already present in DevRites:** Principles, explicit specifications,
  Clarify/Define/Vet, active workspace state, convergence, Prove/Review/Seal, and
  manifest-aware managed install already cover the workflow value.
- **Weaker/missing in DevRites:** Shell and Node acquisition paths lack complete
  response, archive, member, expanded-size, and secure-temp bounds; checksum
  failure can reach an unchecked fallback.
- **Unsuitable concepts:** A generic YAML workflow engine, project-directed hooks,
  extension marketplace, advisory permissions, optional tests, and auto-commit
  are rejected.
- **License implication:** MIT permits reuse with notice. DevRites adopts the
  boundary requirements, not Spec Kit's downloader or templates.
- **Native adaptation/disposition:** Adopt exact-asset plus checksum-sidecar
  acquisition, fixed transfer limits, secure temp failure, full archive preflight,
  and mandatory release sidecars in existing bootstrap and release owners.

### S6: GSD

- **Purpose/philosophy:** GSD is a broad phase-oriented project workflow that
  favors fresh-context agents, file-backed planning, checkpointed execution, and
  user acceptance at the end of each phase.
- **Architecture/components:** Markdown workflows and agents, a Node/CommonJS
  installer/runtime, `.planning` state, UAT predicates, workstreams, capability
  metadata, migration journals, workflow fragments, and multi-host generation
  support fifteen runtimes.
- **Strongest ideas:** Its UAT predicate demands positive evidence rather than
  accepting the absence of failures. Read-only workspace inventory, truthful
  progress metadata, install-script disclosure, and update impact previews are
  also useful.
- **End-to-end workflow:** Initialize a project, research and roadmap it, discuss
  and plan a phase, check the plan, execute tasks, verify and run UAT, then close
  or continue the milestone.
- **Skill/agent patterns:** Researchers, planners, checkers, executors, and
  verifiers operate in fresh contexts with file handoffs; workstreams can fan out
  to several source writers.
- **Spec/planning:** Roadmaps, phase research, context, plans, and verification
  criteria provide durable decomposition, though they do not use DevRites'
  requirement/scenario and decision-coverage contract.
- **Execution controls:** Worktrees, checkpoints, atomic commits, recovery, and
  partial fragment composition are explicit. Fragment conditions are validated
  but were not selected dynamically at the audited revision.
- **Proof/review:** UAT fails when evidence is empty or vacuous, a stronger rule
  than compile-only success. Plan checking and final verification are separate.
- **State/context:** `.planning` state, progress, checkpoints, handoffs, and
  optional external memory provide continuity at the cost of a large state
  vocabulary.
- **Safety/failure handling:** Typed capability trust and migration journals help.
  Executable capability sources, hook bypass, parallel writers, and external
  memory authority widen the trust boundary.
- **DX/docs:** The command set is extensive and well illustrated, but Node 18/22,
  release, hook, and non-Claude parallel documentation disagree with code.
- **Extension model:** Workflows, fragments, capabilities, workstreams, and many
  host targets create an open platform rather than a fixed method.
- **Install/update:** A large installer manages host payloads and migrations;
  recent changes improve empty-cherry-pick and metadata behavior but increase the
  compatibility surface.
- **Weaknesses/tradeoffs:** Coverage is broad, while installer size, documentation
  drift, external capabilities, and parallel mutation make guarantees harder to
  state and test.
- **Already present in DevRites:** The fixed lifecycle, fresh exact agents,
  Markdown workspace, AFK/HITL, task slices, recovery, handoff, Prove, Review,
  Seal, and managed local installation cover the main workflow strengths.
- **Weaker/missing in DevRites:** Proof guidance can accept skipped,
  assertion-free, compile-only, or otherwise non-observable success. Cross-feature
  status and update impact presentation could also improve.
- **Unsuitable concepts:** Parallel source writers, executable capability
  marketplaces, external canonical memory, generic workstreams, and ambiguous
  pseudo-numeric scoring are rejected.
- **License implication:** MIT permits reuse with notice. The admitted proof rule
  is independently stated in DevRites terms.
- **Native adaptation/disposition:** Adopt positive observable/framework evidence
  as a proof minimum. Defer read-only all-workspace inventory and update preview
  until a concrete navigation problem justifies them; add no GSD runtime.

### S7: BMAD Method

- **Purpose/philosophy:** BMAD provides scale-adaptive planning and delivery
  through role-focused workflows, explicit architecture decisions, readiness,
  story execution, review, checkpoints, and retrospective learning.
- **Architecture/components:** A Node installer, module catalogs, YAML/Markdown
  workflows, roles and skills, spec and architecture templates, memlog state,
  verification-gap prompts, and custom-module acquisition form a configurable
  method.
- **Strongest ideas:** Its architecture records bind choices to constraints, spec
  validation includes a claim-preservation pass, and the installer deliberately
  removes inherited Git variables that can retarget repository commands.
- **End-to-end workflow:** Choose a quick or full path, produce analysis and
  product/architecture artifacts, check readiness, create and implement stories,
  review them, checkpoint larger work, and run a retrospective.
- **Skill/agent patterns:** Named personas and concern-oriented reviewers provide
  useful lenses, but inline writer fallback and quotas weaken separation and
  honest null findings.
- **Spec/planning:** The spec kernel, two-pass validation, readiness, and explicit
  architecture consequences are strong. Append-only memlog claims can become a
  competing source of truth.
- **Execution controls:** Story workflows, checkpoints, prior-action follow-up,
  and commits structure delivery; some flows assume authority to mutate and
  commit without DevRites' approval ladder.
- **Proof/review:** Multi-lens gap review is useful. A recent blanket exclusion of
  source-text and model-output checks would miss DevRites' executable guidance,
  schema, and eval surfaces.
- **State/context:** Product, architecture, story, checkpoint, retrospective, and
  memlog artifacts preserve context, with more mutable owners than DevRites.
- **Safety/failure handling:** Git environment filtering addresses a real
  retargeting risk. In contrast, the custom SSH source parser permits shell
  metacharacters before `execSync`, and custom modules can run install scripts.
- **DX/docs:** Many roles and routes cover projects of different sizes. The audited
  HEAD reports 6.10.0 metadata but is 44 commits beyond the v6.10.0 tag.
- **Extension model:** Custom modules and remote sources make the method open, but
  also introduce unverified code and lifecycle execution.
- **Install/update:** The Node installer composes modules and filters Git
  environment state; custom-source and dependency installation remain high-risk
  edges.
- **Weaknesses/tradeoffs:** Configurability, personas, memlog, quotas, and module
  loading add competing authority and supply-chain exposure.
- **Already present in DevRites:** Frame/Spec/Define/Vet, binding decisions,
  exact-role review, checkpointed Build, Prove, Seal/Ship separation, and Learn
  already implement the strongest workflow concepts.
- **Weaker/missing in DevRites:** Both Go Git subprocess sites and production shell
  Git calls inherit retargeting `GIT_*` variables. Capability/source folding can
  also lose earlier claims.
- **Unsuitable concepts:** Finding quotas, blanket source-test exclusion, inline
  writer fallback, auto-commit, memlog authority, remote module loading, and the
  custom SSH parser are rejected.
- **License implication:** MIT permits reuse with notice. DevRites adopts the
  security property and preservation criterion, not BMAD implementation text.
- **Native adaptation/disposition:** Adopt one Go and one shell Git-environment
  policy with dynamic config-pair parity; merge claim preservation into the
  existing capability fold. Add no BMAD module or state surface.

### S8: oh-my-claudecode

- **Purpose/philosophy:** oh-my-claudecode is a hook-driven Claude orchestration
  runtime for autonomous and team execution, iterative recovery, verification,
  provider routing, and cancellation.
- **Architecture/components:** TypeScript commands, skills, agents, ambient hooks,
  Team, Autopilot, Ralph, UltraQA, state and cancellation stores, worktree locks,
  provider routing, installer/update code, and benchmarks form overlapping modes.
- **Strongest ideas:** Named-workflow resume validates content and transcript
  identity, candidate state uses SHA-256 in critical paths, and commit-time checks
  reauthenticate what is about to ship.
- **End-to-end workflow:** Magic keywords or commands choose a mode; Autopilot
  plans and executes, Team fans out work, Ralph iterates, UltraQA checks results,
  and cancel/state commands manage long-running sessions.
- **Skill/agent patterns:** Nested workers and provider-selected agents increase
  throughput, while worker permissions are advisory rather than exact
  host-enforced source boundaries.
- **Spec/planning:** Mode-specific planning exists, but no single requirement,
  decision, traceability, and evidence chain matches the DevRites workspace.
- **Execution controls:** Cross-platform locks, worktrees, resume validation,
  retries, cancellation, and provider routing are useful; overlapping modes make
  authority and stop conditions hard to reason about.
- **Proof/review:** UltraQA and iterative review provide repeated checks, but some
  approvals depend on model text or caller-supplied verdicts rather than frozen
  executable proof.
- **State/context:** Hook state and transcripts support recovery. Documented
  UltraQA paths and actual runtime paths drift.
- **Safety/failure handling:** Security protections default off in places, CI uses
  mutable action tags, external teams have advisory permissions, and silent
  update installs an unpinned npm latest release.
- **DX/docs:** Many modes are discoverable, but skill counts, state paths, and
  claimed benchmark savings disagree with checked-in evidence.
- **Extension model:** Hooks, providers, modes, agents, and keyword routing make an
  extensible runtime at the cost of a large implicit control surface.
- **Install/update:** Plugin installation is convenient; the silent floating
  updater and checksum gaps conflict with DevRites release requirements.
- **Weaknesses/tradeoffs:** Autonomy and provider flexibility add ambient hooks,
  nested modes, heuristic routing, mutable updates, and evidence claims that are
  difficult to reproduce.
- **Already present in DevRites:** AFK/HITL, exact native agents, causal recovery,
  immutable Prove/Review/Seal, cancellation by the host, managed checksummed
  acquisition, locks, and runtime smokes cover the useful intent.
- **Weaker/missing in DevRites:** Evidence freshness binds modification time, not
  candidate bytes, path state, or executable mode. Live reviewer eval fixtures
  are also limited.
- **Unsuitable concepts:** Ambient hooks, magic routing, nested team/autopilot
  modes, heuristic provider routing, security-off defaults, and floating
  auto-update are rejected.
- **License implication:** MIT permits reuse with notice. The selected digest and
  manifest format is original DevRites design.
- **Native adaptation/disposition:** Adopt content-addressed candidate identity
  shared by Candidate and Seal, plus staged/committed reauthentication before
  outbound Git. Defer broader live-model evaluation; add no OMC mode or hook
  runtime.

### S9: ECC

- **Purpose/philosophy:** ECC combines a large cross-harness skill and agent
  catalog with contract-first development, memory guidance, install profiles,
  hooks, dashboards, and an alpha orchestration control plane.
- **Architecture/components:** The audited tree contains 281 skills, 67 agents,
  rules and commands, manual host mirrors, JavaScript install lifecycle code,
  session/memory stores, hooks, and a Rust `ecc2` daemon/TUI/worktree system.
- **Strongest ideas:** Contract-first integration uses one canonical API, event,
  or schema artifact and asks both provider and consumer to test it. Typed review
  outcomes, scoped memory trust, and installer preview are also useful.
- **End-to-end workflow:** Users compose planning, contract, implementation,
  review, and memory skills; profiles install subsets; ECC2 can schedule agents,
  worktrees, output capture, and merge work.
- **Skill/agent patterns:** A broad specialist catalog and cross-host mirrors offer
  coverage, but manual duplication and parallel writers conflict with DevRites'
  generated exact-role model.
- **Spec/planning:** Contract-first guidance grounds an integration slice in one
  boundary artifact; general plans and annotations do not provide a single fixed
  lifecycle schema.
- **Execution controls:** Profiles, hooks, worktrees, a scheduler, merge queue,
  daemon, and TUI provide many controls. The Rust control plane is not part of
  main CI at the audited revision.
- **Proof/review:** Provider and consumer tests against the same artifact close a
  real integration gap. Universal coverage targets and catalog-level review
  recipes are too generic for all projects.
- **State/context:** Session vaults, unified memory, dashboards, and SQLite retain
  context, creating a second authority beside project files.
- **Safety/failure handling:** Written supply-chain and memory trust guidance is
  strong. Floating `npx` MCP packages, clone-time npm install, always-on hooks,
  live-web defaults, and delayed output-persistence errors contradict that
  posture.
- **DX/docs:** Profiles and quick references make the catalog approachable, but
  v2.1.0 metadata, a changelog ending at 2.0.0, and stale alpha docs show drift.
- **Extension model:** Skills, agents, rules, commands, MCPs, dashboards, profiles,
  and the Rust runtime form a broad plugin platform.
- **Install/update:** JavaScript lifecycle code previews and applies profiles;
  manual host mirrors and optional runtime/tool downloads multiply parity work.
- **Weaknesses/tradeoffs:** Breadth creates large validation, documentation,
  supply-chain, and state costs. The daemon/TUI/worktree plane duplicates native
  host orchestration.
- **Already present in DevRites:** Exact reviewers and outcome admission, scoped
  memory promotion in Learn, fresh-context planning and proof, installer
  manifests, one writer, and generated Claude/Codex parity already cover most
  strengths.
- **Weaker/missing in DevRites:** A boundary-changing slice need not prove that
  provider and consumer tests consume the same canonical contract artifact.
- **Unsuitable concepts:** The Rust daemon/TUI/scheduler/merge queue, parallel
  writers, broad hooks, duplicate memory/session stores, floating MCP defaults,
  universal coverage, dashboards, and manual mirrors are rejected.
- **License implication:** MIT permits reuse with notice. The conditional
  provider/consumer rule is independently specified; no ECC content was copied.
- **Native adaptation/disposition:** Adopt a conditional shared-contract proof
  rule in existing Spec/Plan/Prove/testing owners. Add no registry, daemon,
  profile, agent, or state store.

## Cross-project adoption matrix

The matrix is split into linked tables so each cost and proof field remains
readable. `Adopt` means planned by ADR-0026. `Merge` means strengthen an existing
owner without new public surface. `Defer` requires new evidence. `Reject` appears
in each dossier for duplicated or unsafe systems.

### Decision, owner, and validation

| ID | Source | Exact SHA | Concept and problem | Current DevRites handling | Native adaptation | Decision and implementation owner/location | Validation method |
|---|---|---|---|---|---|---|---|
| C1 | Superpowers; oh-my-claudecode | `44c9b2d6e889982ac18c27d05a19fefe335194e1`; `41a4c0f77144c5beb5f5f000a89cff379c680606` | Candidate/evidence identity; mtime and heuristic path extraction do not bind proved bytes. | Seal compares proof and touched-file mtimes; missing manifests and listed files can pass. Ship later changes candidate files. | Explicit `present`/`deleted` manifest, no-files marker, normalized bounded closure, streaming versioned digest, shared Candidate/Seal helper, pre-Review closure, affected re-proof, identical exact digest lines in evidence, review, seal, and browser evidence when present, plus Ship and staged/committed checks. | **Adopt.** Go candidate/evidence owner in `engine/internal/lib/` and `engine/internal/gate/`; native Prove/Polish/Review/Seal/Ship and workspace schema in `pack/.claude/`. | Positive digest stability; byte/path/state/type/mode mutations; malformed/legacy manifests; bounds; symlink/special-file rejection; missing or mismatched evidence/review/seal/browser lines; routing and Git-ladder contract tests. |
| C2 | BMAD Method | `a35e4c30d59a8edd808c834610428714785d5847` | Inherited Git variables can retarget repository, objects, refs, config, or pathspec despite `git -C`. | Two Go subprocess sites and production shell Git sites inherit ambient `GIT_*`. | One shared Go sanitizer and one sourced shell sanitizer; preserve unrelated variables; remove retargeting variables; keep dynamic config key/value pairs in parity. | **Adopt.** Go subprocess owners in `engine/internal/lib/` and `engine/internal/rootfacts/`; shared shell policy under `scripts/` used by all production shell Git callers. | Poisoned two-repository probes; per-variable table tests; dynamic-pair malformed/preserve/drop cases; Go/shell parity; root and secret-scan regression. |
| C3 | Spec Kit | `d1e86f638277a99b82715c22c90558cd58d3cffd` | Unbounded or unchecked acquisition can exhaust resources or bypass provenance. | Release checksums exist, but shell/Node transfers lack complete bounds, temp fallback can be predictable, and unchecked source/tag/raw fallbacks remain. | Exact asset plus mandatory SHA-256 sidecar, fixed transfer ceilings, secure private temp or fail, complete one-prefix archive preflight, no unchecked fallback, mandatory release sidecars. | **Adopt.** `install.sh`, `update.sh`, `scripts/install-lib.sh`, `bin/devrites.mjs`, release builder/check, and existing install/update/release tests. | Oversized metadata/checksum/archive/binary; missing/incorrect sidecar; temp failure; extra prefix/member/type; traversal; symlink/special file; expanded-size limit; release missing-sidecar failure; curl-plus-tar smoke. |
| C4 | OpenSpec; BMAD Method | `45cca5db6137ed209117cc70510eb3e057fb981b`; `a35e4c30d59a8edd808c834610428714785d5847` | MODIFIED capability folding can lose prior scenarios or source claims; no-impact changes are implicit. | `spec-grammar.md` and the capability ledger own deltas, but preservation and no-impact are not explicit enough. | Preserve prior scenarios and claims unless explicitly replaced; require justified no-impact; perform every candidate-affecting fold before Review/Seal. | **Merge.** Existing capability/spec owners in `pack/.claude/skills/rite-polish/`, shared spec grammar/schema, and native contract tests. | Multiple-scenario and foreign-tail fixtures; explicit replacement; duplicate/conflict; no-impact justification; Ship mutation rejection; generated-host parity. |
| C5 | GSD | `640eaee16e08bc5426cec570871cc0503a6c7d4c` | A green command can be skipped, assertion-free, compile-only, or otherwise vacuous. | Prove records commands and acceptance links, but positive observable/framework evidence is not uniformly mandatory. | Require executed positive observation or framework assertion; classify skipped, vacuous, compile-only, assertion-free, and non-executed results as unproven. | **Adopt.** Canonical testing and Prove standards, traceability/evidence schema, reviewer/test analyst, and routing/native contract tests. | Fixtures for each vacuous form plus real positive unit, integration, browser, and textual-artifact evidence; test-integrity review; Seal rejection. |
| C6 | ECC | `e4e4163101f162881e628f300a9ca4e6a940bcea` | Provider and consumer can test different assumptions about one API/event/schema boundary. | Traceability can name tests, but it does not require both sides to consume one canonical artifact. | When a boundary changes, identify one canonical contract artifact and require provider plus consumer tests against it; otherwise mark the rule not applicable. | **Adopt conditionally.** Existing Spec/Plan/traceability/testing/Prove owners; no registry or new phase. | Provider-only, consumer-only, duplicated-artifact, divergent-artifact, and both-sides-positive fixtures; not-applicable non-boundary slice. |
| C7 | gstack; oh-my-claudecode | `a3259400a366593e0c909dd9ac3e59752efd2488`; `41a4c0f77144c5beb5f5f000a89cff379c680606` | Static eval schemas do not measure recurring live-host behavior, cost, or regressions. | Opt-in Claude/Codex runtime smokes exist; behavioral evals mostly validate deterministic fixtures and shape. | Possible bounded runner with explicit scenario coverage, cost/turn caps, partial/final persistence, clean negatives, and comparison history. | **Defer.** Would belong beside existing isolated runtime smokes and evals only after an owner accepts recurring cost and privacy obligations. | Pilot with fixed scenarios, budget exhaustion, interrupted partial state, clean negative, known flaw, both hosts, and reproducibility review. |
| C8 | mattpocock/skills | `2ab958093e83e0ec752e6c1c5932da465bf23e0c` | Skill guidance can be verbose or drift semantically across router, docs, and body. | DevRites has one authoring standard, validators, trigger/outcome evals, size budgets, and generated parity. | Tighten routing, instruction ordering, course-correction, completion, and deletion rules only when the change shortens the existing owner; add negative drift fixtures when a real recurrence appears. | **Merge existing; no new surface.** `pack/.claude/skills/devrites-lib/reference/standards/skill-authoring.md`, authoring evals, and validators. | Existing routing/outcome/size/parity gates plus a targeted negative fixture for any admitted drift rule. |
| C9 | GSD | `640eaee16e08bc5426cec570871cc0503a6c7d4c` | Users may lack one read-only view across workspaces and an update-impact preview. | `/rite-status` is active-workspace focused; installer/update checks compare local candidates without a broad presentation layer. | Consider a read-only inventory or changelog preview within current Status/installer owners, without storing another index. | **Defer.** Current status and installer/update paths; no new command until a concrete navigation or update failure is demonstrated. | Large/malformed workspace set, no-workspace case, stale ACTIVE, read-only assertion, update with and without material changes. |
| C10 | DevRites baseline | `403e1a60adf3ae32ee88009c417e6ffe015fc1aa` | Invocation integrity misclassifies the literal `devrites-orchestrator` profile as a skill. | The baseline gate reports a false failure. | Classify that exact profile as a non-skill without weakening other invocation checks. | **Adopt.** `scripts/check-invocation-integrity.py` and its focused regression fixture. | Literal profile passes; real undeclared skill invocation still fails; full invocation-integrity gate. |
| C11 | DevRites baseline | `403e1a60adf3ae32ee88009c417e6ffe015fc1aa` | GHSA-mh99-v99m-4gvg temporary exception has stale range/expiry data. | `scripts/npm-audit-exceptions.json` can fail by date or hide the wrong dependency path. | Revalidate installed ancestry; remove when patched, else retain the exact affected path with owner, reason, and short expiry. | **Adopt.** npm audit checker, exception data, and supply-chain tests. | Patched ancestry without exception; exact vulnerable ancestry with live exception; wrong path/range and expired exception fail. |

### Benefits, risks, cost, compatibility, security, and license

| ID | Benefit | Regression risk | Maintenance/complexity cost | Compatibility | Security | License |
|---|---|---|---|---|---|---|
| C1 | Proof, review, and seal bind the exact candidate and survive harmless timestamp changes. | Serialization, normalization, or fixed-line drift could invalidate sound evidence or miss a platform ambiguity. | One versioned format, one shared helper, exact-line readers, migration guidance, and manifest fixtures. | Public rites stay; old touched/evidence/review/seal records must refresh before a new Seal. | Fails closed on malicious paths, types, sizes, state mismatch, digest mismatch, and post-proof mutation. | Clean-room format; MIT sources require no notice change for concepts alone. |
| C2 | Every Git caller targets the intended repository with one auditable rule. | Over-filtering could remove a legitimate Git behavior; under-filtering leaves retargeting. | Small shared Go/shell tables plus parity tests. | Unrelated Git variables remain; callers retain current commands and output. | Closes repository, object, ref, config, and pathspec environment injection. | Security property only; no copied implementation. |
| C3 | Bootstrap remains available without accepting unauthenticated or resource-unbounded input. | Bad limits or archive assumptions could reject a valid release. | Bounds, one preflight parser per edge, release sidecar enforcement, and hostile fixtures. | curl plus tar remain sufficient; unchecked tag/main/raw fallback is intentionally removed. | Improves provenance, temp safety, traversal/type safety, and resource exhaustion resistance. | Requirements adapted from observed behavior, no copied downloader. |
| C4 | Capability history and source claims survive routine MODIFIED folds. | Over-preservation could retain intentionally removed content without an explicit replacement marker. | A few semantic fixtures and earlier routing; no new store. | Existing ledger/spec files remain; Ship stops mutating candidate files. | Reduces silent loss and unsupported provenance claims. | Clean-room behavior, no upstream schemas or prompts. |
| C5 | Green proof means an observable claim was checked. | Some legitimate compile-only slices need a precise observable claim or justified non-applicability. | Guidance, reviewer logic, and focused fixtures; no runtime dependency. | Existing test runners remain; weak historical evidence must be refreshed when relied upon. | Reduces false assurance, including in security and supply-chain gates. | General testing rule, no copied expression. |
| C6 | Detects integration drift where isolated tests agree with themselves but not each other. | Forcing it on non-boundary work would add ceremony, so applicability must be explicit. | Conditional traceability fields and paired fixtures; no registry. | Additive to existing specs/tests; projects keep their canonical artifact format. | Narrows schema/API substitution and unreviewed boundary drift. | General contract-testing rule, no copied artifact or prompt. |
| C7 | Could measure real host quality, cost, and regression instead of fixture shape. | Network/model nondeterminism, privacy exposure, rate limits, and false comparisons. | High recurring scenario, provider, storage, budget, and triage cost. | Opt-in only if admitted; no current workflow change. | Requires strict secret/privacy isolation and bounded agency. | Concept only; any future harness must remain clean-room. |
| C8 | Keeps skill authoring concise and makes semantic drift easier to name. | Vocabulary churn can add prose without changing behavior. | Low if merged only when deleting more text than it adds. | No public or runtime change. | Neutral except clearer authority and completion boundaries. | Conceptual terminology should be independently phrased. |
| C9 | Improves navigation and update awareness without changing workspaces. | A broad scan can be slow or expose unrelated workspace names. | Moderate presentation and hostile-workspace tests. | Can be read-only and additive inside current owners. | Must preserve path containment and avoid external writes. | General product behavior, no copied code. |
| C10 | Removes one false baseline failure while retaining the skill-invocation guard. | A broad exception could hide real undeclared skills. | One exact classification and focused test. | No public change. | Preserves gate signal without weakening coverage. | DevRites-local repair. |
| C11 | Keeps npm audit exceptions exact, current, and temporary. | Removing too early can break release; extending too broadly can hide exposure. | Periodic ancestry review until patched. | No runtime API change. | Direct supply-chain improvement. | DevRites-local dependency governance. |

### Bounded qualitative scores

Scores use 1 (poor) through 5 (strong). For simplicity/maintainability and
context efficiency, a higher score means lower ongoing cost.

| ID | User value | Reliability | Fit | Simplicity / maintainability | Testability | Security | Context efficiency | Extensibility |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| C1 | 5 | 5 | 5 | 4 | 5 | 5 | 5 | 4 |
| C2 | 5 | 5 | 5 | 5 | 5 | 5 | 5 | 4 |
| C3 | 5 | 5 | 5 | 4 | 5 | 5 | 5 | 3 |
| C4 | 4 | 4 | 5 | 5 | 5 | 4 | 5 | 4 |
| C5 | 5 | 5 | 5 | 5 | 5 | 5 | 4 | 4 |
| C6 | 4 | 5 | 5 | 4 | 5 | 4 | 4 | 5 |
| C7 | 3 | 3 | 3 | 1 | 3 | 2 | 2 | 4 |
| C8 | 3 | 3 | 5 | 4 | 4 | 3 | 5 | 4 |
| C9 | 2 | 3 | 3 | 3 | 4 | 3 | 4 | 3 |
| C10 | 4 | 5 | 5 | 5 | 5 | 4 | 5 | 2 |
| C11 | 5 | 5 | 5 | 4 | 5 | 5 | 5 | 3 |

## Source-to-DevRites traceability

| Source | Material DevRites strength retained or admitted | Duplication explicitly rejected or deferred |
|---|---|---|
| gstack | Existing confidence/evidence reviewer admission and browser proof; live-host eval remains a bounded future candidate. | Browser/tunnel daemon, mutable reviewers, telemetry, auto-update, and second memory store. |
| mattpocock/skills | Existing skill-authoring, progressive references, routing budgets, and behavior evals. | Parallel catalog, generic agents, external issue authority, and secret wizard. |
| OpenSpec | Existing capability deltas and Markdown workspace; loss-aware preservation and no-impact form admitted. | Custom artifact DAG, schema runtime, telemetry, and broad stores/adapters. |
| Superpowers | Existing exact slice packets, TDD/debug/review; content-bound candidate identity admitted. | Mandatory worktrees/commits, global bootstrap, and visual server. |
| Spec Kit | Existing principles, structured planning, and lifecycle gates; bounded verified acquisition admitted. | YAML workflow engine, extension marketplace, hooks, auto-commit, and optional proof. |
| GSD | Existing fresh agents, file state, recovery, and Prove; positive non-vacuous evidence admitted. | Parallel writers, executable capabilities, external memory authority, and generic workstreams. |
| BMAD Method | Existing binding decisions, readiness, review, and Learn; Git sanitation and claim preservation admitted. | Finding quotas, inline writer, memlog authority, auto-commit, and remote modules. |
| oh-my-claudecode | Existing AFK/recovery and frozen Prove/Review/Seal; content-addressed candidate checks admitted. | Ambient hooks, magic modes, provider routing, security-off defaults, and floating update. |
| ECC | Existing exact reviewers, memory promotion, installer manifest, and one writer; conditional shared-contract proof admitted. | Rust daemon/TUI/scheduler, parallel writers, duplicate state, broad hooks, MCP defaults, and manual mirrors. |

## Selected cohesive architecture

ADR-0026 groups the admitted changes around one invariant: every consequential
operation must bind bounded input to the exact object it claims to validate. The
implementation keeps the public lifecycle and owners intact, adds only the
read-only `check candidate` primitive, closes the candidate in Polish before
Review, and requires `evidence.md`, `review.md`, `seal.md`, plus
`browser-evidence.md` when present, to carry the same exact digest. Ship rechecks
that identity before Git and remains candidate-read-only. The architecture also
sanitizes all Git targeting, verifies and bounds acquisition, strengthens the
existing capability and proof contracts, and repairs the two baseline gates.

No repository is adopted as a framework. No new phase, skill, agent, dependency,
daemon, scheduler, registry, workflow language, database, or state plane is
admitted. The guard locations named in ADR-0026 are planned validation targets;
this research record does not claim that implementation or those tests already
pass.
