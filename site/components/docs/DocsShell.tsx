"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ArrowLeft, ArrowUpRight, ChevronDown } from "lucide-react";
import { type ReactNode } from "react";
import ThemeToggle from "@/components/ThemeToggle";
import { DOCS_NAV } from "@/lib/docs";
import { REPO } from "@/lib/site";
import DocsSearch from "./DocsSearch";
import DocsToc from "./DocsToc";

const norm = (path: string) => (path.endsWith("/") ? path : `${path}/`);

export default function DocsShell({ children }: { children: ReactNode }) {
  const pathname = norm(usePathname() || "/docs/");

  const nav = (
    <nav aria-label="Documentation" className="grid gap-7">
      {DOCS_NAV.map((group) => (
        <div key={group.group}>
          <p className="mono text-xs font-medium text-ink-faint">{group.group}</p>
          <ul className="mt-3 grid gap-1">
            {group.items.map((link) => {
              const active = pathname === norm(link.href);
              return (
                <li key={link.href}>
                  <Link
                    href={link.href}
                    aria-current={active ? "page" : undefined}
                    className={`block rounded-xl border px-3 py-2 text-sm transition-colors ${
                      active
                        ? "border-line-bright bg-surface-2 font-medium text-ink"
                        : "border-transparent text-ink-muted hover:border-line hover:bg-surface-2/65 hover:text-ink"
                    }`}
                  >
                    {link.label}
                  </Link>
                </li>
              );
            })}
          </ul>
        </div>
      ))}
      <div>
        <p className="mono text-xs font-medium text-ink-faint">Source</p>
        <div className="mt-3 grid gap-1">
          <a href={REPO} rel="noopener" className="rounded-xl px-3 py-2 text-sm text-ink-muted hover:bg-surface-2/65 hover:text-ink">
            GitHub repository
          </a>
          <a href={`${REPO}#readme`} rel="noopener" className="rounded-xl px-3 py-2 text-sm text-ink-muted hover:bg-surface-2/65 hover:text-ink">
            README
          </a>
        </div>
      </div>
    </nav>
  );

  return (
    <div className="relative min-h-[100dvh]">
      <header className="sticky top-0 z-30 border-b border-line bg-bg/90 backdrop-blur-xl">
        <div className="wrap flex h-[4.5rem] items-center justify-between gap-3">
          <Link href="/" className="flex shrink-0 items-center gap-2.5">
            <Image src="/assets/img/mark-64.png" width={27} height={27} alt="" priority />
            <b className="tracking-tight">DevRites</b>
            <span className="hidden text-sm text-ink-faint sm:inline">Docs</span>
          </Link>

          <div className="flex items-center gap-2">
            <DocsSearch />
            <ThemeToggle />
            <Link href="/" className="hidden items-center gap-1.5 rounded-full px-3 py-2 text-sm text-ink-muted hover:bg-surface-2 hover:text-ink md:inline-flex">
              <ArrowLeft className="size-4" aria-hidden /> Back to site
            </Link>
            <a href={REPO} rel="noopener" className="hidden items-center gap-1 rounded-full px-3 py-2 text-sm text-ink-muted hover:bg-surface-2 hover:text-ink sm:inline-flex">
              GitHub <ArrowUpRight className="size-3.5" aria-hidden />
            </a>
          </div>
        </div>
      </header>

      <div className="wrap pt-5 lg:hidden">
        <details className="group rounded-card border border-line bg-surface">
          <summary className="flex cursor-pointer list-none items-center justify-between px-4 py-3 font-medium text-ink">
            Browse docs
            <ChevronDown className="size-4 transition-transform group-open:rotate-180" aria-hidden />
          </summary>
          <div className="border-t border-line p-4">{nav}</div>
        </details>
      </div>

      <div className="wrap grid gap-12 py-12 lg:grid-cols-[220px_minmax(0,1fr)] lg:gap-16 lg:py-20 xl:grid-cols-[220px_minmax(0,48rem)_170px] xl:gap-14">
        <aside className="hidden min-w-0 border-r border-line pr-7 lg:sticky lg:top-28 lg:block lg:h-max">
          {nav}
        </aside>
        <main id="docs-content" className="min-w-0 pb-24">{children}</main>
        <div className="hidden xl:block">
          <DocsToc />
        </div>
      </div>
    </div>
  );
}
