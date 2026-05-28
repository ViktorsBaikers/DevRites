# rite-review — anti-patterns

Load this when standing a non-trivial review decision, when tempted to
demote severity to hide a finding, or when crossing out of feature scope.

Pack-wide rationalizations + red flags: see
[rules/anti-patterns.md](../../../rules/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "Polish already cleaned this up; nothing to flag." | Polish finishes; review judges. If review still spots a real issue, flag it — don't re-run polish here. |
| "Tests pass, so they're adequate." | Adequacy = do they prove acceptance, including failure / boundary / empty paths? A passing test is not automatically a good test. |
| "Security audit isn't needed for this feature." | If the feature touches user input, auth, data storage, external integrations, secrets, or permissions — yes it is. |
| "I'll just fix everything I find in scope." | Some fixes belong in `/rite-plan` as drift, not as smuggled redesigns inside review. |
| "Performance is fine without measuring." | No claim without a number or a specified measurement. "Feels fast" is not a finding. |

## Severity gaming

| Excuse | Rebuttal |
|---|---|
| "Demoting a Critical to Important keeps the review clean." | Severity reflects the actual risk to ship — not the review you'd like. |
| "I won't surface this Critical; user can find it later." | Hidden Criticals are the worst-case play. The seal blocks anyway, and now you've burned trust. |

## Red Flags

- A finding without `file:line`.
- A finding without a severity label (Critical / Important / Suggestion / Nit / FYI).
- A Critical recorded but silently fixed instead of surfaced to the seal.
- A fix applied without re-verification afterwards.
- Review crossing into out-of-scope files because they "looked wrong".
