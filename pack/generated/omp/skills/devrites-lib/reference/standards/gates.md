# Gates — the machine-checked acceptance ledger

> Applies when: authoring or checking the machine-verified acceptance ledger (gates.md).

`gates.md` is the acceptance ledger: every required outcome becomes one
independently checkable gate, and the engine — not reviewer prose — decides
whether each gate ran and produced its declared signal. It is the structural
answer to partial compliance: a feature cannot report "done" while a required
outcome is unchecked, stale, or abandoned.

Written at Vet alongside `test-plan.md`, executed at Prove, required all-met at
Seal. Like every required artifact it is phase-bound: vet/build/converge and
every later phase requires its presence; proof phases require every gate met.

## Grammar

```markdown
- [ ] AC-001: <observable outcome>
  CHECK: <exact command from test-plan.md Build-entry preflight>
  EXPECT: <substring | /regex/flags>
  CWD: <repository-relative path; omit for repo root>
  EVIDENCE: pending

- [x] AC-002: <human-judged outcome>
  EVIDENCE: <human attestation, one line>

ABANDON: AC-003 <why the outcome can never be met>
```

- One `- [ ]`/`- [x]` row per gate; `ID:` ends at the first colon, max 64 chars.
- `CHECK` and `EXPECT` are a pair: both present (runnable gate) or both absent
  (manual gate). A partial pair is a parse error.
- Attributes are indented beneath their gate. Fenced code examples never parse.
- `EXPECT` is a substring, or `/pattern/flags` with `i`, `m`, `s` flags.
- `CWD` is repository-relative; absolute paths and `..` segments are rejected.
- `ABANDON:` starts at column 1, names a known gate, and requires a reason.
  An abandoned gate blocks completion and demands a handoff — it can never
  read as a pass.
- An empty ledger (no gates) is malformed.

## Runnable gates and the approval boundary

A runnable gate executes only when its exact `(CHECK, CWD)` pair matches a row
of `test-plan.md`'s `## Build-entry preflight` table (`Command` + `Cwd`
columns). Approval is exact-text equality after whitespace/cwd normalization —
a differently spelled command or different directory is a different oracle and
needs its own vetted row. This is the DevRites approval boundary: the vetted
test plan is the sole list of commands the engine may run, so gates can never
execute unreviewed code.

Execution contract: the command runs through the platform shell in the named
directory with a bounded timeout; pass requires exit 0 **and** output matching
EXPECT. Output is captured but never persisted — only its digest.

After Vet READY the runnable set is fixed: converting a runnable gate to manual,
loosening EXPECT, changing CHECK or CWD, or removing a row is Plan repair plus
re-Vet, never a Prove or Seal edit. A runnable gate that cannot execute stays unmet
with `unavailable: <reason>`. Prove attests only gates that were manual at Vet,
checked by diffing `gates.md` against the Vet record. **Failing case:** AC-003's
runnable gate fails at Prove, the root deletes CHECK/EXPECT and attests "verified
manually", and `gates status` reads `all-met`.

## Evidence binding

A successful run writes `EVIDENCE: automatic-evidence=v1; def=<sha256>; …`
where `def` binds the exact CHECK/EXPECT/CWD the run executed. Editing any of
the three renders prior evidence `stale-unmet` — a stale pass cannot hide
behind an edited oracle. Prose evidence on a runnable gate is also stale:
only automatic evidence bound to the current definition counts.

Manual gates are met when checked with non-empty human evidence (anything
except `pending`); honesty of that evidence stays with the Prove/Seal
reviewers. The engine cannot see who attests, so the note must cite the human's
own words or source (a `q-…` id the human resolved, or their message in this
session) and name what was observed and where. An agent's own inspection, or a
question resolved by AFK or autocomplete auto-pick, is not human evidence: an AFK
run leaves the gate unmet behind an open `gate: validating` question and never
self-attests. Placeholder or tautological notes (`n/a`, `tbd`, `-`, `ok`, `done`,
`yes`, the restated gate title) are unmet at Prove and Seal. **Failing case:**
`gates attest <slug> AC-004 "done"` passes the engine and Seal reaches GO.

## Commands

```text
devrites-engine gates scaffold <slug>     create gates.md from spec.md AC ids
devrites-engine gates status <slug>       reduce without executing; all-met/handoff/not-met/malformed
devrites-engine gates run <slug>          execute unmet runnable gates, write bound evidence
devrites-engine gates reverify <slug>     re-execute every runnable gate; demote stale passes
devrites-engine gates lint <slug>         audit oracle quality without executing
devrites-engine gates attest <slug> <id> <note>   record human evidence on a manual gate
devrites-engine gates abandon <slug> <id> <why>   record a terminal ABANDON handoff
```

Run/reverify hold the feature lock, re-read the ledger before writeback, and
discard a result whose gate definition moved mid-run. Reverify is the
suspicion-killer: when any doubt exists about prior passes, rerun it rather
than trusting recorded evidence.

## Oracle quality (`gates lint`)

Lint audits signs an oracle cannot fail honestly — fixed-output CHECKs
(`echo done`), EXPECT values that also appear in failure output (`pass`,
`ok`, `0`), path-shaped regexes, activity titles instead of outcomes
("Improve coverage"), unmeasured numbers in manual titles, unvetted CHECKs,
and mostly-manual ledgers. Warnings prompt sharpening; an unvetted CHECK is
an error because the runner can never execute it.

## Authoring guidance

- One gate per observable outcome; gate ids should reuse canonical `AC-###`
  ids so `check readiness` coverage and the ledger name the same set.
- Write oracles a stranger could judge: EXPECT a line only success can print,
  never a word failure output also contains.
- Source `CHECK` text from the repository's own wiring:
  `devrites-engine detect commands` resolves test/lint/vet/build commands from
  the Makefile, `package.json` scripts, and language manifests without
  executing them. Cite the resolved command verbatim in the preflight row;
  an `unresolved` slot means Vet must name the concrete command, never guess.
- Prefer runnable gates; reserve manual gates for genuinely judgment-bound
  outcomes and make the title's claim inspectable.
- Pick the strongest oracle the outcome admits: differential (output compared
  against a recorded baseline or the pre-change behavior) over property
  (an invariant the output must always satisfy) over example (one concrete
  input/output case). An example-shaped oracle on a property-shaped outcome is
  a weak gate — reviewers flag it even when it passes.
- When an outcome becomes impossible, `gates abandon` it and route the
  handoff — never delete the row to make the ledger pass.
- The gate set is the coverage denominator: it never shrinks to reach
  `all-met`. Two gates sharing one normalized CHECK+EXPECT+CWD are one oracle
  counted twice — `gates lint` flags them as `duplicate-oracle`. Distinct
  methods and denominator rules live in
  [`verification-methods.md`](verification-methods.md).
