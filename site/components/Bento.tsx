"use client";

import { Reveal, SectionHead } from "./ui";
import { MECHANISMS } from "@/lib/site";

const GROUPS = [
  {
    title: "Model judgment, deterministic gates",
    lead: "The model handles engineering judgment. The engine applies the same state, completeness, evidence, and install rules every time.",
    proof: "readiness: blocked, missing acceptance proof",
    keys: ["engine", "drift", "ledger"],
    span: "lg:col-span-4",
    treatment: "bento-feature--signal",
  },
  {
    title: "Project state survives a cleared chat",
    lead: "The next agent reads the git-diffable project files instead of relying on a reconstructed chat summary.",
    proof: ".devrites/work/<feature>/",
    keys: ["harnesses", "paper-trail"],
    span: "lg:col-span-2",
    treatment: "bento-feature--path",
  },
  {
    title: "Reviewers start with the evidence",
    lead: "Fresh-context specialists inspect the diff independently before release. They do not receive the builder's reasoning.",
    proof: "spec, code, tests, security, performance",
    keys: ["fanout", "security", "learn"],
    span: "lg:col-span-2",
    treatment: "bento-feature--review",
  },
  {
    title: "Git changes wait for a human",
    lead: "AFK runs can keep working on their own, but risky boundaries and release actions still require a human decision.",
    proof: "seal: GO, awaiting literal confirmation",
    keys: ["type-go", "afk"],
    span: "lg:col-span-4",
    treatment: "bento-feature--gate",
  },
];

export default function Bento() {
  return (
    <section id="mechanisms" className="wrap py-32 md:py-48">
      <SectionHead
        title="Use firm boundaries where they matter."
        lead="DevRites backs each workflow rule with an engine result, a file in the repository, an independent review, or a human decision."
      />

      <div className="mt-16 grid grid-flow-dense grid-cols-1 gap-4 lg:grid-cols-6">
        {GROUPS.map((group, index) => {
          const mechanisms = group.keys.map((key) => MECHANISMS.find((item) => item.key === key)!);

          return (
            <Reveal key={group.title} delay={index * 0.06} className={group.span}>
              <article className={`bento-feature relative h-full overflow-hidden rounded-card p-7 md:p-9 ${group.treatment}`}>
                <div className="relative">
                  <code className="mono block max-w-full overflow-hidden text-ellipsis whitespace-nowrap text-xs text-accent">
                    {group.proof}
                  </code>
                  <h3 className="mt-8 max-w-xl text-3xl font-semibold leading-[1.02] tracking-[-0.035em] text-ink md:text-4xl">
                    {group.title}
                  </h3>
                  <p className="mt-4 max-w-xl leading-relaxed text-ink-muted">{group.lead}</p>
                </div>

                <div className="relative mt-10 border-t border-line">
                  {mechanisms.map((mechanism) => (
                    <div key={mechanism.key} className="grid gap-2 border-b border-line py-5 last:border-0 md:grid-cols-[0.38fr_0.62fr] md:gap-6">
                      <h4 className="font-semibold text-ink">{mechanism.title}</h4>
                      <p className="text-sm leading-relaxed text-ink-muted">{mechanism.body}</p>
                    </div>
                  ))}
                </div>
              </article>
            </Reveal>
          );
        })}
      </div>
    </section>
  );
}
