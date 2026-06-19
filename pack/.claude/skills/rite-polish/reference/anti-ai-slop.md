# Anti-AI-slop

Tells of generic LLM-generated work — in **UI** and in **code**. Applies at two stages:

- **At build time** (preventive) — `/rite-build` checks against these as it writes; the
  cheap moment to avoid them.
- **At polish time** (catch) — `/rite-polish` Phase 1 (Code) + Phase 2 (Backend) + Phase 4
  (UI) scrub anything that slipped through.

Avoid these unless the project's existing system explicitly uses them. When in doubt,
match the neighbors.

## UI anti-slop (banned defaults)
- Default **purple/blue gradients** as the brand look. Special case: any
  hero hex in the `#6366f1 → #a855f7 → #ec4899` family applied as a default.
- **Gradient text** (`background-clip: text`) used decoratively on headings.
- **Glassmorphism** (`backdrop-filter: blur(...)` on translucent panels) as
  a default surface.
- **Side-stripe colored borders** on cards/sections — the
  "tiny-bar-of-meaningful-accent-color-on-the-left" pattern. Distinctive
  templating tell.
- **Pure `#000` / `#fff`** as raw text or background — too clinical; use
  near-black/near-white tokens (`oklch(0.18 0 0)` / `oklch(0.98 0 0)` or
  the project's surface tokens).
- **All-CAPS body text** for paragraphs/labels. Reserve uppercase for short
  micro-copy (badges, eyebrows); never for sentences.
- **Em-dash overuse** in UI copy — multiple em-dashes per paragraph, em-
  dashes where a comma or period would do. A clear AI tell.
- **Cards inside cards** — nested bordered/elevated containers.
- **Identical card grids** for everything, regardless of content.
- A **generic rounded-square icon tile** above every heading/section.
- **Gray text on colored backgrounds** (fails contrast, looks templated).
- The **hero-metric cliché** — three big numbers in a row with no real meaning.
- **Decorative bounce / elastic easing** on everything; motion without purpose.
- **Reflex fonts** picked because they're the default in a tutorial:
  - Inter for every product when the project has its own choice.
  - **DM Sans**, **Plus Jakarta Sans**, **Fraunces**, **Newsreader** when
    they're not the project's actual type system.
  Match the project; don't reach for the "tasteful default" of 2024.
- **Modal-first thinking** — reaching for a modal as the answer to every interaction.

### Category-reflex check — run at two altitudes

Most generic-AI design fails one of these two reflex tests. Run both — the
second one catches what the first one misses.

- **First-order:** if someone could guess the theme + palette *from the
  category alone* — "observability → dark blue", "healthcare → white +
  teal", "fintech → navy + gold", "AI tool → black with a violet accent",
  "crypto → neon on black" — the styling is on the first training-data
  reflex. Rework the scene sentence
  (`devrites-frontend-craft/reference/design-references.md`) and the
  colour-commitment strategy
  (`devrites-frontend-craft/reference/quality-standards.md`) until the
  answer isn't obvious from the domain.
- **Second-order:** if a stranger looked at the surface with *no copy
  visible* and confidently said "this is a CRM / fitness tracker / fintech
  / AI workflow tool", the styling is still on a category template — just
  one tier deeper. The first reflex was avoided, the second wasn't.
  Re-shape until the surface doesn't telegraph its category from looks
  alone.

Both pass = the surface looks like *this product*, not "an app in this category".

## Code anti-slop (UI **and** backend)
- **Over-defensive checks** — `if (x && x.length > 0)` repeated, layered null guards,
  belt-and-braces nullability the surrounding code already proves. Signals lack of
  confidence in the flow.
- **Blanket `catch` / "robust" error handling** that swallows errors or wraps them in
  generic "Something went wrong." Hides bugs. Catch narrow; rethrow with context; fail
  closed on auth/permission/transaction.
- **Useless wrapper functions** — `function getUser(id){ return User.find(id); }` adds a
  hop with no value. Inline or remove.
- **Over-engineered abstractions** for trivial problems — a factory + interface + plugin
  registry for a 10-line function. **Don't add abstraction before two real callers**
  (see `coding-style.md`, `patterns.md`).
- **Convention-blind** code — ignores the repo's naming, file layout, error patterns,
  validation style. "Generic good code" beats the project's idiom; reuse first (see
  `coding-style.md`).
- **Going beyond the spec** — features/options/configs/flags the spec didn't ask for.
  Implement exactly what was specified; flag extras as follow-ups.
- **Sycophant / fake-helpful comments** — `// helper function`, `// increment i by 1`,
  `// TODO: improve this later`. Rename or delete.
- **Tutorial-style comments** explaining basic syntax: `// loop through the array` over
  a `.forEach`. The code already says it.
- **Generic AI naming** — `process_data`, `handle_thing`, `do_it`, `result`, `data`,
  `temp`. Name for *intent*, not for the action's category.
- **Premature config** — feature flags / config knobs / extension points with no current
  user.
- **Dead leftovers** — TODOs without an owner/issue, commented-out code, unused imports,
  `console.log`s, debug prints.

## Why these are banned
They signal "a model generated this" rather than "this team designed/wrote this." They
ignore the product's register and the project's idiom, add noise, hide bugs (defensive
catches), bloat the diff (over-engineering, beyond-spec), and often fail accessibility
or correctness review. They're cargo-cult, not craft.

## What to do instead
- **UI**: project tokens / shared components / consistent type & spacing
  (`design-system-discovery.md`); content shapes layout; motion serves feedback;
  reserve modals for focused interrupting tasks.
- **Code**: validate at trust boundaries (don't sprinkle null checks); catch narrow,
  recover or rethrow; one clear name per concept; one responsibility per function;
  reuse before write (`coding-style.md`); implement exactly the spec; let inherent
  complexity be — don't pad with ceremony.
- If the project **does** use one of these intentionally, follow the project. Consistency
  beats the rule.

## When in doubt: ask
A "robust" check or shiny abstraction you can't justify in one sentence is probably slop.
Delete it; or ask the user if it should exist.
