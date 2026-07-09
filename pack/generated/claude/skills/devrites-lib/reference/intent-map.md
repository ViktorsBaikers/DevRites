# Using DevRites: intent map

This is an explicit reference, not a session-start autoload. Load it when the user asks which DevRites rite fits.

| User intent | Route | Notes |
|---|---|---|
| New feature / vague product idea | `/rite-spec` (Codex: `$rite-spec`) | Investigate first; do not plan from vibes. |
| Existing codebase / resume reality | `/rite-adopt` or `/rite-converge` | Adopt derives intent from code; converge appends remaining slices. |
| Review plan before code | `/rite-vet` | Every defined plan; depth scales to stakes. |
| Small safe fix | `/rite-quick` | Escalates on auth, migration, public API, destructive, ambiguous, or multi-slice work. |
| Prove UI/runtime | `/rite-prove` plus `devrites-browser-proof` | Capture real evidence. |
| Stuck / unfamiliar area | `/rite-zoom-out` | Map structure before changing code. |
| Teach me | `/rite-explain` | Human learning loop. |
| Ship? decide readiness | `/rite-seal` | Seal decides; it does not mutate git. |
| Execute ship after GO | `/rite-ship` | Requires GO/type-GO. |
| Whole lifecycle unattended | `/rite-autocomplete` | Clean baseline + checkpoint mode; stops on hard gates. |
