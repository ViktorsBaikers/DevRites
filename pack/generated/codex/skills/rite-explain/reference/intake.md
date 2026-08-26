# rite-explain intake: classify the input, then ground and check-in by shape

Owned by `$rite-explain`. Loaded at Phase 1 because the four input shapes ground, compose, and
check-in differently, and this detail would bloat the SKILL if inlined. This file is the single
source for classification; the SKILL improvises none of it.

## The four shapes

| Shape | The input is… | Grounds in | Composes as | Check-in |
| --- | --- | --- | --- | --- |
| **concept** | a named idea / pattern / technology ("explain optimistic locking", "how does our gate engine work") | this repo's footprint of the concept (codegraph first) + external sources only if they sharpen it | build the mental model from a known part of *this* codebase outward | **checked exercise** |
| **diff** | a specific change: a ref, a slice, a PR, "this diff" | `git diff` / the hunks + `decisions.md` + `seal.md` for the *why* + `touched-files.md` `Review trail` when present | explainer, or **walkthrough** when the user asks to review/approve/checkpoint the change | **predict-then-reveal** |
| **idea** | a hypothesis or "what if" with no code yet | the user's framing + prior art (external, date-weighted; year is 2026) | steelman the idea, name its hinge and its failure mode | **checked exercise** |
| **recap** | a time window of the user's own work ("what did I ship this week") | the shipped archive + `git log`/`git diff` over the window + `evidence.md` | narrate what changed and what was learned, ranked by leverage | **predict-then-reveal** |

## Token overrides: explicit beats inferred

If the user's input carries any of these tokens, they **override** shape inference:

| Token | Meaning |
| --- | --- |
| `diff:<ref>` | force the **diff** shape against `<ref>` (a commit, range, or slug) |
| `walkthrough:<ref>` | force the **diff** shape and compose a human review walkthrough instead of a teaching explainer |
| `since:<when>` | force the **recap** shape over the window (`since:1w`, `since:last-ship`, an ISO date) |
| `output:<path>` | write the explainer to `<path>` instead of the default `.devrites/explainers/<date>-<slug>/` |

No token → infer the shape from the phrasing and the referenced object. If the phrasing says
"walk me through", "checkpoint", "human review", "help me approve", or "review this change",
use the diff shape with walkthrough composition.

## Tiebreaks

- **concept vs diff:** if the input names both a concept *and* a specific change ("explain how
  this diff does optimistic locking"), it is a **diff** explainer that teaches the concept through
  the change. Ground in the diff; use the concept as the model to build.
- **idea vs concept:** if the thing exists in the codebase or the wider world already, it is a
  **concept** (teach what is). If it is a proposal not yet built, it is an **idea** (teach whether
  it holds).
- **conflict:** a token and the prose disagree (`diff:` set but the text asks to explain a
  concept) → the token wins for shape; honor the prose for emphasis. State the resolution in one
  line at the top of the explainer.

## Check-in mechanics

Retention comes from retrieval. Offer exactly one check-in via the harness's blocking-question tool
(`AskUserQuestion`, or `request_user_input` on Codex); the user answers **before** any reveal. Grade honestly: a wrong prediction that gets corrected teaches more than a
confirmed one, so do not lead the witness.

## Walkthrough composition (diff only)

A walkthrough is for human review, not retention. Write it under the normal run directory as
`walkthrough.md` and include:

1. **Intent:** one sentence naming what the change is for, grounded in `spec.md` / `decisions.md` when available.
2. **Scope:** files changed, modules touched, and boundary crossings.
3. **Concern walkthrough:** group `path:line` stops by design intent, not file order. Prefer the existing `touched-files.md` `Review trail`; generate one from the diff only when absent.
4. **Risk stops:** 2-5 highest-blast-radius places to inspect, tagged `[auth]`, `[data]`, `[schema]`, `[public API]`, `[migration]`, `[ui]`, `[perf]`, or `[test]` where relevant.
5. **Manual observations:** 2-5 concrete ways a human can observe the change working. If there is no user-visible behavior, say so and name the automated evidence instead.
6. **Decision options:** approve / rework / ask for a targeted review, each with the next command.

Completion criterion: every concern has at least one clickable repo-relative `path:line` stop, or the walkthrough states why the change has no source stops.

## Visual dual-read (when composition earns it)

If the explainer or walkthrough needs a spatial/relational page, follow the SKILL's
"Visual where it earns it" branch: load matching playbooks from
[`../../devrites-lib/reference/visual-playbooks/index.md`](../../devrites-lib/reference/visual-playbooks/index.md),
emit `visual/<name>.html` + `visual/<name>.outline.md` (workspace or `$RUN_DIR`), treat
outline as SSOT, and never invent a new phase or Lavish dependency. Classification still
owns shape; the visual branch does not change which shape you are in.

### Predict-then-reveal (diff / recap)

1. Pick the single most load-bearing hunk or decision in the explainer.
2. Ask the user to predict: "what does this change do?" / "why was `<decision>` made here?":
   offer 2-4 plausible options where one is the real answer and the distractors are *reasonable*
   (a distractor no one would pick teaches nothing).
3. On answer, reveal: confirm or correct, pointing at the exact `file:line` that settles it.

### Checked exercise (concept / idea)

1. Pose one small problem the developer solves using the model just taught: "given `<situation>`,
   what would `<mechanism>` do?" or "where would this break?".
2. Let them answer, then grade against the taught model: name what they got right, correct what
   they missed, and cite the part of the explainer (or the repo) that covers it.

Skip the check-in entirely when the material is a genuine one-off with nothing worth retaining, or
the user declines. Offer once; never nag.
