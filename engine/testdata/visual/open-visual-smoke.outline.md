# open-visual smoke fixture

## Purpose

Prove that `devrites-engine open-visual` resolves a portable HTML+outline pair under `engine/testdata/visual/`, prints agent tips, runs a non-fatal inventory↔HTML id consistency check, and honors `--no-open` without launching a browser. The HTML teaches the dual-read → opener flow for humans reviewing the smoke surface.

## Playbooks used

| ID | Why loaded |
| --- | --- |
| diagram | Small relationship diagram for the opener smoke path |

## ID inventory

| HTML `id` | Meaning |
| --- | --- |
| `viz-title` | Page header / question the fixture answers |
| `diagram-overview` | Core relationship figure |
| `diagram-title` | Accessible SVG title |
| `node-html` | HTML fixture node |
| `node-outline` | Outline companion node |
| `node-opener` | open-visual command node |
| `edge-html-outline` | Dual-read pairing |
| `edge-outline-opener` | Opener tip path |
| `diagram-detail` | Prose steps mirroring the overview |
| `diagram-legend` | Symbol / behavior key |
| `viz-open-questions` | Open questions with confidence |
| `diagram-citations` | Repo path citations |

## Relationships

| From | To | Relationship / decision / open question |
| --- | --- | --- |
| `node-html` | `node-outline` | Dual-read pair; outline is SSOT on conflict |
| `node-outline` | `node-opener` | Opener prints outline path + playbook tip |
| `open-visual --no-open` | browser | Must not launch; tips still print |
| `diagram-overview` | `diagram-detail` | Same three-step story: pair → SSOT → opener |
| `#devrites-outline` | `.outline.md` | JSON is a mirror; outline.md wins on conflict |

## Citations

| Claim | Path |
| --- | --- |
| Opener implementation | `engine/internal/lib/open_visual.go` |
| Playbook router | `pack/.claude/skills/devrites-lib/reference/visual-playbooks/index.md` |
| Outline template | `pack/.claude/skills/devrites-lib/reference/visual-playbooks/outline-template.md` |
| rite-explain tip hook | `pack/.claude/skills/rite-explain/SKILL.md` |

## Open questions

| id | question | confidence | status |
| --- | --- | --- | --- |
| `q-json-ssot` | Should agents ever prefer `#devrites-outline` JSON over `.outline.md` when both exist? | high | open |
| `q-browser-ci` | Do all interactive review hosts have a usable OS opener, or is `--no-open` the default smoke path? | medium | open |
