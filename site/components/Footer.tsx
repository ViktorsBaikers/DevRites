import Image from "next/image";
import { REPO, VERSION } from "@/lib/site";

const COLS = [
  {
    h: "Product",
    links: [
      ["Workflow", "#workflow"],
      ["Mechanisms", "#mechanisms"],
      ["Drive anywhere", "#anywhere"],
      ["FAQ", "#faq"],
      ["Install", "#install"],
    ],
  },
  {
    h: "Docs",
    links: [
      ["Overview", "/docs/"],
      ["Command map", "/docs/command-map/"],
      ["Flow", "/docs/flow/"],
      ["Architecture", "/docs/architecture/"],
    ],
  },
  {
    h: "Source",
    links: [
      ["GitHub", REPO],
      ["Releases", `${REPO}/releases`],
      ["License (MIT)", `${REPO}/blob/main/LICENSE`],
    ],
  },
];

export default function Footer() {
  return (
    <footer className="border-t border-line bg-bg-deep">
      <div className="wrap py-14">
        <div className="grid gap-10 md:grid-cols-[1.4fr_2fr]">
          <div>
            <a href="#top" className="flex items-center gap-2.5">
              <Image src="/assets/img/mark-64.png" width={28} height={28} alt="" />
              <b className="text-lg">DevRites</b>
            </a>
            <p className="mt-4 max-w-sm text-sm leading-relaxed text-ink-muted">
              A disciplined senior-engineer workflow for Claude Code. Spec, build, prove, seal,
              ship, with the receipts on disk.
            </p>
          </div>
          <div className="grid grid-cols-2 gap-8 sm:grid-cols-3">
            {COLS.map((c) => (
              <div key={c.h}>
                <h3 className="mono text-xs uppercase tracking-[0.14em] text-ink-faint">{c.h}</h3>
                <ul className="mt-3 space-y-2.5">
                  {c.links.map(([label, href]) => (
                    <li key={label}>
                      <a
                        href={href}
                        rel={href.startsWith("http") ? "noopener" : undefined}
                        className="text-sm text-ink-muted transition-colors duration-200 hover:text-ink"
                      >
                        {label}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>

        <div className="mt-12 flex flex-col items-center justify-between gap-3 border-t border-line pt-6 text-sm text-ink-faint sm:flex-row">
          <span>© 2026 Viktors Baikers · DevRites v{VERSION}</span>
          <span className="mono">spec → build → prove → seal → ship</span>
        </div>
      </div>
    </footer>
  );
}
