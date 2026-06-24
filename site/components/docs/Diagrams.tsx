"use client";

import { motion, useReducedMotion } from "framer-motion";
import { ChevronDown, RotateCcw, Check } from "lucide-react";
import { Reveal, EASE } from "@/components/ui";
import { PHASES } from "@/lib/site";

const ACT_TONE: Record<string, { text: string; ring: string; bg: string; track: string }> = {
  shape: { text: "text-accent", ring: "border-accent/60", bg: "bg-accent/15", track: "bg-accent/60" },
  build: { text: "text-warn", ring: "border-warn/60", bg: "bg-warn/15", track: "bg-warn/60" },
  ship: { text: "text-go", ring: "border-go/60", bg: "bg-go/15", track: "bg-go/60" },
};

/* ---- 1. The 10-phase pipeline, grouped into 3 acts ---- */
export function PipelineDiagram() {
  const reduce = useReducedMotion() ?? false;
  const acts = [
    { key: "shape", label: "shape", span: 4 },
    { key: "build", label: "build", span: 4 },
    { key: "ship", label: "ship", span: 2 },
  ];

  return (
    <Reveal>
      <div className="tile overflow-x-auto p-5 sm:p-6">
        <div className="min-w-[640px]">
          {/* act bands */}
          <div className="mb-3 flex gap-2">
            {acts.map((a) => {
              const t = ACT_TONE[a.key];
              return (
                <div key={a.key} style={{ flex: a.span }}>
                  <div className={`mono text-[0.7rem] uppercase tracking-[0.16em] ${t.text}`}>{a.label}</div>
                  <div className={`mt-1 h-1 rounded-full ${t.track}`} />
                </div>
              );
            })}
          </div>

          {/* nodes */}
          <div className="flex items-start">
            {PHASES.map((p, i) => {
              const t = ACT_TONE[p.act];
              const last = i === PHASES.length - 1;
              return (
                <div key={p.name} className="flex flex-1 items-start">
                  <motion.div
                    initial={reduce ? false : { opacity: 0, scale: 0.5 }}
                    whileInView={reduce ? undefined : { opacity: 1, scale: 1 }}
                    viewport={{ once: true }}
                    transition={{ duration: 0.4, ease: EASE, delay: i * 0.06 }}
                    className="flex flex-col items-center gap-1.5 text-center"
                    style={{ width: 48 }}
                  >
                    <span className={`flex size-9 items-center justify-center rounded-full border ${t.ring} ${t.bg} ${t.text} text-sm font-bold ${last ? "rotate-45" : ""}`}>
                      <span className={last ? "-rotate-45" : ""}>{last ? <Check className="size-4" strokeWidth={3} /> : i + 1}</span>
                    </span>
                    <span className="mono text-[0.62rem] text-ink-muted">{p.name}</span>
                  </motion.div>
                  {!last && (
                    <div className="mt-4 h-px flex-1 bg-line" />
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </Reveal>
  );
}

/* ---- 2. How one phase works: read -> work -> write, on disk ---- */
export function PhaseLoop() {
  const steps = [
    { t: "Reads", b: "the previous phase's files" },
    { t: "Works", b: "investigate, build, or review" },
    { t: "Writes", b: "its own Markdown artifacts" },
  ];
  return (
    <Reveal>
      <div className="tile p-5 sm:p-6">
        <div className="grid gap-3 sm:grid-cols-[1fr_auto_1fr_auto_1fr] sm:items-center">
          {steps.map((s, i) => (
            <div key={s.t} className="contents">
              <div className="rounded-tile border border-line bg-surface-2/40 p-4 text-center">
                <div className="font-bold text-accent">{s.t}</div>
                <div className="mt-1 text-[0.82rem] text-ink-muted">{s.b}</div>
              </div>
              {i < steps.length - 1 && (
                <ChevronDown className="mx-auto size-5 rotate-[-90deg] text-line-bright max-sm:rotate-0" />
              )}
            </div>
          ))}
        </div>
        <div className="mono mt-4 rounded-lg border border-dashed border-line bg-bg-deep/50 px-4 py-2.5 text-center text-[0.78rem] text-ink-faint">
          all of it on disk under <span className="text-accent">.devrites/work/&lt;slug&gt;/</span>. It survives /clear and new sessions
        </div>
      </div>
    </Reveal>
  );
}

/* ---- 3. Architecture: who drives what, down to the source of truth ---- */
export function StackDiagram() {
  const layers = [
    { label: "Anyone drives it", items: ["Claude Code", "Cursor", "Codex", "Gemini CLI", "CI", "a human"], tone: "text-ink-muted" },
    { label: "Three surfaces", items: ["rite-* commands", "devrites CLI", "MCP server"], tone: "text-accent" },
    { label: "Orchestration", items: ["13 review agents", "internal specialists", "engineering rules"], tone: "text-warn" },
  ];
  return (
    <Reveal>
      <div className="tile p-5 sm:p-6">
        <div className="flex flex-col items-stretch gap-2">
          {layers.map((l, i) => (
            <div key={l.label}>
              <div className="rounded-tile border border-line bg-surface-2/30 p-4">
                <div className={`mono mb-2 text-[0.66rem] uppercase tracking-[0.14em] ${l.tone}`}>{l.label}</div>
                <div className="flex flex-wrap gap-1.5">
                  {l.items.map((it) => (
                    <span key={it} className="mono rounded-md border border-line bg-surface/60 px-2 py-1 text-[0.72rem] text-ink-muted">
                      {it}
                    </span>
                  ))}
                </div>
              </div>
              {i < layers.length - 1 && <ChevronDown className="mx-auto my-1 size-5 text-line-bright" />}
            </div>
          ))}
          <ChevronDown className="mx-auto my-1 size-5 text-accent" />
          <div className="tile--lit rounded-tile p-5 text-center">
            <div className="mono text-[0.66rem] uppercase tracking-[0.14em] text-go">Single source of truth</div>
            <div className="mt-1.5 font-bold text-ink">
              .devrites/ <span className="text-ink-muted">· Markdown workspace on disk</span>
            </div>
          </div>
        </div>
      </div>
    </Reveal>
  );
}

/* ---- 4. The Spec Drift Guard loop ---- */
export function DriftLoop() {
  return (
    <Reveal>
      <div className="tile p-5 sm:p-6">
        <div className="flex flex-col items-center gap-3 sm:flex-row sm:justify-center">
          <Box tone="accent" title="build slice" />
          <Arrow />
          <Box tone="warn" title="drift?" icon={<RotateCcw className="size-3.5" />} />
          <Arrow />
          <Box tone="warn" title="record drift.md" />
          <Arrow />
          <Box tone="accent" title="plan repair" />
        </div>
        <div className="mt-3 flex items-center justify-center gap-2 text-[0.78rem] text-ink-faint">
          <RotateCcw className="size-3.5 text-accent" />
          repaired plan loops back into <span className="mono text-accent">build</span>, so the wrong turn never compounds
        </div>
      </div>
    </Reveal>
  );
}

/* ---- 5. The evidence (proof) ladder ---- */
export function ProofLadder() {
  const rungs = [
    { t: "browser-harness", n: "real browser: screenshots, console, network, viewport" },
    { t: "Chrome DevTools MCP", n: "scripted browser checks" },
    { t: "/run + /verify", n: "Claude Code's built-in runtime checks" },
    { t: "project E2E", n: "Playwright · Cypress · Capybara" },
    { t: "manual fallback", n: "exact steps recorded; seal weighs the risk" },
  ];
  return (
    <Reveal>
      <div className="tile p-5 sm:p-6">
        <div className="mb-3 flex items-center justify-between">
          <span className="mono text-[0.66rem] uppercase tracking-[0.14em] text-go">strongest proof</span>
          <span className="mono text-[0.66rem] uppercase tracking-[0.14em] text-ink-faint">fallback</span>
        </div>
        <div className="space-y-1.5">
          {rungs.map((r, i) => (
            <div
              key={r.t}
              className="flex items-center gap-3 rounded-lg border border-line bg-surface-2/30 px-4 py-2.5"
              style={{ opacity: 1 - i * 0.1 }}
            >
              <span className="mono w-6 shrink-0 text-center text-xs text-ink-faint">{i + 1}</span>
              <span className="mono text-sm font-medium text-ink">{r.t}</span>
              <span className="ml-auto hidden text-[0.8rem] text-ink-muted sm:block">{r.n}</span>
            </div>
          ))}
        </div>
        <p className="mt-3 text-[0.8rem] leading-relaxed text-ink-faint">
          A screenshot path counts as unproven until it is opened and described. DevRites detects what is
          available and degrades down the ladder; it never installs tooling for you.
        </p>
      </div>
    </Reveal>
  );
}

function Box({ tone, title, icon }: { tone: "accent" | "warn" | "go"; title: string; icon?: React.ReactNode }) {
  const map = { accent: "border-accent/50 text-accent", warn: "border-warn/50 text-warn", go: "border-go/50 text-go" };
  return (
    <div className={`mono flex items-center gap-1.5 rounded-lg border ${map[tone]} bg-surface-2/40 px-3 py-2 text-sm`}>
      {icon}
      {title}
    </div>
  );
}

function Arrow() {
  return <ChevronDown className="size-4 shrink-0 rotate-[-90deg] text-line-bright max-sm:rotate-0" />;
}
