# Forge — competing candidate builds for one slice

How `/rite-build` builds a slice that `/rite-vet` flagged `Forge: yes`: instead of one wright,
**K = 2–3 candidate wrights** each build the slice a genuinely different way in **isolation**, a
read-only judge scores them, and exactly **one** winner's diff lands in the working tree. Loaded
on demand by `/rite-build` step 3; the sibling of single-path
[`wright-dispatch.md`](wright-dispatch.md).

Forge is the **rare** path. Most slices are single-path (cheaper, and the default). A slice
forges only when the work is a genuine architecture fork — two or three approaches that actually
differ, no clear winner on paper, at high enough stakes that building the wrong one costs more
than building all of them. That judgment is made at `/rite-vet` and recorded as the slice's
`Forge:` field; build acts on the flag, it does not invent it.

## Why this doesn't break the single-writer invariant

DevRites forbids a parallel fan-out of writers **sharing one tree** — concurrent writers on one
working tree make conflicting implicit decisions and produce incoherent code
([`wright-dispatch.md`](wright-dispatch.md)). Forge keeps that invariant intact by **isolation**:
every candidate writes in its own worktree (or its own throwaway branch), never touching another
candidate's tree, and the orchestrator lands exactly one. No tree ever has two authors; exactly
one author's work ships. "N isolated complete attempts, winner-takes-all" is not "N writers on
one slice."

## Trust the flag, but clear a stale one

Before competing anything, confirm the flag still earns its cost — `/rite-vet` set it, but the
plan may have moved:

- **No objective scorecard** — the slice has no acceptance criteria and no `test-plan.md`
  coverage the judge can score against → clear `Forge` to `no`, build single-path. A competition
  with no rubric is a coin toss with K× the cost.
- **Can't name two genuinely different strategies** — if the candidates would be variations of
  one approach, there is nothing to compete → single-path.
- **Slice shrank below the bar** (now Complexity ≤3, or a dependency landed that picks the
  approach) → single-path.

A cleared flag is recorded in `decisions.md` (one line: why forge was dropped). Never forge a
slice you can't score, and never forge to avoid making a decision the plan already made.

## Mechanics

The orchestrator runs F1–F7. Steps 4–7 of the [one-slice-cycle](one-slice-cycle.md) (doubt,
fail-on-red, reconcile, record, stop) then run **unchanged** on the winner.

### F1 — Confirm the K strategies
Take the 2–3 candidate strategies `/rite-vet` named (in the slice brief). Each must be a
**distinct, complete approach** to the same slice contract — a different seam, data shape,
reuse-vs-build call, or algorithm — not a tweak of one. Name them `A`, `B`, (`C`) with a
one-line description each. K is capped at 3: a fourth candidate rarely changes the winner and
multiplies cost.

### F2 — Snapshot, then isolate each candidate
Snapshot the working tree first (the winner's landing is reconciled against it later), then give
each candidate its own isolated tree. **Prefer parallel isolated worktrees** when the harness can
dispatch a sub-agent with worktree isolation; the **always-available** path is one throwaway git
branch per candidate, built sequentially (slower, universally works — the
[`tooling.md`](../../devrites-lib/reference/standards/tooling.md) "fallback is first-class" discipline):

```bash
SLUG="$(cat .devrites/ACTIVE 2>/dev/null)"
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || echo "(not a git repo — forge needs git for isolation; degrade to single-path)"
BASE="$(git rev-parse --abbrev-ref HEAD)"
# one isolated worktree per candidate (parallel-capable); falls back to branches if worktrees are unavailable
for C in A B; do
  git worktree add ".devrites/work/$SLUG/forge/cand-$C" -b "forge/$SLUG/cand-$C" 2>/dev/null \
    || git branch "forge/$SLUG/cand-$C" "$BASE"
done
```

### F3 — Dispatch K candidate wrights
Send **each** candidate the identical single-path slice contract from
[`wright-dispatch.md`](wright-dispatch.md), with **one** line added naming its assigned strategy
and its isolated tree:

```
Forge candidate <A|B|C> of <K>. Build this slice using ONLY this approach:
  Strategy: <the distinct approach, one or two sentences — the seam / data shape / reuse call>
Work in your isolated tree: <worktree path | branch name>. Do not consult or merge another
candidate's work. Same discipline as always — orient → RED → implement smallest complete →
verify → return your structured artifact. Code + tests only.
```

Each is still **one wright, one tree** — the invariant holds per candidate. Parallel where the
harness isolates them; otherwise build candidate A to green, capture its diff, reset to `BASE`,
then candidate B. A candidate that hits an irreversible-risk item escalates exactly as a
single-path wright does — **forge never bypasses a gate** (see AFK below).

### F4 — Judge (read-only, fresh context)
Dispatch [`devrites-forge-judge`](../../../agents/devrites-forge-judge.md) with the K finished
candidate diffs and the scorecard inputs (slice acceptance, `test-plan.md`, `.devrites/principles.md`,
the anti-slop charter). It scores each candidate, ranks them, names the **winner** and the
specific runner-up ideas worth grafting, and returns the structured verdict. The judge **never
writes code** — it reads the diffs and returns findings, like every reviewer agent. If sub-agent
dispatch is unavailable, do the judge's rubric pass yourself in a fresh read, discarding your
authoring reasoning (a flagged fallback, not an independent judgment).

### F5 — Land the winner, graft sparingly
Apply **only** the winner's diff to the working tree (merge its worktree / cherry-pick its
branch). If the judge named a cheap, specific improvement in a runner-up, graft it by
**continuing the winning wright once** with that instruction — never hand-merge two candidates'
code (that re-creates the incoherent-tree failure the invariant exists to prevent). The landed
tree has exactly one author: the winner (plus its own grafted follow-up).

### F6 — Write `forge-report.md`
The durable record of the competition (template below). One per forged slice; archived with the
feature.

### F7 — Clean up, then return to the cycle
Remove the losing worktrees / delete the candidate branches:

```bash
SLUG="$(cat .devrites/ACTIVE 2>/dev/null)"
for C in A B; do
  git worktree remove ".devrites/work/$SLUG/forge/cand-$C" --force 2>/dev/null || true
  git branch -D "forge/$SLUG/cand-$C" 2>/dev/null || true
done
```

Then hand the **winner's** structured artifact to [one-slice-cycle](one-slice-cycle.md) step 4 as
if a single wright produced it. Doubt, fail-on-red, reconcile (against the F2 snapshot, claimed =
the winner's files), record, and stop all run unchanged.

## `forge-report.md` template

```markdown
# Forge report: <slice id — name>
Forged on <iso>. Reason: <the architecture fork /rite-vet flagged>.

## Candidates
| # | Strategy | Gates | Score | Notes |
|---|---|---|---|---|
| A | <approach> | green/red | <judge score> | <one line> |
| B | <approach> | green/red | <judge score> | <one line> |

## Verdict
Winner: <A|B|C> — <why it won, in the judge's terms: acceptance coverage, simplicity, principle
fit, anti-slop, reuse>.
Grafted from runner-up: <specific idea + which candidate | none>.
Discarded: <the losing approaches + the one-line reason each lost — load-bearing for a later
slice that might be tempted to retry one>.
```

The **Discarded** section matters: it is a dead-end record at the slice level, the same role
`decisions.md` "Dead ends" plays for the feature — a later slice shouldn't re-litigate an
approach forge already rejected.

## AFK & budget

- **Cost.** Forge multiplies the build by K. Under `.devrites/AFK`, each candidate counts against
  the slice budget (`devrites-engine tick-afk` once per candidate dispatched), so a forge slice can exhaust the
  cap faster — that is intended back-pressure, not a bug.
- **Gates are unchanged.** Forge changes *how many candidates build*, never *what pauses*. An
  irreversible-risk item, a blocking/escalating gate, or a red-on-completion in **any** candidate
  pauses per [`afk-hitl.md`](../../devrites-lib/reference/standards/afk-hitl.md) exactly as single-path. AFK widens what is
  automatic, never what is irreversible — forge included.
- **Stuck loop still applies.** Re-dispatching the same candidate without progress trips
  `devrites-engine stuck` the same as a single wright.

## When isolation is impossible

No git, or neither worktrees nor throwaway branches are usable → **degrade to single-path** and
say so. Forge is an accelerator for choosing between real alternatives; it is never a requirement
for building the slice. A slice always has a single-path build available — the competition is the
optional part.

## Anti-patterns

See [`anti-patterns.md`](anti-patterns.md) § Forge — the short version: don't forge a decided or
trivial slice, don't forge to dodge a decision, don't hand-merge candidates, don't exceed K=3,
and don't let a forged slice skip the post-return doubt because "the judge already looked."
