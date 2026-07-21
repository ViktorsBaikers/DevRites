"use client";

import Image from "next/image";
import { useEffect, useState } from "react";
import { ArrowUpRight, Menu, X } from "lucide-react";
import { MagneticLink } from "./ui";
import ThemeToggle from "./ThemeToggle";
import { REPO } from "@/lib/site";

const LINKS = [
  { href: "#workflow", label: "Workflow" },
  { href: "#mechanisms", label: "Mechanisms" },
  { href: "#anywhere", label: "Hosts" },
  { href: "#faq", label: "FAQ" },
  { href: "/docs/", label: "Docs" },
];

export default function Nav() {
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState<string | null>(null);

  useEffect(() => {
    const sections = LINKS.flatMap((link) => {
      if (!link.href.startsWith("#")) return [];
      const section = document.querySelector<HTMLElement>(link.href);
      return section ? [section] : [];
    });
    const observer = new IntersectionObserver(
      (entries) => {
        const current = entries.find((entry) => entry.isIntersecting);
        if (current) setActive(`#${current.target.id}`);
      },
      { rootMargin: "-30% 0px -60% 0px" },
    );
    sections.forEach((section) => observer.observe(section));
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    if (!open) return;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("keydown", closeOnEscape);
    return () => document.removeEventListener("keydown", closeOnEscape);
  }, [open]);

  return (
    <header className="pointer-events-none fixed inset-x-0 top-3 z-30">
      <div className="nav-shell wrap pointer-events-auto flex h-16 items-center justify-between gap-5 rounded-card px-3 sm:px-4">
        <a href="#top" className="flex shrink-0 items-center gap-2.5" aria-label="DevRites home">
          <Image src="/assets/img/mark-64.png" width={28} height={28} alt="" priority />
          <b className="text-[1.05rem] tracking-tight">DevRites</b>
        </a>

        <nav className="hidden items-center gap-1 lg:flex" aria-label="Primary">
          {LINKS.map((link) => (
            <a
              key={link.href}
              href={link.href}
              aria-current={active === link.href ? "location" : undefined}
              className={`rounded-full px-3.5 py-2 text-sm transition-colors ${
                active === link.href
                  ? "bg-surface-2 text-ink"
                  : "text-ink-muted hover:bg-surface-2/65 hover:text-ink"
              }`}
            >
              {link.label}
            </a>
          ))}
        </nav>

        <div className="flex items-center justify-end gap-2">
          <a
            href={REPO}
            rel="noopener"
            className="hidden items-center gap-1 rounded-full px-3 py-2 text-sm text-ink-muted transition-colors hover:bg-surface-2/65 hover:text-ink sm:inline-flex"
          >
            GitHub <ArrowUpRight className="size-3.5" aria-hidden />
          </a>
          <ThemeToggle />
          <MagneticLink href="#install" className="btn btn-primary px-4 py-2 text-sm">
            Install
          </MagneticLink>
          <button
            type="button"
            className="inline-flex size-9 cursor-pointer items-center justify-center rounded-full border border-line text-ink-muted lg:hidden"
            aria-label={open ? "Close menu" : "Open menu"}
            aria-controls="mobile-menu"
            aria-expanded={open}
            onClick={() => setOpen((value) => !value)}
          >
            {open ? <X className="size-4" aria-hidden /> : <Menu className="size-4" aria-hidden />}
          </button>
        </div>
      </div>

      {open ? (
        <nav
          id="mobile-menu"
          aria-label="Mobile"
          className="pointer-events-auto absolute inset-x-3 top-[4.5rem] grid rounded-card bg-surface p-2 lg:hidden"
        >
          {LINKS.map((link) => (
            <a
              key={link.href}
              href={link.href}
              aria-current={active === link.href ? "location" : undefined}
              onClick={() => {
                setActive(link.href.startsWith("#") ? link.href : null);
                setOpen(false);
              }}
              className={`rounded-xl px-4 py-3 text-base transition-colors ${
                active === link.href ? "bg-surface-2 text-ink" : "text-ink-muted hover:bg-surface-2/65 hover:text-ink"
              }`}
            >
              {link.label}
            </a>
          ))}
          <a href={REPO} rel="noopener" className="rounded-xl px-4 py-3 text-ink-muted hover:bg-surface-2/65 hover:text-ink">
            GitHub
          </a>
        </nav>
      ) : null}
    </header>
  );
}
