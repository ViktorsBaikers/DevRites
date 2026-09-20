"use client";

import { useEffect, useState } from "react";
import { Menu, Search, X } from "lucide-react";
import { CopyButton } from "./ui";
import { INSTALL_CMD, REPO } from "@/lib/site";

const LINKS = [
  { href: "/docs/", label: "Docs" },
  { href: "/docs/getting-started/", label: "Guides" },
  { href: "https://github.com/ViktorsBaikers/DevRites/releases", label: "Changelog" },
  { href: REPO, label: "GitHub" },
  { href: "/docs/command-map/", label: "Commands" },
];

export default function Nav() {
  const [open, setOpen] = useState(false);
  useEffect(() => {
    if (!open) return;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("keydown", closeOnEscape);
    return () => document.removeEventListener("keydown", closeOnEscape);
  }, [open]);

  return (
    <header className="site-nav">
      <div className="site-nav-inner">
        <a href="#top" className="site-wordmark" aria-label="DevRites home">
          DevRites
        </a>

        <nav className="site-nav-links" aria-label="Primary">
          {LINKS.map((link) => (
            <a key={link.href} href={link.href}>{link.label}</a>
          ))}
        </nav>

        <div className="site-nav-tools">
          <a href="/docs/" className="site-search" aria-label="Search documentation"><Search aria-hidden /><span>Search docs…</span><kbd>/</kbd></a>
          <div className="site-install-command"><code>&gt;&nbsp; {INSTALL_CMD}</code><CopyButton text={INSTALL_CMD} label="Copy install command" /></div>
          <p>Repository local<br />Spec to ship</p>
          <button
            type="button"
            className="site-menu-button"
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
          className="site-mobile-menu"
        >
          {LINKS.map((link) => (
            <a
              key={link.href}
              href={link.href}
              onClick={() => setOpen(false)}
            >
              {link.label}
            </a>
          ))}
          <a href="#install" onClick={() => setOpen(false)}>Install</a>
        </nav>
      ) : null}
    </header>
  );
}
