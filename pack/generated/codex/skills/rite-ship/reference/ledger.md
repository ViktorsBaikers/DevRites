# Capability ledger — the living "what the system does now" layer

The ledger is DevRites' cumulative spec store: `.devrites/specs/<capability>/spec.md`, a flat
list of proven `### Requirement:` blocks per capability. Unlike a feature workspace — which is
archived away on close — the ledger **persists** and accumulates, so feature N+1 starts from the
proven contract of every feature before it instead of re-deriving from code.

It lives outside `.devrites/work/`, so close-out's archival never touches it. It is written only
here, on ship, and only from behavior that just passed the seal GO + evidence-freshness gate — so
the ledger records *proven* truth, never merely proposed.

## Fold semantics

A feature's `spec.md` carries **deltas** grouped under capability-tagged H2 sections
(`## ADDED Requirements — capability: <c>`, `MODIFIED`, `REMOVED`) — see
[`spec-grammar.md`](../../devrites-lib/reference/standards/spec-grammar.md) § Delta form.
`devrites-engine ledger sync` groups them by capability and folds each into its ledger spec:

- **ADDED** → append the block (upsert: replaces in place on a re-sync, so sync is idempotent)
- **MODIFIED** → replace the same-named block with the full new version
- **REMOVED** → delete the block (an emptied capability spec is dropped)

Matching is by **requirement header identity**. A flat feature spec (no delta H2 sections) folds
as all-ADDED into the capability named by the feature slug.

## Sync workflow (rite-ship step 2b)

Runs **before the commit** so the ledger change ships in the same commit as the code that proves it.

1. Preview the fold — deterministic, zero-token:
   ```bash
   devrites-engine ledger diff .devrites/work/<slug>
   ```
2. **Opt-in and confirmed** (mirrors the design-memory rollup): present the option set — default
   is to sync, since a shipped feature's proven behavior belongs in the ledger; the escape hatch
   is skip (the feature ships without updating the living spec — a deliberate call, e.g. an
   internal-only change with no capability contract).
3. On yes:
   ```bash
   devrites-engine ledger sync .devrites/work/<slug>
   ```
   Then append each written `.devrites/specs/<capability>/spec.md` to `touched-files.md` so it is
   staged and committed at the git ladder (step 4). Record the synced capabilities in `ship.md`.
   **The ledger must be git-tracked.** Most of `.devrites/` is runtime state a project gitignores;
   `specs/` is the exception (durable shared truth). If a `git status` / `git check-ignore` shows
   the ledger path ignored, the commit would silently drop it — carve it out first:
   `.devrites/*` + `!.devrites/specs/`. Surface this rather than shipping a ledger that never lands.
4. Skip when the feature declares no requirements (a pure refactor / chore with no spec deltas).

## Reading the ledger (earlier phases)

`$rite-spec` and `$rite-adopt` consult the ledger before writing a new spec:
```bash
devrites-engine ledger list            # capabilities + requirement counts
devrites-engine ledger show <capability>   # the proven contract to write deltas against
```
