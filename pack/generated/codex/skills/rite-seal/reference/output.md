# rite-seal output

This is the normative output contract for `$rite-seal`. `../SKILL.md` points
here so the root skill can stay focused without weakening response shape.

Run `devrites-engine progress` first, then use the GO or NO-GO typed template
from the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../../devrites-lib/reference/reply-contract.md)).

GO shape:
```text
GO: feature cleared to ship
Follow-ups: <none | non-blocking count>
Next: $rite-ship
Record: .devrites/work/<slug>/seal.md
↻ Hygiene: /clear before $rite-ship
```

NO-GO shape:
```text
NO-GO: <short verdict>
Blockers: <count + top 1-3 blockers>
Fix: <single next command>
Record: .devrites/work/<slug>/seal.md
↻ Hygiene: /compact (seal blockers) if fixing now; /clear if stopping
```

Do not imply anything shipped. `$rite-seal` decides only; `$rite-ship` executes.
