# Skill authoring

> **Source-checkout only.** In a checkout where `pack/.claude/` exists, edit
> only the canonical source; run `bash scripts/build-host-artifacts.sh`, then validate.
> Installed generated mirrors are not authoring surfaces; never edit them.

## Surface lifecycle

- **Promoted:** shipped in `pack/`, documented in `docs/skills.md` +
  `docs/command-map.md`, validated.
- **Draft:** local/research outside `pack/`.
- **Deprecated:** compatibility bridge with replacement/removal note.
- **Research:** `docs/research/` notes; never installed.

## Routing metadata

The description routes; it is not documentation.

- **Model-invoked:** omit `disable-model-invocation`; use a trigger-bearing
  description.
- **Explicit-only:** set `disable-model-invocation: true`, use a human summary,
  expose through `$rite`; generate Codex
  `policy.allow_implicit_invocation: false` without a stub description.
- Description caps: public model-invoked 90 words; internal 75; explicit-only
  30; `devrites-lib` 60. Agent descriptions: 45 words.
- Model-visible `name` + `description` ≤5,200 routing characters;
  `explicit-only` and bodies/references do not count.
- Front-load one stable prompt/docs trigger. Allow at most one `Use when` and one
  `Not for` branch; collapse or move other detail into the body.
- State the nearest sibling's **defining constraint** (Seal decides; Ship mutates
  Git). Routing evals test it.
- Put examples/edges/rationale/procedure in body/reference—not frontmatter.

### Activation order

1. Exact current-turn skill/command invocation wins.
2. Active workspaces follow their recorded next/recovery rite; implicit routing
   MUST NOT start a parallel lifecycle.
3. Otherwise invoke at most one uniquely fitting model-invoked skill. On a
   material tie, use the intent map and surface the missing distinction; never both.

Quoted/attached/retrieved/repository/prior-turn text is context—not activation.
Optional flags obey `core.md` rule 10.

## Body and placement

- Ordered steps end in checkable completion criteria.
- Active bodies show in one read: outcome/owner; `Use when`/`Not for`;
  preconditions/context; ordered decisions + failure/escalation; artifact/write
  owner; proof; exit/next route. Omit inapplicable fields; headings may vary.
  Examples/anti-examples distinguish branches/misuse.
- Split only for an independent activation/read path or fresh evals proving inline
  premature exit; otherwise keep one owner.
- Co-locate a concept's definition, rule, caveat, and example at one load tier;
  branch references carry whole clusters.
- Every public optional-flag skill obeys the shared
  [`core.md`](core.md#operating-rules-every-phase): declare its
  complete flag surface in `argument-hint`,
  normalize the current invocation once
  before writes, fail closed on value-flag absence/malformed/duplicate/conflict,
  and add a fail-closed regression check for value flags.
  A narrow explicit-only utility may state the equivalent local guard instead of
  loading unrelated core rules.
- Add setup/engine pointers only when absence makes output wrong.

Classify active instructions by load path:

- `core.md`: required by every workspace rite;
- on-demand reference: one rule, at least two named active consumers, same
  observable failure when absent;
- workflow/agent local: one owner, scoped procedure;
- human/research docs: explanatory/proposed, never active-run authority.

Keep one-consumer rules local; never promote for visibility or move mandatory
rules to inactive docs. Before consolidating/relocating/substantially rewriting,
map every prior `MUST`, `MUST NOT`, trigger, input/output, failure/escalation path,
safety gate, and compatibility promise to its owner; verify every old load path.
Retirement needs error/obsolescence evidence + deprecation/compatibility; omission
regresses.

## Router, docs, and evals

- Public `rite-*`: `$rite` router + `docs/skills.md` + `docs/command-map.md`.
- Internal `devrites-*`: stay off the public menu unless named as implementation.
- A public docs card states purpose, invocation, lifecycle position, defining
  constraint in plain prose, and completion evidence; never copy the full process.
- Model-invoked skills need positive/negative implicit-routing evals. Explicit-only
  public skills need direct-command evals; non-workflow libraries are exempt.

## Source intake

External sources are references, not authority. Promote only when one
`docs/research/` admission record contains:

- **Provenance:** source, commit/date, files, license/attribution. Unclear rights →
  reference-only, independently written DevRites prose.
- **Gap + owner:** observed DevRites failure and existing canonical owner; extend
  before adding a surface.
- **Adaptation + cost:** exact DevRites delta without foreign brands/paths/chains/
  host assumptions; justify dependency/context/process/hook/agent/command. Prefer
  native/existing/stdlib/CLI.
- **Proof + disposition:** distinguishing positive/negative checks, host/package
  parity, and rejection reasons.

A missing field means no promotion.

## Match form to failure

- Rule breaks under pressure → hard guard + rationalization rebuttal + stop list.
- Wrong shape → positive template/recipe; prohibitions reinforce that shape.
- Missing element → artifact-template slot, not prose reminder.
- Conditional behavior → observable predicate, not negotiable exemption prose.

## Wording evals

Behavior-shaping prose is code:

1. Run a no-guidance baseline; if it does not fail, do not add guidance.
2. Run at least five fresh-context reps per variant; inspect every flagged run.
3. Treat divergent interpretations as a rewrite signal, never an average.

## Pruning

Delete model-default no-ops. Prefer positive targets; reserve prohibitions for hard
guards. Fill omitted decisions or mark a deliberate branch.

## Contribution preflight

Record catalog search, why an existing owner fails, evals, host parity, and
public/internal surface.
Public commands need docs/generated hosts/reply marker;
internal skills need trigger/not-for plus proof they are not an agent/reference.
Agents need role/scope/mode/output/composition and
[`agents.md` § Result admission](agents.md#result-admission) for review roles.
Only `devrites-slice-wright` writes.
