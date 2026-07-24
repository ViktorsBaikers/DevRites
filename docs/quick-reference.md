# DevRites quick reference

Install DevRites with `npx devrites ...`. The installer generates project-local
Claude Code and Codex artifacts; plugin stores do not distribute them.

## Lifecycle

<!-- authority:lifecycle:start -->
`FRAME → SPEC → CLARIFY → TEMPER → DEFINE → PLAN → VET → BUILD → CONVERGE → PROVE → POLISH → REVIEW → SEAL → SHIP → DONE`
<!-- authority:lifecycle:end -->

Seal makes the release decision, Ship mutates git, Build handles one slice per
run, and Autocomplete is opt-in.

## Standing checklists

- Definition of Done: `pack/.claude/skills/devrites-lib/reference/standards/definition-of-done.md`
- Review: `pack/.claude/skills/devrites-lib/reference/standards/review-checklist.md`
- Security: `pack/.claude/skills/devrites-lib/reference/standards/security-checklist.md`
- Test proof: `pack/.claude/skills/devrites-lib/reference/standards/test-proof-checklist.md`
- Browser proof: `pack/.claude/skills/devrites-lib/reference/standards/browser-proof-checklist.md`
- Ship: `pack/.claude/skills/devrites-lib/reference/standards/release/ship-checklist.md`

## Host command forms

Claude: `/rite <verb>` or `/rite-<verb>`. Codex: `$rite <verb>` or `$rite-<verb>`.
