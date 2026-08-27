# Visual HTML artifacts (lavish-inspired, DevRites-native)

> **For agentic workers:** REQUIRED SUB-SKILL: Use `mstar-sdd` (recommended) or inline execution. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Let agents produce rich, reviewable HTML visualizations (plans, diagrams, tables, comparisons, code, inputs, slides) that humans can open correctly and agents can re-read without ambiguity — integrated into existing DevRites steps, **no new lifecycle phase**.

**Architecture:** Portable single-file HTML + companion machine outline under `.devrites/work/<slug>/visual/`; on-demand playbook references loaded by skills; thin local `devrites-engine open-visual` opener (no network, no annotation poll). Improve on lavish-axi by making dual-read first-class (outline), stripping Lavish-only APIs, and binding playbooks into existing Spec/Define/Explain surfaces.

**Tech Stack:** Pack skills (`pack/.claude/skills` → dual-host regen), `devrites-engine` CLI, workspace artifact schema, Mermaid/SVG in HTML, progressive skill references.

**Execution:** mstar-sdd

**plan_id:** `002-visual-html-artifacts`  
**workflow_id:** `wf-visual-html-artifacts`  
**primary_spec:** `.mstar/specs/visual-html-artifacts.md` (to be authored in Prepare)  
**Phase:** Done — T1–T5 Approved; T6 QC Approve + QA Pass (`reports/T6-qa.md`)
**Task category:** `docs` (primary) + `logic` (engine helper) + `visual` (artifact craft)

---

## Intent gate

| Item | Statement |
| ------ | ----------- |
| Real goal | Humans and agents understand spatial/structural ideas (flows, plans, diffs, tradeoffs) from the same durable visual artifact without screenshot ping-pong or prose-only loss |
| Success | Agents load playbooks on demand; write HTML+outline into workspace `visual/`; `open-visual` opens the page and prints agent-facing paths; `/rite-explain` and `flows.md` authors use the format; dual-host packs stay in parity; no new rite phase |
| Non-goals | New lifecycle step; lavish-axi dependency/poll/share/ht-ml.app; DESIGN.md ownership change; auto-required HTML for every feature; MCP server; annotation whiteboard editor in v1 |

---

## Locked clarify (complete — 2026-08-26)

1. **Product shape:** Format + playbooks + thin open/view helper (not a full Lavish clone; not Markdown-only).
2. **Primary hooks (empty answer → PM locked recommended):** `/rite-explain` + `flows.md` / Spec–Define diagram path. Infrastructure also updates workspace schema + open helper.
3. **Agent dual-read:** Single HTML file **plus** machine outline block/file (not HTML-only, not Markdown twin).
4. **HTML home:** `.devrites/work/<slug>/visual/<name>.html` (+ outline) beside `flows.md`.
5. **Open helper:** `devrites-engine open-visual` (local resolve + OS browser open + agent tip stdout). No annotation poll, no share host.
6. **Playbooks v1:** Full lavish-like 7 — `diagram`, `table`, `comparison`, `plan`, `code`, `input`, `slides` — **adapted** to DevRites (no `window.lavish.*` / `data-lavish-*`; input collects via native forms → outline/`answers` companion readable by agents).

---

## Global Constraints

- **No new phase.** Wire into existing writers only.
- **Portable HTML:** Saved file must render identically in a normal browser (no DevRites injection required). Prefer self-contained CSS; CDN only when a playbook explicitly requires it (e.g. Mermaid / `@pierre/diffs`) and outline notes the dependency.
- **Outline is SSOT for agents:** If HTML and outline disagree, outline wins for machine understanding until the author regenerates both.
- **Engine stays local/no-network:** `open-visual` may spawn the OS opener only; never fetch remote hosts.
- **Dual-host:** Author in `pack/.claude/skills`; regenerate Codex/Claude mirrors via existing host-artifact build; no second schema.
- **Do not collide with** project `DESIGN.md` (design memory) or Impeccable/Stitch DESIGN schemas.
- **Conditional artifact:** `visual/` is optional; never inflate readiness/candidate gates unless a playbook author explicitly links a visual path from an existing artifact.
- **Strip Lavish coupling:** Playbooks keep craft guidance; drop Lavish poll/queue/annotation APIs; replace with DevRites dual-read + optional `answers.md` / pointer into `questions.md`.

---

## Target design

### Artifact contract (`visual/`)

```text
.devrites/work/<slug>/visual/
  <name>.html           # human-viewable page
  <name>.outline.md     # machine dual-read (required companion)
  README.md             # optional index of visuals for the workspace
```

**HTML minimum:**

- Explicit page background / color-scheme (avoid blank/self-paint)
- Semantic landmarks (`header`, `main`, labeled sections with stable `id`s)
- Figures: prefer hand-authored SVG; Mermaid only when flowchart/sequence/state is clearer **and** source Mermaid text is also embedded for agent read (and mirrored in outline)
- Stable ids on annotatable nodes (future-proof; no Lavish required)

**Outline minimum (Markdown):**

- Title, purpose, playbook ids used
- Node/section inventory (`id` → meaning)
- Relationships / decisions / open questions in prose tables
- File/path citations when claims touch the repo
- Optional `## Answers` when `input` playbook was used

### Playbooks (on-demand references)

Location (proposed): `pack/.claude/skills/devrites-lib/reference/visual-playbooks/`

| ID | Use when |
| ---- | ---------- |
| `diagram` | Relationships, flows, state, architecture |
| `table` | Dense comparable records |
| `comparison` | Options / before-after / tradeoffs |
| `plan` | Product or technical plan before build |
| `code` | Snippets, files, diffs (prefer focused ranges) |
| `input` | Structured choices the human should make visually |
| `slides` | Only when a paced deck/presentation is requested |

Router stub skill or lib reference: agents **must** open each matching playbook before writing HTML (same discipline as lavish).

Improvements vs lavish:

- Outline dual-read (agents don't guess from HTML alone)
- DevRites path/home conventions + workspace schema entry
- Input → durable answers without a poll server
- Progressive disclosure under `devrites-lib` (token-cheap stub + per-id refs)

### Open helper

```text
devrites-engine open-visual <path-or-name> [--slug <slug>] [--no-open]
```

Behavior:

1. Resolve under active/`DEVRITES_WORKSPACE` `visual/` or absolute path
2. Require `.html`; warn if sibling `.outline.md` missing
3. Unless `--no-open`, launch OS default browser
4. Print compact agent tip: absolute HTML path, outline path, playbook hint

### Skill hooks (existing steps)

| Surface | Change |
| --------- | -------- |
| `workspace-artifact-schema.md` | Document conditional `visual/` + outline contract; budgets |
| `rite-explain` | When visual earns it → write `visual/` HTML+outline (or under explainers run dir with same contract) and offer `open-visual` |
| Spec/Define `flows.md` | Keep Mermaid in `flows.md`; when richer visual needed, **also** emit `visual/<flow>.html`+outline and link from `flows.md` |
| Skill routing | Model-invocable stub or lib pointer so agents load playbooks when about to visualize |

### Deferred roadmap (Durable Roadmap Gate)

| Batch | Scope | Trigger |
| ------- | -------- | --------- |
| B2 | Dogfood journey visuals using playbooks | After B1 ships + dogfood authors request |
| B3 | Optional lavish-axi adapter (annotate/poll) as **opt-in external** tool | User asks; never default dependency |
| B4 | Additional phase hooks (temper strategy boards, vet coverage boards) | After B1 usage evidence |
| B5 | Engine validation of outline↔HTML id parity | If drift becomes a real failure mode |

---

## Tasks

### T1 — Spec + schema contract

- **Owner:** architect (or prompt-engineer with architect review)
- **Do:** Author `.mstar/specs/visual-html-artifacts.md`; update `workspace-artifact-schema.md` conditional `visual/` rows + ownership table; outline template
- **Done when:** Schema documents paths, budgets, non-candidate status, dual-read rule; spec matches locked clarify
- **Interfaces:** schema SSOT for all later writers

### T2 — Playbook pack (7 refs + router)

- **Owner:** prompt-engineer
- **Do:** Add `devrites-lib/reference/visual-playbooks/{index,diagram,table,comparison,plan,code,input,slides}.md`; DevRites-adapted guidance (no Lavish APIs); router load rules in index
- **Done when:** Each id has use_when / structure / design_rules / pitfalls / DevRites notes; index lists ids; skill-trust clean
- **Deps:** T1 for path names

### T3 — Hook `/rite-explain` + `flows.md` writers

- **Owner:** prompt-engineer
- **Do:** Soft-required visual branch in `rite-explain`; Spec/Define/`flows.md` guidance to link `visual/` when diagrams need richer presentation; reply-contract `Record` may cite HTML+outline
- **Done when:** Skills instruct load playbooks → write pair → optional `open-visual`; dual-host regen
- **Deps:** T2

### T4 — `devrites-engine open-visual`

- **Owner:** fullstack-dev
- **Do:** CLI subcommand + tests; path resolve; missing-outline warn; `--no-open`; usage in `main.go` / docs/cli.md
- **Done when:** Unit/CLI tests green; help text accurate; no network
- **Deps:** T1 path contract

### T5 — Host artifact parity + smoke example

- **Owner:** fullstack-dev (+ prompt-engineer check)
- **Do:** Regenerate `pack/generated` host skills; add one fixture/example visual under testdata or docs example; skill-budget/reference-governance green
- **Done when:** Claude/Codex mirrors match; example opens via `open-visual`
- **Deps:** T2–T4

### T6 — Plan QC + light QA

- **Owner:** QC tri + pm-acceptance (docs/skills; engine behavior needs QA if T4 lands)
- **Do:** Review contract drift, Lavish leakage, dual-host, readiness non-inflation
- **Deps:** T1–T5

---

## Verification plan

- Schema + playbooks reachable from `devrites-lib` / explain / flows writers
- `devrites-engine open-visual` opens fixture HTML; warns without outline; `--no-open` prints paths only
- `check skill-trust` on new/changed skills
- Host artifact build parity
- Manual: produce one plan/diagram HTML+outline; agent re-reads outline and restates relationships correctly

## Risks / rollback

| Risk | Mitigation |
| ------ | ------------ |
| Token bloat from playbooks | Progressive refs; stub router; no auto-load all 7 |
| Agents emit pretty HTML without outline | Schema + skill completion criteria require companion |
| Engine scope creep | open-visual only; no poll/share |
| Dual-host drift | Single author tree + regen in T5 |
| Readiness inflation | visual/ non-required; not candidate |

Rollback: revert skill/schema/CLI commits; leftover `visual/` dirs are harmless optional artifacts.

## Findings cleanup

`Findings cleanup: zero-residual` for this plan (fixable skill/engine issues must be fixed before Done).
