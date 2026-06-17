# `strategy.md` template + fold-back rules

`/rite-temper` produces two kinds of output. The `strategy.md` artifact is the durable
**record** of the review; the **fold-back edits** are what the build actually follows. The
record without the fold-back is dead prose — the build reads `spec.md`, not `strategy.md`.

## `strategy.md` — the record
Write to `.devrites/work/<slug>/strategy.md`. If one exists for the slug, **update** it, don't
clobber.

```markdown
# Strategy: <slug>
Tempered: <iso>

## 1. Significance
full | skipped — low stakes (<trigger>)        # if skipped, stop here (see significance.md)

## 2. Scope mode
Mode: expand | selective | hold-rigor | reduce-to-MVP
Rationale: <why this mode>
Hinge: <what evidence/answer would change the call>

## 3. Forward pass (ambition)
10-star outcome (the deliberate overshoot): <one or two lines>
Premises challenged: <load-bearing assumption(s) + verdict>
Converged sweet spot: <where we landed and why — this, not the overshoot, is what folds in>

## 4. Pre-mortem (inversion)
| Failure mode (past tense) | Likelihood | Mitigation | Owning slice/criterion |
|---|---|---|---|
| It shipped and <…> | high/med/low | <mitigation> | <slice or FR it binds to> |

## 5. YAGNI ledger
| Candidate scope item | later-refactor cost | Verdict |
|---|---|---|
| <item> | trivial / large | build-now / defer (revisit: <trigger>) |

## 6. Cross-cutting coverage
| Concern | Addressed? |
|---|---|
| Security / trust boundary | <how | N/A — why> |
| Data & migration | <… | N/A> |
| Observability | <… | N/A> |
| Modifiability | <… | N/A> |

## 7. Dimension scores (floor-gate)
| Dimension | Band | Evidence |
|---|---|---|
| Problem altitude & ambition | strong/adequate/thin/broken | <spec line / absence> |
| … (all 9) | … | … |
Floor verdict: <weakest band> on <dimension> → pass | blocked

## 8. Deferred / Won't-have-this-time
- <item> — <one-line reason> (revisit when: <trigger>)

## 9. Reviewer loop
- iter 1: <findings → resolution> · iter 2: … (≤3)
- Final: floor = <band>; unresolved → <none | blocking qid>
```

## Fold-back — the part the build follows
Apply through the **Spec Drift Guard** (these are spec changes, not free edits). If a plan
already exists, record in `drift.md` and route via `/rite-plan repair` instead of editing
`spec.md` blind.

- **`spec.md`** (these are the spec template's actual section headings — edit each named one,
  don't collapse pairs):
  - *Success criteria* **and** *Acceptance criteria* (separate sections) — add to **both** for
    every opt-in **expansion**; remove (via the Guard) from both for every **reduction**. Each
    must stay measurable + technology-agnostic.
  - *Non-goals* — append every deferred item (from the YAGNI ledger + the Deferred register), so
    it is neither silently dropped nor silently re-injected mid-build.
  - *Constraints* **and** *Risks* (separate sections) — tighten *Constraints* where the pre-mortem
    demands; add the top failure modes + mitigations to *Risks*.
  - *Gaps, issues & decisions* table — one row per scope call (options offered → decision → owner).
  - **Re-run the Readiness gate** at the bottom of the spec after edits; it must still pass, and
    **every folded scope delta must trace to a recorded decision** (a `questions.md` qid or a
    `decisions.md` ADR) — an untraceable change is the batch-dump failure.
- **`decisions.md`** — one ADR-style entry per scope call and accepted trade-off:
  `context · decision · why-not-the-alternative · what-would-change-it`.
- **`assumptions.md`** — every "we'll probably need X" demoted to an explicit
  **assumption-to-verify**, never smuggled into scope as a fact.

## Write discipline
- **Single canonical writer.** The skill writes `strategy.md` and edits the canonical files; the
  `devrites-strategy-reviewer` agent is **read-only** and only returns findings + bands.
- **No silent scope.** An expansion that the human didn't opt into, or a reduction that drops an
  acceptance criterion without a recorded decision, is a defect — not a convenience.
- Add `strategy.md` to the workspace between `spec.md` and `plan.md`; `/rite-define`,
  `/rite-review`, and `/rite-seal` read it.
