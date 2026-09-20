# Windows — the deferral ledger

> Applies when: a change introduces a deferral marker (TODO, FIXME, XXX, HACK,
> a skipped test, a stub body) or `windows.md` waivers are authored or judged.

`windows.md` is the broken-windows ledger. A review finding can describe a new
`TODO`, a skipped test, or a stubbed body; this artifact makes the register
countable instead: every deferral marker the change *introduces* must either
be removed or carry an explicit waiver row, or `check windows` fails. The
target is not zero deferrals — it is zero *silent* deferrals. Pre-existing
markers on the baseline are pre-existing debt and are never attributed to the
current change; only added lines count.

Optional until a marker is introduced: a feature with no new markers needs no
`windows.md`. Once any hit exists, Seal treats an unwaived marker as NO-GO.

## Grammar

```markdown
- <project-relative path> | <marker token> | <reason>
```

- One waiver per row; each row waives exactly one hit at that path carrying
  that marker. Five new TODOs in one file need five rows — a blanket row
  cannot cover them.
- `marker token` is the literal marker the checker reported (`TODO`, `t.Skip(`,
  `NotImplementedError`, `panic("TODO`, …).
- `reason` must be non-empty and should name the follow-up owner or slice; a
  row with an empty reason warns and waives nothing.
- Rows never suppress coverage — they explain it. Reviewers still judge
  whether the deferral itself is acceptable.

## Command

```text
devrites-engine check windows <slug> [--worktree|--staged|--base <ref>]
```

Exit `0` when no marker is introduced or every hit is waived; `3` when any
introduced marker lacks a waiver or the workspace/ledger is unreadable; `2`
on usage errors. The check parses the unified diff (`-U0`) for added lines,
treats untracked files as wholly new, ignores binary files, deletions, and
`.devrites/` paths, and runs on worktree, staged, or base-reference diffs.

## Reviewer posture

A waived marker is disclosed, not approved. The Seal reviewer asks whether
each deferral belongs in this change at all, whether the reason is honest
(owner and follow-up named, not "later"), and whether the diff could simply
remove the marker instead. A waiver that exists only to silence the gate is
an `Important` finding.
