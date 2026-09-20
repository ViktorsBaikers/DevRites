# Glossary

> Applies when: any DevRites term is unclear — resolve it here once instead of
> inferring from context. Alphabetical. The owning file holds the full contract.

| Term | Meaning |
| --- | --- |
| AFK / HITL | Unattended vs human-in-the-loop execution modes. [`afk-hitl.md`](afk-hitl.md) owns pause/resume rules. |
| Agent | Fresh-context `devrites-*` role spawned by the host; never the root chat. Roster: [`agents.md`](agents.md). |
| Allowlist | The exact project-relative path set a wright may write. Checked mechanically by `devrites-engine check diff-scope`. |
| Applicability map | `spec.md` table routing risk families (topology, data, integration, security, delivery) to their owning standard. A routed standard is mandatory, not optional. |
| Binding | A recorded hash tying evidence or review to one exact candidate; stale binding = unproven. |
| Bundle (context) | One deduplicated read-set file emitted by `devrites-engine context` into `ctx/`; a dispatch target reads it instead of selecting files. |
| Candidate | The content-bound project state (digest + file set) reviews and proof run against. [`candidate-integrity.md`](../candidate-integrity.md). |
| Consumptive action | An action whose failed attempt is not safely retryable (spends quota, mutates external state, or deletes its own failure evidence). [`one-shot-actions.md`](one-shot-actions.md). |
| Controlling caller | The rite that invoked an earlier rite inline for repair; it keeps authority and resumes after the nested stop. |
| Cursor | The `state.md` key/value table (phase, status, next_action, …) that is the authoritative lifecycle position. |
| Decision coverage | Clarify's ledger that every material decision has current evidence and an owner; `CLEAR` unblocks later phases. |
| Detected commands | `devrites-engine detect commands`: resolves the repo's own test/lint/vet/build commands (Makefile → package.json → manifests, lockfile-derived package manager) without executing them; cite resolved text verbatim, `unresolved` means name it in the plan. |
| Diff-scope gate | `devrites-engine check diff-scope`: changed paths ⊆ allowlist, run before reviewer dispatch. |
| Drift baseline | `readiness-inputs.json`: per-input SHA-256 snapshot recorded by `check drift --record` after Vet's binding; `check drift` attributes later input changes to the exact artifact. Advisory, never a gate. |
| Dup ignore ledger | `.devrites/dup-ignore`: one `check dup` cluster hash per line with a `#` reason; content-derived hashes resurface on structural edits. [`duplicate-code.md`](duplicate-code.md). |
| Evidence freshness | The rule that proof must postdate the candidate it claims to cover; [`candidate-integrity.md`](../candidate-integrity.md) § evidence validity. |
| Finding shape (C2) | Canonical `Outcome / Finding / Basis` result rows every reviewer must return; [`agents.md`](agents.md) § Result admission. |
| Gap | Result state naming missing/stale/unreadable input or a skipped required check; a required gap blocks, it is never `no-findings`. |
| Gates ledger | `gates.md`: machine-checked acceptance rows (scaffold/status/run/reverify/lint/attest/abandon). [`gates.md`](gates.md). |
| Loads manifest | The `<!-- loads: {...} -->` block in a SKILL.md declaring its exact read-set (`always`, `triggers`, `workspace`) for `devrites-engine context`. |
| Metrics ledger | Append-only `metrics.jsonl` in a feature workspace recording dispatches, returns, bundle bytes, repair rounds. |
| Nested phase boundary | A `STOP` inside a caller-invoked repair rite; it returns control to the caller, not the human. |
| Read next | The per-phase bounded read order for workspace artifacts; [`workspace-artifact-schema.md`](../workspace-artifact-schema.md) § Read next by phase. |
| Regression baseline | `regression-baseline.json`: the durable high-water mark of workspace progress facts; `check regression` compares against it, `--update` ratchets it at a checkpoint. |
| Seal | Final GO/NO-GO decision phase; a GO never authorizes irreversible mutation by itself. |
| Slice | One vertical, independently provable unit of the plan (`SLICE-###` in `tasks.md`). |
| Spec Drift Guard | The only legal path to change a sealed spec/plan: batch-sweep violations, record drift, repair through the owning rite. `rite-build/reference/spec-drift-guard.md`. |
| Standards | The rule files in this directory; `core.md` is always-on, the rest load on trigger (`README.md` table). |
| Transfer commit | The isolated wright's local unpushed `WIP(<slug>):` transport commit; never the control-branch checkpoint. |
| Trigger | A named conditional read-set in a `loads:` manifest (e.g. `tdd`, `applicability`, `ui`); engine fails closed on unknown names. |
| Windows ledger | `windows.md`: one waiver row per deferral marker the change introduces; `check windows` fails on unwaived hits. [`windows.md`](windows.md). |
| Workflow Artifact | A Vet-ready, root-authored executable file under `.devrites/work/<slug>/`; [`workflow-artifacts.md`](workflow-artifacts.md). |
| Wright | `devrites-slice-wright`: the sole product source/test writer under a path-bounded contract. |
