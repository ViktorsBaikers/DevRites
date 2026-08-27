# Browser proof checklist

- Open the real UI (never screenshot-only); check console and network for errors.
- Interactive slices capture each relevant state (default/hover/focus-visible/active/disabled/loading/empty/error) at 320 + 768 px, +1024/1440 when adaptive; state floor: [`../../../devrites-frontend-craft/reference/quality-standards.md`](../../../devrites-frontend-craft/reference/quality-standards.md). Omission needs a one-line `not-needed` reason; states must exist in source — an unreachable state's capture proves nothing.
- Review browser-default surfaces once per slice (selection, caret, scrollbars, focus ring); no horizontal overflow at captured widths; 200% zoom spot-check; compare to `design-brief.md` → Visual Verdict.
- Tooling unavailable ⇒ record fallback + limitation. Backend-only changes record that disposition instead of capturing quietly; UI copy follows `devrites-frontend-craft`, long-form prose follows [`prose-style.md`](prose-style.md).

Detailed skill: `devrites-browser-proof`.
