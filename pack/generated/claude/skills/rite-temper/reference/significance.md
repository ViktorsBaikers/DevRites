# Significance test — fire the full review, or skip it

`/rite-temper` is **significance-gated**: full strategic rigor only when the work is big
enough to earn it, a one-line skip otherwise. This is the shared definition used by both the
optional `/rite-temper` path and the always-run (significance-gated) `/rite-autocomplete` step,
so they agree on when it fires.

A strategic review of a five-line, reversible change is theatre; a strategic review of an
auth rewrite is the cheapest insurance you'll buy. Gate on stakes and shape, not uniformly.

## Fire the FULL review when ANY trips
- **Irreversible-risk contact** — the slice/spec touches anything on the `afk-hitl.md`
  irreversible-risk list: destructive data migration, auth/authz boundary, public-API break,
  external-service contract, filesystem destruction outside the workspace.
- **Data model** — new/changed entities, relationships, or persistence shape.
- **Cross-module blast radius** — the impact (from a code-intelligence index if available — see
  `../../devrites-lib/reference/standards/tooling.md` — or an honest estimate) crosses
  module/service boundaries or has many dependents.
- **Greenfield / ambiguous scope** — a new surface with no existing seam to follow, or a spec
  whose "right size" is genuinely open (multiple defensible scopes).
- **Multi-slice / multi-day** — the work decomposes into several slices (large enough that a
  wrong scope call is expensive to unwind).
- **The user asked** — "think bigger", "is this ambitious enough", "scope check", "pre-mortem",
  an explicit `/rite-temper` invocation, or a chosen `--mode`. An explicit ask always fires.

## Skip (low stakes) when ALL hold
- Single-module, reversible, behavior-shaped-not-structural change.
- No irreversible-risk contact; blast radius contained to the touched files.
- Scope is unambiguous — there's one obvious right size and the spec already has it.

On skip, **don't run the passes**. Write only the one-line verdict to `strategy.md`:

```markdown
# Strategy: <slug>
Significance: skipped — low stakes (<the trigger that was NOT met, e.g. "single-module, reversible, scope unambiguous">)
Next: /rite-define
```

Then recommend `/rite-define`. A skip is a *recorded* decision, not a silent omission — the
seal can see the call was made deliberately.

## In `/rite-autocomplete`
Autocomplete runs the significance test every feature. On skip → straight to `/rite-define`
(no pause). On fire → run the full review under the AFK gate ceiling: `hold-rigor` and
`reduce-to-MVP` auto-apply (they never grow acceptance); **any `expand` is a blocking pause**;
irreversible-risk findings always pause. Expansion is never auto-grown unattended.
