# rite-build output

This is the normative output contract for `$rite-build`. `../SKILL.md` points
here so the root skill can stay focused without weakening response shape.

Run `devrites-engine progress` first. It prints the deterministic header, slice
meter, and flow ribbon. Put the compact fact lines below it using the shared
completion reply contract
([`devrites-lib/reference/reply-contract.md`](../../devrites-lib/reference/reply-contract.md)).

**Slice built; slices remain:**
```text
Done: built slice <n> — <name>.
Changed: <files touched>; state.md, touched-files.md, evidence.md
Evidence: acceptance met; tests <cmd -> pass>; browser <summary | n/a>; drift <none|handled>
Open: <none | blockers | questions>
Next: $rite-build
Record: .devrites/work/<slug>/evidence.md
↻ Hygiene: /clear between slices; $rite-handoff if away > a few hours
```

**Slice built; all slices built:**
```text
Done: built slice <n> — <name>; all slices built.
Changed: <files touched>; state.md, touched-files.md, evidence.md
Evidence: acceptance met; tests <cmd -> pass>; browser <summary | n/a>; drift <none|handled>
Open: <none | blockers | questions>
Next: $rite-prove
Record: .devrites/work/<slug>/evidence.md
↻ Hygiene: /clear before $rite-prove; $rite-handoff if away > a few hours
```

For awaiting-human or stopped states, use the typed templates from the reply contract.
Keep fact lines terse. The meter and ribbon carry progress; don't duplicate them in
prose.

For a **forged slice**, the `Built` line names the competition and points at the record, e.g.
`Done: built slice 3 — csv-streaming (forged: 3 candidates, winner B).` and
`Record: .devrites/work/<slug>/forge-report.md`.

**DO NOT continue to the next slice automatically**: even at `✅ ALL BUILT`, `$rite-prove` is the user's call.
