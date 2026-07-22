# Issue tracker: local Markdown

Issues and PRDs for this repo live as markdown files in `.scratch/`. For the end-to-end local issue → rites flow, see [issue-driven-rites.md](issue-driven-rites.md).

## Conventions

- One feature per directory: `.scratch/<feature-slug>/`
- The PRD is `.scratch/<feature-slug>/PRD.md`
- Implementation issues are `.scratch/<feature-slug>/issues/<NN>-<slug>.md`, numbered from `01`
- Triage state is recorded as a `Status:` line near the top of each issue file (see `triage-labels.md` for the role strings)
- Comments and conversation history append to the bottom of the file under a `## Comments` heading

## When a skill says "publish to the issue tracker"

Create a new file under `.scratch/<feature-slug>/` (creating the directory if needed).

## When a skill says "fetch the relevant ticket"

Read the file at the referenced path. The user will normally pass the path or the issue number directly.

## Investigation maps

For huge, foggy efforts, `/rite-pressure-test` may create an **investigation map** instead of pretending one session can settle the whole thing. No separate map command exists.

- **Map**: `.scratch/<effort>/map.md` with `Destination`, `Decisions so far`, `Not yet specified`, and `Out of scope`.
- **Child ticket**: `.scratch/<effort>/issues/NN-<slug>.md`, numbered from `01`, with one research/prototype/grilling/task question in the body. A `Type:` line records the ticket type; a `Status:` line records `open`/`claimed`/`resolved`.
- **Frontier**: one ticket per session. Pick the first open, unblocked, unclaimed ticket by number.
- **Blocking**: `Blocked by: NN, NN` near the top. A ticket is unblocked when every listed ticket is `resolved`.
- **Resolve**: append `## Answer`, set `Status: resolved`, then add a one-line context pointer to `map.md` under `Decisions so far`.
