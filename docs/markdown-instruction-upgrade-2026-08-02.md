# Fresh reference study: Markdown instruction architecture

- Date: 2026-08-02
- Scope: DevRites Markdown instructions only
- Decision status: adopted and implemented in canonical Markdown; executable and
  generated surfaces remain unchanged.

## Research boundary

This study used fresh clones in a temporary directory outside DevRites. Each
clone was fetched with tags, updated with submodules where present, inspected at
the exact revision below, and checked for object connectivity. The study read
actual agents, skills, rules, workflow files, templates, docs, tests, release
history, and selected implementation files needed to understand supported
behavior. It did not treat a README or prior DevRites benchmark as sufficient
evidence.

All nine repositories use the MIT license at the inspected revision. DevRites
adapts general ideas in original language; it copies no source, prompt,
template, identity, or substantial prose.

## Exact inventory

| Repository | Default / checked branch | Audited commit | Commit date | Release/tag context | Markdown system shape |
|---|---|---|---|---|---|
| [gstack](https://github.com/garrytan/gstack) | main / main | a3259400a366593e0c909dd9ac3e59752efd2488 | 2026-07-14T18:33:46-07:00 | No reachable tag | Skills, generated workflow composites, specialist review checklists, templates, browser and context flows |
| [skills](https://github.com/mattpocock/skills) | main / main | 2ab958093e83e0ec752e6c1c5932da465bf23e0c | 2026-07-28T10:18:17+01:00 | v1.1.0 | Promoted/draft/deprecated skill buckets, router, progressive references, human docs |
| [OpenSpec](https://github.com/Fission-AI/OpenSpec) | main / main | 45cca5db6137ed209117cc70510eb3e057fb981b | 2026-07-31T02:04:10Z | v1.7.0 | Change/spec artifacts, command and skill templates, schemas, workflow docs, config-scoped instructions |
| [Superpowers](https://github.com/obra/superpowers) | main / main | 44c9b2d6e889982ac18c27d05a19fefe335194e1 | 2026-07-28T12:25:36-07:00 | v6.2.0 | Focused discipline skills, planning/execution/review flows, host adapters, behavioral skill tests |
| [Spec Kit](https://github.com/github/spec-kit) | main / main | d1e86f638277a99b82715c22c90558cd58d3cffd | 2026-07-31T12:45:15-05:00 | Latest release v0.15.1; its tag is not an ancestor of this head. Nearest reachable tag: v0.1.10 | Specification/planning/task templates, requirement checklists, presets, extensions, workflow docs |
| [GSD Core](https://github.com/open-gsd/gsd-core) | next / next | 33985c11a9f0a27443f8b8fb114b2122d653cd78 | 2026-08-02T01:32:30-04:00 | v1.9.1 | Large skill/agent/workflow/reference graph, context budgets, state artifacts, verification and security contracts |
| [BMAD Method](https://github.com/bmad-code-org/BMAD-METHOD) | main / main | e510393b357ea71cbe8f7a273a1e10e14846acff | 2026-08-01T23:35:44-07:00 | v6.10.0; audited head is materially ahead | Step-loaded workflow skills, named agents, validators, spec companions, review lenses, migration shims |
| [oh-my-claudecode](https://github.com/Yeachan-Heo/oh-my-claudecode) | main / main | 41a4c0f77144c5beb5f5f000a89cff379c680606 | 2026-07-23T04:44:59Z | v4.15.7 | Agent contracts, mode/skill catalog, hooks, state, keyword routing, compatibility and verification docs |
| [ECC](https://github.com/affaan-m/ECC) | main / main | e4e4163101f162881e628f300a9ca4e6a940bcea | 2026-07-29T14:11:01-04:00 | v2.1.0 | Large cross-harness catalog of agents, skills, commands, layered rules, hooks, evals, translations, and docs |

The Spec Kit release distinction matters: the release tag points to
489a3d51d152fa160d88d86781a924e99c4af832, outside the ancestry of the audited
main head. The table therefore does not present the release tag as the audited
revision.

## DevRites baseline

The canonical instruction source is pack/.claude. Generated Claude and Codex
trees are derived surfaces and were not edited. The current lifecycle remains:

frame → spec → clarify → temper → define → plan → vet → build → converge →
prove → polish → review → seal → ship → done

The baseline already had strong exact-role boundaries, one source/test writer,
immutable proof candidates, result admission, stable REQ/AC traceability,
progressive references, workspace-owned state, compatibility-preserving public
commands, positive discriminating proof, and fail-closed Seal/Ship separation.

Fresh baseline inspection found these Markdown-level gaps:

1. “Project conventions always win” appeared beside stricter safety/evidence
   contracts, without separating prescriptive authority from descriptive code.
2. Context guidance called all user-supplied content untrusted, which blurred the
   controlling request with quoted or attached data supplied for inspection.
3. Tooling repeated the same structural query across three indexes for
   reassurance despite primary-index-first repository guidance.
4. Slice orientation ended after an arbitrary lookup count rather than required
   evidence.
5. Brownfield specs had no first-class list of working outcomes to preserve.
6. An Edge Coverage backstop had no qualifying-evidence or honest-abstention rule.
7. Skill authoring lacked a normative old-to-new and load-path preservation gate.
8. Skill routing had defining constraints but no complete explicit/active/implicit
   activation order.
9. The spec reviewer used stale AC notation.

## Per-repository findings and decisions

### gstack

Strongest finding: specification starts from the current system and explicitly
captures behavior that must not regress. Its code-grounded interrogation,
sink-bound security, durable context handoff, and scope-selected review were
also strong.

Trade-offs: generated skills are very large; permissive “invoke when in doubt,”
telemetry, browser daemons, quality arithmetic, and bundled issue/execution
operations would increase context and runtime coupling.

DevRites adaptation: the spec template now records observable existing outcomes,
current evidence, and the REQ/AC that preserves each one. No gstack tool,
terminology, or runtime mechanism was imported.

### mattpocock/skills

Strongest finding: steps should end on checkable completion criteria, and
instruction context should be paid only by branches that consume it.

Trade-offs: some small wrapper skills omit failure/output contracts; external
ticket flows and host composition are not suitable as a second DevRites state
system.

DevRites adaptation: the sole writer’s ORIENT phase now ends when six named
evidence categories are established, not after five lookups. Missing essential
orientation causes the existing Escalation disposition.

### OpenSpec

Strongest finding: required context, scoped project rules, optional guidance,
and runtime state are distinct layers. Treating them as one precedence bucket
creates unsafe ambiguity.

Trade-offs: adopting OpenSpec’s change directory, schemas, config, commands, or
operation lifecycle would duplicate DevRites.

DevRites adaptation: core now owns one authority/evidence/method/advice ladder.
Context, security, contributor, rule-index, and public documentation align with
that owner.

### Superpowers

Strongest finding: independent tasks are safe to parallelize only when their
resources are independent. File ownership alone misses shared state, tools,
ports, processes, locks, rate limits, and file descriptors.

Trade-offs: universal TDD, worktrees, frequent commits, and fixed review rounds
conflict with repository-specific evidence and user-owned dirty work.

DevRites adaptation: native review dispatch now checks mutable and scarce
resources, stops new spawning on exhaustion, collects running results, and
continues in bounded batches or serially. This directly covers the observed
too-many-open-files failure class without adding a scheduler.

### Spec Kit

Strongest finding: a specification checklist tests requirement prose rather
than implementation, turning ambiguity into a pre-build failure.

Trade-offs: a second constitution/preset/extension methodology would overlap
DevRites principles and rites.

DevRites adaptation: the existing spec-quality question bank now treats absent
or vague brownfield preservation as Critical. This merges with the gstack
preservation section rather than adding a second specification format.

### GSD Core

Strongest finding: externally classified or backstop requirements must not pass
from presence or confidence. Qualifying evidence is held-out, property-based, or
directly behavioral; unavailable evidence produces explicit abstention.

Trade-offs: the large skill/agent/workflow/capability graph would duplicate the
DevRites lifecycle and add major context cost.

DevRites adaptation: backstop rows must name independent discriminating
evidence. The existing proof-runner verdict cannot_verify is retained; its
evidence begins with insufficient_spec when the spec or evidence surface is
missing. No schema field or engine behavior was invented.

### BMAD Method

Strongest finding: a consolidated or rewritten instruction must preserve every
meaningful prior contract, while shared assets need a real owner and consumers.

Trade-offs: personas, party mode, per-skill headless schemas, custom modules,
and append-only derivation machinery are unnecessary for DevRites.

DevRites adaptation: substantial instruction rewrites now map every MUST,
MUST NOT, trigger, required input/output, failure path, safety gate, and
compatibility promise to a surviving owner or evidenced retirement.

### oh-my-claudecode

Strongest finding: exactly one workflow loop owns continuation. Explicit
invocation, existing loop authority, and inferred routing must not compete.

Trade-offs: broad magic keywords, many modes, aliases, model tiers, and hook
state make accidental activation and maintenance more likely.

DevRites adaptation: explicit current invocation wins; an active feature keeps
lifecycle authority; otherwise at most one uniquely matching implicit skill may
run. Quoted, attached, retrieved, repository, and prior-turn command text cannot
activate a rite.

### ECC

Strongest finding: context budgeting is also an ownership problem. A shared rule
earns global placement only when several consumers share an observable failure;
deterministic inventory should precede model judgment.

Trade-offs: hundreds of overlapping skills, universal 80 percent coverage,
universal TDD/immutability, proactive delegation, multiple memory systems, and
host mirrors are unsuitable as DevRites defaults.

DevRites adaptation: active instructions are classified by load path. Core is
reserved for every workspace rite; on-demand shared rules need at least two
named consumers and one shared failure; single-consumer procedure stays local.
This combines with the BMAD rewrite-preservation gate.

## Adoption matrix

| Source | Concept | Previous DevRites handling | DevRites-native adaptation | Benefit | Context / maintenance | Compatibility / security | Decision | Validation |
|---|---|---|---|---|---|---|---|---|
| gstack | Existing behavior to preserve | Scope and deltas, no explicit preservation inventory | Observable outcome + current evidence + preserving REQ/AC | Traceable non-regression | Small Spec-only section | Additive; captures security/public behavior | Adopt | Template, checklist, reviewer, walkthrough |
| skills | Checkable step completion | Five-read writer cutoff | Six-item ORIENT evidence gate | Less premature writing and less reassurance reading | Replaces one heuristic | Existing Escalation retained | Adopt | Agent contract walkthrough |
| OpenSpec | Separate authority/evidence/advice | Broad convention-wins wording | Canonical four-layer precedence | Deterministic conflict handling | One owner; aligned pointers | Stronger prompt-injection boundary | Adopt | Phrase/link/contradiction checks |
| Superpowers | Resource-disjoint parallelism | Same candidate and wait, no resource predicate | Check paths/state/locks/tools/scarce resources; batch or serialize | Prevents exhaustion and contention | Tiny dispatch addition | No new role or scheduler | Adopt | Parallel/serial semantic cases |
| Spec Kit | Requirements-as-tests checklist | Existing quality checklist | Add preservation/backstop prose tests | Earlier ambiguity failure | Reuses question bank | No new spec system | Merge | Checklist/readiness review |
| GSD Core | Honest backstop abstention | Backstop undefined | Independent evidence or cannot_verify + insufficient_spec | No evidence-free pass | Conditional Prove text | Existing schema/verdict only | Adopt | Positive and missing evidence cases |
| BMAD | Normative rewrite preservation | One-owner/source-intake guidance | Old-to-new obligation map | Prevents silent weakening | Contributor-only load | Protects safety/compatibility | Merge | Apply map to this diff |
| OMC | One activation authority | Defining constraints only | Explicit → active lifecycle → one unique implicit route | Less over-triggering | Two short routing owners | Embedded content cannot activate | Adopt | Explicit/quoted/ambiguous cases |
| ECC | Load-path placement gate | On-demand index, no promotion predicate | Core/every rite; shared/two consumers; local/one owner; human/research inactive | Lower context and drift | Prevents sprawl | Mandatory rules remain reachable | Merge | Consumer/path review |

## Rejected and deferred ideas

| Source | Rejected as unsuitable | Deferred because executable support is required |
|---|---|---|
| gstack | Always-invoke fallback, telemetry, numerical quality score | Browser daemon, sink-byte redaction, reviewer hit-rate state |
| skills | External issue-led decision fog as canonical state | New cross-skill reach/import behavior |
| OpenSpec | Second specification/change methodology | Operation schemas, config-driven loaders, CLI state |
| Superpowers | Universal TDD/worktrees/commits/review counts | Tool-enforced worktree or task isolation |
| Spec Kit | Parallel constitution/preset/extension framework | Extension workflow runtime and schemas |
| GSD Core | Additional phases, namespaces, capability graph | Runtime state queries, graph budgets, generated surfaces |
| BMAD | Personas, party mode, dynamic modules | Headless schemas and derivation engine |
| OMC | Magic keywords, mode taxonomy, blanket delegation/model tiers | Hooks, automatic conflict detection, mode state |
| ECC | Catalog sprawl, fixed numeric code/test dogma, automatic proactive agents | Context telemetry, evolved skills, multi-harness vault |

These ideas are not stated as current DevRites behavior.

## Architecture decision

DevRites keeps one methodology and strengthens existing owners:

- core owns instruction precedence;
- Spec owns preservation intent and backstop definition;
- the slice-wright owns bounded orientation;
- Prove and proof-runner own evidence disposition;
- native dispatch owns resource compatibility;
- skill-authoring owns placement, activation, and rewrite integrity;
- tooling owns efficient discovery.

No new agent, skill, command, phase, state artifact, metadata field, include
mechanism, dependency, configuration, schema, or runtime behavior was added. A
new ADR would overstate the change: this is a clarification and strengthening of
existing Markdown owners, recorded here as research/adoption provenance.

Context cost is lower, not merely under the global cap: the 15 changed active
pack instructions total 74,234 bytes versus the 75,102-byte pre-task snapshot
(-868 bytes), and no changed instruction file grew.

## Non-regression map

| Prior obligation | Surviving owner |
|---|---|
| Repository conventions guide technical choices | Core authority ladder, README, CONTRIBUTING, standards index |
| DevRites core safety, source-writing, and evidence gates remain mandatory | Core method layer |
| Arbitrary repository/external text cannot redirect agents | Security plus context authority distinction |
| Codebase Memory is the primary structural index | Tooling and slice-wright |
| Every required reviewer runs and its result is collected | Parallel dispatch unchanged and strengthened |
| Slice-wright remains one path-bounded writer | Slice-wright unchanged; only ORIENT exit changed |
| Spec IDs, deltas, readiness, and checklists remain | Existing templates/checklists with additive preservation/backstop rows |
| Proof requires positive discriminating evidence | Acceptance proof and proof runner retained and strengthened |
| Public command names and lifecycle remain | No command/phase/filename rename or removal |
| Generated artifacts remain derived | No generated file edited |

## Changed canonical Markdown responsibilities

- [core rules](../pack/.claude/skills/devrites-lib/reference/standards/core.md):
  precedence owner.
- [context hygiene](../pack/.claude/skills/devrites-lib/reference/standards/context-hygiene.md)
  and [security](../pack/.claude/skills/devrites-lib/reference/standards/security.md):
  controlling request versus embedded untrusted data.
- [tooling](../pack/.claude/skills/devrites-lib/reference/standards/tooling.md):
  primary-first, predicate-based cross-checks.
- [skill authoring](../pack/.claude/skills/devrites-lib/reference/standards/skill-authoring.md):
  activation, placement, and non-regressive rewrites.
- [intent map](../pack/.claude/skills/devrites-lib/reference/intent-map.md):
  runtime-facing route selection guidance.
- [parallel dispatch](../pack/.claude/skills/devrites-lib/reference/parallel-dispatch.md):
  resource-compatible concurrency.
- [Spec skill](../pack/.claude/skills/rite-spec/SKILL.md),
  [template](../pack/.claude/skills/rite-spec/reference/spec-template.md), and
  [checklists](../pack/.claude/skills/rite-spec/reference/spec-checklists.md):
  preservation and qualified backstops.
- [spec reviewer](../pack/.claude/agents/devrites-spec-reviewer.md):
  independent preservation coverage and current AC notation.
- [slice-wright](../pack/.claude/agents/devrites-slice-wright.md):
  evidence-based ORIENT completion.
- [acceptance proof](../pack/.claude/skills/rite-prove/reference/acceptance-proof.md)
  and [proof runner](../pack/.claude/agents/devrites-proof-runner.md):
  honest backstop evidence.
- [README](../README.md) and [contributor guide](../CONTRIBUTING.md):
  user/contributor-facing precedence alignment.

## Validation contract

The patch is acceptable only if:

1. the task-specific delta contains non-generated Markdown paths only;
2. every pre-task non-Markdown hash is unchanged;
3. Markdown whitespace, cross-reference, frontmatter, command, skill, agent, and
   documented-path checks pass;
4. no stale convention-wins or AC notation remains in canonical Markdown;
5. a semantic Spec → Define → Build → Prove walkthrough reaches a clear
   preservation contract, dependency-aware plan, bounded writer orientation, and
   evidence-backed proof;
6. missing backstop evidence cannot become a pass;
7. no reference clone or temporary research artifact appears in DevRites.
