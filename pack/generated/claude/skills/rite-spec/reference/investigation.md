# Investigation before specification

Understand the requirement, decide where it belongs, define the outcome, and identify
every issue and gap. Resolve material gaps with the user before the spec is ready. A
gap first found during `/rite-build` is a drift event.

Use a code-intelligence index if available (see
`../../devrites-lib/reference/standards/tooling.md`) for structural questions such as
where code lives, what calls it, and what it could break. With none present, use
Read/Grep/Glob. When a gap depends on an external fact, such as a standard, UX pattern,
or comparable product, **search the web if available** (brave MCP preferred;
`../../devrites-lib/reference/standards/tooling.md`). Cite the finding in the option
presented to the human.

## Check the archive first
Before external research, check whether the project already shipped related work. Search
the archive for the feature's key nouns. A hit may indicate an extension, conflict, or
replacement and provides prior decisions:
Use native file search over `.devrites/archive/*/{spec,decisions}.md`.
- **Overlap found** → read the overlapping `spec.md` + its `decisions.md`, then put it to
  the human as a ranked option (*extend the shipped feature* · *this supersedes it* ·
  *genuinely distinct*) same option-set contract as a gap.
- **No archive / no hit** → skip silently; never block a spec on absent history (the
  brownfield / principles no-op discipline).

## Produce these findings (write into spec.md)
1. **The request and problem:** restate the goal and the problem behind it (people ask
   for "a dashboard" when they want an answer to a question). Who hits it, how often,
   what they do today instead.
2. **Current behavior:** how it works today, or what's absent. Read the actual code and
   flows; don't assume.
3. **Placement**
   - Which module / layer / file / component should own this; the right seam.
   - Existing patterns/components/utilities to **extend or reuse** instead of duplicating.
   - **Integration points**: callers and dependents, the data it reads/writes, the
     APIs/events/contracts it touches (interface analysis: how it interacts with the
     rest of the system).
4. **Outcome:** the result and how to observe it (feeds
   success + acceptance criteria).
5. **Issues:** conflicts with existing code/UX/data/permissions, constraints, and
   anything that makes the obvious approach wrong. Each issue gets a disposition.
6. **Gaps:** unknowns, ambiguities, undecided behavior, missing inputs. **Every gap
   becomes a question** (next section).
7. **Blast radius:** what this change could break (use the code graph's impact/callers).
   Informs risks, test strategy, and rollback.
8. **Human prerequisites:** credentials, accounts, approval windows, or irreversible
   action-time decisions the acceptance path requires. Separate these from agent-owned
   implementation and diagnostic work.

Design/reference materials the human supplies: see [references-intake](references-intake.md).

## Gap analysis (present → desired)
State the present and desired states. Their delta defines the work; unknowns in that
delta are gaps. Resolve them before `/rite-define`. Mark each gap inline with
`[NEEDS CLARIFICATION: question]`.

## Present gaps and issues as options
For each material gap or issue that changes scope, placement, data model, UX, security,
migration risk, or acceptance, **ask the human** one gap at a time with a ranked
option set with the recommended option **first and marked `(Recommended)`** plus an escape
hatch (via `AskUserQuestion` in HITL):
```
<gap/issue stated in one line>. Why it matters: <...>
1. <recommended option> (Recommended) — <implication / where it places the work; cite any web/docs finding>
2. <alternative> — <implication>
3. <alternative> — <implication>
4. Something else — I'll describe it
```
Investigate and recommend, but do not settle a material decision. High confidence may
make the answer a one-pick confirmation; it does not change the decision owner.
Only a **genuinely reversible, low-impact** gap is decided and recorded in `assumptions.md`
without asking. Full render contract + AFK behaviour: [`afk-hitl.md`](../../devrites-lib/reference/standards/afk-hitl.md).

## Done when
- The shipped archive was checked for prior art; any overlap was surfaced to the human.
- The problem, current behavior, placement, and outcome are written down.
- Every issue has a disposition; every material gap is resolved **by a human pick** from its
  option set (or explicitly deferred as non-blocking), not settled silently on your confidence.
- Every foreseeable build-time human prerequisite is resolved, assigned, or justified as an
  action-time gate; objective repair/retry work is not disguised as a question.
- No blocking `[NEEDS CLARIFICATION]` remains. The spec covers the gaps found during
  authoring and records correct placement. This is the `/rite-spec` readiness gate
  before `/rite-clarify` performs the systematic topology scan.
