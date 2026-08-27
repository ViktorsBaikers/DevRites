# Spec: Visual HTML artifacts (DevRites-native)

**plan_id:** `002-visual-html-artifacts`  
**SSOT plan:** `.mstar/plans/002-visual-html-artifacts.md`  
**Status:** Execute — plan LOCKED 2026-08-26; T1 contract authored  
**Primary home:** `.devrites/work/<slug>/visual/`  
**Schema SSOT:** `pack/.claude/skills/devrites-lib/reference/workspace-artifact-schema.md`

## One-line product contract

Agents produce portable single-file HTML visualizations plus a required machine outline companion under workspace `visual/`; humans open via local `open-visual`; agents re-read the outline (outline wins on conflict) — wired into existing Spec/Define/`flows.md` and `/rite-explain` surfaces only, **no new lifecycle phase**, **no Lavish runtime dependency**.

## Locked clarify (authoritative)

1. **Product shape:** Format + on-demand playbooks + thin open/view helper (not a full Lavish clone; not Markdown-only).
2. **Primary hooks:** `/rite-explain` and `flows.md` / Spec–Define diagram path; infrastructure also updates workspace schema + open helper.
3. **Agent dual-read:** Single HTML file **plus** machine outline companion (not HTML-only, not a full Markdown twin page).
4. **HTML home:** `.devrites/work/<slug>/visual/<name>.html` (+ `<name>.outline.md`) beside `flows.md`.
5. **Open helper:** `devrites-engine open-visual` — local resolve + OS browser open + agent tip stdout. No annotation poll, no share host, no network fetch.
6. **Playbooks v1:** Seven ids — `diagram`, `table`, `comparison`, `plan`, `code`, `input`, `slides` — **adapted** to DevRites (no `window.lavish.*` / `data-lavish-*`; `input` collects via native forms → outline/`Answers` or pointer into `questions.md`).

## Goals / non-goals

| Goals | Non-goals |
| --- | --- |
| Dual-read HTML+outline contract agents and humans share | New rite lifecycle phase |
| On-demand playbook references under `devrites-lib` | Lavish-axi / ht-ml.app / poll / share as required deps |
| Local `open-visual` opener with agent-facing path tips | MCP server; annotation whiteboard editor in v1 |
| Soft hooks into existing explain + flows writers | Auto-required HTML for every feature |
| Conditional workspace artifact; never inflate readiness | Changing ownership of project `DESIGN.md` / Impeccable DESIGN schemas |

## Artifact contract

```text
.devrites/work/<slug>/visual/
  <name>.html           # human-viewable portable page
  <name>.outline.md     # machine dual-read (required companion)
  README.md             # optional index of visuals for the workspace
```

### Dual-read rule (SSOT)

- Writers always emit the HTML/outline **pair** for a named visual.
- Agents treat `.outline.md` as the machine source of truth.
- **If HTML and outline disagree, outline wins** for machine understanding until the author regenerates both.
- Humans view HTML in a normal browser; no DevRites injection is required for correct render.

### HTML minimum

- Explicit page background / color-scheme (avoid blank/self-paint).
- Semantic landmarks (`header`, `main`, labeled sections with stable `id`s).
- Figures: prefer hand-authored SVG; Mermaid only when flowchart/sequence/state is clearer **and** Mermaid source is embedded for agent read **and** mirrored in the outline.
- Stable ids on annotatable nodes (future-proof; Lavish not required).
- Prefer self-contained CSS; CDN only when a playbook explicitly requires it (e.g. Mermaid / `@pierre/diffs`) and the outline notes the dependency.

### Outline minimum

Required headings (see `visual-playbooks/outline-template.md`):

1. Title
2. Purpose
3. Playbooks used
4. ID inventory (`id` → meaning)
5. Relationships / decisions / open questions
6. Citations (repo paths when claims touch the tree)
7. Optional `## Answers` when the `input` playbook was used

### Conditional / readiness / candidates

- `visual/` is an **optional** workspace artifact family.
- Paths under `visual/` are **not** candidate paths and must not appear in `touched-files.md` candidate manifests.
- Presence or absence of `visual/` **does not** inflate native readiness gates unless an author explicitly links a visual from an existing required artifact.
- Do not create `visual/` before a writer has a concrete visual need; absence is meaningful.

## Playbooks (v1 ids)

Author location: `pack/.claude/skills/devrites-lib/reference/visual-playbooks/`  
Agents **must** open each matching playbook before writing HTML (progressive disclosure; do not preload all seven).

| ID | Use when |
| --- | --- |
| `diagram` | Relationships, flows, state, architecture |
| `table` | Dense comparable records |
| `comparison` | Options / before-after / tradeoffs |
| `plan` | Product or technical plan before build |
| `code` | Snippets, files, diffs (prefer focused ranges) |
| `input` | Structured choices the human should make visually |
| `slides` | Only when a paced deck/presentation is requested |

**Adaptation rule:** Keep craft guidance; strip Lavish poll/queue/annotation APIs; replace with DevRites dual-read + optional `## Answers` / pointer into workspace `questions.md`.

## Open helper (contract for T4)

```text
devrites-engine open-visual <path-or-name> [--slug <slug>] [--no-open]
```

1. Resolve under active / `DEVRITES_WORKSPACE` `visual/` or absolute path.
2. Require `.html`; warn if sibling `.outline.md` is missing.
3. Unless `--no-open`, launch OS default browser.
4. Print compact agent tip: absolute HTML path, outline path, playbook hint.
5. Engine stays local / no-network — may spawn OS opener only; never fetch remote hosts.

## Skill hooks (existing steps only)

| Surface | Contract |
| --- | --- |
| `workspace-artifact-schema.md` | Documents conditional `visual/` + outline, budgets, non-candidate / non-readiness |
| `/rite-explain` | When a visual earns it → write HTML+outline (workspace `visual/` or explainers run dir with the same pair contract) and offer `open-visual` |
| Spec/Define `flows.md` | Keep Mermaid in `flows.md`; when richer presentation is needed, **also** emit `visual/<flow>.html`+outline and link from `flows.md` |
| Skill routing | Stub / lib pointer so agents load matching playbooks before visualizing |

No new lifecycle phase is introduced.

## Dual-host

Author under `pack/.claude/skills`; regenerate Codex/Claude mirrors via existing host-artifact build. Single schema; no second visual contract.

## Deferred roadmap pointer (B2–B5)

| Batch | Scope | Trigger |
| --- | --- | --- |
| B2 | Dogfood journey visuals using playbooks | After B1 ships + dogfood authors request |
| B3 | Optional lavish-axi adapter (annotate/poll) as **opt-in external** tool | User asks; never default dependency |
| B4 | Additional phase hooks (temper strategy boards, vet coverage boards) | After B1 usage evidence |
| B5 | Engine validation of outline↔HTML id parity | If drift becomes a real failure mode |

## Acceptance (product — T1 contract)

- Spec matches locked clarify (shape, dual-read, home, open helper, 7 playbook ids, no Lavish requirement).
- Schema documents paths, ownership, budgets, non-candidate status, and non-readiness inflation.
- Outline template lists required headings for dual-read companions.
- No new phase language appears as a requirement.

## Interfaces for later tasks

| Task | Consumes from this spec |
| --- | --- |
| T2 | Playbook path + ids + DevRites adaptation rule; outline template colocated |
| T3 | Hook surfaces + dual-read pair requirement |
| T4 | `open-visual` behavior + path home |
| T5 | Dual-host author tree + fixture under same contract |
