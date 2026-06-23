"use client";

import { CountUp, Reveal, Stagger, staggerChild, useGitHubStars } from "./ui";
import { motion, useReducedMotion } from "framer-motion";
import { STATS, TOOLS } from "@/lib/site";

export default function Trust() {
  const reduce = useReducedMotion() ?? false;
  const stars = useGitHubStars();

  return (
    <section className="wrap py-8 sm:py-12">
      <Reveal>
        <p className="text-center text-sm text-ink-faint">
          {stars !== null ? (
            <>
              Trusted by engineers tired of plausible-but-wrong code. Join{" "}
              <b className="text-ink">{stars.toLocaleString()}</b> of them on GitHub
            </>
          ) : (
            "Trusted by engineers tired of plausible-but-wrong code"
          )}
        </p>
      </Reveal>

      <Stagger className="mt-8 grid grid-cols-2 gap-4 lg:grid-cols-4">
        {STATS.map((s) => (
          <motion.div
            key={s.label}
            variants={reduce ? undefined : staggerChild}
            className="tile flex flex-col gap-2 p-5"
          >
            <div className="font-black text-ink [font-size:clamp(2rem,4vw,2.75rem)] leading-none">
              {s.prefix}
              <CountUp value={s.value} suffix={s.suffix} />
            </div>
            <p className="text-sm text-ink-muted leading-snug">{s.label}</p>
          </motion.div>
        ))}
      </Stagger>

      <Reveal delay={0.1}>
        <div className="mt-10 flex flex-col items-center gap-4">
          <span className="mono text-xs uppercase tracking-[0.16em] text-ink-faint">
            same files, same gates, driven from
          </span>
          <div className="flex flex-wrap items-center justify-center gap-x-3 gap-y-2">
            {TOOLS.map((t) => (
              <span
                key={t}
                className="rounded-full border border-line bg-surface/50 px-3.5 py-1.5 text-sm text-ink-muted"
              >
                {t}
              </span>
            ))}
          </div>
        </div>
      </Reveal>
    </section>
  );
}
