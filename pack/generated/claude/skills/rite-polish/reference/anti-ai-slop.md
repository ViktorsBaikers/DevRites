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
- **Comment noise — the most common code tell.** Default to **zero** comments; the code and
  the names carry the meaning. A comment earns its place only by answering *why* in one
  sentence (intent, a trade-off, a non-obvious constraint, a "here be dragons" warning). Cut
  every comment that restates the code:
  - *What-comments* — `// increment i by 1`, `// set the user name`, `// return the result`.
  - *Tutorial comments* — `// loop through the array` over a `.forEach`; `// check if null`.
  - *Sycophant / filler* — `// helper function`, `// this is important`, `// magic happens here`.
  - *Ownerless TODOs* — `// TODO: improve this later` with no issue/owner.
  - *Meta / edit-process comments* — the agent narrating its own edit: `// Now I'll add error
    handling`, `// Updated to handle the edge case`, `// As requested`, `// Step 1: … Step 2:`.
    The reader doesn't care how the diff was produced.
  - *Hedging / apologetic / overconfident comments* — `// should work`, `// hopefully handles
    this`, `// I think`, `// hacky`, `// sorry`, or `// obviously` / `// just` / `// trivial`.
    They admit the code is unverified (or paper over it). Verify the code; delete the doubt.
  Density smell: more than roughly **one comment per ~10 lines of straightforward code** means
  you are narrating, not explaining. Rename the thing or delete the comment.
- **Names must match the contract.** A name is a promise about what the symbol does — keep it
  honest. A `validateUser()` that actually checks payment status, an `isReady` that mutates
  state, a `getUser()` that also writes a log: the name lies. Models pick names that *sound*
  plausible for the category; verify the name says what the code does.
- **Generic AI naming** — `process_data`, `handle_thing`, `do_it`, `result`, `data`,
  `temp`, `manager`, `helper`, `util2`. Name for *intent*, in the repo's casing and idiom —
  not for the action's category.
- **Premature config** — feature flags / config knobs / extension points with no current
  user.
- **Dead leftovers** — TODOs without an owner/issue, commented-out code, unused imports,
  `console.log`s, debug prints.
- **Hallucinated imports & APIs** — an import of a package or module the project doesn't
  declare (absent from the manifest/lockfile) or that doesn't exist, and invented methods or
  parameters on a real library. Unused imports are dead code; *non-existent* ones are the model
  inventing dependencies. Verify every import resolves and every unfamiliar API exists at the
  source (`devrites-source-driven`) — never invent one.
- **Placeholder bodies posing as complete** — a function that looks implemented but only does
  `pass` / `...` / `return None` / `throw NotImplementedError` / returns a constant. It promises
  functionality that isn't there. A genuine `@abstractmethod` / interface / Protocol stub is
  fine — the slop is the stub pretending to be the real implementation.
- **Fake or inflated docstrings** — a generic "This function does X" / "Handles the logic" that
  restates the signature, or a 10-line docstring over a 2-line body. A real public-API docstring
  earns its place; the slop is the one that inflates or says nothing. Follow the project's doc
  convention either way.
- **Oversized units (smell, not a hard gate)** — a function past ~50 logical lines, cyclomatic
  complexity >10, more than ~4 parameters, or nesting ≥4 deep is a god-function smell. Split it
  or flatten with guard clauses; judge in context, don't game the metric.
- **Unexplained magic constants** — a bare literal, a hardcoded URL/endpoint, or an embedded
  account/provider/test ID inline with no name or source. Give it a name (a const) or a home
  (config/env). The mirror image of premature config, and just as much a tell.
- **Copy-paste duplication** — a near-identical block pasted and tweaked instead of reused.
  Reuse → extend → build new (`coding-style.md`, `patterns.md`); duplication beats the *wrong*
  abstraction, but pasted clones are slop, not a deliberate AHA call.

### Comment density — before / after

```js
// Before (slop): a comment narrating almost every line
function calculateTotal(items) {
  // initialize the total to zero
  let total = 0;
  // loop through each item in the items array
  for (const item of items) {
    // add the item price to the total
    total += item.price;
  }
  // return the final total
  return total;
}

// After: the names carry it; no comment needed
function sumPrices(items) {
  return items.reduce((total, item) => total + item.price, 0);
}

// A comment that earns its place — it explains WHY, not what:
// Prices are in minor units (cents); the gateway rejects fractional amounts.
const total = sumPrices(items);
```

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
