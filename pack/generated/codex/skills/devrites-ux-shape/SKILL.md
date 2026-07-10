---
name: devrites-ux-shape
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
user-invocable: false
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers — NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# devrites-ux-shape — plan the UX/UI before code

Decide what the UI *is* and how it should look and behave **before** code, and write it to
the feature's `design-brief.md` — the contract the build (`devrites-frontend-craft`),
polish, and seal all check against. This is "shape before code" raised to a **feature-level
artifact**: produced once at spec time, refined per slice. It is **woven into the
lifecycle** (`$rite-spec` calls it when UI is detected; `$rite-build` refines it), not a
separate phase the user runs.

## When it runs
- `$rite-spec` invokes it after references-intake when the feature touches UI
  (`../rite-build/reference/frontend-trigger.md`). Output: `design-brief.md` + a
  confirmation pause.
- `$rite-build` / `devrites-frontend-craft` read the brief as the **build target** and
  refine it for the slice's surface — they don't re-derive it.
- Skip entirely for backend / data / CLI / infra-only features.

## 1. Foundation — discover, don't impose
Reuse what's there first. Read the design system + register and any references the spec
gathered:
- Design system + register (tokens, components, type, spacing, neighbors; brand-vs-product)
  → `../devrites-frontend-craft/reference/design-references.md`.
- `PRODUCT.md` / `DESIGN.md` / `CLAUDE.md` if present — anchors that reduce questions.
  `DESIGN.md` is the project's **rolled-up design memory** (tokens, calibration baseline,
  proven component behaviors) earlier features sealed via `../rite-ship/reference/design-memory.md`;
  treat it as the inherited system — read it before re-discovering, depart only on signal.
- `references.md` + `references/` — the screenshots / Figma / video / links the human
  supplied. Honor each recorded role: **target** = fidelity contract, **constraint** =
  required rule, **inspiration** = extract only the cited principle.

## 2. Discovery — one round, assert-then-confirm
Understand the feature deeply enough to make excellent design calls — **no code, no
markup**. Use the `devrites-interview` cadence: 2-3 questions per round, best-guess
attached, stop when answers converge. One round is the default; add a second only for
material gaps. When `PRODUCT.md` + the spec already pin an answer, **assert it and ask to
confirm** ("reads as Restrained — confirm?"), don't offer a four-option menu. Cover:
- **Purpose & user** — who, in what state of mind (rushed / exploring / anxious / focused).
- **Content & data** — realistic ranges (0 / typical / many), dynamic content, real
  media/assets the surface needs.
- **Preserve + anti-goals** — identity, IA, behavior, content voice, SEO/analytics hooks,
  existing accessibility wins, or assets that must survive; what this must NOT become;
  the biggest risk of getting it wrong.

## 3. Design direction — the direction set (commit, don't hedge)
One deliberate visual decision on five fronts, each anchored in an existing reference so
the call is checkable, not taste:
- **Scene sentence** — who / where / ambient light / mood, per
  `../devrites-frontend-craft/reference/design-references.md`. Forces dark-vs-light and
  tone from the scene, not the category.
- **Color strategy** — Restrained / Committed / Multi-role / Saturated, per the
  color-commitment table in `../devrites-frontend-craft/reference/quality-standards.md`.
  Pick from the scene; register doesn't decide it.
- **Calibration** — density (Airy / Balanced / Dense) and motion (Minimal / Standard /
  Expressive), per the calibration table in the same quality-standards file. Set both from
  the scene so the build targets a calibration, not a guess (a 2am SRE → Dense + Minimal; a
  launch hero → Airy + Expressive).
- **Named anchor references** — 2-3 *specific* products / brands / objects to steer toward
  (not adjectives like "modern" or "clean"), plus the saved `references/` files.
- **Visual thesis** — the focal point and hierarchy in one sentence, plus one memorable
  move (or `none — system continuity`) and the obvious-but-wrong direction rejected. This
  creates intentional distinction without forcing novelty into product UI.

Respect the existing identity (default, ~90%); depart only on an explicit signal.

## 4. Scope — task-scoped, never persisted
Name the output target so sketch-vs-ship isn't guessed: **fidelity** (sketch / mid-fi /
high-fi / production — DevRites default is production), **breadth** (one screen / a flow /
a surface), **interactivity**, **time intent**. These ride in the brief only; they do not
change the project's design system, and never get written to `PRODUCT.md` / `DESIGN.md`.

## 5. Key states + interaction model
List every state the feature needs and what the user must see/feel in each (default,
loading initial+subsequent, empty→next-action, error→recovery, success, disabled/
no-permission, long-content/overflow), the information hierarchy (what's seen 1st / 2nd /
3rd; primary action unmistakable), responsive reflow, a11y must-haves, and the interaction
model (inline vs navigated vs — rarely — modal; optimistic vs pending; feedback). Name the
representative states, viewports, input modes, and target R-ids that will prove the result;
"looks polished" is not a proof target. Canonical state list:
`../devrites-frontend-craft/reference/shape.md`.

## 6. Visual-direction probe — capability-gated
When the work is net-new or directionally ambiguous and fidelity ≥ mid-fi, pressure-test
the lane with something concrete instead of words — pull Figma context, generate image
probes, screenshot reference sites, or route a code-fidelity question to `$rite-prototype`.
**Capability-gated**: announce the skip in one line if no tool is available. Flow:
[reference/visual-direction-probe.md](reference/visual-direction-probe.md).

## 7. AI-slop pre-check
Run the two-altitude category-reflex check (`../rite-polish/reference/anti-ai-slop.md`) on
the chosen direction *before* writing the brief — if the palette/theme is guessable from
the category, rework the scene sentence and color strategy. Cheaper to catch the slop in
the brief than in the build.

## 8. Write the brief + gate
Write `design-brief.md` ([reference/brief-template.md](reference/brief-template.md)) —
**compact** (3-5 bullets) when discovery was crisp, **full** when the surface is ambiguous
or multi-screen. Don't pad a clear brief to look thorough; don't skip the pause to look
fast. Then honor the run mode (`../devrites-lib/reference/standards/afk-hitl.md`):
- **HITL** — present the brief and **STOP for explicit confirmation**. The pause is the
  point: shape ends at the user's "go", not at your own certainty. Disagreement → revisit
  the relevant discovery question.
- **AFK** — assert the best-guess direction, record it in `decisions.md` + an advisory
  `questions.md` entry, and proceed. A direction touching the irreversible-risk list still
  pauses.

Completion: the smallest useful brief exists; primary action, states, interaction, proof
targets, and reference role/usage are explicit; the run-mode gate is resolved. Only then may
UI code start.

## Output
```
UX/UI shaped: <slug>
Direction: <color strategy> · density <airy|balanced|dense>/motion <minimal|standard|expressive> · "<scene sentence>" · anchors: <ref A, ref B>
States: <n listed>   Probe: <figma | images | prototype | skipped — no tool>
Brief: design-brief.md (<compact | full>)
Gate: <confirmed | awaiting confirmation | AFK-asserted>
Next: $rite-define (UI slices map to the brief's states) — or $rite-build refines it per slice
```

## NEVER (ux-shape)
- Never write code, markup, or a component here — produce the thinking, not the UI.
- Never finalize the brief without naming the **primary action** and the **full state set**
  (not just the happy path).
- Never decide visual direction by taste when the project has a system — discover first.
- Never treat an **inspiration** reference as a fidelity target — honor its recorded role.
- Never skip the HITL confirmation pause "because the brief is obviously right" — ask once,
  wait.
- Never re-derive the brief from scratch in `$rite-build` — refine the existing one.
