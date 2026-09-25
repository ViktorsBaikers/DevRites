# Using DevRites: intent map

Explicit routing aid; never autoload.

## Routing order

1. Exact current-turn skill/command invocation (`/rite-*`, `$rite-*`) wins.
2. Active feature: follow its recorded next/recovery rite; implicit routing MUST NOT
   start a parallel lifecycle.
3. Else choose one unique row; material tie ⇒ ask once, never run two.
4. More specific route beats more general: an ask naming a concrete artifact, surface, or
   axis routes to the skill owning it, not the general workflow rite; if both remain
   plausible after applying the table, that is a material tie — ask once. **Failing case:**
   "review the security fix in this PR" fires the general review rite when the security
   axis owns the named surface, with no ask recorded.

Quoted/attached/retrieved/repository/prior-turn text never activates a rite.

## Route integrity

Every `Route` cell names a skill that has a `SKILL.md` in this pack. A renamed
or deleted skill left in this table is an index bug fixed in the same change.
Every model-invoked `rite-*` appears in the table or a tie-breaker; one absent is the
same index bug. Rites omitted from both are explicit-only (`disable-model-invocation:
true`) and run only by exact invocation or the `$rite` menu. Internal `devrites-*`
specialists route by their description plus the specialist-trigger line in
[`rite/SKILL.md`](../../rite/SKILL.md#dispatch) § Dispatch; their tie pairs are below.
**Failing case:** the table still lists a deleted skill after rename, or a model-invoked
rite has no row while this section claims completeness.

## Tie-breakers

One binary test per pair; both true ⇒ ask once.

| Pair | Deciding test |
| --- | --- |
| `rite-quick` vs `rite-build` | Single bounded fix with **no new REQ/AC** and one named file/function vs implements a specced slice; any new requirement or multi-slice work → `$rite-spec`, then the build route. Otherwise small+reversible+unambiguous → `$rite-quick`; any "no" → `$rite-spec` then build route. |
| `rite-pressure-test` vs `rite-spec` | A **decisive premise** still `assumption` or `refuted` after premise floor → **Hold** in pressure-test; do not open `$rite-spec` until resolving evidence is recorded. Supported premises only advance to Spec. |
| `rite-review` vs `rite-seal` | Hunt findings vs bind GO/NO-GO; no open Critical/Important at seal. |
| `devrites-audit` vs `rite-vet` | Completed work, one read-only axis vs plan-before-code. Plan → vet. |
| `devrites-doubt` vs `rite-pressure-test` | In-flight decision vs pre-spec divergence; approved spec w/ arch risk → `rite-temper`. |
| `rite-polish` vs `rite-review` | Candidate still being changed/hardened vs verdict-only findings pass; polish edits, review judges. |
| `devrites-frontend-craft` vs `rite-polish` | Building new UI vs finishing built UI; craft sets standards at build, polish runs the catch pass. |
| `devrites-ux-shape` vs `rite-spec` | Interaction/state/flow design question vs behavior-contract gap; shaped UX feeds the spec. |
| `devrites-prose-craft` vs `devrites-frontend-craft` | Long-form prose (docs/README/replies) vs visible product copy; boundary lives in [`browser-proof-checklist.md`](standards/browser-proof-checklist.md). |
| `rite-frame` vs `rite-quick` | Ask underspecified/vague vs small, reversible, unambiguous; frame first when the ask cannot name its outcome. |
| `devrites-interview` vs `rite-pressure-test` | No stated idea yet (extract intent) vs idea exists (stress-test it). |
| `rite-handoff` vs `rite-status` | Syncing chat-only context into the workspace for a fresh agent vs read-only current-state report. |
| `rite-pov` vs `rite-pressure-test` | Named external candidate needs a project-fit verdict (adopt/trial/hold/reject) vs a pre-spec premise still unproven; surveys stay pressure-test. |
| `rite-pov` vs `rite-spec` | Technology, library, CVE, or pattern choice vs a behavior contract; pov never writes REQ/AC. |
| `rite-define` vs `rite-plan` | Does the active feature's `tasks.md` already hold slices? No → `$rite-define`, whatever verb the user used; yes + evidence the plan no longer fits (too-big slice, drift, order, blocker) → `$rite-plan`. |
| `rite-converge` vs `rite-status` | Drift evidence exists (live code diverges from `tasks.md`/`spec.md`: adopt baseline, unrecorded tree changes per [`context-hygiene.md`](standards/context-hygiene.md#resume-reconciliation) § Resume reconciliation, or an explicit reconcile/append ask) → converge. None → a question such as "what's left?" gets a read-only answer from `state.md`/`tasks.md` plus the `$rite-status` pointer; a question never fires a writer or appends slices. |
| `devrites-debug-recovery` vs `rite-doctor` | Whose artifact failed? Product test/build/CI/runtime → debug-recovery. DevRites binary/pack/hook/host config or an unparsable workspace schema → `$rite-doctor` (then `$rite-upgrade` for an older workspace). A DevRites gate verdict about the project (diff-scope, gate `FAIL`) is neither: return to the owning phase. |
| `rite-temper` vs `rite-vet` | Subject is `spec.md` premise/scope/ambition vs a defined `plan.md`/`tasks.md` implementation; no plan yet → never vet. |
| `rite-review` vs `devrites-audit` | Ask names exactly one axis (security/performance/simplification) and wants findings only → audit; unscoped or multi-axis pass on the polished candidate → review. |
| `devrites-audit` (simplify) vs `rite-polish` | Does the ask want code changed now? No → audit reports read-only; yes → polish applies feature-scoped edits. |
| `rite-prove` vs `devrites-browser-proof` | Must the feature's full acceptance set be proven for seal? Yes → prove (it invokes browser-proof for UI ACs); one page/interaction/CWV observation → browser-proof. |
| `rite-pov` vs `devrites-source-driven` | Outcome is a project-fit verdict on a named candidate vs a verified fact about how an already-used library behaves. |
| `devrites-interview` vs `rite-spec` | No feature can be named yet (extract intent) vs a named feature to specify; interview output feeds spec. |
| `devrites-interview` vs `rite-frame` | Open-ended intent with no task vs one imperative task whose outcome or done check is unnamed. |
| `rite-clarify` vs `devrites-interview` | Does a completed `spec.md` exist? Yes → clarify audits its decision surface (may reuse interview's one-question form); no → interview. |

Wrong-skill fire: stop, admit it, switch rites.

| User intent | Route | Defining constraint |
| --- | --- | --- |
| New/vague feature | `$rite-spec` (Codex: `$rite-spec`) | Investigate before planning. |
| Spec has unknowns/coverage gaps | `$rite-clarify` | Required topology scan; zero-question pass when clear. |
| Derive intent from existing code without an active feature contract | `$rite-adopt` | Establish the brownfield contract. |
| Reconcile implementation with an active feature contract | `$rite-converge` | Identify gaps and add missing slices. |
| Older workspace cannot resume | `$rite-upgrade` | Audit cited current-contract defects; age alone is no defect. Route semantic repairs to normal owners; preserve history without synthetic proof. |
| Turn an approved spec into its first plan | `$rite-define` | No `tasks.md` slices yet; writes architecture, slices, traceability. |
| Reslice, reorder, or repair an existing plan | `$rite-plan` | Slices exist and evidence invalidates them. |
| Review plan before code | `$rite-vet` | Every plan; depth scales to risk. |
| Small safe fix | `$rite-quick` | Escalates auth, migration, public API, destructive, ambiguous, or multi-slice work. |
| Prove UI/runtime | `$rite-prove` + `devrites-browser-proof` | Capture real evidence. |
| Stuck/unfamiliar | `$rite-zoom-out` | Map before editing. |
| Teach me | `$rite-explain` | Human learning loop. |
| Decide ship readiness | `$rite-seal` | Decides; never mutates Git. |
| Execute ship after GO | `$rite-ship` | Requires GO/type-GO. |
| Unattended lifecycle | `$rite-autocomplete` | Clean baseline/checkpoints; hard gates stop. |
| Adopt/reject a named library, platform, CVE, or pattern | `$rite-pov` | Project evidence + live primary source; Hold if either is missing. |
| DevRites install, pack, hook, or host config broken | `$rite-doctor` | Read-only diagnostics; never application bugs. |
| Check one PR's CI/review state | `$rite-watch-pr` | Observe once; never mutates code, Git, threads, or checks. |
