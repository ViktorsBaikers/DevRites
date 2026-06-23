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
    <header className="mb-10">
      <Reveal>
        <p className="mono text-xs text-ink-faint">
          DevRites <span className="text-line-bright">/</span> docs{" "}
          <span className="text-line-bright">/</span>{" "}
          <span className="text-accent">{crumb}</span>
        </p>
      </Reveal>
      <Reveal delay={0.05}>
        <h1 className="mt-3 font-black [font-size:clamp(2.2rem,5vw,3.25rem)]">{title}</h1>
      </Reveal>
      <Reveal delay={0.1}>
        <p className="mt-4 text-pretty text-ink-muted [font-size:var(--text-lead)] leading-relaxed">
          {lead}
        </p>
      </Reveal>
    </header>
  );
}
