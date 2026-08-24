# $rite-clarify anti-patterns

Load this when tempted to skip clarify, assume product choices, or write
`Decision coverage: CLEAR` before every material row has an owner.

Pack-wide rationalizations: see
[standards/anti-patterns.md](../../devrites-lib/reference/standards/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
| --- | --- |
| "We already discussed this in chat." | Chat is not durable decision coverage. Scan topology and record owners in workspace artifacts. |
| "The answer is obvious — assume it." | Product, scope, policy, and irreversible-risk choices are human-owned unless already recorded in an accepted decision. |
| "Most rows are clear — good enough for AFK." | Partial/Missing material rows block CLEAR and block AFK arming. |
| "Skip clarify for this small change." | Clarify is mandatory and adaptive; it may ask no questions, but it still runs the scan. |
| "I'll clarify during build if needed." | Unowned decisions discovered in build are expensive drift. Close them before plan/vet. |

## Red flags in your own output

- `Decision coverage: CLEAR` with Partial/Missing material rows still open.
- Product/policy choices recorded without a human packet or cited prior decision.
- AFK armed or `$rite-define` started before CLEAR is durable.
- Clarify skipped because the user said they were "in a hurry".
