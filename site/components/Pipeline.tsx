import { ArrowRight, FolderGit2, LockKeyhole, ShieldCheck } from "lucide-react";
import { PHASES } from "@/lib/site";

const STAGE_DEFS = [
  {
    key: "shape",
    label: "Shape",
    title: "Agree on the contract.",
    body: "Scope, acceptance criteria, and the build plan are agreed before implementation starts.",
    names: ["spec", "define", "vet"],
    gate: "Implementation starts after vet passes.",
  },
  {
    key: "build",
    label: "Build",
    title: "Build one slice.",
    body: "A fresh-context writer implements one bounded slice, runs focused tests, records its work, then stops.",
    names: ["build"],
    gate: "Unclaimed source writes are rejected.",
  },
  {
    key: "prove",
    label: "Prove",
    title: "Update the evidence.",
    body: "Tests, runtime checks, browser evidence, polish, and independent review are tied to the current diff.",
    names: ["prove", "polish", "review"],
    gate: "A code change expires old evidence.",
  },
  {
    key: "release",
    label: "Release",
    title: "Record the release decision.",
    body: "DevRites seals the evidence first. Commit, push, and tag remain locked until a human types GO.",
    names: ["seal", "ship"],
    gate: "A missing criterion means NO-GO.",
  },
].map((stage) => ({
  ...stage,
  phases: stage.names.map((name) => PHASES.find((phase) => phase.name === name)!).filter(Boolean),
}));

export default function Pipeline() {
  return (
    <section id="workflow" className="release-process py-28 md:py-40" aria-labelledby="workflow-title">
      <div className="wrap">
        <header className="release-heading">
          <h2
            id="workflow-title"
            className="font-bold text-ink [font-size:clamp(3rem,5.4vw,5.4rem)] leading-[0.92] tracking-[-0.04em]"
          >
            Keep the full release record with the feature.
          </h2>
          <p className="mt-7 max-w-2xl text-ink-muted [font-size:var(--text-lead)] leading-relaxed">
            One project-local folder holds the agreed scope, implementation record, checks, review, and release decision.
          </p>
        </header>

        <div className="release-conveyor mt-12">
          <div className="release-conveyor-head">
            <div className="release-feature">
              <FolderGit2 className="size-6" strokeWidth={1.7} aria-hidden />
              <span>Feature workspace</span>
              <strong>auth-tokens</strong>
            </div>
            <div className="release-path">
              <span>Stored in the repository</span>
              <code>.devrites/work/auth-tokens/</code>
            </div>
          </div>

          <div className="release-route" aria-hidden="true">
            <span className="release-route-line" />
            {STAGE_DEFS.map((stage) => (
              <span key={stage.key} className="release-route-node" />
            ))}
          </div>

          <ol className="release-panels">
            {STAGE_DEFS.map((stage) => (
              <li key={stage.key} className={`release-panel release-panel--${stage.key}`}>
                <div className="release-panel-copy">
                  <span>{stage.label}</span>
                  <h3>{stage.title}</h3>
                  <p>{stage.body}</p>
                  <div className="release-panel-commands" aria-label={`${stage.label} commands`}>
                    {stage.phases.map((phase) => <code key={phase.name}>{phase.cmd}</code>)}
                  </div>
                </div>

                <div className="release-panel-writes">
                  <span>Written to disk</span>
                  <div>
                    {stage.phases.map((phase) => <code key={phase.name}>{phase.out}</code>)}
                  </div>
                </div>

                <p className="release-panel-gate">
                  <LockKeyhole className="size-3.5" strokeWidth={2} aria-hidden />
                  {stage.gate}
                </p>
              </li>
            ))}
          </ol>

          <div className="release-conveyor-foot">
            <p>
              <ShieldCheck className="size-5" strokeWidth={1.8} aria-hidden />
              <span>Release record is complete</span>
              <strong>A human still has to approve it.</strong>
            </p>
            <a href="/docs/flow/">Read the full flow <ArrowRight className="size-4" aria-hidden /></a>
          </div>
        </div>
      </div>
    </section>
  );
}
