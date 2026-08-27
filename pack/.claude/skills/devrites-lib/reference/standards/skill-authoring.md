# Skill authoring

> **Source-checkout only:** edit canonical `pack/.claude/`; run
> `bash scripts/build-host-artifacts.sh`, then validate. Installed generated mirrors are not authoring surfaces.

## Surface lifecycle

- **Promoted:** validated in `pack/`, `docs/skills.md`, `docs/command-map.md`.
- **Draft:** local, outside `pack/`.
- **Deprecated:** bridge with replacement/removal note.
- **Research:** `docs/research/`, never installed.

## Routing metadata

Description routes; it is not documentation.

- **Model-invoked:** omit `disable-model-invocation`; use a trigger-bearing
  description.
- **Explicit-only:** set `disable-model-invocation: true`, use a human summary,
  expose through `/rite`; generate Codex
  `policy.allow_implicit_invocation: false` without a stub description.
- Caps: public model-invoked 90 words; internal 75; explicit-only
  30; `devrites-lib` 60. Agent descriptions: 45 words.
- Model-visible `name` + `description` ≤5,200 routing characters;
  `explicit-only` and bodies/references do not count.
- Front-load one stable prompt/docs trigger. Allow at most one `Use when` and one `Not for` branch;
  move other detail into the body.
- State the nearest sibling's **defining constraint** (Seal decides; Ship mutates
  Git). Routing evals test it.
- A routing/tie-breaker change cites the mis-route it fixes and passes trigger corpora; no failing case, no change.
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

- Ordered steps end in checkable criteria.
- One read shows outcome, triggers, preconditions, decisions/failure, write owner,
  proof, exit; omit irrelevant fields. Examples distinguish branches.
- Split only for independent load path or eval-proven inline failure; keep one owner; co-locate each rule/caveat/example cluster.
- Every public optional-flag skill obeys the shared
  [`core.md`](core.md#operating-rules-every-phase): declare its
  complete flag surface in `argument-hint`,
  normalize the current invocation once
  before writes, fail closed on value-flag absence/malformed/duplicate/conflict,
  and add a fail-closed regression check for value flags.
  - A narrow explicit-only utility may state the equivalent local guard instead of loading core.
- Add setup/engine pointers only when absence makes output wrong.

Classify active instructions by load path:

- `core.md`: required by every workspace rite;
- on-demand reference: one rule, ≥2 named active consumers, same observable failure when absent;
- workflow/agent local: one owner, scoped procedure;
- human/research docs: explanatory/proposed, never active-run authority.

Keep one-consumer rules local; never promote for visibility or move mandatory
rules to inactive docs. Before consolidating/relocating/substantially rewriting,
map every prior `MUST`, `MUST NOT`, trigger, input/output, failure/escalation path,
safety gate, and compatibility promise to its owner; verify every old load path.
Retirement needs error/obsolescence evidence + deprecation/compatibility; omission
regresses.

## Router, docs, and evals

- Public `rite-*`: `/rite` router + `docs/skills.md` + `docs/command-map.md`.
- Internal `devrites-*`: stay off the public menu unless named as implementation.
- A public docs card states purpose, invocation, lifecycle position, defining
  constraint in plain prose, and completion evidence; never copy the full process.
- Model-invoked skills need positive/negative implicit-routing evals; explicit-only public skills need direct-command evals; non-workflow libraries are exempt.

## Source intake

External sources are references, not authority. Promote only when one
`docs/research/` admission record contains:

- **Provenance:** origin, review date/files, adaptation, derived targets; external assets add
  source URL/SHA/path/license, local/user assets add relative path/digest/owner. Unverified
  external origin/rights → reference-only, independently written prose.
- **Gap + owner:** observed failure and existing canonical owner; extend before adding.
- **Adaptation + cost:** native delta, no foreign brands/paths/host assumptions; justify every
  dependency, context, process, hook, agent, or command.
- **Proof + disposition:** positive/negative checks, host/package parity, rejection reasons.

Missing field → no promotion.

## Skill trust tiers

Every skill or agent surface belongs to exactly one trust tier. Higher tiers may
constrain lower ones; nothing may weaken shipped gates or permissions.

| Tier | Source | Authority | Install check |
| --- | --- | --- | --- |
| **shipped** | `pack/.claude/` built by CI | Full workflow authority | manifest hash + host parity |
| **project-local** | Repo-scoped customization approved by a human | May extend project rules; cannot weaken DevRites method | `devrites-engine check skill-trust` on the path |
| **imported** | External skill with `docs/research/` admission record | Read/adapt only after provenance review | skill-trust scan + admission record required |
| **untrusted** | Unknown origin or failed scan | Reference-only; never executable authority | block on any HIGH finding |

Before promoting/installing project-local/imported Markdown, run:

```bash
devrites-engine check skill-trust <path>
```

HIGH findings (injection override prose, suspicious Unicode, credential exfil, sensitive paths) block install; MEDIUM requires explicit human acknowledgment in the diff, not silent merge.

## Match form to failure

- Rule breaks under pressure → hard guard + rationalization rebuttal + stop list.
- Wrong shape → positive template/recipe; prohibitions reinforce that shape.
- Missing element → artifact-template slot, not prose reminder.
- Conditional behavior → observable predicate, not negotiable exemption prose.

## Wording evals

Behavior-shaping prose is code:

1. Baseline without guidance; if it passes, add none.
2. Run ≥5 fresh-context reps/variant; inspect every flagged run.
3. Divergent interpretations require rewrite, not averaging.
4. Pin host/model/build, corpus, grader, and candidate digest or commit+path. Report tasks/trials,
   arms, same-build A/A noise before A/B, sanitized per-trial verdicts/metrics, invalid/null results,
   variance, process versus job outcome, and supported/unproved claims. Never capture raw transcripts;
   lost grading signal is `cannot_verify`.

CI validates only corpora/deterministic artifacts—never paid sessions or lexical claims.

## Pruning

Delete model-default no-ops. Prefer positive targets; reserve prohibitions for hard
guards. Fill omitted decisions or mark a deliberate branch.

## Contribution preflight

Record catalog search, owner gap, evals, host parity, and public/internal surface. Public
commands need docs/generated hosts/reply marker; internal skills need trigger/exclusion and
skill-not-agent proof. Agents need role/scope/mode/output/composition plus
[Result admission](agents.md#result-admission) for reviewers. Only `devrites-slice-wright`
writes product source/tests; root-owned bounded `.devrites/**` follows `workflow-artifacts.md`.

## Coverage-gap review (maintainer pass)

1. Verdict each candidate domain `covered`/`partial`/`absent` against named owners.
2. Gap needs consumer evidence: frequency × purpose (observable failure without it); unverifiable ⇒ no adoption.
3. ≤2 net-new guidance files per round; prefer extending a standard; accepted file names load trigger + non-trigger before shipping.
4. Rejections record reasons; revisit only on changed evidence.
