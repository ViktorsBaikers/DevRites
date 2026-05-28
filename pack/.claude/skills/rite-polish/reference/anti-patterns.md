# rite-polish — anti-patterns

Load this when standing a non-trivial polish decision, or when tempted to
skip normalize, polish without Chesterton's Fence, or cite clean lint/build
as proof of quality.

Pack-wide rationalizations + red flags (incl. lint-pass-as-quality): see
[rules/anti-patterns.md](../../../rules/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "Tests pass; the feature is done." | Tests don't measure ship-quality, design drift, anti-slop, or backend polish. |
| "UI looks fine to me." | Must align to the design system + meet CWV/WCAG 2.2 — not match a personal taste. |
| "Code is simple enough; no need to audit." | Measure first. If there's no hotspot, that's fine — but record "no hotspots found", don't skip silently. |
| "It's a small UI change; polish without normalize is fine." | **NO** — decoration on drift is banned. Phase 3 runs before Phase 4, always. |
| "Backend looks OK; skip Phase 2." | If the diff touched BE, Phase 2 runs — error responses, logging hygiene, queries, anti-slop. |

## Red Flags

- About to polish UI (Phase 4) without running normalize (Phase 3) first.
- No browser evidence saved for a UI polish.
- Code polish ran without naming a single technique (guard clauses, Extract Method, ...).
- Backend was touched but `polish-report.md` shows no Phase 2 section.
- A "simplification" that changes observable behavior — that's not behavior-preserving.
- Reading a Chesterton's Fence as "looks dead" and deleting without explaining what it guards.
