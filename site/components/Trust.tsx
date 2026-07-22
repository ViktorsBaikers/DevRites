import { Check, LockKeyhole } from "lucide-react";
import { STATS } from "@/lib/site";

const ACTORS = [
  ["Claude Code", "project-local skills"],
  ["Codex", "project-local skills"],
  ["CI and scripts", "engine commands"],
  ["Humans", "review and release"],
];

const ARTIFACTS = [
  ["spec.md", "scope agreed"],
  ["plan.md", "slice mapped"],
  ["evidence.md", "checks fresh"],
  ["review.md", "independent pass"],
];

const GUARANTEES = [
  "Scope stays explicit",
  "Acceptance stays traceable",
  "Proof survives context resets",
  "Release authority stays human",
];

export default function Trust() {
  return (
    <section
      data-scroll-scene="manual"
      aria-labelledby="alignment-title"
      className="pb-28 pt-20 md:pb-40 md:pt-28"
    >
      <div className="wrap">
        <div className="alignment-heading max-w-5xl">
          <h2
            id="alignment-title"
            className="font-bold text-ink [font-size:clamp(2.8rem,5.4vw,5.6rem)] leading-[0.94] tracking-[-0.04em]"
          >
            Give every agent the same project record.
          </h2>
          <p className="mt-7 max-w-2xl text-lg leading-relaxed text-ink-muted md:text-xl">
            DevRites records scope, checks, and release decisions in your repository. A new context can resume from those files instead of rebuilding the story from chat history.
          </p>
        </div>

        <figure className="alignment-board mt-14" aria-labelledby="alignment-caption">
          <figcaption id="alignment-caption" className="sr-only">
            Claude Code, Codex, automation, and humans read the same DevRites feature files, including the scope, acceptance criteria, evidence, and release decision.
          </figcaption>

          <div className="flex flex-col gap-3 border-b border-line px-5 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-7">
            <div className="flex min-w-0 flex-wrap items-baseline gap-x-3 gap-y-1">
              <span className="mono text-xs text-ink-faint">Shared feature record</span>
              <code className="mono truncate text-sm font-medium text-ink">.devrites/work/&lt;feature&gt;/</code>
            </div>
            <div className="flex items-center gap-4 text-xs text-ink-muted">
              <span>local</span>
              <span>git-diffable</span>
              <span>host-neutral</span>
            </div>
          </div>

          <div className="alignment-map">
            <section aria-labelledby="alignment-actors" className="min-w-0">
              <h3 id="alignment-actors" className="text-xl font-semibold text-ink">Who uses the record</h3>
              <p className="mt-2 text-sm leading-relaxed text-ink-muted">Each tool can take the next step without creating its own workflow.</p>
              <div className="mt-7">
                {ACTORS.map(([name, detail]) => (
                  <div key={name} className="alignment-row">
                    <strong className="text-sm text-ink">{name}</strong>
                    <span className="mono text-[0.68rem] text-ink-faint">{detail}</span>
                  </div>
                ))}
              </div>
            </section>

            <div className="alignment-transfer" aria-hidden><span /></div>

            <div className="alignment-record-stage min-w-0">
              <span className="alignment-record-layer alignment-record-layer--back" aria-hidden />
              <span className="alignment-record-layer alignment-record-layer--front" aria-hidden />
              <section aria-labelledby="alignment-record" className="alignment-record min-w-0">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <p className="mono text-[0.68rem] text-accent-ink/80">What every actor reads</p>
                    <h3 id="alignment-record" className="mt-2 text-2xl font-semibold text-accent-ink">Shared feature files</h3>
                  </div>
                  <LockKeyhole className="size-5 shrink-0 text-accent-ink/75" strokeWidth={1.8} aria-hidden />
                </div>

                <div className="mt-7">
                  {ARTIFACTS.map(([file, state]) => (
                    <div key={file} className="alignment-artifact">
                      <code className="mono text-sm font-medium text-accent-ink">{file}</code>
                      <span className="flex items-center gap-2 text-xs text-accent-ink/75">
                        {state}
                        <Check className="size-3.5" strokeWidth={2.4} aria-hidden />
                      </span>
                    </div>
                  ))}
                  <div className="alignment-artifact">
                    <code className="mono text-sm font-medium text-accent-ink">seal.md</code>
                    <span className="flex items-center gap-2 text-xs text-accent-ink/75">
                      human decision pending
                      <LockKeyhole className="size-3.5" strokeWidth={2} aria-hidden />
                    </span>
                  </div>
                </div>
              </section>
            </div>

            <div className="alignment-transfer alignment-transfer--delay" aria-hidden><span /></div>

            <section aria-labelledby="alignment-guarantees" className="min-w-0">
              <h3 id="alignment-guarantees" className="text-xl font-semibold text-ink">What carries over</h3>
              <p className="mt-2 text-sm leading-relaxed text-ink-muted">Each handoff keeps the agreed boundaries and definition of done.</p>
              <ul className="mt-7">
                {GUARANTEES.map((guarantee) => (
                  <li key={guarantee} className="alignment-guarantee">
                    <Check className="size-4 shrink-0 text-accent" strokeWidth={2.4} aria-hidden />
                    <span className="text-sm font-medium text-ink">{guarantee}</span>
                  </li>
                ))}
              </ul>
            </section>
          </div>

          <div className="border-t border-line px-5 py-4 text-sm text-ink-muted sm:px-7">
            Chat history can disappear. The project files remain.
          </div>
        </figure>

        <dl className="mt-10 grid grid-cols-2 gap-x-8 gap-y-8 border-t border-line pt-8 lg:grid-cols-4">
          {STATS.map((stat) => (
            <div key={stat.label} className="alignment-stat">
              <dt className="mono text-3xl font-semibold leading-none text-ink tabular-nums md:text-4xl">
                {stat.prefix}{stat.value}{stat.suffix}
              </dt>
              <dd className="mt-3 max-w-[16rem] text-sm leading-relaxed text-ink-muted">{stat.label}</dd>
            </div>
          ))}
        </dl>
      </div>
    </section>
  );
}
