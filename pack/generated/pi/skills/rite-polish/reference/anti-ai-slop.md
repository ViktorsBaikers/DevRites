# Anti-AI-slop

Tells of generic LLM-generated work — in **UI** and in **code**. Applies at two stages:

- **At build time** (preventive) — `/rite-build` checks against these as it writes; the
  cheap moment to avoid them.
- **At polish time** (catch) — `/rite-polish` Phase 1 (Code) + Phase 2 (Backend) + Phase 4
  (UI) scrub anything that slipped through.

Avoid these unless the project's existing system explicitly uses them. When in doubt,
match the neighbors.

## UI admission rule (tells are leads, not findings)
Build avoids every UI entry; review and polish admit a hit only by this rule. Classes:
**ENG** (engineering/realism), **A11Y** (WCAG 2.2 SC or platform requirement), **DS**
(binding only where project tokens, components or `design-brief.md` declare it), **AES**
(optional aesthetic). A hit is a finding only with cited mismatch evidence: it contradicts
a declared token, brief line or nearest neighbour (cite both locations), or it has a
measured user consequence (contrast ratio, overflow, CLS, lost or hidden focus). Otherwise
it is an AES proposal under "Proposals (non-blocking)": never a Visual Verdict FAIL, never
scored, capped at Suggestion/Minor, never blocking unless `design-brief.md` adopts it (then
DS). Act on clusters per [`prose-style.md`](../../devrites-lib/reference/standards/prose-style.md).
A broken task, data loss or misleading state is reported and fixed before any AES item.
**Failing case:** `tokens.css` declares Inter and a 20 px card radius, yet review files
"reflex font" and "balloon radius" as Important; or three eyebrows on six sections become
a FAIL row with no brief conflict.

**UI-copy boundary.** Craft owns functional copy: action-named controls, errors with cause
and recovery, next-step empty states, product vocabulary, fit under expansion, and an
accessible name containing the visible label (SC 2.5.3, A11Y). Long-form product text
routes to `devrites-prose-craft`; translation catalogs stay untouched without their owner.

## UI anti-slop (banned defaults)
- [AES] Default **purple/blue gradients** as the brand look. Special case: any
  hero hex in the `#6366f1 → #a855f7 → #ec4899` family applied as a default.
- [AES] **Gradient text** (`background-clip: text`) used decoratively on headings.
- [AES] **Glassmorphism** (`backdrop-filter: blur(...)` on translucent panels) as
  a default surface.
- [AES] **Side-stripe colored borders** on cards/sections — the
  "tiny-bar-of-meaningful-accent-color-on-the-left" pattern. Distinctive
  templating tell.
- [AES] **Pure `#000` / `#fff`** as raw text or background — too clinical; use
  near-black/near-white tokens (`oklch(0.18 0 0)` / `oklch(0.98 0 0)` or
  the project's surface tokens).
- [AES] **All-CAPS body text** for paragraphs/labels. Reserve uppercase for short
  micro-copy (badges, eyebrows); never for sentences.
- [AES] **Em dashes in UI copy** are a lead, not a ban: DS only where the project's
  style guide bans them; otherwise no rule applies to product copy, because
  [`prose-style.md`](../../devrites-lib/reference/standards/prose-style.md) governs DevRites
  prose, not product UI.
- [AES] **Cards inside cards** — nested bordered/elevated containers.
- [AES] **Identical card grids** for everything, regardless of content.
- [AES] A **generic rounded-square icon tile** above every heading/section.
- [A11Y] **Gray text on colored backgrounds** — a finding with the measured ratio (SC 1.4.3).
- [AES] The **hero-metric cliché** — three big numbers in a row with no real meaning.
- [AES] **Decorative bounce / elastic easing** on everything; motion without purpose.
- [AES] **Reflex fonts** picked because they're the default in a tutorial:
  - Inter for every product when the project has its own choice.
  - **DM Sans**, **Plus Jakarta Sans**, **Fraunces**, **Newsreader** when
    they're not the project's actual type system.
  Match the project; don't reach for the "tasteful default" of 2024.
- [AES] **Modal-first thinking** — reaching for a modal as the answer to every interaction.
- [AES] **Ghost-card** — a `1px` border *and* a soft (blur ≥16px) shadow on the same element.
  Borders separate; shadows lift — pick one
  ([`quality-standards.md`](../../devrites-frontend-craft/reference/quality-standards.md) — Materiality).
- [AES] **Balloon radius** — `border-radius` above ~16px on cards, inputs, panels. Cards top
  out around 12–16px; full pills are for tags and buttons only.
- [AES] **Uppercase tracked eyebrow above every section** — one named kicker is voice; one per
  section is template grammar (countable cap below).
- [AES] **Numbered-section scaffolding** (`01 / 02 / 03`) when the sections aren't a sequence.
- [ENG] **Fake UI-in-a-div** — a "product screenshot" assembled from nested divs, or hand-drawn/
  sketchy SVG scenery. Ship a real capture/asset or nothing (quality-standards — Materiality).
- [AES] **Hero prop badges** — version labels (`V0.6`, `BETA`) and decorative pulsing status
  dots as set dressing.
- [AES] **UA chrome** — default selection, caret or scrollbar; the UA focus ring is a
  finding only when SC 2.4.7/1.4.11 fails (quality-standards § Browser chrome).
- [ENG] **Invented metrics** — precise-looking numbers with no source (`+247% faster`,
  `99.99% uptime`). [DS] **Off-token color** — a hex used inline that the token set doesn't
  define. Both are realism failures, not styling choices.

### Required remediations (fix the hit, don't just flag it)
A slop finding names its remediation from this table; a ban without a stated fix is
an incomplete finding.

| Signature | Required remediation |
| --- | --- |
| Default purple/blue gradient; gradient text; glassmorphism default | Re-derive from the scene sentence and colour commitment, then re-run both category-reflex tests |
| Invented metric | Replace with the state lattice's missing-data placeholder (quality-standards § Focus & states) plus a "metric to confirm" question, or delete the proof slot |
| Off-token color | Lift into the token set as a named color; replace every inline use |
| Wrapping CTA / nav overflow | Shorter label or a collapsed nav, verified at 200% text zoom — never `white-space: nowrap` or a shrunken tap target |
| Horizontal scroll in 320–1920 | Fix intrinsic sizing (`minmax(0, 1fr)`, wrapping, media constraints) with the bounded exception of quality-standards § Responsive — never `overflow-x: clip` on `html`/`body` |
| Fake UI-in-a-div screenshot | Ship a real capture/asset or remove the block |
| Sticky sub-nav hidden by a banner | Offset by the banner's token height; split z-index roles instead of one raised value |
| UA focus ring failing SC 2.4.7/1.4.11 | Apply the DS focus-ring token per quality-standards § Browser chrome |

### Copy & data realism
Placeholder content is a tell even when the layout is clean: fake-perfect numbers
(`99.99%`, `10,000+` — real data is ragged), placeholder people/brands ("John Doe",
"Acme"), filler verbs (Elevate / Seamless / Unleash). Re-read every visible string before
shipping; AI-cute copy is worse than boring copy.

### Category-reflex check — run at two altitudes

Most generic-AI design fails one of these two reflex tests. Run both — the
second one catches what the first one misses.

- **First-order:** if someone could guess the theme + palette *from the
  category alone* — "observability → dark blue", "healthcare → white +
  teal", "fintech → navy + gold", "AI tool → black with a violet accent",
  "crypto → neon on black" — the styling is on the first training-data
  reflex. Rework the scene sentence
  ([`design-references.md`](../../devrites-frontend-craft/reference/design-references.md)) and the
  colour-commitment strategy
  ([`quality-standards.md`](../../devrites-frontend-craft/reference/quality-standards.md)) until the
  answer isn't obvious from the domain.
- **Second-order:** if a stranger looked at the surface with *no copy
  visible* and confidently said "this is a CRM / fitness tracker / fintech
  / AI workflow tool", the styling is still on a category template — just
  one tier deeper. The first reflex was avoided, the second wasn't.
  Re-shape until the surface doesn't telegraph its category from looks
  alone.

Both pass = the surface looks like *this product*, not "an app in this category".

### Mechanical pre-flight (countable — run, don't vibe)
Each is counted or grepped, not judged. ENG and DS rows pass or fail; an AES count past
its cap is a lead for the admission rule, not a failure:
- [AES] **Em dashes** in visible UI strings: counted; DS only where the style guide bans them.
- [AES] **Eyebrows** (uppercase-tracked kickers): ≤ `ceil(sections / 3)`.
- [AES] **Layout families** (hero, image+text split, card grid, bento, table…): no family more
  than twice per page; never 3 consecutive image+text zigzags.
- [DS] **Icons**: 0 emoji-as-icon; exactly one icon set imported.
- [ENG] **State presence in code**: every state the role lattice requires (quality-standards § Focus & states) — an unreachable state's screenshot proves nothing.
- [ENG] **Form inputs**: no border-width shifts between states; focus ring from outline/ring (not border swap); consistent control height; reserved helper slot; disabled beyond opacity alone.

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
  (see [`coding-style.md`](../../devrites-lib/reference/standards/coding-style.md), [`patterns.md`](../../devrites-lib/reference/standards/patterns.md)).
- **Convention-blind** code — ignores the repo's naming, file layout, error patterns,
  validation style. "Generic good code" beats the project's idiom; reuse first (see
  [`coding-style.md`](../../devrites-lib/reference/standards/coding-style.md)).
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
  Reuse → extend → build new ([`coding-style.md`](../../devrites-lib/reference/standards/coding-style.md), [`patterns.md`](../../devrites-lib/reference/standards/patterns.md)); duplication beats the *wrong*
  abstraction, but pasted clones are slop, not a deliberate AHA call.

## Why banned, what instead
They signal model-generated rather than team-designed work: they ignore register and
idiom, add noise, hide bugs (defensive catches), bloat diffs, and often fail a11y or
correctness review. Instead: project tokens/components, validate at trust boundaries,
catch narrow and rethrow, one clear name per concept, reuse first ([`coding-style.md`](../../devrites-lib/reference/standards/coding-style.md)),
implement exactly the spec. If the project intentionally uses one of these, follow the
project — consistency beats the rule. A check or abstraction you can't justify in one
sentence is slop: delete it or ask.
