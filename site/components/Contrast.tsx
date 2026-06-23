"use client";

import { Reveal, SectionHead } from "./ui";
import { X, Check, ArrowRight } from "lucide-react";

const BEFORE = [
  "Writes plausible code, then calls it done.",
  "“Done” with no tests run and a half-finished migration.",
  "Loses the thread the moment you /clear.",
  "Quietly changes auth or drops a column.",
  "Hands you one giant diff to review.",
];

const AFTER = [
  "Asks the right questions before writing a line.",
  "Proof on disk: tests, build, browser evidence.",
  "Resumes cold from .devrites/ after a clear.",
  "Stops at a typed gate before anything irreversible.",
  "Ships one verified slice at a time.",
];

export default function Contrast() {
  return (
    <section id="why" className="wrap py-20 sm:py-28">
      <SectionHead
        center
        eyebrow="Plausible is not correct"
        title="The difference shows up on the first feature."
        lead="A wrong turn three steps back costs you a day. DevRites makes the agent work like a careful engineer: one slice at a time, with decisions recorded and receipts on disk."
      />

      <div className="mt-12 grid items-stretch gap-4 md:grid-cols-[1fr_auto_1fr]">
        <Reveal className="rounded-card border border-line bg-surface/40 p-6 sm:p-7">
          <span className="chip chip--danger">
            <span className="dot" />
            your agent today
          </span>
          <ul className="mt-5 space-y-3.5">
            {BEFORE.map((t) => (
              <li key={t} className="flex gap-3 text-ink-muted">
                <X className="mt-0.5 size-4 shrink-0 text-danger" strokeWidth={2.4} />
                <span className="[&_:where(code)]:font-mono">{render(t)}</span>
              </li>
            ))}
          </ul>
        </Reveal>

        <div className="flex items-center justify-center" aria-hidden>
          <div className="flex size-11 items-center justify-center rounded-full border border-line-bright bg-surface text-accent">
            <ArrowRight className="size-5 max-md:rotate-90" />
          </div>
        </div>

        <Reveal delay={0.12} className="tile--lit rounded-card p-6 sm:p-7">
          <span className="chip chip--go">
            <Check className="size-3" strokeWidth={3} />
            with DevRites
          </span>
          <ul className="mt-5 space-y-3.5">
            {AFTER.map((t) => (
              <li key={t} className="flex gap-3 text-ink">
                <Check className="mt-0.5 size-4 shrink-0 text-go" strokeWidth={2.6} />
                <span>{render(t)}</span>
              </li>
            ))}
          </ul>
        </Reveal>
      </div>
    </section>
  );
}

/* render inline /commands and .files in mono */
function render(text: string) {
  return text.split(/(\/clear|\.devrites\/|auth)/g).map((part, i) =>
    part === "/clear" || part === ".devrites/" ? (
      <code key={i} className="k">
        {part}
      </code>
    ) : (
      <span key={i}>{part}</span>
    ),
  );
}
