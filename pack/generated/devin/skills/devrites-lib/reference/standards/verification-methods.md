# Verification methods and coverage

> Applies when: choosing how to check work — Prove, Review, Seal, Vet oracles,
> audit sweeps, or any claim that a scope was verified.

A check earns its place by what it can catch. This file is the selection
discipline: distinct methods, an explicit coverage denominator, honest status
vocabulary, and named fallbacks when a capability is missing. The `gates.md`
acceptance ledger ([`gates.md`](gates.md)) binds runnable oracles to evidence;
this file governs how methods are *chosen* and how coverage is *counted*.

## A method is distinct only when its failure target differs

A method names five things: the **failure** it hunts, the **procedure**, the
**oracle** that separates pass from fail, the **material/scope** inspected, and
the **evidence** it leaves. Count distinct failure targets supported by the
observed checks, not command or tool names. Re-running the same check with
cosmetic flags adds no method. If tests, build, and lint all stop at the same
dead import, those runs supply no evidence of their later checks.

Overlapping detection does not make different checks identical: executed tests
asserting retry exhaustion, a build checking interface compatibility, and lint
checking unsafe shell interpolation can support three distinct targets. Record
each target and observation; neither three command names nor a shared ability
to catch bad imports decides the count.

Different methods may share the authoritative oracle (the spec, the ledger,
the diff). Inventing a second expected answer to look independent is not
independence. Where a requirement supports it, prefer complementary families:

- **functional** — execute the behavior at boundaries and failure paths;
- **structural** — inspect the changed surface, its interfaces, and callers;
- **adversarial** — metamorphic properties, differential baselines, fault
  injection, or a different render/interaction path.

## Enumerate the denominator before claiming coverage

Before work is called covered, write the inventory it must cover: every
unit × check × environment row. Examples of rows a denominator may need:

- Code: changed behaviors, affected callers, edge cases, supported runtimes.
- UI: routes × viewport boundaries × interactive states (loading, empty,
  error, disabled, success, focus) × agreed browsers.
- Fixed-layout documents: every page × required visual check on the final
  render — a thumbnail sheet or extraction pass is not visual inspection.
- Reflowable documents: content units × renderer/viewport/font states;
  reflow invalidates prior visual coverage even when the text is unchanged.
- Data: schema, reconciliation, missing-data treatment, transformation classes.

Two rules keep the denominator honest:

1. **Never shrink it to reach 100%.** An exclusion needs a recorded decision
   with authority; a unit that cannot be inspected stays visible as missing.
   A percentage without the inventory proves nothing.
2. **Sampling is not coverage.** A sample can surface a defect; only a row per
   required unit satisfies a complete-inspection claim. If exhaustive coverage
   exceeds budget, narrow scope with the human — do not quietly redefine done.

## Status and result are separate facts

Record two fields per row, never one:

- **status** — `fresh` (run on this candidate), `reused` (carried forward after
  a recorded applicability check), `failed` (the check ran and could not
  complete — environment, timeout, parse), `missing` (never run), `blocked`
  (cannot run: no tool, no authority).
- **result** — `pass`, `fail`, `unverified`.

A missing check is not a negative result; a failed check is not an unavailable
tool; freshness alone is not a pass. Only applicable `fresh` or `reused`
evidence with a supported `pass` satisfies a required row. Reuse keeps the
original observation and records the new applicability check separately —
see [`candidate-integrity.md`](../candidate-integrity.md#evidence-validity).
When change impact cannot be bounded, recheck the full affected scope.

## Judge evidence before counting a method

Before any check counts, answer five questions:

1. What distinct failure could this method reveal?
2. What material and environment did it actually inspect?
3. What independent observation separates pass from fail?
4. Where is the observation recorded, and which candidate does it bind?
5. What does this method not establish?

Duplicate answers mean one method, not two. A property no available tool can
verify is `unverified` — name the fallback, never manufacture a pass.

## Capability fallbacks

A missing capability gets a named fallback, not a fabricated check:

| Missing | Honest fallback |
| --- | --- |
| Network | Dated local sources with their limits stated; a decision that needs current facts blocks or narrows scope with approval. |
| Question tool | Numbered text options with a marked recommendation and a free-text path; a skipped answer stays `unanswered` ([`elicitation.md`](elicitation.md)). |
| Fresh-context reviewer | `gap` under [`agents.md`](agents.md) result admission; required review waits for an authorized reviewer — self-review is labeled, never relabeled independent. |
| Delegation permission | No dispatch. Prepare a neutral handoff; never bypass the restriction through another CLI.<!-- pack-scan-ignore: defensive rule prohibiting bypass, not instructing it --> |
| Renderer/vision | Mechanical checks run; the visual-coverage rows stay `blocked` until real inspection or a recorded narrower scope. |
| Budget | Stop or reduce scope with approval; failed attempts stay in the record, never deleted to look clean. |

## Exercise the recipient's path

Verification ends at the artifact as its recipient meets it: install the
package, open the archive, run the binary, follow the entry point, navigate
the UI. Installation, host discovery, activation, and actual behavior are four
different facts — each proves only itself. Check relative paths, bundled
resources, links, metadata, licensing, and instructions on the packaged form,
from a clean location where feasible. A check run only against the working
tree proves the working tree.
