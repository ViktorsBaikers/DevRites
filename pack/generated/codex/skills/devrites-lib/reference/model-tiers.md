# Model tiers — dispatch by task shape, never by model name

The single source for how DevRites skills choose a model for a dispatched subagent. A skill
names a **tier** by the *shape of the work*; it never hardcodes `haiku` / `sonnet` / `opus` into
skill content. Model names change and differ per harness; a tier is stable. Loaded on demand by
any skill that dispatches subagents (review fan-out, scouts, judges); not a skill itself.

## The three tiers

| Tier | For work that is… | Examples in DevRites | Model policy |
|---|---|---|---|
| **extraction** | search-and-quote: retrieval, grep, verbatim quoting, mechanical scanning — no judgement | archive-search scouts, footprint scanning, repo-profile gathering, an evidence-dossier scout | cheapest capable model **when the harness exposes a per-agent override**; otherwise inherit |
| **generation** | evidence-driven or mechanical verification: apply a rule to text, draft to a contract, check a claim against a source | drafting to a `design-brief`, mechanical convention checks, first-pass claim verification | mid-tier when the harness allows; otherwise inherit |
| **ceiling** | adversarial judgement and composition: the calls that decide GO/NO-GO or write the artifact a human reads | every reviewer in the seal/review fan-out, `devrites-forge-judge`, `devrites-plan-reviewer`, `$rite-explain` composition | the orchestrator's own model — inherited by declaring **no** model, never dispatched down |

**Reviewers are ceiling on purpose.** Adversarial review is where a cheap model costs the most —
a missed vulnerability or a rubber-stamp is far more expensive than the tokens saved. DevRites
agent definitions therefore declare **no** `model:` field: they inherit the ceiling. Do not add
one to a reviewer to save cost. The tier a scout runs on is where savings are safe.

## The degradation rule

Tiers are an optimization, never a correctness dependency. A skill written against tiers must run
correctly on a harness that cannot select models per agent, or has no subagent primitive at all:

1. **Per-agent model override unavailable** → dispatch the scout on the **inherited** model and
   keep its **read budget and output cap**. The cost control falls back to structure, not model
   choice — this is why every scout dispatch also carries an explicit read budget.
2. **No subagent primitive at all** → run the scout work **inline** in the orchestrator, with the
   same budgets. The tier still tells you how much to spend, even with nothing to dispatch to.

A skill that only works when it can pick a cheap model is broken. The tier names the *intent*; the
budget enforces it regardless of what the harness can do.

## How a skill references a tier

Name the tier and the budget where you dispatch; never the model:

> Dispatch an **extraction-tier** scout (read budget: 20 files, output cap: the dossier path + a
> 3-line gist) to quote every call site of `X` into `<scratch>/dossier.md`.

Not: "dispatch a Haiku agent to…". If a future reviewer needs to know why a scout is cheap, this
file is the answer; the skill stays legible and portable.
