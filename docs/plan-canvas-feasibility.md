# Plan/spec/design canvas feasibility

Status: feasible as a review aid, not as an authorization surface.

## Decision

DevRites may add a local review canvas for `spec.md`, `plan.md`, `tasks.md`, and `design-brief.md`, but it must map every verdict back into existing `.devrites/` artifacts and gates.

## Constraints

- DevRites remains installed through `npx devrites ...` only.
- Canvas output is project-local review evidence, not a Claude/Codex plugin package.
- Canvas approval must never bypass `/rite-seal`, `/rite-ship`, typed `GO`, or engine gates.
- Canvas annotations are untrusted user/reviewer input until summarized into `questions.md`, `decisions.md`, `drift.md`, or `review.md`.

## Minimal future implementation

1. `devrites-engine canvas export [slug]` renders a static HTML review artifact from current workspace Markdown.
2. Human annotations are saved as JSON under `.devrites/work/<slug>/canvas/`.
3. `devrites-engine canvas import [slug]` summarizes annotations into normal DevRites artifacts.
4. `/rite-status` and `snapshot` report pending imported/unimported annotations.

## Supported verdict mapping

| Canvas verdict | DevRites effect |
| --- | --- |
| approve spec | Mark spec review evidence only; next command still comes from engine phase. |
| request changes | Add/update `questions.md` or `drift.md`; do not advance phase. |
| approve plan | Add plan-review evidence; `/rite-vet` still decides readiness. |
| reject design | Add `design-brief.md` comments and route back to `/rite-spec`/`/rite-define`. |

## Non-goals

- No browser dashboard before `devrites.workspace.v1` and doctor readiness are stable.
- No remote service.
- No marketplace/plugin delivery.
