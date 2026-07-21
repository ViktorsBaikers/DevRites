"use client";

import Link from "next/link";
import { Search, X } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { DOCS_LINKS } from "@/lib/docs";

const DESCRIPTIONS: Record<string, string> = {
  "/docs/": "How the workflow, engine, hosts, and feature files fit together.",
  "/docs/getting-started/": "Install DevRites and run the first verified feature.",
  "/docs/concepts/": "Understand slices, gates, evidence, workspaces, and run modes.",
  "/docs/usage/": "Use build, spec drift, HITL, AFK, and autocomplete in realistic workflows.",
  "/docs/command-map/": "Find the right command for the job in Claude or Codex.",
  "/docs/flow/": "Follow the lifecycle, drift handling, run modes, checks, and release.",
  "/docs/cli-mcp/": "Reference deterministic engine commands and exit behavior.",
  "/docs/architecture/": "Inspect the generated host surfaces and Go control plane.",
};

export default function DocsSearch() {
  const dialog = useRef<HTMLDialogElement>(null);
  const input = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState("");
  const matches = DOCS_LINKS.filter((link) =>
    `${link.label} ${DESCRIPTIONS[link.href]}`.toLowerCase().includes(query.trim().toLowerCase()),
  );

  const open = useCallback(() => {
    const modal = dialog.current;
    if (!modal) return;
    if (!modal.open) modal.showModal();
    requestAnimationFrame(() => input.current?.focus());
  }, []);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target instanceof HTMLElement ? event.target : null;
      const typing = target?.matches("input, textarea, select, [contenteditable='true']") ?? false;
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        open();
      } else if (event.key === "/" && !typing) {
        event.preventDefault();
        open();
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [open]);

  return (
    <>
      <button
        type="button"
        onClick={open}
        aria-label="Search documentation"
        className="inline-flex h-9 items-center gap-2 rounded-full border border-line px-3 text-sm text-ink-muted transition-colors hover:border-line-bright hover:bg-surface-2 hover:text-ink"
      >
        <Search className="size-4" aria-hidden />
        <span className="hidden sm:inline">Search docs</span>
        <kbd className="mono hidden rounded-md bg-surface-2 px-1.5 py-0.5 text-[0.65rem] text-ink-faint md:inline">/</kbd>
      </button>

      <dialog
        ref={dialog}
        aria-label="Search documentation"
        className="docs-search elevated m-auto w-[min(42rem,calc(100%-2rem))] rounded-card border border-line bg-surface p-0 text-ink"
        onClose={() => setQuery("")}
      >
        <div className="flex items-center gap-3 border-b border-line p-4">
          <Search className="size-5 shrink-0 text-ink-faint" aria-hidden />
          <label htmlFor="docs-search" className="sr-only">Search documentation</label>
          <input
            ref={input}
            id="docs-search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search documentation"
            className="min-w-0 flex-1 bg-transparent text-base text-ink outline-none placeholder:text-ink-faint"
          />
          <button
            type="button"
            onClick={() => dialog.current?.close()}
            className="inline-flex size-8 items-center justify-center rounded-full text-ink-muted hover:bg-surface-2 hover:text-ink"
            aria-label="Close search"
          >
            <X className="size-4" aria-hidden />
          </button>
        </div>

        <div className="max-h-[65dvh] overflow-y-auto p-2">
          {matches.length ? (
            matches.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                onClick={() => dialog.current?.close()}
                className="block rounded-xl px-4 py-3 transition-colors hover:bg-surface-2"
              >
                <strong className="font-semibold text-ink">{link.label}</strong>
                <span className="mt-1 block text-sm leading-relaxed text-ink-muted">{DESCRIPTIONS[link.href]}</span>
              </Link>
            ))
          ) : (
            <div className="px-4 py-12 text-center">
              <p className="font-semibold text-ink">No matching page</p>
              <p className="mt-2 text-sm text-ink-muted">Try a command, workflow phase, or architecture term.</p>
            </div>
          )}
        </div>
      </dialog>
    </>
  );
}
