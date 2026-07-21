"use client";

import type { ReactNode } from "react";
import { Reveal } from "@/components/ui";

export default function DocsHeader({
  crumb,
  title,
  lead,
}: {
  crumb: string;
  title: string;
  lead: ReactNode;
}) {
  return (
    <header className="mb-14 border-b border-line pb-10 md:mb-16 md:pb-12">
      <Reveal>
        <p className="mono text-xs text-ink-faint">
          DevRites <span className="text-line-bright">/</span> docs{" "}
          <span className="text-line-bright">/</span>{" "}
          <span className="text-accent">{crumb}</span>
        </p>
      </Reveal>
      <Reveal delay={0.05}>
        <h1 className="mt-5 max-w-3xl font-bold [font-size:clamp(2.7rem,5.2vw,5rem)] leading-[0.94] tracking-[-0.04em]">{title}</h1>
      </Reveal>
      <Reveal delay={0.1}>
        <p className="mt-6 max-w-[65ch] text-pretty text-ink-muted [font-size:var(--text-lead)] leading-relaxed">
          {lead}
        </p>
      </Reveal>
    </header>
  );
}
