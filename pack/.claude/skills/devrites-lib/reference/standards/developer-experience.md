# Developer experience

When a change ships a surface another engineer has to *use* — a public API, a CLI, an SDK
or library, a webhook, a config/env contract, an error message, or the getting-started path
that introduces all of them — the developer using it is a user, and their experience is part
of the change. Code that compiles and passes its tests can still strand the next developer at
a cryptic error, a getting-started snippet that doesn't run, or an endpoint whose shape you
have to read the source to learn. Un-tested DX is a claim, the same way un-run code is a claim.

This rule is the developer-facing counterpart to [`performance.md`](performance.md): perf says
*measure the speed before you assert it*; this says *measure the experience before you assert
it*. It pairs with the [`devrites-api-interface`](../../../devrites-api-interface/SKILL.md)
skill (which shapes the contract) — this rule **proves** the shape is usable.

## Scope — when this applies

Conditional, like [`performance.md`](performance.md) and [`observability.md`](observability.md).
It applies when the diff changes a **developer-facing surface**:

- a public/external **API** endpoint, response shape, or status-code contract;
- a **CLI** command, flag, output, or exit code;
- an **SDK / library / package** export, signature, or type;
- a **webhook**, event payload, or integration contract;
- a **config / env-var** interface a consumer must set;
- an **error message, exit code, or failure response** a developer reads to recover;
- the **getting-started path** — README, install steps, quickstart, the first-run docs.

Skip it for pure-internal refactors, private helpers, app-only UI (that's
[`devrites-frontend-craft`](../../../devrites-frontend-craft/SKILL.md) /
`devrites-frontend-reviewer`), docs-only typo fixes, and config that no external consumer
touches. Don't DX-review a change with no developer-facing surface — the same scope discipline
as the rest of the conditional rules. End-user UX is a different axis; this rule is about the
*developer* consuming the surface.

## The boomerang — predict at plan, measure at prove, reconcile at seal

DevRites already runs a predict-then-prove spine (the vetted `test-plan.md` coverage target at
`/rite-vet`, proven at `/rite-prove`, gated at `/rite-seal`). DX rides the same spine, and the
gap between the two ends is the signal:

1. **Predict** — at `/rite-vet`, when a developer-facing surface is in scope, score the *planned*
   surface against the dimensions below and write a predicted scorecard to `devex.md` (the
   estimate, e.g. "time-to-hello-world: ~3 min"). This is cheap and pre-build; it forces the
   ergonomics question before the contract sets.
2. **Measure** — at `/rite-prove`, actually exercise the surface (run the getting-started flow,
   call the endpoint, invoke the CLI, trigger the error) and record the *measured* scorecard with
   evidence: real time-to-hello-world, the verbatim error text, the screenshot of the docs page.
3. **Reconcile (the boomerang)** — at `/rite-seal`, compare predicted against measured. A material
   gap — "the plan said 3 minutes, the getting-started flow actually took 8 and step 4 errored" —
   is a finding, not a rounding error. The estimate was wrong *or* the surface regressed; either
   way the developer using it pays, and the gap is exactly what a single end-state score would hide.

The boomerang is what makes DX a measured axis rather than an opinion: the prediction is on
record, so the measured reality can falsify it.

## The scorecard — what to score

One entity, one name: call the metric **time-to-hello-world (TTHW)** consistently, not "onboarding
time" in one place and "setup time" in another. Score these dimensions; only the ones the diff
touches:

- **Discoverability** — can a developer find the entry point without reading the source? Is the
  new surface named, exported, and documented where they'll look?
- **Time-to-hello-world (TTHW)** — wall-clock from "I have the repo/package" to "I got one
  successful call/response/render". The headline number; measure it, don't estimate it at seal.
- **Getting-started friction** — does the quickstart run *as written*, copy-pasted, on a clean
  checkout? Every undocumented prerequisite, wrong command, or missing step is friction.
- **Error-message quality** — does a failure say *what* failed, *why*, and *how to recover*, with
  the relevant ids (never secrets — see [`security.md`](security.md))? A bare stack trace, a
  silent exit, or "an error occurred" is a defect on a developer-facing path.
- **Ergonomics & consistency** — does the surface match the project's existing conventions
  (naming, argument order, pagination, error shape)? An inconsistent new endpoint taxes everyone
  who learned the old ones. Sensible defaults; the common case is one call.
- **Docs accuracy** — examples are copy-pasteable and correct, the signature in the docs matches
  the code, and changed behavior updated its docs in the same change ([`documentation.md`](documentation.md)).

## Measure, don't assert

The same discipline as `performance.md` "measure first" and `testing.md` "see it fail first":

- **Run it, don't read it.** A scorecard backed by "the code looks fine" is Source mode and says
  so. The graded scorecard comes from actually invoking the surface — the getting-started flow on
  a clean state, the real CLI `--help`, the real error path — and recording what happened.
- **Quote the artifact.** Paste the verbatim error string, the exact failing command, the measured
  TTHW; for a docs/quickstart page, capture it through the browser-proof ladder
  ([`../skills/devrites-browser-proof/SKILL.md`](../../../devrites-browser-proof/SKILL.md)) and
  describe the screenshot. A path is not proof; the observation is.
- **No measurement → no DX claim**, and usually no finding above Suggestion. "Feels confusing" is
  a hypothesis to test, not a verdict.

## The gate — severity by who-pays

DX findings carry the standard labels (Critical / Important / Suggestion / Nit / FYI), scaled to
the cost the consumer bears, not to how polished it feels:

- A **public/external contract that ships broken or wrong** — a documented command that errors, an
  endpoint whose response contradicts its contract, a getting-started flow that can't complete — is
  **Important**, and **Critical** when it's a frozen public surface (the principles-gated public-API
  invariant, [`principles.md`](principles.md)) or an irreversible break.
- A **measured DX regression** against the predicted scorecard with no recorded reason is a
  finding the boomerang surfaces at seal — at least **Important** when it lands on a public surface.
- An **unactionable error message** on a developer-facing failure path is **Important** — the
  on-call test ([`observability.md`](observability.md)) applied to the consumer instead of the operator.
- Inconsistent-but-working ergonomics, a thin doc, a missable default → **Suggestion**, unless the
  spec made it acceptance.

## The artifact — `devex.md`

When a developer-facing surface is in scope, the workspace carries `devex.md`: the **predicted**
scorecard (from `/rite-vet`), the **measured** scorecard with evidence (from `/rite-prove`), and
the **boomerang delta** (reconciled at `/rite-seal`). Absent surface → no `devex.md`, and the gate
passes silently — never block a change for the *absence* of a DX surface, the same no-op discipline
as the principles and spec-grammar gates.

## Scope discipline

Review the developer-facing surface the change *touches*. A surface the diff didn't change is not
this change's job; a project-wide DX audit (every endpoint, the whole CLI) is its own effort —
record it as a follow-up, don't smuggle it into an unrelated change.
