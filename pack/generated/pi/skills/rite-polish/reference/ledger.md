# Capability ledger

`.devrites/specs/<capability>/spec.md` is the canonical ledger of proven capability
`### Requirement:` blocks across feature archival.
Host filesystem I/O; no engine command.

## Fold semantics

A feature spec may group deltas under capability-tagged H2s (`## ADDED
Requirements — capability: <c>`, `MODIFIED`, or `REMOVED`); see
[`spec-grammar.md`](../../devrites-lib/reference/standards/spec-grammar.md).

| Delta | Rule |
|---|---|
| ADDED | Append an absent block. Identical existing content is a no-op; the same header with different content conflicts. |
| MODIFIED | Replace the named block in full. Equal content is a no-op; an absent target conflicts. |
| REMOVED | Delete the named block. An absent target is a no-op; preview deletion if the capability becomes empty. |

Match exact `### Requirement:` headers. Rename = REMOVED + ADDED. Flat specs
seed the slug capability as all-ADDED.

For MODIFIED, compare the complete current requirement block to its full replacement.
It MUST preserve
every existing scenario and normative or source-grounded claim unless an exact accepted `DEC-###`
explicitly names its replacement or removal and rationale. For MODIFIED or REMOVED,
reconciliation MUST account for load-bearing adjacent notes,
examples, and comments whose meaning depends on changed text. Ambiguous ownership blocks
reconciliation: preserve and show the content for an explicit decision; MUST NOT silently
relocate or drop it. An unexplained omission is a blocking conflict,
not cleanup or simplification.

## Native preview/write workflow

Run during Polish before Review so ledger edits enter the proved candidate.

1. **Read.** Read the feature spec and only affected ledger specs. Reject a
   capability escaping `.devrites/specs`, symlink/special file, duplicate header,
   or malformed delta.
2. **Preview, do not write.** Show exact target paths; every delta with before/after
   header; full requirement order; and changed files. Apply the table to current
   bytes and the preservation rule to MODIFIED and REMOVED. Conflicts stop.
3. **Check feature grammar.** Apply `spec-grammar.md`'s Native grammar re-read
   checklist to the saved feature spec. Any miss stops. Step 2's native comparison owns delta reconciliation; an
   already-applied empty preview remains a no-op.
4. **Confirm.** Name exact paths and deltas. A later Seal or Ship approval does
   not confirm this write. Skip only when no capability contract exists.
5. **Write safely.** Use `git check-ignore` and `git status` to prove targets will
   be tracked; surface a needed ignore carve-out. Immediately re-read and refuse
   bytes changed since preview. Touch only `.devrites/specs/`, preserve unrelated
   bytes/order, and apply only previewed replacements or empty-file deletion.
   **Failing case:** Polish rewrites feature `spec.md` acceptance criteria "to match
   code." Capability fold is the non-trigger; feature `spec.md` stays sealed
   (`core.md` rule 4).
6. **Verify.** Re-read targets; require unrelated blocks byte-identical and the
   same preview empty. Apply the same Native grammar re-read checklist to each
   resulting capability spec.
   Add changed paths to the authoritative candidate manifest, run affected real
   re-proof, and refresh evidence/browser digest bindings before Review.

Skip all six steps for a refactor/chore with no requirements.

## Reading in Spec and Adopt

List ledger specs with the host filesystem and read only touched capabilities.
Classify against current file content, never memory.
