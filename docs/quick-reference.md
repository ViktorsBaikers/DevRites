# DevRites quick reference

Install DevRites with `npx devrites ...`. The installer generates project-local
Claude Code and Codex artifacts; plugin stores do not distribute them.

## Lifecycle

<!-- authority:lifecycle:start -->
`FRAME → SPEC → CLARIFY → TEMPER → DEFINE → PLAN → VET → BUILD → CONVERGE → PROVE → POLISH → REVIEW → SEAL → SHIP → DONE`
<!-- authority:lifecycle:end -->

Frame is the optional non-gating preflight lens represented in the machine state
vocabulary; Spec begins the required feature-definition path. Seal makes the
release decision and Ship mutates git. A direct Build handles one slice per run
in HITL; an explicit `.devrites/AFK` sentinel may chain bounded low-risk slices,
while Autocomplete owns full-lifecycle repetition. Build maintains the strict
candidate manifest. Prove, Review, and Seal bind to its content digest; Polish
completes durable rollups before Review; Ship is candidate-read-only.

Conditional compatibility: `/rite-upgrade [slug]` audits an older active
workspace against current contracts. It repairs only a cited defect through
Clarify, Plan repair, Converge, Vet, Prove, Polish, Review, or Seal; it is not a
phase or migration and never synthesizes old proof.

Candidate check: `devrites-engine check candidate <slug>` prints
`candidate-sha256: <64 lowercase hex>` and `candidate-files: <row count>` on a
pass. See [`candidate-integrity.md`](candidate-integrity.md).

## Standing checklists

- Definition of Done: `pack/.claude/skills/devrites-lib/reference/standards/definition-of-done.md`
- Review: `pack/.claude/skills/devrites-lib/reference/standards/review-checklist.md`
- Security: `pack/.claude/skills/devrites-lib/reference/standards/security-checklist.md`
- Test proof: `pack/.claude/skills/devrites-lib/reference/standards/test-proof-checklist.md`
- Browser proof: `pack/.claude/skills/devrites-lib/reference/standards/browser-proof-checklist.md`
- Ship: `pack/.claude/skills/devrites-lib/reference/standards/release/ship-checklist.md`

## Host command forms

Claude: `/rite <verb>` or `/rite-<verb>`. Codex: `$rite <verb>` or `$rite-<verb>`.
