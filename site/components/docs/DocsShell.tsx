"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ArrowLeft, ArrowRight, ArrowUpRight, ChevronDown } from "lucide-react";
import { useRef, type ReactNode } from "react";
import { DOCS_NAV, DOCS_LINKS } from "@/lib/docs";
import { REPO } from "@/lib/site";
import DocsSearch from "./DocsSearch";
import DocsToc from "./DocsToc";

const norm = (path: string) => (path.endsWith("/") ? path : `${path}/`);

export default function DocsShell({ children }: { children: ReactNode }) {
  const pathname = norm(usePathname() || "/docs/");
  const browse = useRef<HTMLDetailsElement>(null);
  const current = DOCS_LINKS.findIndex((link) => norm(link.href) === pathname);
  const previous = DOCS_LINKS[current - 1];
  const next = DOCS_LINKS[current + 1];
  const nav = (
    <nav aria-label="Documentation" className="docs-index">
      {DOCS_NAV.map((group) => <div key={group.group}><h2>{group.group}</h2><ul>{group.items.map((link) => (
        <li key={link.href}><Link href={link.href} aria-current={pathname === norm(link.href) ? "page" : undefined}
          onClick={() => { if (browse.current) browse.current.open = false; }}>{link.label}</Link></li>
      ))}</ul></div>)}
      <a className="docs-source-link" href={REPO}>View the source <ArrowUpRight size={16} aria-hidden /></a>
    </nav>
  );
  return (
    <div className="docs-shell docs-register">
      <a href="#docs-content" className="docs-skip">Skip to documentation</a>
      <header className="docs-topbar"><div className="docs-topbar-inner">
        <Link href="/" className="docs-brand"><b>DevRites</b><span>Documentation</span></Link>
        <div className="docs-tools"><DocsSearch /><a href={REPO} className="docs-github">GitHub <ArrowUpRight size={16} aria-hidden /></a></div>
      </div></header>
      <div className="docs-mobile-browse"><details ref={browse}>
        <summary>Browse docs <span>{DOCS_LINKS[current]?.label}</span><ChevronDown size={18} aria-hidden /></summary>{nav}
      </details></div>
      <div className="docs-layout">
        <aside className="docs-sidebar">{nav}<p className="docs-sidebar-note">The workflow is in your repository.<br />This is the field guide.</p></aside>
        <main id="docs-content" className="docs-article">{children}
          <nav className="docs-page-turn" aria-label="Previous and next pages">
            {previous ? <Link href={previous.href}><span><ArrowLeft size={16} aria-hidden /> Previous</span><strong>{previous.label}</strong></Link> : <Link href="/"><span><ArrowLeft size={16} aria-hidden /> Back to</span><strong>The change register</strong></Link>}
            {next && <Link href={next.href}><span>Continue <ArrowRight size={16} aria-hidden /></span><strong>{next.label}</strong></Link>}
          </nav>
          <footer className="docs-article-footer">Repository-local. Evidence-bound. Human-approved.<a href={REPO}>Source on GitHub <ArrowUpRight size={14} aria-hidden /></a></footer>
        </main>
        <div className="docs-toc-column"><DocsToc /></div>
      </div>
    </div>
  );
}
