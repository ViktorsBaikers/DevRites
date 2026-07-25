---
name: rite-spec
description: Spec new or high-risk behavior before code and write its `.devrites/work/<slug>/` workspace. Use for a feature/app, vague product idea, auth or migration work, or a public-API change. Not for approved-spec planning.
argument-hint: "<feature or idea>"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- Inspect the current `spawn_agent` role list. When the named `devrites-<role>` is exposed, dispatch it with `fork_turns="none"`; full-history forks inherit the parent type. The matching project contract is `.codex/agents/devrites-<role>.toml`.
- If a named role is not exposed, use generic `explorer` for every read-only role with `fork_turns="none"`. Tell it to read `.codex/agents/devrites-<role>.toml`, follow its `developer_instructions`, and execute the unchanged packet. Trusted `.codex/hooks.json` binds `agent_type=explorer` to the fail-closed reviewer read-only guard.
- For `devrites-slice-wright`, trusted `.codex/hooks.json` binds generic `worker` (`agent_type=worker`) to the active reconcile window and exact `.wright-allowlist`. Dispatch that worker with `fork_turns="none"`, tell it to read `.codex/agents/devrites-slice-wright.toml`, and execute the unchanged packet. Never create `.reconcile-inline` when this safe rung is available.
- A missing custom role is not evidence that spawning is unavailable. Only when the project hooks are unavailable or untrusted, no spawn primitive exists, or higher-priority policy rejects a safe spawn may the root run the documented discipline inline. Label it `independence: fallback`, never call it independent, create `.reconcile-inline` only for that path, and apply every fallback risk gate.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-spec: investigate and write the spec

Turn a request into a **fully covered, correctly placed `spec.md`** by investigating
the existing system and resolving every material gap found during authoring.
`$rite-clarify` audits the full topology before `$rite-define` plans it. **Do not write
a plan, tasks, or code here.** Those belong to `$rite-define` and `$rite-build`.

> **Use `$rite-quick` for a small change.** A typo, copy edit, config bump, or
> one-function fix does not need a full workspace and lifecycle. Run
> `$rite-quick <change>`. It returns here if the work touches auth, data, a migration, a
> public API, or more than one slice.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
Pull `documentation.md` via `Read`
when capturing significant spec decisions (why-not-what, ADR-style notes in `decisions.md`);
pull `principles.md` when the project has declared invariants (`.devrites/principles.md`): a
new spec must respect them, and a requirement that can only be met by breaking one is a blocking gap.
Pull `spec-grammar.md` and `devrites-lib/reference/workspace-artifact-schema.md` when writing
acceptance for a behavioral / high-risk requirement (auth,
data model, state machine, public API, money, migration): the structured `### Requirement:` /
`#### Scenario:` (SHALL · WHEN/THEN) form, lint-checked by `devrites-engine spec-validate`. Simple criteria
stay flat `AC-###` bullets; the grammar is opt-in by rigor, never forced. Use
[`reference/acceptance-criteria.md`](reference/acceptance-criteria.md) to keep each
criterion independently observable and binary.

## Operating rules (DevRites core)
- No silent assumptions · no guessing through confusion · prefer existing conventions ·
  ask the human when an answer changes scope, placement, data model, UX, security,
  migration risk, or acceptance.
- **Root authority:** the controlling chat asks every human question, makes decisions, and
  writes the workspace. Read-only evidence work follows the fresh-context contract in
  [`agents.md`](../devrites-lib/reference/standards/agents.md).
- **Author one section at a time.** Draft problem → goal → requirements → acceptance →
  edge cases, pausing after each section. If a section contains a contested requirement,
  boundary, or unstated assumption, apply a relevant technique from
  [`elicitation.md`](../devrites-lib/reference/standards/elicitation.md) before continuing.

## Workflow
0. **Read `.agents/skills/devrites-lib/reference/standards/core.md`:** the always-on operating rules and anti-rationalizations.
   Then run `devrites-engine preamble` for deterministic workspace orientation.
0a. **Check whether existing code needs adoption.** If this is an
   **existing codebase** that has **never been adopted** (no
   `.devrites/conventions.md`, no prior `.devrites/work`, `.devrites/features`, or `.devrites/archive`) the build has no
   conventions ledger, route through `$rite-adopt` **first** and pass `$ARGUMENTS` as its
   next objective. Adopt derives the baseline `spec.md`, seeds conventions, and proposes
   principles; `$rite-spec` only detects and routes. In **HITL**, present a ranked option:
   recommend adoption first and include a spec-only escape hatch. In **AFK**, when adoption
   is allowed, run `$rite-adopt` automatically. **Skip this check silently for greenfield or
   already-onboarded projects.** Never block a spec only because adoption is absent. Probe:
   ```bash
   if [ ! -f .devrites/conventions.md ] && [ ! -d .devrites/archive ] \
      && [ -z "$(ls .devrites/work .devrites/features 2>/dev/null)" ] \
      && [ -n "$(git ls-files 2>/dev/null | grep -vE '^\.(devrites|claude)/' | head -1)" ]; then
     echo "brownfield, not yet adopted → recommend $rite-adopt first (carry this idea as its next objective)"
   else echo "greenfield or already onboarded → continue spec"; fi
   ```
1. **Understand the request** (`$ARGUMENTS`). State the requested outcome and the
   underlying problem in one or two sentences.
   **Completion:** one sentence includes both.
1a. **Local dedupe.** Search local issues/PRDs and archived specs before creating a new workspace:
   ```bash
   devrites-engine spec-dedupe "$ARGUMENTS"
   ```
   If it finds a close match, ask the user: extend existing / adopt / new spec. Record the choice in
   `decisions.md` once the workspace exists. No match → continue silently.
2. **Investigate:** follow [investigation](reference/investigation.md) through its
   complete findings and done-when gate. Also discover the project's **test /
   build/typecheck/lint** commands, frontend/backend systems, and declared project guidance
   (`PRODUCT.md`, `DESIGN.md`, `CLAUDE.md`, `AGENTS.md`, and `.devrites/principles.md` when
   present).
   **Consult the capability ledger**, which records current system behavior
   ([`ledger.md`](../rite-ship/reference/ledger.md)): `devrites-engine ledger list` for the
   capabilities on record, then `devrites-engine ledger show <capability>` for any this feature
   touches. Also search prior decisions with `devrites-engine decisions search "<2-4 feature nouns>"`
   before asking the human to revisit a settled architecture, API, or auth choice. The
   ledger shows whether each requirement is new or changes existing behavior, which
   determines the delta kind in step 5.
   Identify proof constraints now: human-only credentials, unavailable environments, approval
   windows, or acceptance not observable through existing test/runtime/browser surfaces.
   Split independent placement, blast-radius, and external-fact questions into at most three
   bounded `devrites-evidence-scout` packets on one frozen baseline. Await and reconcile every
   cited dossier before step 4. The scout supplies facts only; it never asks the human or writes
   the spec.
3. **Gather design references when provided:** [references-intake](reference/references-intake.md).
   The human may attach screenshots, mockups, a Figma link, a video, links, or nothing.
   Skip this step when none are provided. Otherwise, **view or fetch**
   them, **save local files** into `.devrites/work/<slug>/references/`, and index them in
   `references.md` as target, constraint, or inspiration. Later phases honor that role
   rather than treating every reference as a fidelity target.
   **Completion:** every supplied reference is saved and classified, or absence is explicit.
3a. **Shape UX/UI before code when the feature is frontend**
   ([frontend-trigger](../rite-build/reference/frontend-trigger.md)). Apply
   `devrites-ux-shape` within the spec phase. It turns the
   references and spec into a feature-level **`design-brief.md`** (design direction, key
   states, interaction model, optional Figma/image visual-direction probe) that `$rite-build`
   targets for the build. In HITL it pauses for the human to
   confirm the direction; in AFK it asserts the best guess and logs it. Pure
   backend/data/CLI features skip this.
4. **Resolve human-owned gaps.** Recommend an option and let the human decide. Apply
   [question-protocol](reference/question-protocol.md), the shared
   [`afk-hitl.md` option-set and decision-ownership rules](../devrites-lib/reference/standards/afk-hitl.md#decision-ownership-search-before-asking),
   and [`interview-patterns.md`](reference/interview-patterns.md) for a vague ask. Every
   material dimension is resolved by a human pick or explicitly deferred as non-blocking;
   only genuinely reversible, low-impact details go to `assumptions.md`. A paper-only
   uncertainty may take one scoped `$rite-prototype` detour. A declared-principle conflict
   remains blocking until the human approves a recorded, scoped exception.
4a. **Build-interruption forecast.** Search first, then list and close foreseeable human needs:
   product/acceptance ambiguity, irreversible/external approval, or human-only access. Record
   owned prerequisites. Keep a build checkpoint only for unavailable pre-code evidence or a
   mandatory action-time approval. **Completion:** no foreseeable human choice is deferred.
5. **Create the workspace** + set `.devrites/ACTIVE` from
   [state-workspace](reference/state-workspace.md). Write every required artifact and
   conditional annex exactly from [spec-template](reference/spec-template.md), including its
   grammar/delta, coverage-seed, edge/prohibition, UI, and AI rules. Then refresh any managed
   project context block so `AGENTS.md` / `CLAUDE.md` point at the new active workspace:
   ```bash
   devrites-engine context sync || true
   ```
5a. **Check the spec prose** with [spec-checklists](reference/spec-checklists.md).
   Emit every applicable domain checklist and fix each CRITICAL by correcting the spec,
   never by softening the question.
6. **Run the complete readiness gate** at the bottom of
   [spec-template](reference/spec-template.md), then validate structure and ledger deltas:
   ```bash
   devrites-engine spec-skeleton ".devrites/work/<slug>"
   devrites-engine spec-validate ".devrites/work/<slug>" --against .devrites/specs
   ```
   **Do not run `devrites-engine analyze` in this phase:** `tasks.md` deliberately does not
   exist yet. `$rite-define` owns the first analyze pass after it writes the slices.
   Any failure blocks. The interruption forecast must be resolved, owned, or a justified
   action-time gate. Then write `Spec gate: passed <iso>`.
6a. **Review-before-code digest.** Before planning, render the compact human review:
   `Intent` (one sentence), `Done means` (top acceptance/scenario IDs), `Scope/risk` (what is in/out
   plus the hard gates), and `Build exactly this?` (yes → next phase; no → revise now). The digest
   is a view over `spec.md`, not a new artifact. **Stop** after the digest.

> **Mid-flight discipline.** Do not skip investigation, gap resolution, or placement
> decisions. See [`anti-patterns`](reference/anti-patterns.md).

## Output

**Progress first**: run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: spec ready for <slug>; placement decided and gaps closed.
Changed: spec.md, decisions.md, assumptions.md, questions.md, references/ <updated|n/a>
Evidence: checklists passed; grammar <valid | n/a flat acceptance>; design brief <path | n/a>
Open: <none | n non-blocking questions | Alternative: $rite-quick if express-lane eligible>; review digest: intent + done-means + scope/risk rendered
Next: $rite-clarify
Record: .devrites/work/<slug>/spec.md
↻ Hygiene: /clear before $rite-clarify; $rite-handoff if away > a few hours
```
If a workspace with the slug already exists, update its spec rather than overwriting it,
and **show the human a short diff of what changed** in `spec.md` (acceptance criteria added /
removed / reworded) before proceeding. A spec edit reviewed as a diff catches silent scope
drift that a full re-read buries; this is the spec-review view (`$rite-spec --review` renders
just the diff + the open-question delta, no re-investigation).

When the ask overlaps the ACTIVE feature (same intent, >50% scope), route to
`$rite-plan revise` (its gate decides) rather than minting a parallel workspace.
