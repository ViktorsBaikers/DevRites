# Using DevRites: intent map

This is an explicit reference, not a session-start autoload. Load it when the user asks which DevRites rite fits.

| User intent | Route | Notes |
|---|---|---|
| New feature / vague product idea | `$rite-spec` (Codex: `$rite-spec`) | Investigate before planning. |
| Written spec still has unknowns / missing decision coverage | `$rite-clarify` | Required before planning; topology scan, zero-question fast path when clear. |
| Existing codebase / resume reality | `$rite-adopt` or `$rite-converge` | Adopt derives intent from code; converge appends remaining slices. |
| Active workspace created under older DevRites rules | `$rite-upgrade` | Explicit-only semantic reconciliation; preserves completed work and history. `devrites-engine migrate` handles structure only. |
| Review plan before code | `$rite-vet` | Every defined plan; depth scales to stakes and emits the implementation-readiness verdict. |
| Small safe fix | `$rite-quick` | Escalates on auth, migration, public API, destructive, ambiguous, or multi-slice work. |
| Prove UI/runtime | `$rite-prove` plus `devrites-browser-proof` | Capture real evidence. |
| Stuck / unfamiliar area | `$rite-zoom-out` | Map structure before changing code. |
| Teach me | `$rite-explain` | Human learning loop. |
| Ship? decide readiness | `$rite-seal` | Seal decides; it does not mutate git. |
| Execute ship after GO | `$rite-ship` | Requires GO/type-GO. |
| Whole lifecycle unattended | `$rite-autocomplete` | Clean baseline + checkpoint mode; stops on hard gates. |
