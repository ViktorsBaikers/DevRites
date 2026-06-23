"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import { ArrowLeft } from "lucide-react";
import { GithubMark } from "@/components/ui";
import { DOCS_NAV } from "@/lib/docs";
import { REPO, VERSION } from "@/lib/site";

const norm = (p: string) => (p.endsWith("/") ? p : p + "/");

export default function DocsShell({ children }: { children: ReactNode }) {
  const pathname = norm(usePathname() || "/docs/");

  return (
    <div className="relative min-h-screen">
      <div aria-hidden className="grid-field pointer-events-none absolute inset-x-0 top-0 -z-10 h-[40rem] opacity-40" />

      {/* top bar */}
      <header className="sticky top-0 z-30 border-b border-line bg-bg/80 backdrop-blur-xl">
        <div className="wrap flex items-center justify-between py-3">
          <div className="flex items-center gap-3">
            <Link href="/" className="flex items-center gap-2.5">
              <Image src="/assets/img/mark-64.png" width={26} height={26} alt="" />
              <b className="tracking-tight">DevRites</b>
            </Link>
            <span className="mono hidden rounded-full border border-line bg-surface-2/60 px-2 py-0.5 text-[0.62rem] text-ink-faint sm:inline">
              docs · v{VERSION}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Link
              href="/"
              className="hidden items-center gap-1.5 rounded-xl border border-line px-3 py-1.5 text-sm text-ink-muted transition-colors hover:border-line-bright hover:text-ink sm:inline-flex"
            >
              <ArrowLeft className="size-4" /> Back to site
            </Link>
            <a
              href={REPO}
              rel="noopener"
              className="inline-flex items-center gap-1.5 rounded-xl border border-line px-3 py-1.5 text-sm text-ink-muted transition-colors hover:border-line-bright hover:text-ink"
            >
              <GithubMark className="size-4" /> GitHub
            </a>
          </div>
        </div>
      </header>

      <div className="wrap grid gap-10 py-10 lg:grid-cols-[210px_1fr] lg:gap-14">
        {/* sidebar */}
        <aside className="lg:sticky lg:top-24 lg:h-max">
          <nav aria-label="Docs">
            <span className="mono text-[0.62rem] uppercase tracking-[0.16em] text-ink-faint">Docs</span>
            <ul className="mt-3 flex gap-2 overflow-x-auto pb-1 lg:flex-col lg:gap-1 lg:overflow-visible">
              {DOCS_NAV.map((l) => {
                const active = pathname === norm(l.href);
                return (
                  <li key={l.href} className="shrink-0">
                    <Link
                      href={l.href}
                      aria-current={active ? "page" : undefined}
                      className={`block whitespace-nowrap rounded-lg px-3 py-2 text-sm transition-colors ${
                        active
                          ? "bg-surface-2/70 font-medium text-ink"
                          : "text-ink-muted hover:bg-surface/50 hover:text-ink"
                      }`}
                    >
                      {l.label}
                    </Link>
                  </li>
                );
              })}
            </ul>

            <span className="mono mt-7 hidden text-[0.62rem] uppercase tracking-[0.16em] text-ink-faint lg:block">
              Source
            </span>
            <ul className="mt-3 hidden lg:block">
              <li>
                <a href={REPO} rel="noopener" className="block rounded-lg px-3 py-2 text-sm text-ink-muted transition-colors hover:text-ink">
                  GitHub repo
                </a>
              </li>
              <li>
                <a href={`${REPO}#readme`} rel="noopener" className="block rounded-lg px-3 py-2 text-sm text-ink-muted transition-colors hover:text-ink">
                  README
                </a>
              </li>
            </ul>
          </nav>
        </aside>

        {/* content */}
        <main className="min-w-0 max-w-3xl">{children}</main>
      </div>
    </div>
  );
}
