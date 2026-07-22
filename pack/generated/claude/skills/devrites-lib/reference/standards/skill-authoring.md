# Skill authoring

Use this when creating or editing DevRites skills.

## Distribution

DevRites is installed through the npm package (`npx devrites ...`). Claude Code
and Codex files are generated host artifacts copied by that installer, never
Claude/Codex plugin-store surfaces. Edit the canonical Claude-authored pack sources,
rebuild host artifacts, then validate.

## Surface lifecycle

- **Promoted:** shipped in `pack/`, documented in `docs/skills.md` and
  `docs/command-map.md`, and covered by validation.
- **Draft:** local/research material outside the shipped pack.
- **Deprecated:** shipped only as a compatibility bridge with a replacement and
  removal note.
- **Research:** notes under `docs/research/`; never installed.

## Description

The description is an invocation pointer, not documentation.

- **Model-invoked** skills pay context load so the agent or another skill can reach them;
  omit `disable-model-invocation` and give them trigger-bearing descriptions.
- **Explicit-only** skills pay human cognitive load instead; set
  `disable-model-invocation: true`, keep the description a human summary, and expose them
  through `/rite`. The Codex generator must map this to
  `policy.allow_implicit_invocation: false` without stubbing a public description.
- Keep public model-invoked skills under 90 words, internal specialists under 75,
  explicit-only skills under 30, and `devrites-lib` under 60.
- Front-load one stable leading word that is also used in prompts/docs when that concept
  should trigger the skill.
- Use one clear trigger branch per phrase; repeated `Use when` or `Not for` means the branch should collapse or move into the body.
- State the **defining constraint**: the one fact that separates this skill from its nearest sibling (e.g. `/rite-seal` decides, `/rite-ship` mutates git). It is the strongest trigger discriminator the routing evals measure.
- Put examples, edge cases, and rationale in `SKILL.md` body or a reference file, not in frontmatter.

## Body

- Put ordered work as steps, each ending in a checkable completion criterion.
- Move branch-only reference behind a direct file pointer.
- Keep one meaning in one place; prefer a shared reference over repeated prose.
- Add an explicit setup/engine pointer only where the skill produces *wrong* output without
  the config; where it merely sharpens output, plain prose ("the conventions ledger, if
  present") is enough: cargo-culted pointers spread as sediment.

## Router and docs

- Public `rite-*` skills must appear in the `/rite` router, `docs/skills.md`,
  and `docs/command-map.md`.
- Internal `devrites-*` skills must stay out of the public command menu unless
  named as implementation detail.
- A public skill's docs card states purpose, when to invoke, where it fits,
  its defining constraint (as plain prose, never a labelled aside), and what
  evidence proves completion. Do not copy the full `SKILL.md` process into docs.
- Model-invoked skills need positive/negative implicit-routing evals. Explicit-only
  public skills need direct-command evals; non-workflow libraries are exempt explicitly.

## Source intake

External skill packs, articles, and examples are references, not authority.

- Record source, commit/date, and files read in `docs/research/`.
- Adopt the DevRites principle, not foreign names or workflow chains.
- Name rejected ideas so future maintainers do not re-litigate them.
- Add a validator or eval when the adoption creates a durable product contract.

## Match the form to the failure

Pick the instruction form from the *observed* failure, not by habit:

- Agent **violates a rule under pressure** → hard guardrail + rationalization rebuttal
  (the `anti-patterns.md` table form) + a red-flag stop list.
- Output has the **wrong shape** (bloated, buried, missing emphasis) → a positive recipe or
  template with REQUIRED slots. Prohibitions backfire here: wording tests show a "don't"
  list produces *more* of the unwanted shape than no guidance at all.
- Agent **omits a required element** → a structural slot in the artifact template, not a
  prose reminder.
- Behavior should **depend on a condition** → a conditional keyed to an observable
  predicate, not an unconditional rule with exemption clauses ("unless it matters" reopens
  the negotiation).

## Wording evals

A wording change to behavior-shaping content is a code change: prove it:

1. **Baseline first (no-guidance control).** Run the scenario without the new wording; if
   the control doesn't exhibit the failure, the guidance is a no-op: don't author it.
2. **≥5 reps per variant, fresh context each.** Single samples lie; read every flagged run.
3. **Variance is a signal.** Five runs, five interpretations = the wording isn't binding:
   rewrite, don't average.

## Pruning

Delete no-op instructions the model already follows. Keep positive target behavior; use prohibitions only for hard guardrails.

Read a draft for its **negative space**: every decision the skill declines to make is
delegated to the model's priors, not left neutral. Decide each silence deliberately: fill
it, or leave it open as a real branch.


## Contribution preflight

New skills are expensive routing surface. Before adding one, document the catalog search, why a reference inside an existing skill is insufficient, required eval coverage, host command parity, and whether the surface is public `rite-*` or internal `devrites-*`. Public commands need docs, evals, generated Claude/Codex artifacts, and a reply-contract marker. Internal skills need a clear trigger boundary and "when not to use" section. Agents need role/scope, read/write mode, output format, and composition block; only `devrites-slice-wright` may write.
