# Investigation: understand deeply before specifying

The spec is only as good as the investigation behind it. Goal: understand the
requirement **completely**, decide **where it correctly belongs**, name what it
**resolves**, and surface every **issue and gap**, then drive the gaps to closure with
the user so the spec ships fully-covered and correctly-placed. A gap found here is cheap;
a gap found in `/rite-build` is a drift event.

Use a code-intelligence index if available (see
`../../devrites-lib/reference/standards/tooling.md`) for the structural parts below; it answers "where does this
live / what calls it / what would it break" far more cheaply and accurately than reading files.
With none present, fall back to Read/Grep/Glob. When a gap turns on a fact the codebase can't
answer: a standard, a prevailing UX pattern, how comparable products solve it: **search the
web if available** (brave MCP preferred; `../../devrites-lib/reference/standards/tooling.md`) and cite the finding in
the option you present, so the human's pick is informed rather than guessed.

## Prior art: check our own archive first (cheap, silent when empty)
Before investigating outward, check whether we already shipped this. Grep the shipped
archive for the feature's key nouns: a hit means an extension, a conflict, or a re-spec of
solved work, and you inherit its decisions instead of re-deriving them:
```bash
devrites-engine archive-search "<key nouns>" 2>/dev/null \
  || grep -rliE '<noun1>|<noun2>' .devrites/archive/*/spec.md 2>/dev/null
```
- **Overlap found** → read the overlapping `spec.md` + its `decisions.md`, then put it to
  the human as a ranked option (*extend the shipped feature* · *this supersedes it* ·
  *genuinely distinct*) same option-set contract as a gap.
- **No archive / no hit** → skip silently; never block a spec on absent history (the
  brownfield / principles no-op discipline).

## Produce these findings (write into spec.md)
1. **The real ask:** restate the goal and the *problem behind the request* (people ask
   for "a dashboard" when they want an answer to a question). Who hits it, how often,
   what they do today instead.
2. **Current behavior:** how it works today, or what's absent. Read the actual code and
   flows; don't assume.
3. **Placement, where it should live** *(so it's correctly placed to be used)*
   - Which module / layer / file / component should own this; the right seam.
   - Existing patterns/components/utilities to **extend or reuse** instead of duplicating.
   - **Integration points**: callers and dependents, the data it reads/writes, the
     APIs/events/contracts it touches (interface analysis: how it interacts with the
     rest of the system).
4. **What it resolves:** the outcome/value and how we'll *observe* it's resolved (feeds
   success + acceptance criteria).
5. **Issues:** conflicts with existing code/UX/data/permissions, constraints, and
   anything that makes the obvious approach wrong. Each issue gets a disposition.
6. **Gaps:** unknowns, ambiguities, undecided behavior, missing inputs. **Every gap
   becomes a question** (next section).
7. **Blast radius:** what this change could break (use the code graph's impact/callers).
   Informs risks, test strategy, and rollback.

Also gather any **design/reference materials** the human supplies (screenshots, Figma,
links, video): see [references-intake](references-intake.md). They're part of
understanding the requirement and become the target later phases verify against.

## Gap analysis (present → desired)
State the **present state** and the **desired state**; the delta is the work, and the
**unknowns in the delta are the gaps**. Drive the count of open gaps toward zero before
handing off to `/rite-define`. Mark each gap inline with `[NEEDS CLARIFICATION: question]`.

## Turn gaps & issues into questions WITH options: you recommend, the human decides
For each material gap/issue (one that changes scope, placement, data model, UX, security,
migration risk, or acceptance), **put it to the human**: one gap at a time, as a ranked
option set with the recommended option **first and marked `(Recommended)`** plus an escape
hatch (via `AskUserQuestion` in HITL):
```
<gap/issue stated in one line>. Why it matters: <...>
1. <recommended option> (Recommended) — <implication / where it places the work; cite any web/docs finding>
2. <alternative> — <implication>
3. <alternative> — <implication>
4. Something else — I'll describe it
```
Investigate and recommend; don't settle a material decision yourself. Confidence in the answer
lowers the *cost* of the question (a one-pick confirm), not its owner: present the set anyway.
Only a **genuinely reversible, low-impact** gap is decided and recorded in `assumptions.md`
without asking. Full render contract + AFK behaviour: [`afk-hitl.md`](../../devrites-lib/reference/standards/afk-hitl.md).

## Done when
- The shipped archive was checked for prior art; any overlap was surfaced to the human.
- The real problem, current behavior, placement, and what-it-resolves are written down.
- Every issue has a disposition; every material gap is resolved **by a human pick** from its
  option set (or explicitly deferred as non-blocking), not settled silently on your confidence.
- No blocking `[NEEDS CLARIFICATION]` remains: the spec is **fully covered and correctly
  placed**. This is the `/rite-spec` readiness gate before `/rite-define` plans it.
