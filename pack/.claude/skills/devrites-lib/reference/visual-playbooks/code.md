# Visual playbook: code

## use_when

Render source snippets, files, patches, PR diffs, or before/after code inside a visual — when the claim needs readable code next to explanation (prefer focused ranges, not whole unrelated files).

## Structure / landmarks / stable ids

| Landmark | Suggested `id` | Role |
| --- | --- | --- |
| Claim / reason | `code-why` | Why inspect this code |
| Path header | `code-path-<slug>` | Path, language, scope |
| File or diff surface | `code-view-<slug>` | Rendered file or diff |
| Annotations | `code-notes` | Line-tied notes beside the claim |

Place path, language, and reason immediately before each render. Group multi-file changes by user-facing area or task, not raw repo order.

## design_rules

- Prefer **focused ranges** and parsed patches over dumping huge files.
- Keep evidence next to claims (path + line references in HTML and outline Citations).
- **Simple snippets:** semantic `<pre><code>` (or equivalent) with language class and wrap-friendly CSS is acceptable when no interactive diff is needed.
- **Diffs / multi-file review:** may use `@pierre/diffs` from a pinned CDN (e.g. esm.sh) when side-by-side or unified diff UX is needed. If used:
  - Pin the version in the script URL.
  - Note the CDN dependency in the outline.
  - Prefer themes that match the page light/dark scheme.
  - Choose split vs unified for width; keep wrap unless alignment is essential.
- Prefer self-contained CSS for chrome around the code surface.
- Explicit background / color-scheme; stable ids on each file/diff block.

## Pitfalls / anti-patterns

- Screenshots of code instead of text the agent can re-read.
- Showing huge unrelated files when a range would do.
- Separating a claim from the lines that prove it.
- Hard-requiring Lavish annotation / queue APIs around the code surface.
- Using a CDN without recording it in the outline.
- HTML without `.outline.md`.

## DevRites notes

- **Home:** `.devrites/work/<slug>/visual/<name>.html` + `<name>.outline.md`.
- Outline: [`outline-template.md`](outline-template.md); list `code` under Playbooks used; note CDN if `@pierre/diffs` or similar is used.
- **Outline wins** on conflict — include path/line claims in Citations even when HTML renders diffs.
- Often combines with `plan`, `table`, or `comparison` — open every match ([`index.md`](index.md)).
- **No new phase**; optional; not readiness-required.
