"use client";

import { useState } from "react";
import { ArrowRight } from "lucide-react";
import { Reveal } from "./ui";
import { FAQ } from "@/lib/site";

export default function Faq() {
  const [active, setActive] = useState(0);
  const item = FAQ[active];

  return (
    <section id="faq" className="wrap py-32 md:py-48">
      <div className="grid gap-14 lg:grid-cols-[0.72fr_1.28fr] lg:gap-24">
        <div>
          <h2 className="font-bold text-ink [font-size:clamp(2.8rem,5vw,5.4rem)] leading-[0.94] tracking-[-0.04em]">
            See what DevRites installs and changes.
          </h2>
          <p className="mt-6 max-w-xl text-ink-muted [font-size:var(--text-lead)] leading-relaxed">
            Project files stay local and inspectable, and the source is available on GitHub.
          </p>

          <Reveal delay={0.08}>
            <div className="mt-10 border-y border-line" aria-label="Frequently asked questions">
              {FAQ.map((question, index) => {
                const selected = active === index;
                return (
                  <button
                    key={question.q}
                    type="button"
                    aria-pressed={selected}
                    aria-controls="faq-answer"
                    onClick={() => setActive(index)}
                    className={`group grid w-full cursor-pointer grid-cols-[1fr_auto] items-center gap-4 border-b border-line py-4 text-left transition-colors duration-300 last:border-0 ${
                      selected ? "text-ink" : "text-ink-muted hover:text-ink"
                    }`}
                  >
                    <span className="font-medium leading-snug">{question.q}</span>
                    <ArrowRight
                      className={`size-4 transition-transform ${selected ? "translate-x-1 text-accent" : "text-ink-faint group-hover:translate-x-1"}`}
                      aria-hidden
                    />
                  </button>
                );
              })}
            </div>
          </Reveal>
        </div>

        <Reveal delay={0.12} className="lg:pt-24">
          <div className="min-h-[25rem] border-t border-line pt-8 md:pt-10">
            <article id="faq-answer" key={item.q} className="faq-answer" aria-live="polite">
              <h3 className="max-w-2xl text-3xl font-semibold leading-tight text-ink md:text-5xl">
                {item.q}
              </h3>
              <p className="mt-8 max-w-2xl text-lg leading-[1.75] text-ink-muted">{item.a}</p>
            </article>
          </div>
        </Reveal>
      </div>
    </section>
  );
}
