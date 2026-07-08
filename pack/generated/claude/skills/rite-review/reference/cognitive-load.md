# Cognitive load — a UX review lens

Cognitive load is the total mental effort the interface demands. Overloaded users
hesitate, misclick, abandon. Reviewing UI without this lens misses a whole class of
defects that no a11y checker or perf tool will flag.

DevRites' `/rite-review` uses this reference when the feature touches UI. It is
descriptive (helps you spot defects) rather than scored — findings raised here
roll up under the UX axis of the multi-axis review with the usual severity labels.

## Three kinds of load

| Type | Source | Reviewer's job |
|---|---|---|
| **Intrinsic** | The task itself — its inherent steps, decisions, and required knowledge. | Can't be eliminated, only **structured**: step the user through it, provide defaults, group related decisions, defer what isn't needed yet. |
| **Extraneous** | Bad design — friction the task doesn't require: noisy hierarchy, redundant choices, jargon, layout that hides the next step. | **Eliminate it ruthlessly.** It's pure waste, and the easiest class of finding to land. |
| **Germane** | Effort the user spends *forming a mental model* of the product. | **Invest deliberately.** Consistent flows, shared vocabulary, predictable affordances — anything that lets the user re-use what they already learned. |

A surface can be intrinsic-heavy *and* extraneously-fine — that's a hard task done
right. A simple surface that still feels heavy is extraneous load disguised as
"complexity."

## Symptoms reviewers should flag

Treat each as a finding when present; classify by load type so the fix is obvious.

### Extraneous (reduce / remove)
- **Visual noise** that doesn't earn its place: redundant borders, shadows,
  dividers, gratuitous icons next to every label, colour used decoratively.
- **Competing primary actions** — two or three buttons of equal weight when one
  is clearly the next step.
- **Jargon and abbreviations** the user doesn't share — names from the codebase
  or internal team that leaked into the UI.
- **Restated headings** (the body of a section repeats the section title in
  long-form).
- **Decision paralysis** — too many options when one default + a "more options"
  toggle would do.
- **Form fields without obvious priority** — required vs optional, primary vs
  secondary mixed in a single column with no rhythm.
- **Premature error messaging** (validation that fires before the user has
  finished typing).

### Intrinsic (structure, don't try to eliminate)
- **No progressive disclosure** on a genuinely complex task — every field
  visible at once because "the user might need it".
- **No safe defaults** on multi-step flows — the user has to choose at every step
  even when one path is the obvious answer.
- **Decisions grouped wrong** — "billing address" sandwiched between two
  unrelated sections rather than next to "shipping address".
- **No scaffolding** (no template, no example input, no recently-used value)
  for tasks that have a high blank-page cost.

### Germane (invest in)
- **Inconsistent vocabulary** across the feature — same concept named two ways
  (e.g. "member" in one screen, "user" in the next).
- **Flow-shape divergence** from neighbouring features (modal where the project
  uses inline; route-change where the project uses an overlay).
- **Affordances that look interactive but aren't** (and vice versa) — destroys
  the mental model the user is trying to build.

## Reviewing a flow — three-pass discipline

1. **First pass: silent.** Walk the flow without reading any explainer copy.
   Anything that stops you cold without copy is a finding — either extraneous
   load you can't dismiss, or a germane-load problem (the user is missing a
   mental model the UI assumes).
2. **Second pass: with copy.** Read every label, helper, and error. Does the
   copy resolve the friction the first pass surfaced, or just paper over it?
   "Tooltip explaining a confusing button" is rarely the right fix — usually
   the button needs a different label.
3. **Third pass: empty + error.** Run the flow with no data, partial data, and
   the worst inputs (offline / 500 / permission-denied). Most cognitive-load
   defects hide in non-happy paths because nobody designs them.

## The cheap-fix test

If a finding would take more than 30 minutes to fix, classify it (token gap /
component miss / flow misalignment per `rite-polish/reference/ui.md`) and
route to the appropriate bucket. Cognitive-load review surfaces problems; it
doesn't decide the rewrite strategy.

## What this lens **doesn't** cover

- **Accessibility** — separate axis (WCAG 2.2; see
  [`devrites-frontend-craft/reference/quality-standards.md`](../../devrites-frontend-craft/reference/quality-standards.md)).
- **Performance** — separate axis (Core Web Vitals; see
  [`performance-checklist.md`](performance-checklist.md)).
- **Code quality** — covered by the simplification audit
  (`devrites-audit simplify`).
- **Security** — covered by the security audit (`devrites-audit security`).

Cognitive-load findings live on the **UX axis** of the multi-axis review.
