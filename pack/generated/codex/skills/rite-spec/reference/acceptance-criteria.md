# Acceptance criteria

Acceptance criteria are the contract the seal checks. Write them so each is
**independently verifiable by evidence**, not opinion.

## A good criterion
- Is observable: a test passes, a command outputs X, a screenshot shows Y.
- Is binary: met or not: no "mostly".
- Names the evidence: "`npm test path/x.test.ts` passes" or "POST /export returns 200
  with a CSV body" or "empty state renders 'No exports yet' at 375px".
- Belongs to a slice: every criterion maps to at least one slice (mapped in `$rite-define`).

## Forms
| Type | Template |
|---|---|
| Functional | "Given <state>, when <action>, then <observable result>." |
| API | "<METHOD> <route> with <input> returns <status> and <shape>." |
| UI | "<screen> shows <element/state> at <viewport>; <interaction> does <result>." |
| UI vs reference | "<screen> matches design reference <Rn> at <viewport>." |
| Non-functional | "<operation> completes under <budget> / handles <volume>." |
| Negative | "<invalid input> is rejected with <message>, no <side effect>." |

## Anti-patterns
- "Works well" / "is fast" / "looks good": unverifiable. Attach a number or an
  observation.
- Criteria with no slice: orphaned scope.
- Criteria only provable by reading the code: require runtime/test evidence instead.

## At seal time
Each criterion gets a checkbox + the evidence that proves it. Unproven critical
criteria force **NO-GO**.
