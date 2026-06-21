# Investigation — understand deeply before specifying

The spec is only as good as the investigation behind it. Goal: understand the
requirement **completely**, decide **where it correctly belongs**, name what it
**resolves**, and surface every **issue and gap** — then drive the gaps to closure with
the user so the spec ships fully-covered and correctly-placed. A gap found here is cheap;
a gap found in `/rite-build` is a drift event.

Use a code-intelligence index if available — codebase-memory-mcp first, cross-checked with codegraph + graphify, else standard methods (LSP / Read/Grep/Glob)
(see `../../../rules/tooling.md`) — for the structural parts below; it answers "where does this
live / what calls it / what would it break" far more cheaply and accurately than reading files.
With none present, fall back to Read/Grep/Glob.

## Produce these findings (write into spec.md)
1. **The real ask** — restate the goal and the *problem behind the request* (people ask
   for "a dashboard" when they want an answer to a question). Who hits it, how often,
   what they do today instead.
2. **Current behavior** — how it works today, or what's absent. Read the actual code and
   flows; don't assume.
3. **Placement — where it should live** *(so it's correctly placed to be used)*
   - Which module / layer / file / component should own this; the right seam.
   - Existing patterns/components/utilities to **extend or reuse** instead of duplicating.
   - **Integration points**: callers and dependents, the data it reads/writes, the
     APIs/events/contracts it touches (interface analysis — how it interacts with the
     rest of the system).
4. **What it resolves** — the outcome/value and how we'll *observe* it's resolved (feeds
   success + acceptance criteria).
5. **Issues** — conflicts with existing code/UX/data/permissions, constraints, and
   anything that makes the obvious approach wrong. Each issue gets a disposition.
6. **Gaps** — unknowns, ambiguities, undecided behavior, missing inputs. **Every gap
   becomes a question** (next section).
7. **Blast radius** — what this change could break (use the code graph's impact/callers).
   Informs risks, test strategy, and rollback.

Also gather any **design/reference materials** the human supplies (screenshots, Figma,
links, video) — see [references-intake](references-intake.md). They're part of
understanding the requirement and become the target later phases verify against.

## Gap analysis (present → desired)
State the **present state** and the **desired state**; the delta is the work, and the
**unknowns in the delta are the gaps**. Drive the count of open gaps toward zero before
handing off to `/rite-define`. Mark each gap inline with `[NEEDS CLARIFICATION: question]`.

## Turn gaps & issues into questions WITH options
For each material gap/issue (one that changes scope, placement, data model, UX, security,
migration risk, or acceptance), ask the user — one question at a time, **best guess
attached**, with structured options and an escape hatch:
```
<gap/issue stated in one line>. Why it matters: <...>
1. <option A> — <implication / where it places the work>
2. <option B> — <implication>
3. <option C> — <implication>
4. Something else — I'll describe it
   (My best guess: #2, because <reason>.)
```
Resolve material gaps before finalizing the spec. Reversible, low-impact gaps: decide,
record in `assumptions.md`, and move on — don't interrogate.

## Done when
- The real problem, current behavior, placement, and what-it-resolves are written down.
- Every issue has a disposition; every material gap is resolved (or explicitly deferred
  as non-blocking).
- No blocking `[NEEDS CLARIFICATION]` remains — the spec is **fully covered and correctly
  placed**. This is the `/rite-spec` readiness gate before `/rite-define` plans it.
