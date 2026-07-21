import Image from "next/image";
import { REPO } from "@/lib/site";

const PRIMARY_LINKS = [
  ["Workflow", "#workflow"],
  ["Mechanisms", "#mechanisms"],
  ["Hosts", "#anywhere"],
  ["Docs", "/docs/"],
  ["Install", "#install"],
];

const SOURCE_LINKS = [
  ["GitHub", REPO],
  ["Releases", `${REPO}/releases`],
  ["MIT license", `${REPO}/blob/main/LICENSE`],
];

export default function Footer() {
  return (
    <footer className="border-t border-line bg-bg-deep/80">
      <div className="wrap py-20 md:py-28">
        <p className="max-w-5xl font-semibold text-ink [font-size:clamp(2.5rem,5vw,5.5rem)] leading-[0.95] tracking-[-0.04em]">
          AI can write the diff. You decide whether it ships.
        </p>

        <div className="mt-20 grid gap-12 border-t border-line pt-10 lg:grid-cols-[0.85fr_1.15fr] lg:items-end">
          <div>
            <a href="#top" className="inline-flex items-center gap-2.5">
              <Image src="/assets/img/mark-64.png" width={28} height={28} alt="" />
              <b className="text-lg">DevRites</b>
            </a>
            <p className="mt-4 max-w-md text-sm leading-relaxed text-ink-muted">
              A shared engineering workflow for Claude Code and Codex, with recorded checks, independent review, and human approval before release.
            </p>
          </div>

          <nav aria-label="Footer" className="flex flex-wrap gap-x-7 gap-y-4 lg:justify-end">
            {PRIMARY_LINKS.map(([label, href]) => (
              <a
                key={label}
                href={href}
                className="text-sm font-medium text-ink-muted transition-colors duration-200 hover:text-ink"
              >
                {label}
              </a>
            ))}
          </nav>
        </div>

        <div className="mt-12 flex flex-col items-start justify-between gap-5 border-t border-line pt-6 text-sm text-ink-faint md:flex-row md:items-center">
          <span>© 2026 Viktors Baikers</span>
          <div className="flex flex-wrap gap-x-5 gap-y-2">
            {SOURCE_LINKS.map(([label, href]) => (
              <a key={label} href={href} rel="noopener" className="transition-colors duration-200 hover:text-ink">
                {label}
              </a>
            ))}
          </div>
        </div>
      </div>
    </footer>
  );
}
