# `ship.md` template

Write before close-out; it moves to `.devrites/archive/<slug>/ship.md`.

```markdown
# Ship: <slug>

- Shipped at: <iso>
- Verdict: GO (seal.md)
- Branch: <branch>
- Commit(s): <sha> [<sha> …]
- Tag/PR: <value|none>

## What shipped
<project-vocabulary paragraph>

## Acceptance
- Proven: <n/total> (evidence.md/seal.md)
- Outstanding: <rows below | none>

## Evidence
- seal.md — verdict/reconciliation
- evidence.md — acceptance; browser-evidence.md if UI
- review.md — review findings

## Follow-up reconciliation
Checked: strategy.md (if present), decisions.md, review.md, seal.md.

| Source | Item | Disposition | Destination / rationale |
|---|---|---|---|
| <file + section> | <item> | tracked | <durable path/ID> |
| <file + section> | <item> | no-action | <decision or explicit human no-action approval + reason/revisit> |

<If empty: No residual items found.>
```

Keep pointers short. Acceptance MUST match `seal.md`. Each source residual appears
once; a missing target, unsupported `no-action`, vague disposition, or conflict blocks
close-out.
