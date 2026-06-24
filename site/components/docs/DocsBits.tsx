"use client";

import type { ReactNode } from "react";
import { Reveal } from "@/components/ui";

export function H2({ id, children }: { id?: string; children: ReactNode }) {
  return (
    <Reveal>
      <h2 id={id} className="mt-14 scroll-mt-24 text-2xl font-bold first:mt-0">
        {children}
      </h2>
    </Reveal>
  );
}

export function H3({ children }: { children: ReactNode }) {
  return (
    <Reveal>
      <h3 className="mt-8 text-lg font-bold text-ink">{children}</h3>
    </Reveal>
  );
}

export function P({ children }: { children: ReactNode }) {
  return (
    <Reveal delay={0.04}>
      <p className="mt-3 text-pretty text-ink-muted leading-relaxed">{children}</p>
    </Reveal>
  );
}

export function Panel({ children }: { children: ReactNode }) {
  return (
    <Reveal delay={0.06}>
      <div className="mt-5 overflow-hidden rounded-tile border border-line">{children}</div>
    </Reveal>
  );
}

export function Row({ left, tag, body }: { left: string; tag?: string; body: string }) {
  return (
    <div className="flex flex-col gap-1.5 border-b border-line p-4 last:border-0 sm:flex-row sm:items-baseline sm:gap-4">
      <div className="flex w-full shrink-0 items-center gap-2 sm:w-56">
        <code className="mono text-sm text-accent">{left}</code>
        {tag && (
          <span className="mono rounded border border-line px-1.5 py-0.5 text-[0.6rem] uppercase tracking-wide text-ink-faint">
            {tag}
          </span>
        )}
      </div>
      <p className="text-[0.9rem] leading-relaxed text-ink-muted">{body}</p>
    </div>
  );
}

export function Code({ children }: { children: ReactNode }) {
  return (
    <Reveal delay={0.06}>
      <pre className="mono mt-5 overflow-x-auto rounded-tile border border-line bg-bg-deep/60 p-4 text-[0.82rem] leading-relaxed text-ink-muted">
        {children}
      </pre>
    </Reveal>
  );
}

export function Callout({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Reveal delay={0.06}>
      <div className="tile--lit mt-6 rounded-tile p-5">
        <h3 className="font-bold text-ink">{title}</h3>
        <p className="mt-2 text-[0.9rem] leading-relaxed text-ink-muted">{children}</p>
      </div>
    </Reveal>
  );
}
