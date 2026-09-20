import type { ReactNode } from "react";
import { CircleAlert } from "lucide-react";

export function H2({ id, first = false, children }: { id?: string; first?: boolean; children: ReactNode }) {
  return <h2 id={id} className={`docs-heading${first ? " docs-heading-first" : ""}`}>{children}</h2>;
}
export function H3({ children }: { children: ReactNode }) {
  return <h3 className="docs-subheading">{children}</h3>;
}
export function P({ children }: { children: ReactNode }) {
  return <p className="docs-paragraph">{children}</p>;
}
export function Panel({ children }: { children: ReactNode }) {
  return <div className="docs-panel">{children}</div>;
}
export function Row({ left, tag, body }: { left: string; tag?: string; body: string }) {
  return <div className="docs-reference-row"><div><code>{left}</code>{tag && <span className="docs-row-tag">{tag}</span>}</div><p>{body}</p></div>;
}
export function Code({ children }: { children: ReactNode }) {
  return <pre className="docs-code" tabIndex={0}><code>{children}</code></pre>;
}
export function Callout({ title, children }: { title: string; children: ReactNode }) {
  return <aside className="docs-callout"><CircleAlert className="docs-callout-mark" aria-hidden /><div className="docs-callout-copy"><h3>{title}</h3><p>{children}</p></div></aside>;
}
