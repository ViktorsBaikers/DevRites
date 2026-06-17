# Scope modes + the two passes

The heart of `/rite-temper`: two ordered passes (heat, then quench), converging on exactly
one scope mode. Ambition is a **generative** move followed by a **required convergence** — the
divergence is never the spec; the converged sweet spot is.

## Pass 1 — FORWARD (heat: raise ambition on the outcome)
Diverge on the *real problem*, not the obvious path. Ask:
- What would an order-of-magnitude-better **outcome** look like (the 10-star version)? Design
  the extreme, then come back to the feasible sweet spot — don't ship the extreme.
- Is a **premise** load-bearing and wrong? Name the assumptions the spec rests on; challenge
  the one that, if false, changes everything.
- Where is **under-reaching** the real risk — a timid improvement to a path we should reframe?

Ambition aims at the **outcome** (problem definition, success bar, user value) and at
complexity-**neutral** future-proofing. It never aims at the **solution surface** (that's Pass 2's
job to prune). Ground every ambitious idea in the codebase's real seams and blast radius
(`codegraph`/`graphify`) — don't invent greenfield scope a constrained codebase can't absorb.

## Pass 2 — INVERSION (quench: pre-mortem + prune)
Invert. Two subtractive moves:
- **Pre-mortem (prospective hindsight).** State it in past tense: *"It's six months out. This
  shipped and failed. What went wrong?"* List the top failure modes; for each: likelihood,
  mitigation, and the slice that will own the mitigation. Unmitigated top risks are **gating**.
- **YAGNI ledger.** Each candidate scope item gets the "imagine the later refactor" test: if
  adding it *later* isn't materially more expensive, **defer it** (with a revisit note).
  Bias hard toward defer — most presumed-needed features are never actually used. Reuse the
  pack's existing standard — `patterns.md` ("no abstraction before two real callers",
  "speculative generality") — don't invent a new YAGNI rule.

## The four modes — commit exactly one
Mode selection is the first convergence move. Record the chosen mode + rationale + the
**hinge** (what would change the call) in `strategy.md` and `decisions.md`.

| Mode | When | What it does | YAGNI / Drift Guard |
|---|---|---|---|
| **EXPAND** *(opt-in, default OFF)* | Under-reaching on the outcome is the real risk | Raise the success bar / reframe to the deeper problem; thinner-but-more-complete path to a bigger target | Expands the *problem + acceptance*, not machinery. Each added criterion is a Drift-Guard-recorded, human-confirmed (HITL) decision — never auto-grown |
| **SELECTIVE** | One or two high-leverage dimensions deserve more; the rest is right | Expand only the high-upside dimension; hold the rest at spec scope | Each selected expansion routes through the Guard; everything not selected is parked in Non-goals so it's neither dropped nor re-injected mid-build |
| **HOLD-RIGOR** | The spec is right-sized (default for irreversible-risk-heavy or already-tight specs) | No scope change — pure hardening: pre-mortem, risk mitigations, acceptance sharpening, cross-cutting coverage | Adds zero surface; trivially Drift-Guard-clean (no acceptance change) |
| **REDUCE-TO-MVP** | Over-scoped: second-system bloat, ~all must-haves, gold-plating | Hammer scope to the Minimum Usable Subset; move the rest to Non-goals with one-line reasons + revisit notes | Strongest YAGNI alignment. Cutting acceptance is **still a spec change** — record it via the Guard, never a silent trim |

## Bound ambition by reversibility
The bigger the bet, the more it must be reversible. Irreversible-risk areas (auth, migrations,
public API, data model) get the **opposite** of ambition — maximum conservatism, explicit
rollback, and they **always pause** regardless of mode (the `afk-hitl.md` irreversible-risk
list is honored verbatim). Size big bets so that if the whole thing vanished tomorrow, you'd be
fine.

## Effort foresight (frame it now, not "later")
While choosing scope, surface the late-stage questions early: *"What will surprise us at
integration? What will we wish we'd planned for at polish/test time?"* Turn each into a
constraint, a risk, or a deferred Non-goal **now** — not a "figure it out later".
