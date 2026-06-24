"use client";

import { ArrowRight } from "lucide-react";
import { CopyButton, Reveal, GithubMark, MagneticLink } from "./ui";
import { INSTALL_CMD, REPO } from "@/lib/site";

const POINTS = [
  ["No global writes, ever.", "Installs into the target project; ~/.claude is refused."],
  ["Every file is recorded", "in .claude/devrites.manifest. Uninstall removes exactly those."],
  ["Your work is preserved.", "Feature data in .devrites/work/ is never touched by uninstall."],
  ["Preview first", "with --dry-run. Pin a release with DEVRITES_REF=v2.3.0."],
];

export default function Install() {
  return (
    <section id="install" className="relative overflow-hidden py-24 sm:py-32">
      <div aria-hidden className="pointer-events-none absolute inset-0 -z-10">
        <div className="grid-field absolute inset-0 rotate-180 opacity-50" />
        <div className="absolute left-1/2 top-10 size-[40rem] -translate-x-1/2 rounded-full bg-accent/[0.07] blur-[150px]" />
      </div>

      <div className="wrap text-center">
        <Reveal>
          <h2 className="mx-auto mt-3 max-w-2xl font-bold [font-size:var(--text-h2)]">
            One command. One project. Zero babysitting.
          </h2>
        </Reveal>
        <Reveal delay={0.12}>
          <p className="mx-auto mt-4 max-w-xl text-pretty text-ink-muted [font-size:var(--text-lead)]">
            Free and open source. Read every line before you run it, then feel the difference on
            your first feature.
          </p>
        </Reveal>

        <Reveal delay={0.18}>
          <div className="mx-auto mt-8 flex max-w-xl items-center gap-2 rounded-xl border border-line-bright bg-surface/80 py-3 pl-5 pr-3 backdrop-blur">
            <code className="mono flex-1 text-left text-sm text-ink sm:text-[0.95rem]">
              <span className="text-accent">npx</span> devrites@latest
            </code>
            <CopyButton text={INSTALL_CMD} />
          </div>
        </Reveal>

        <Reveal delay={0.24}>
          <div className="mt-6 flex flex-col items-center justify-center gap-3 sm:flex-row">
            <MagneticLink href={REPO} rel="noopener" className="btn btn-primary group px-6 py-3">
              <GithubMark className="size-4" />
              Star it on GitHub
              <ArrowRight className="size-4 transition-transform duration-200 group-hover:translate-x-1" />
            </MagneticLink>
            <a href="/docs/" className="btn btn-ghost px-6 py-3 font-medium">
              Read the docs
            </a>
          </div>
        </Reveal>

        <Reveal delay={0.3}>
          <p className="mt-6 text-sm text-ink-faint">
            No Node? <code className="k">curl -fsSL https://devrites.com | bash</code> runs the same
            installer. Then run <code className="k k--accent">/rite-spec &lt;feature&gt;</code> to begin.
          </p>
        </Reveal>

        <div className="mx-auto mt-12 grid max-w-3xl gap-3 text-left sm:grid-cols-2">
          {POINTS.map(([h, b], i) => (
            <Reveal key={h} delay={Math.min(i * 0.06, 0.3)}>
              <div className="flex gap-3 rounded-tile border border-line bg-surface/40 p-4">
                <span className="mono mt-0.5 text-accent">→</span>
                <p className="text-sm leading-relaxed text-ink-muted">
                  <b className="text-ink">{h}</b> {b}
                </p>
              </div>
            </Reveal>
          ))}
        </div>
      </div>
    </section>
  );
}
