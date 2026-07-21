import { ArrowRight, FileCheck2, LockKeyhole, RefreshCw, ShieldCheck } from "lucide-react";
import { CopyButton, MagneticLink } from "./ui";
import { INSTALL_CMD } from "@/lib/site";

const RELEASE_INPUTS = [
  { file: "spec.md", label: "Contract", detail: "criteria locked", icon: FileCheck2 },
  { file: "evidence.md", label: "Proof", detail: "current diff", icon: RefreshCw },
  { file: "review.md", label: "Review", detail: "independent", icon: ShieldCheck },
];

export default function Hero() {
  return (
    <section id="top" className="hero-section">
      <div aria-hidden className="hero-ambient" />

      <div className="hero-shell wrap">
        <div className="hero-copy">
          <h1>
            Verify AI code before <span>release.</span>
          </h1>
          <p className="hero-lead">
            DevRites gives Claude Code and Codex a shared release process. Each feature starts
            with a spec, moves through bounded builds and recorded checks, then waits for human approval.
          </p>

          <p className="hero-start">Start in the repository you already have.</p>
          <div className="hero-install-command">
            <code>{INSTALL_CMD}</code>
            <CopyButton text={INSTALL_CMD} label="copy" className="hero-install-copy" />
          </div>
          <div className="hero-actions">
            <MagneticLink href="#install" className="btn btn-primary group px-6 py-3.5">
              Install
              <ArrowRight className="size-4 transition-transform group-hover:translate-x-1" aria-hidden />
            </MagneticLink>
            <a href="#workflow" className="btn btn-ghost group px-5 py-3.5">
              See the workflow
              <ArrowRight className="size-4 transition-transform group-hover:translate-x-1" aria-hidden />
            </a>
          </div>
        </div>

        <aside className="hero-verdict" aria-label="How DevRites assembles a release decision">
          <header className="hero-verdict-head">
            <p>What DevRites checks before GO</p>
            <code>/rite-seal auth-tokens</code>
          </header>

          <div className="hero-verdict-stage">
            <div className="hero-evidence-stack">
              {RELEASE_INPUTS.map(({ file, label, detail, icon: Icon }) => (
                <article key={file} className="hero-evidence-signal">
                  <Icon className="size-5" strokeWidth={1.8} aria-hidden />
                  <div>
                    <span>{label}</span>
                    <code>{file}</code>
                  </div>
                  <strong>{detail}</strong>
                </article>
              ))}
            </div>

            <div className="hero-decision-gate">
              <LockKeyhole className="hero-decision-icon size-5" strokeWidth={1.8} aria-hidden />
              <span>Human decision</span>
              <strong>GO?</strong>
              <p>Git stays locked until approval.</p>
            </div>
          </div>

          <footer className="hero-verdict-foot">
            <code>devrites-engine seal auth-tokens</code>
            <span>release decision based on current evidence</span>
          </footer>
        </aside>
      </div>
    </section>
  );
}
