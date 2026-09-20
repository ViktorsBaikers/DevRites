import type { ReactNode } from "react";

export default function DocsHeader({ crumb, title, lead }: { crumb: string; title: string; lead: ReactNode }) {
  return <header className="docs-article-header"><p className="docs-crumb">{crumb}</p><h1>{title}</h1><p className="docs-article-lead">{lead}</p></header>;
}
