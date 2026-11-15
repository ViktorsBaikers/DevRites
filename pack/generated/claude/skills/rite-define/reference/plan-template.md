# `plan.md` template

Build plan for `spec.md`; tech choices live here, not in spec. `/rite-plan` revises on
new evidence.

Canonical references replace optional prose copies only. Preserve required verbatim
spec Prohibitions in slices, `Dependencies`/`depends_on` equality, `Proves` quotations,
and runnable gate `CHECK`/`CWD` equality with test-plan preflight Command/Cwd.
Resolve every reference to its owning section/ID; a missing or ambiguous target is
a gap, never permission to drop the obligation.

For canonical future-pool plans, write one standalone declaration below the plan identity:
`Consumed planning inputs: none` or `Consumed planning inputs: <slug>, <slug>`. List
unique related workspaces whose implementation design is actually consumed, including
nonadjacent producers; roadmap scope ownership alone does not count. Native reviewers
verify the declaration under the
[consumed-input schema](../../devrites-lib/reference/workspace-artifact-schema.md#consumed-planning-inputs),
which owns Vet digest bindings and drift handling. Required design not yet available
parks the affected source under
[parallel-workspaces policy](../../devrites-lib/reference/parallel-workspaces.md),
never an invented contract. Keep explicit `Workspace dependencies` and BUILT producer
checkpoints separate; explicit separate-session lookahead retains its predecessor binding.

```markdown
# Plan: <Feature>
Spec: ./spec.md   Decision coverage: ./decision-coverage.md   Date: <date>

## Summary
<primary requirement + chosen approach, 1–2 sentences>

## Technical context
- Language/runtime/version: <...>
- Frameworks/libraries: <...>
- Storage/data: <...>
- Test commands: <discovered runner commands; discovery contract: .claude/skills/rite-prove/reference/test-command-discovery.md>
- Target/platform: <...>; constraints: <exact spec.md section/IDs; implementation consequences here>
- Approach-affecting product/constraint unknowns route to `/rite-clarify`;
  `[NEEDS CLARIFICATION]` blocks approval.

## Applicability and system ownership

Validate `spec.md` applicability against live evidence. Each `applies` row includes its
owner's **Required plan output** here or in architecture:

- topology → root/deployable/owner table;
- data/migration/tenant/retention → invariant/recovery table;
- API/webhook/queue/job/cache/service → boundary table;
- security/UI/delivery → focused standard + proof owner.

Record feature owner, failure/recovery, slice, and evidence. Changed `not applicable`
uses Spec Drift Guard.

## Global constraints
Reference exact `spec.md` section/IDs for constraints inherited by every slice;
record their implementation consequences here, not a prose copy of the requirements.

## Approach
State the implementation strategy and reference exact `decisions.md` `DEC-###` entries
for rationale. For costly/hard-to-reverse boundaries, models, contracts, or dependencies,
those entries compare ≥2 `Option · Drivers · Trade-offs · Consequence`.

## Slice strategy
For explicit workflow-only scope, use zero product slices and the existing empty
candidate manifest (`No project files.` plus `## Touched files`). Map AC/REQ to
Workflow Artifact behaviors and proof, not invented slices; follow
[`workflow-artifacts.md`](../../devrites-lib/reference/standards/workflow-artifacts.md#workflow-only-planning).
The following slice strategy applies to product work.

Reference exact `tasks.md` `SLICE-###` entries for vertical increments, AC coverage,
order, and dependencies; explain risk-first strategy within dependency tiers here.
Wide refactors use expand → green migrate batches → contract, or an integration
branch + final verify slice.

## Architecture admission

Promote to ADR only **irreversible cross-boundary** choices (public contract,
security invariant, migration that cannot roll back). Reversible config, helper
placement, library version, or env default stays in `plan.md` / `decisions.md` as
implementation-local or planning-owned — not architecture.

**Failing case:** "Use env `FOO=bar` as default" recorded under Architecture decisions
→ Vet requests downgrade to implementation-local horizon with observable trigger.

## Architecture decisions
Reference exact `decisions.md` `DEC-###` entries; that log owns decisions and rationale.
Record implementation consequences here. Prefer reuse and invariants over scaffolding.
Medium+ entries add `Binds:`/`Prevents:`. Interfaces name invariants, I/O,
ordering/idempotency, errors, versioning, config, and budgets.

## Decision horizons

Classify every known unresolved/action item. Keep `HZN-###` across replans; never delete
silently. Resolution/supersession needs evidence.

| Horizon | Includes | Required disposition |
|---|---|---|
| Human-owned blocker | Product, acceptance, policy, irreversible risk | `/rite-clarify`; unresolved blocks approval/readiness. |
| Planning-owned | Architecture, boundary, dependency, sequence, proof | Resolve from source. If executable evidence is necessary, plan a bounded risk spike with discriminating criteria + fallback branches. |
| Implementation-local | Reversible detail unknowable before code/tests: exact helper name, final query shape after live evidence, or a refactor that may disappear | Owner slice + observable trigger + bounds/fallback + resolution proof; never “ask later.” |
| Action-time checkpoint | Approval/evidence mandatory when acting | Owner + gate/signal + bounds/fallback + proof; cannot hide an earlier decision. |

Never local (SSOT; mirrored by rite-vet review-axes §0.8): public contracts, security/data invariants, acceptance, migration/rollback,
dependency choice, cross-slice interfaces. Output per item: `HZN-### · item · horizon ·
owner/slice · evidence · trigger/checkpoint · bounds/fallback/branches · resolution proof ·
status`. Only a complete sweep may write `Decision horizons: none — <evidence>`.

## Shared contract proof
Changed provider/consumer boundary: one table:

| Boundary | Canonical contract artifact | Provider-side asserting test | Consumer-side asserting test |
|---|---|---|---|
| <provider → consumer surface> | <existing schema/type/fixture path> | <test path + assertion> | <test path + assertion> |

Tests consume the same artifact. Reuse an existing canonical artifact; no ceremony-only
artifact. Without boundary change write exactly:

Shared contract impact: none — <specific justification>

## Dependency graph
Reference exact `tasks.md` slice entries for local prerequisites and explicit
`Workspace dependencies` under the
[canonical slice grammar](../../devrites-lib/reference/workspace-artifact-schema.md#canonical-slice-grammar);
tasks owns slice dependencies. Record non-code prerequisites and deployable order here:
contract/schema/config → old/new app/worker → migration/backfill → exposure → removal.
Name shared mutable resources that force serialization or explicit isolation.
Each temporary flag, shim/adapter, dual read/write, retained old column/endpoint, or deprecated
alias the plan adds names its removal `SLICE-###` here, or a Non-goal continuation
(`/rite-spec <slug>-N`) with measurable trigger + owner; otherwise Vet returns NEEDS REPLAN.

## Implementation order
Reference ordered `tasks.md` slice IDs; retain risk-first rationale within dependency
tiers here rather than a second authoritative slice order.
`MVP cut: after SLICE-00N — <what ships if we stop here>` — the earliest coherent,
shippable prefix. Every acceptance criterion above it is proven there, with no dependency
below it.

## Validation strategy
Name test/build/browser proof points. UI uses `design-brief.md` targets. Each proof names
portable command/cwd/signal, prerequisites, and mutable provenance inputs.

**Key links** — assembled wiring, one row each:
`<from> → <to> via <mechanism>`. List wiring no slice test catches; `/rite-prove` walks it.
`Key links: none` is deliberate.

## Complexity & deviations gate
Justify deviations from reuse, simplicity, scope, dependency/design-system rules;
otherwise simplify.
| Deviation | Why needed | Simpler option rejected because |
|-----------|-----------|---------------------------------|
| <e.g. new dependency X> | <reason> | <why the in-repo option won't work> |

## Rollback
Every risky step (migration, destructive write, flag widening, contract change) names its
backout before Build: **trigger** (what aborts it), **procedure** (down-migration / flag
off / revert / restore), and **rollback-verification proof** (command + observed state).
"Revert if needed" is not a rollback plan.

## Scope boundaries
Untouched scope; reference exact `spec.md` sections/IDs for “Ask first”/“Never do.”
Record implementation consequences, not optional prose mirrors.

## Source docs needed
Framework/library sources (triggers source-driven).

## Readiness gate  *(must pass before /rite-build)*
- [ ] `decision-coverage.md` says `Decision coverage: CLEAR`
- [ ] Every AC maps to a slice
- [ ] Dependencies are acyclic/risk-first
- [ ] Applicability matches live evidence; outputs name owner, recovery, slice, proof
- [ ] `MVP cut` is shippable/self-contained: ACs proven, no dependency below
- [ ] Deviations are justified
- [ ] Destructive/migration steps have rollback (trigger + procedure + verification proof); spec Prohibitions carry into slices verbatim
- [ ] Each `Mode: HITL` slice has `Gate`, `SLA`, `Checkpoint`
- [ ] Human choices resolved; checkpoints need unavailable pre-code evidence/action approval
- [ ] All horizon items remain; blockers/planning items resolved or validly spiked;
      local/checkpoint register fields fully populated
- [ ] Proof command/cwd/prereqs/provenance portable + preflightable
- [ ] UI slices name `Design brief states` + binary `Visual acceptance`
- [ ] `Key links` cover cross-slice wiring (or `none`)
- [ ] Contracts name producer, consumer, invariants/errors/order, proof
- [ ] Deploy/config/schema/app/worker/flag order is safe across old/new or N/A
- [ ] Every temporary flag/shim/old path names its removal slice or continuation + trigger
- [ ] `Shared contract proof` has one consuming table or specific no-impact
- [ ] Wide refactor is expand → green migrate batches → contract, or branch + final verify
- [ ] No hidden `Gate: blocking` dependency behind an AFK slice
```
