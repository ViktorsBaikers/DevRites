"use client";

import { ArrowRight } from "lucide-react";
import { CopyButton, Reveal, MagneticLink } from "./ui";
import { CURL_CMD, INSTALL_CMD, REPO } from "@/lib/site";

const POINTS = [
  ["Installed in the project", "Skills, agents, standards, and hooks stay in the target repository."],
  ["Tracked by a manifest", "Install, update, and uninstall use the same list of managed host files."],
  ["Feature work stays", "Uninstall leaves the work under .devrites/work/ in place."],
  ["Preview before installing", "Use --dry-run to preview the changes, or --no-binary to skip the shared executable."],
];

export default function Install() {
  return (
    <section id="install" className="wrap py-32 md:py-48">
      <Reveal>
        <div className="install-stage overflow-hidden rounded-card p-7 md:p-12 lg:p-16">
          <div className="grid gap-16 lg:grid-cols-[1.08fr_0.92fr] lg:items-end">
            <div>
              <h2 className="max-w-4xl font-bold [font-size:clamp(3rem,5.8vw,5.8rem)] leading-[0.9] tracking-[-0.04em]">
                Use DevRites for your next feature.
              </h2>
              <p className="mt-7 max-w-xl text-lg leading-relaxed opacity-80">
                DevRites is free and open source. It gives Claude Code and Codex the same project-local workflow.
              </p>

              <div className="mt-9 flex max-w-xl items-center gap-2 rounded-xl bg-bg-deep py-3 pl-5 pr-3 text-ink">
                <code className="mono min-w-0 flex-1 truncate text-left text-sm sm:text-base">
                  <span className="text-accent">npx</span> devrites@latest
                </code>
                <CopyButton text={INSTALL_CMD} label="copy" className="border-line-bright bg-surface text-ink" />
              </div>

              <div className="mt-6 flex flex-col gap-3 sm:flex-row">
                <MagneticLink href={REPO} rel="noopener" className="btn btn-dark group px-6 py-3">
                  GitHub
                  <ArrowRight className="size-4 transition-transform duration-300 group-hover:translate-x-1" aria-hidden />
                </MagneticLink>
                <a href="/docs/getting-started/" className="btn px-5 py-3 text-accent-ink hover:bg-bg-deep/10">
                  Installation guide
                </a>
              </div>
            </div>

            <div>
              <h3 className="text-xl font-semibold">What installation changes</h3>
              <div className="mt-5 border-t border-current/25">
                {POINTS.map(([title, body]) => (
                  <div key={title} className="grid gap-1 border-b border-current/20 py-4 sm:grid-cols-[0.4fr_0.6fr] sm:gap-5">
                    <h4 className="font-semibold">{title}</h4>
                    <p className="text-sm leading-relaxed opacity-80">{body}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <p className="mt-12 border-t border-current/25 pt-5 text-sm opacity-80">
            If you do not use Node, run <code className="break-all font-mono">{CURL_CMD}</code>
          </p>
        </div>
      </Reveal>
    </section>
  );
}
