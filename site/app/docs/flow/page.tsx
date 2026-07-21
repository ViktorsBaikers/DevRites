import type { Metadata } from "next";
import { Check } from "lucide-react";
import { Reveal } from "@/components/ui";
import DocsHeader from "@/components/docs/DocsHeader";
import { PipelineDiagram, DriftLoop } from "@/components/docs/Diagrams";
import { PHASES } from "@/lib/site";

export const metadata: Metadata = {
  title: "Flow",
  description: "The feature lifecycle, the Spec Drift Guard loop, and the two run modes.",
  alternates: { canonical: "/docs/flow/" },
};

export default function Flow() {
  return (
    <>
      <DocsHeader
        crumb="flow"
        title="Flow"
        lead="A feature normally moves through spec, define, vet, build, prove, polish, review, seal, and ship. Temper adds an optional strategy review. Plan repair and converge recover work that no longer matches the plan."
      />

      <Reveal>
        <h2 id="lifecycle" className="scroll-mt-28 text-2xl font-bold">Feature lifecycle</h2>
      </Reveal>
      <div className="mt-6">
        <PipelineDiagram />
      </div>
      <ol className="mt-6 space-y-3">
        {PHASES.map((p, i) => (
          <Reveal as="li" key={p.name} delay={Math.min(i * 0.03, 0.2)}>
            <div className="tile flex gap-4 p-5">
              <span className="mono flex size-8 shrink-0 items-center justify-center rounded-full border border-line bg-surface-2/60 text-sm font-bold text-accent">
                {i + 1}
              </span>
              <div className="min-w-0">
                <div className="flex flex-wrap items-baseline gap-x-3">
                  <h3 className="font-bold capitalize text-ink">{p.name}</h3>
                  <span className="mono text-xs text-ink-faint">
                    {p.cmd}{p.optional ? " (optional)" : ""} → {p.out}
                  </span>
                </div>
                <p className="mt-1.5 text-[0.9rem] leading-relaxed text-ink-muted">{p.body}</p>
              </div>
            </div>
          </Reveal>
        ))}
      </ol>

      <Reveal delay={0.1}>
        <div className="tile mt-5 p-5">
          <h3 className="font-bold text-ink">Recorded recovery steps</h3>
          <p className="mt-2 text-[0.9rem] leading-relaxed text-ink-muted">
            <code className="k">/rite-plan repair</code> corrects a plan after recorded drift.{" "}
            <code className="k">/rite-converge</code> compares live code with the recorded spec and plan, then
            appends only the remaining work as new <code className="k">SLICE-###</code> entries.
          </p>
        </div>
      </Reveal>

      <Reveal>
        <h2 id="drift" className="mt-14 scroll-mt-28 text-2xl font-bold">Spec Drift Guard</h2>
      </Reveal>
      <Reveal delay={0.05}>
        <p className="mt-3 text-ink-muted leading-relaxed">
          If implementation shows that the plan is wrong, the build stops and records the mismatch.
          DevRites asks for a decision when product behavior would change, then repairs the plan before
          resuming.
        </p>
      </Reveal>
      <Reveal delay={0.1}>
        <pre className="mono mt-5 overflow-x-auto rounded-tile border border-line bg-bg-deep/60 p-4 text-[0.82rem] leading-relaxed">
          <span className="text-ink">/rite-build</span>{"  "}<span className="text-ink-faint"># slice 3/5</span>{"\n"}
          <span className="text-warn">⚠ drift</span>
          <span className="text-ink-muted"> the auth model assumes sessions; spec needs stateless tokens</span>{"\n"}
          <span className="text-ink-faint">  → recorded to drift.md, build paused</span>{"\n"}
          <span className="text-ink">/rite-plan repair</span>{"  "}<span className="text-ink-faint"># reslice around the corrected model</span>{"\n"}
          <span className="text-ink">/rite-build</span>{"  "}<span className="text-ink-faint"># resume from the repaired plan</span>
        </pre>
      </Reveal>
      <div className="mt-5">
        <DriftLoop />
      </div>
      <Reveal delay={0.12}>
        <div className="tile--lit mt-5 rounded-tile p-5">
          <h3 className="font-bold text-ink">Why repair happens here</h3>
          <p className="mt-2 text-[0.9rem] leading-relaxed text-ink-muted">
            Fixing a bad assumption at slice 3 requires one plan repair. Finding it at seal can require
            reworking the whole feature, so the build stops as soon as it detects the mismatch.
          </p>
        </div>
      </Reveal>

      <Reveal>
        <h2 id="run-modes" className="mt-14 scroll-mt-28 text-2xl font-bold">Run modes</h2>
      </Reveal>
      <Reveal delay={0.05}>
        <p className="mt-3 text-ink-muted leading-relaxed">
          HITL and AFK use the same workflow with different pause rules. Each slice records its mode,
          so you control how much work can run unattended.
        </p>
      </Reveal>
      <div className="mt-6 grid gap-4 md:grid-cols-2">
        <Reveal>
          <div className="tile flex h-full flex-col gap-3 p-6">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-bold text-ink">HITL</h3>
              <span className="chip chip--warn">
                <span className="dot" />
                human in the loop
              </span>
            </div>
            <p className="text-[0.9rem] leading-relaxed text-ink-muted">
              Default. Risky slices pause before code is written, at a typed checkpoint that escalates
              through advisory, validating, blocking, and escalating. Answer once with{" "}
              <code className="k">/rite-resolve &lt;qid&gt; &quot;&lt;answer&gt;&quot;</code> and the build resumes.
            </p>
          </div>
        </Reveal>
        <Reveal delay={0.08}>
          <div className="tile flex h-full flex-col gap-3 p-6">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-bold text-ink">AFK</h3>
              <span className="chip chip--run">
                <span className="dot" />
                away from keyboard
              </span>
            </div>
            <p className="text-[0.9rem] leading-relaxed text-ink-muted">
              Drop a <code className="k">.devrites/AFK</code> file in the project. AFK slices run
              unattended; discretionary pauses downgrade to advisory notes in{" "}
              <code className="k">questions.md</code> so the loop keeps moving.
            </p>
          </div>
        </Reveal>
      </div>
      <Reveal delay={0.1}>
        <div className="mt-4 flex gap-3 rounded-tile border border-danger/40 bg-danger/5 p-5">
          <Check className="mt-0.5 size-5 shrink-0 text-danger" strokeWidth={2.4} />
          <p className="text-[0.9rem] leading-relaxed text-ink-muted">
            <b className="text-ink">Some changes always require a pause.</b> This includes destructive
            migrations, auth and authz changes, public API breaks, and failing tests, types, or lint.
            The optional <code className="k">max_slices</code> setting caps the run, and{" "}
            <code className="k">notify</code> can send a message when it pauses.
          </p>
        </div>
      </Reveal>
    </>
  );
}
