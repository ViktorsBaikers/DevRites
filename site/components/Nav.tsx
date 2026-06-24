"use client";

import Image from "next/image";
import { useState } from "react";
import { useScroll, useMotionValueEvent } from "framer-motion";
import { Menu, X } from "lucide-react";
import { GithubMark, MagneticLink } from "./ui";
import { REPO, VERSION } from "@/lib/site";

const LINKS = [
  { href: "#workflow", label: "Workflow" },
  { href: "#mechanisms", label: "Mechanisms" },
  { href: "#anywhere", label: "Drive anywhere" },
  { href: "#faq", label: "FAQ" },
  { href: "/docs/", label: "Docs" },
];

export default function Nav() {
  const [scrolled, setScrolled] = useState(false);
  const [open, setOpen] = useState(false);

  // Motion's useScroll instead of a raw window scroll listener (no per-frame work)
  const { scrollY } = useScroll();
  useMotionValueEvent(scrollY, "change", (y) => setScrolled(y > 12));

  return (
    <header className="fixed inset-x-0 top-0 z-30 flex justify-center px-3 pt-3">
      <div
        className={`wrap flex items-center justify-between rounded-2xl px-4 py-2.5 transition-all duration-300 ${
          scrolled ? "glass shadow-lg shadow-black/20" : "border border-transparent"
        }`}
      >
        <a href="#top" className="flex items-center gap-2.5" aria-label="DevRites home">
          <Image src="/assets/img/mark-64.png" width={28} height={28} alt="" priority />
          <b className="text-[1.05rem] tracking-tight">DevRites</b>
          <span className="mono hidden rounded-full border border-line bg-surface-2/60 px-2 py-0.5 text-[0.65rem] text-ink-faint sm:inline">
            v{VERSION}
          </span>
        </a>

        <nav className="hidden items-center gap-7 md:flex" aria-label="Primary">
          {LINKS.map((l) => (
            <a
              key={l.href}
              href={l.href}
              className="text-sm text-ink-muted transition-colors duration-200 hover:text-ink"
            >
              {l.label}
            </a>
          ))}
        </nav>

        <div className="flex items-center gap-2">
          <a
            href={REPO}
            rel="noopener"
            className="hidden items-center gap-1.5 rounded-xl border border-line px-3 py-1.5 text-sm text-ink-muted transition-colors duration-200 hover:border-line-bright hover:text-ink sm:inline-flex"
          >
            <GithubMark className="size-4" /> GitHub
          </a>
          <MagneticLink href="#install" className="btn btn-primary px-3.5 py-1.5 text-sm">
            Install
          </MagneticLink>
          <button
            type="button"
            className="ml-1 cursor-pointer rounded-lg border border-line p-1.5 text-ink-muted md:hidden"
            aria-label={open ? "Close menu" : "Open menu"}
            aria-expanded={open}
            onClick={() => setOpen((o) => !o)}
          >
            {open ? <X className="size-5" /> : <Menu className="size-5" />}
          </button>
        </div>
      </div>

      {open && (
        <div className="glass absolute inset-x-3 top-[4.5rem] z-30 rounded-2xl p-3 md:hidden">
          <nav className="flex flex-col" aria-label="Mobile">
            {LINKS.map((l) => (
              <a
                key={l.href}
                href={l.href}
                onClick={() => setOpen(false)}
                className="rounded-lg px-3 py-2.5 text-ink-muted transition-colors hover:bg-surface-2/60 hover:text-ink"
              >
                {l.label}
              </a>
            ))}
            <a
              href={REPO}
              rel="noopener"
              className="rounded-lg px-3 py-2.5 text-ink-muted transition-colors hover:bg-surface-2/60 hover:text-ink"
            >
              GitHub
            </a>
          </nav>
        </div>
      )}
    </header>
  );
}
