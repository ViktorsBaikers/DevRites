# Orchestration patterns

How DevRites coordinates multiple agents — the patterns it uses, the ones it deliberately avoids,
and where Claude Code's Agent Teams and worktree isolation fit. The rule that governs this lives in
[`pack/.claude/skills/devrites-lib/reference/standards/agents.md`](../pack/.claude/skills/devrites-lib/reference/standards/agents.md); this doc is the map.

## The model

DevRites separates three roles and never blurs them:

- **Orchestrator** — the active `rite-*` skill (chiefly `/rite-build` and `/rite-seal`). It owns
  the gates and the `.devrites/` workspace, dispatches the other agents, and is the *single
  canonical writer* of workspace state.
- **Reviewers** — fresh-context, **read-only** subagents under `.claude/agents/`. Each gets the
  workspace path + the diff and returns labelled findings; read-only is enforced at the tool layer
  (`devrites-engine hook reviewer-readonly`), not merely promised.
- **Executor** — `devrites-slice-wright`, the one **write-capable** agent. It implements a single
  fully-specified slice in a fresh context and returns code + tests; it never writes the `.devrites/`
  bookkeeping (the orchestrator does).

Fresh, undirected context is the point: an agent gets the contract, not the author's reasoning, so
its judgment is independent.

## Endorsed patterns

1. **Direct (no orchestration).** Most phases are one skill doing one job. Don't spawn an agent for
   work the phase can do inline.
2. **Parallel read-only fan-out.** At `/rite-seal` the relevant reviewers run *in parallel*, then
   the orchestrator reconciles — banding by confidence and surfacing genuine disagreement explicitly
   rather than averaging it away.
3. **Single writer.** Exactly one `devrites-slice-wright` per slice, never a parallel fan-out of
   writers *sharing a tree* — concurrent writers on one tree make conflicting implicit decisions
   that corrupt a coherent design. The one sanctioned exception is a **forge** slice
   (`Forge: yes`, flagged by `/rite-vet`): K=2–3 candidate wrights build it on distinct strategies
   in **isolated** worktrees, `devrites-forge-judge` scores them, and exactly one winner's diff
   lands — no tree ever has two authors, so the invariant holds
   ([`rite-build/reference/forge.md`](../pack/.claude/skills/rite-build/reference/forge.md)).
4. **Lifecycle as user-driven verbs.** Each verb performs one mutation and stops; chaining is
   explicit (the user types the next command), so there are no hidden side effects between phases.
   `/rite-autocomplete` is the one deliberate exception — an opt-in unattended driver.
5. **Adversarial single-claim check.** `devrites-doubt` spawns a fresh reviewer to try to refute one
   load-bearing decision, rather than asking the author to re-grade their own work.

## Anti-patterns DevRites avoids

- **A persona that paraphrases another.** Passing one agent's summary to the next is lossy telephone;
  reviewers read the raw diff and contract, not a digest of the author's reasoning.
- **Parallel writers.** See single-writer above — two agents editing the same feature concurrently
  is a merge of conflicting decisions, not a speed-up. (Forge is *not* this: its candidates write in
  **isolated** worktrees and exactly one lands, so no tree is ever co-authored.)
- **A router that does the work.** `/rite` is a *thin dispatcher* — it renders the menu, resolves a
  verb to a skill, and gets out of the way. It holds no phase logic and produces no artifact itself.
  (This is the distinction that makes a router fine: the anti-pattern is a "meta-orchestrator"
  persona that paraphrases and re-decides on every call, not a dispatch table.)
- **Deep persona trees.** Agents don't call agents that call agents. The orchestrator dispatches one
  level deep — reviewers and the wright — and reconciles the results itself.

## Agent Teams and worktree isolation

Two Claude Code capabilities sit adjacent to DevRites' model; the stance on each is deliberate.

- **Agent Teams.** DevRites does **not** use Agent Teams to run the lifecycle. The on-disk workspace
  plus fresh-context read-only fan-out already give independent context per agent without the
  coordination overhead, and the lifecycle is intentionally single-slice / single-writer. Reach for
  Agent Teams for work *outside* that discipline — competing-hypothesis debugging, or exploring
  several genuinely different approaches in parallel — not to parallelise a DevRites build. The one
  in-lifecycle form of "compete several approaches" is **forge** (a `Forge: yes` slice), and it is
  deliberately bounded: vet-gated, K≤3, isolated, winner-takes-all.
- **Worktree isolation.** A single feature is single-writer on one branch by design, so DevRites
  doesn't spawn worktrees for ordinary builds — but you can run DevRites *inside* a git worktree to
  drive two features in parallel without them colliding (the `.devrites/ACTIVE` sentinel is
  per-working-tree, so each carries its own active feature). The one place DevRites spawns worktrees
  itself is a **forge** slice: each candidate build gets an ephemeral worktree, auto-removed after
  the winner lands.

## See also

- [`pack/.claude/skills/devrites-lib/reference/standards/agents.md`](../pack/.claude/skills/devrites-lib/reference/standards/agents.md) — the reviewer / executor roster
  and the when-to-fan-out rules.
- [`architecture.md`](architecture.md) — the full layer model.
- [`flow.md`](flow.md) — phase-by-phase flow and the public/internal namespace.
