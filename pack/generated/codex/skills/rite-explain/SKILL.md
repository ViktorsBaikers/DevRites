---
name: rite-explain
description: User-invoked explainer that teaches one concept, diff, idea, or recent-work recap with an optional retrieval check-in.
argument-hint: "[a concept | a diff ref | an idea | \"what did I do this week?\"], or bare to be asked"
user-invocable: true
disable-model-invocation: true
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


# $rite-explain: the human half of the learning loop

Agent-driven development removed the learning that writing code by hand used to give the
developer. When the agent writes the code, the human stops absorbing the codebase. `$rite-explain`
is the replacement: it teaches the developer **one** thing well (a concept, a change, an idea, or
a window of their own recent work) so they keep learning while the agents do the writing. For a
specific change, it can instead write a **walkthrough**: a human review guide organized by concern,
risk stops, and manual observations.

This is the **complement of [`$rite-learn`](../rite-learn/SKILL.md)**, and the two together are
the whole compounding story. `$rite-learn` teaches the **repo**. It promotes recurring lessons
into rules and principles the next feature reads. `$rite-explain` teaches the **person**: the
mental model the next feature needs a human to hold. A team that only runs `$rite-learn`
compounds a codebase whose own maintainers understand it less each week. Run both.

Read-only against source. It reads artifacts and code; it writes exactly one explainer file to
`.devrites/explainers/` and never edits source, specs, or rules.

## Who the explainer is for

The developer, personally. Dense, technical, one voice: no audience adaptation, no "for
beginners" softening, no executive summary. It is a display artifact: no embedded quizzes or
widgets: the check-in (below) happens live in the session, where an answer can be
graded. If the user asked for prep for a meeting or a teammate, it preps **them** to explain the
thing; it does not produce the deck.

## Model tiers

Dispatch by task shape, per [`devrites-lib/reference/model-tiers.md`](../devrites-lib/reference/model-tiers.md):

- **extraction tier:** the work-recap scout and any repo-profiling: search-and-quote, run under a
  read budget, write findings to the run's scratch dossier, return only a gist.
- **ceiling tier:** the explainer composition and the check-in reasoning. These run inline in the
  orchestrator's own model; nothing is dispatched down. Teaching quality is the whole product: do
  not cheapen it.

The degradation rule in that file applies: no per-agent model control → run the scout inline under
the same budget.

## Workflow

### 1. Classify the input: load the intake reference

The four input shapes (concept · diff · idea · work-recap) each ground and compose differently,
and each drives a different check-in. Diff inputs also branch into explainer vs walkthrough
composition. Getting the shape wrong wastes the whole artifact.

**Load [`reference/intake.md`](reference/intake.md) now**. It owns the classification rules, the
`diff:` / `since:` / `output:` token table, the concept-vs-diff tiebreak, and the check-in
mechanics per shape. Do not improvise classification from this file; the detail lives there so this
one stays legible, and skipping it means guessing the shape.

**Bare invocation** (no input): ask **one** blocking question (`AskUserQuestion` when the harness
has it, else the harness's blocking-question tool: `request_user_input` on Codex). "What should I
explain?": offering "a recap of my recent work in this repo" as a shortcut option alongside free
text. Never emit a default explainer unprompted.

### 2. Ground

Match grounding to the shape (full rules in the intake reference). DevRites gives you grounding for
free: prefer it over re-deriving:

- A **diff** or **recap** → the workspace and archive: `seal.md`, `evidence.md`, `decisions.md`,
  `traceability.md`, the shipped `.devrites/archive/<slug>/`, and `git log` / `git diff`.
- A **concept** with footprint in this repo → the live code (codegraph / graphify first), plus any
  `.devrites/principles.md` or ADRs that already take a position on it.
- An **idea** or a concept with no repo footprint → the user's framing plus, only if it sharpens
  the teaching, current external sources (weight by date; the year is 2026).

Create the run directory before composing so scout dossiers have a home:

```bash
RUN_DIR=".devrites/explainers/$(date +%Y%m%d)-<slug>"; mkdir -p "$RUN_DIR"
```

### 3. Compose the explainer

For diff inputs that ask for `walkthrough:<ref>`, "checkpoint", "walk me through", or human review,
write `$RUN_DIR/walkthrough.md` using the walkthrough composition in `reference/intake.md`, then skip
the active-recall check-in unless the user asks for teaching too.

Otherwise write one dense artifact at `$RUN_DIR/explainer.md`. It must **teach**, not summarize:

1. **One thing.** A single clear takeaway named in the first two lines. If the input sprawls,
   teach the highest-leverage slice and say what you cut.
   **Completion:** the first two lines state one takeaway and any deferred topics.
2. **Build the model, don't list facts.** Start from what the developer already knows in *this*
   codebase and move to the new thing. Concrete before abstract: a real symbol, file, or line
   from the grounding beats a generic example every time.
   **Completion:** the explanation connects a known project anchor to the new model without a fact dump.
3. **Show the load-bearing detail.** Quote the actual diff hunk, the real function, the specific
   config, with `file:line` pointers so the developer can go read it.
4. **Visual where it earns it.** A small diagram, a before/after, or a worked trace when the shape
   is spatial or sequential. Not decoration: only when it carries the idea faster than prose.
   **Completion:** the visual carries a named relationship faster than prose, or this branch is explicitly skipped.
5. **Human voice.** Follow [`prose-style.md`](../devrites-lib/reference/standards/prose-style.md):
   no throat-clearing, no false-binary contrast, no marketing adjectives. One senior engineer
   explaining to another.

### 4. Offer the check-in (optional, active recall)

Retention comes from *retrieving*, not re-reading. After the explainer, offer one check-in via the
harness's blocking-question tool (`AskUserQuestion`, or `request_user_input` on Codex): the user
answers **first**, then you confirm or correct. The shape sets the
form (mechanics in the intake reference):

- **diff / recap** → **predict-then-reveal**: ask what a specific hunk changes or why a decision
  was made *before* showing the reveal.
- **concept / idea** → a **checked exercise**: one small problem the developer solves, then you
  grade it against the model you just taught.

Skippable when the material does not warrant retention work (a one-off, or the user declines). Do
not force it; offer once.

## Gotchas

- **Teach one thing.** An explainer that covers five things teaches none. Cut ruthlessly; queue the
  rest as "next time".
- **Ground it or don't claim it.** A concept explainer with no `file:line` or real example is a
  Wikipedia paragraph. If it has footprint in this repo, quote the repo.
- **Not a review.** `$rite-explain` never judges the code or files findings, that is `$rite-review`.
  It explains what *is*, adversarially neutral.
- **Not `$rite-learn`.** It writes to `.devrites/explainers/`, never to `learnings.md`,
  `principles.md`, or any rule file. Teaching the human is not promoting a repo rule.

## Output

Reply-contract exception: cross-feature learning utility; may run with no active feature, so it
skips `devrites-engine progress` when no workspace exists. Otherwise follows
[`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md).

```
Done: explained <the one thing> as a <concept|diff|idea|recap> explainer OR walked through <change> for human review.
Changed: .devrites/explainers/<date>-<slug>/<explainer.md|walkthrough.md>
Evidence: grounded in <artifacts/files quoted>; check-in <offered+result | skipped>; walkthrough stops <count>
Open: <none | next-time topics deferred | check-in awaiting the user>
Next: <single command — usually back to the calling phase, or $rite-learn if a repo rule surfaced>
Record: .devrites/explainers/<date>-<slug>/explainer.md
↻ Hygiene: /clear after reading; the explainer is on disk
```
