"use client";

import { CountUp, Reveal, Stagger, staggerChild, useGitHubStars } from "./ui";
import { motion, useReducedMotion } from "framer-motion";
import { STATS } from "@/lib/site";

export default function Trust() {
  const reduce = useReducedMotion() ?? false;
  const stars = useGitHubStars();

  return (
    <section className="wrap py-10 sm:py-14">
      <Reveal>
        <p className="max-w-2xl text-sm leading-relaxed text-ink-faint">
          {stars !== null ? (
            <>
              Trusted by engineers tired of plausible-but-wrong code. Join{" "}
              <b className="text-ink">{stars.toLocaleString()}</b> of them on GitHub.
            </>
          ) : (
            "Trusted by engineers tired of plausible-but-wrong code."
          )}
        </p>
      </Reveal>

      {/* one panel, split by hairlines — telemetry, not a stat-card grid */}
      <Stagger className="mt-7 grid grid-cols-2 gap-px overflow-hidden rounded-card border border-line bg-line/60 sm:grid-cols-4">
        {STATS.map((s) => (
          <motion.div
            key={s.label}
            variants={reduce ? undefined : staggerChild}
            className="flex flex-col gap-2 bg-bg p-5 sm:p-6"
          >
            <div className="mono text-ink [font-size:clamp(1.6rem,3vw,2.15rem)] font-semibold leading-none tabular-nums">
              {s.prefix}
              <CountUp value={s.value} suffix={s.suffix} />
            </div>
            <p className="text-[0.8rem] leading-snug text-ink-muted">{s.label}</p>
          </motion.div>
        ))}
      </Stagger>
    </section>
  );
}
