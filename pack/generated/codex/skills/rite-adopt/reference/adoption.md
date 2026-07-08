# Adoption — reverse-investigate + seed (the on-ramp contract)

Loaded on demand by `$rite-adopt`. Two jobs: document what the code already does (so the
lifecycle has a spec), and seed the conventions ledger (so the first slice knows the idioms).

## Reverse-investigation — what to capture

Document the **durable shape** of the project, not a line-by-line tour:

- **Current behavior** — what the adopted area does today, from the user's perspective.
- **Architecture + placement** — the layers and seams; where each kind of thing lives
  (where endpoints / components / models / migrations / tests go). Prefer a code-intelligence
  index if available — codebase-memory-mcp first, cross-checked with codegraph + graphify, else standard methods (LSP / Read/Grep/Glob) (see
  `../../devrites-lib/reference/standards/tooling.md`) — for structure, callers, and impact.
- **Commands** — the real test / build / typecheck / lint commands (run or read them; don't
  guess). Verify uncertain framework facts at the source.
- **Idioms** — naming + casing, layering, the error model, the result/exception style, the
  http/data-access pattern, the test framework + file layout.
- **Gotchas** — non-obvious constraints the code encodes (ordering requirements, "don't call
  X before Y", a framework quirk).

This feeds `spec.md` (current behavior as the baseline + the next objective) and the ledger
seeds below.

## Seeding the conventions ledger

The deliberate **bootstrap exception** to evidence-gated promotion. Normally a convention is
promoted only when a *sealed slice proved it* (`$rite-seal`); adoption seeds from **observed
existing code** so the first slice isn't blind. Keep the seeds honest:

- Seed only what the code **actually and consistently** follows — not an aspiration, not a
  one-off, not something you assumed.
- Each seed starts at the base band (one corroboration) and is provenance-tagged as an
  onboarding observation, **not** a sealed-slice proof. Real slices later corroborate it
  (raising the band) or, per fresh-observation-wins, contradict it.

```bash
SLUG="$(cat .devrites/ACTIVE 2>/dev/null | tr -d '[:space:]')"
if command -v devrites-engine >/dev/null 2>&1; then
  devrites-engine conventions promote --slug "${SLUG}-adopt" \
    --key test-runner --kind test \
    --statement "tests run with <runner>, <file layout>" \
    --evidence "observed during $rite-adopt onboarding (not yet slice-proven)"
  # …one promote per durable convention the investigation actually observed
  # (test-runner, build-cmd, error-model, http-client, endpoint-placement, …).
else
  echo "(conventions ledger unavailable — devrites-engine missing; skipping seed)"
fi
```

The `-adopt` suffix on the slug marks the provenance as onboarding. Because the ledger is
read at orient as a *prior* and the live code always overrides it
([`.agents/skills/devrites-lib/reference/standards/security.md`](../../devrites-lib/reference/standards/security.md) § Prompt-injection resistance),
an over-eager seed is self-correcting — but seed conservatively anyway; a wrong seed costs a
needless contradiction later.
