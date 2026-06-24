"use client";

import { Reveal, SectionHead } from "./ui";
import { TOOLS } from "@/lib/site";

const CLI = [
  { cmd: "devrites orient", note: "workspace digest", tone: "text-ink-faint" },
  { cmd: "devrites ready", note: "exit 0 · ready", tone: "text-go" },
  { cmd: "devrites evidence-fresh", note: "exit 3 · stale", tone: "text-warn" },
  { cmd: "devrites acceptance", note: "exit 0 · proven", tone: "text-go" },
];

const MCP_TOOLS = [
  "devrites_orient", "devrites_ready", "devrites_evidence_fresh", "devrites_acceptance",
  "devrites_status", "devrites_active", "devrites_list", "devrites_use",
];

export default function Anywhere() {
  return (
    <section id="anywhere" className="wrap py-20 sm:py-28">
      <SectionHead
        eyebrow="Tool-agnostic core"
        title="The discipline lives in files, not in one harness."
        lead="A portable CLI and a dependency-free MCP server expose the same deterministic gates to any agent, CI job, or human. One verdict, everywhere."
      />

      <div className="mt-12 grid gap-4 lg:grid-cols-2">
        {/* CLI panel */}
        <Reveal className="glass overflow-hidden rounded-card">
          <div className="flex items-center gap-2 border-b border-line bg-surface-2/40 px-4 py-3">
            <span className="flex gap-1.5">
              <i className="size-2.5 rounded-full bg-danger/60" />
              <i className="size-2.5 rounded-full bg-warn/60" />
              <i className="size-2.5 rounded-full bg-go/60" />
            </span>
            <span className="mono ml-2 text-xs text-ink-muted">
              the <b className="text-ink">devrites</b> CLI · exit code is the gate
            </span>
          </div>
          <div className="space-y-2.5 p-5 font-mono text-sm">
            {CLI.map((r) => (
              <div key={r.cmd} className="flex items-center justify-between gap-3">
                <span className="text-ink">
                  <span className="text-accent">$</span> {r.cmd}
                </span>
                <span className={r.tone}>{r.note}</span>
              </div>
            ))}
            <p className="mono pt-2 text-[0.78rem] leading-relaxed text-ink-faint">
              A non-zero ready, evidence-fresh, or acceptance is a hard stop you can script into
              any agent loop or a pre-merge CI step.
            </p>
          </div>
        </Reveal>

        {/* MCP panel */}
        <Reveal delay={0.1} className="glass overflow-hidden rounded-card">
          <div className="flex items-center gap-2 border-b border-line bg-surface-2/40 px-4 py-3">
            <span className="flex gap-1.5">
              <i className="size-2.5 rounded-full bg-danger/60" />
              <i className="size-2.5 rounded-full bg-warn/60" />
              <i className="size-2.5 rounded-full bg-go/60" />
            </span>
            <span className="mono ml-2 text-xs text-ink-muted">
              the MCP server · dependency-free stdio
            </span>
          </div>
          <div className="p-5">
            <div className="flex flex-wrap gap-1.5">
              {MCP_TOOLS.map((t) => (
                <span
                  key={t}
                  className="mono rounded-md border border-line bg-surface-2/50 px-2 py-1 text-[0.72rem] text-ink-muted"
                >
                  {t}
                </span>
              ))}
            </div>
            <p className="mono pt-3 text-[0.78rem] leading-relaxed text-ink-faint">
              Register it in a project&rsquo;s .mcp.json and any MCP client can ask &ldquo;is this
              feature ready to ship?&rdquo; and get the same verdict /rite-seal would compute.
            </p>
          </div>
        </Reveal>
      </div>

      <Reveal delay={0.15}>
        <div className="mt-8 flex flex-wrap items-center gap-x-3 gap-y-2">
          <span className="mono text-xs uppercase tracking-[0.16em] text-ink-faint">
            same files, same gates, from
          </span>
          {TOOLS.map((t) => (
            <span
              key={t}
              className="rounded-full border border-line bg-surface/50 px-3 py-1 text-sm text-ink-muted"
            >
              {t}
            </span>
          ))}
        </div>
      </Reveal>
    </section>
  );
}
