"use client";

import { useRef } from "react";
import { useReducedMotion } from "framer-motion";
import { Reveal, SectionHead } from "./ui";
import { PHASES } from "@/lib/site";
import { gsap, useGSAP } from "@/lib/gsap";

const ACTS = [
  { key: "shape", label: "shape", span: 4, tone: "text-accent" },
  { key: "build", label: "build", span: 4, tone: "text-warn" },
  { key: "ship", label: "ship", span: 2, tone: "text-go" },
] as const;

// the lit state a node tweens into as the scrubbed rail passes it
const LIT = {
  borderColor: "#3ccfe0",
  backgroundColor: "rgba(60, 207, 224, 0.15)",
  color: "#3ccfe0",
  scale: 1.12,
};

export default function Pipeline() {
  const sectionRef = useRef<HTMLElement>(null);
  const railRef = useRef<HTMLDivElement>(null);
  const reduce = useReducedMotion() ?? false;

  useGSAP(
    () => {
      if (reduce) {
        gsap.set(".pl-fill", { scaleX: 1 });
        gsap.set(".pl-dot", LIT);
        return;
      }
      const dots = gsap.utils.toArray<HTMLElement>(".pl-dot");
      const mm = gsap.matchMedia();

      // tablet/desktop: pin the rail, scrub the fill + light nodes in sequence
      mm.add("(min-width: 768px)", () => {
        const tl = gsap.timeline({
          defaults: { ease: "none" },
          scrollTrigger: {
            trigger: railRef.current,
            start: "center 65%",
            end: "+=85%",
            pin: true,
            scrub: 0.5,
          },
        });
        tl.to(".pl-fill", { scaleX: 1, duration: 1 }, 0);
        dots.forEach((d, i) => {
          tl.to(d, { ...LIT, duration: 0.4, ease: "power2.out" }, (i / dots.length) * 0.9);
        });
      });

      // mobile: same choreography, no pin
      mm.add("(max-width: 767px)", () => {
        const tl = gsap.timeline({
          defaults: { ease: "none" },
          scrollTrigger: {
            trigger: railRef.current,
            start: "top 80%",
            end: "bottom 45%",
            scrub: 0.5,
          },
        });
        tl.to(".pl-fill", { scaleX: 1, duration: 1 }, 0);
        dots.forEach((d, i) => {
          tl.to(d, { ...LIT, duration: 0.4, ease: "power2.out" }, (i / dots.length) * 0.9);
        });
      });
    },
    { dependencies: [reduce], scope: sectionRef },
  );

  return (
    <section ref={sectionRef} id="workflow" className="wrap py-20 sm:py-28">
      <SectionHead
        title="Ten phases. One verified slice at a time."
        lead="Every feature walks the same path, grouped into three acts. Each phase reads the last one's files and writes its own, so nothing important lives only in a chat window you can lose."
      />

      {/* the rail — pins and scrubs as you scroll through it */}
      <div ref={railRef} className="relative mt-14">
        {/* act labels */}
        <div className="mb-3 hidden grid-cols-10 sm:grid">
          {ACTS.map((a) => (
            <div
              key={a.key}
              className={`mono text-xs uppercase tracking-[0.16em] ${a.tone}`}
              style={{ gridColumn: `span ${a.span}` }}
            >
              {a.label}
            </div>
          ))}
        </div>

        {/* track */}
        <div className="relative h-1.5 rounded-full bg-line/60">
          <div
            className="pl-fill blade absolute inset-y-0 left-0 w-full origin-left rounded-full"
            style={{ transform: "scaleX(0)" }}
          />
        </div>

        {/* nodes */}
        <div className="mt-3 grid grid-cols-5 gap-2 sm:grid-cols-10">
          {PHASES.map((p, i) => (
            <Reveal
              key={p.name}
              delay={Math.min(i * 0.04, 0.3)}
              className="flex flex-col items-center gap-1.5 text-center"
            >
              <span className="pl-dot flex size-7 items-center justify-center rounded-full border border-line bg-surface text-[0.65rem] font-bold text-ink-muted">
                {i + 1}
              </span>
              <span className="mono text-[0.65rem] text-ink-faint">{p.name}</span>
            </Reveal>
          ))}
        </div>
      </div>

      {/* phase detail cards */}
      <div className="mt-14 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        {PHASES.map((p, i) => {
          const ship = p.act === "ship";
          return (
            <Reveal
              key={p.name}
              delay={Math.min(i * 0.04, 0.25)}
              className={`tile flex flex-col gap-2.5 p-6 ${ship ? "tile--lit" : ""}`}
            >
              <div className="flex items-center justify-between">
                <span className="mono text-sm font-medium text-accent">
                  {p.n} · {p.name}
                </span>
                <span
                  className={`mono rounded-md border border-line px-1.5 py-0.5 text-[0.62rem] uppercase tracking-wide ${
                    p.act === "shape"
                      ? "text-accent"
                      : p.act === "build"
                        ? "text-warn"
                        : "text-go"
                  }`}
                >
                  {p.act}
                </span>
              </div>
              <h3 className="text-lg font-bold text-ink">{p.title}</h3>
              <p className="text-[0.92rem] leading-relaxed text-ink-muted">{p.body}</p>
              <div className="mono mt-auto flex items-center gap-2 pt-2 text-[0.78rem]">
                <b className="text-ink">{p.cmd}</b>
                <span className="text-ink-faint">→ {p.out}</span>
              </div>
            </Reveal>
          );
        })}
      </div>

      <Reveal delay={0.1}>
        <p className="mt-8 text-pretty text-center text-ink-muted">
          In a hurry?{" "}
          <code className="k k--accent">/rite-autocomplete</code> drives the whole lifecycle
          unattended, pausing only at irreversible gates. Lost?{" "}
          <code className="k">/rite-status</code> shows where the active feature stands. Dropping
          into an existing codebase? <code className="k">/rite-adopt</code> reads it first.
        </p>
      </Reveal>
    </section>
  );
}
