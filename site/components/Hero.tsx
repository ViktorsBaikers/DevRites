"use client";

import { motion, useReducedMotion } from "framer-motion";
import { useEffect, useState } from "react";
import { ArrowRight, BookOpen, Star, Check } from "lucide-react";
import { CopyButton, useGitHubStars, EASE } from "./ui";
import { INSTALL_CMD, REPO } from "@/lib/site";

const PIPE = ["spec", "temper", "define", "vet", "build", "prove", "polish", "review", "seal", "ship"];
const HERE = 4; // building

function useTypewriter(text: string, start: boolean) {
  const reduce = useReducedMotion() ?? false;
  const [out, setOut] = useState(reduce ? text : "");
  useEffect(() => {
    if (reduce || !start) {
      setOut(text);
      return;
    }
    let i = 0;
    const id = setInterval(() => {
      i += 1;
      setOut(text.slice(0, i));
      if (i >= text.length) clearInterval(id);
    }, 55);
    return () => clearInterval(id);
  }, [text, start, reduce]);
  return out;
}

export default function Hero() {
  const reduce = useReducedMotion() ?? false;
  const stars = useGitHubStars();
  const [poweredOn, setPoweredOn] = useState(reduce);
  const typed = useTypewriter("/rite-build", poweredOn);
  const [showOut, setShowOut] = useState(reduce);

  useEffect(() => {
    if (reduce) return;
    const t1 = setTimeout(() => setPoweredOn(true), 650);
    const t2 = setTimeout(() => setShowOut(true), 1850);
    return () => {
      clearTimeout(t1);
      clearTimeout(t2);
    };
  }, [reduce]);

  return (
    <section id="top" className="relative overflow-hidden pt-32 pb-16 sm:pt-40 sm:pb-24">
      {/* backdrop */}
      <div aria-hidden className="pointer-events-none absolute inset-0 -z-10">
        <div className="grid-field absolute inset-0 opacity-60" />
        <div className="float-slow absolute -top-24 left-[12%] size-[34rem] rounded-full bg-accent/15 blur-[120px]" />
        <div className="float-slow absolute -top-10 right-[8%] size-[28rem] rounded-full bg-accent-blue/20 blur-[130px] [animation-delay:-4s]" />
      </div>

      <div className="wrap flex flex-col items-center text-center">
        {/* status chips */}
        <motion.div
          initial={reduce ? false : { opacity: 0, y: 14 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, ease: EASE }}
          className="flex flex-wrap items-center justify-center gap-2"
        >
          <span className="chip chip--run">
            <span className="dot pulse-dot" />
            running
          </span>
          <span className="chip chip--warn">
            <span className="dot" />
            paused
          </span>
          <span className="chip chip--go">
            <Check className="size-3" strokeWidth={3} />
            sealed
          </span>
        </motion.div>

        {/* headline */}
        <motion.h1
          initial={reduce ? false : { opacity: 0, y: 22 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.75, ease: EASE, delay: 0.08 }}
          className="mt-6 max-w-4xl font-black [font-size:var(--text-display)] leading-[0.98]"
        >
          Ship AI code you don&rsquo;t have to{" "}
          <span className="relative whitespace-nowrap">
            babysit
            <span aria-hidden className="blade absolute -bottom-1 left-0 h-[0.14em] w-full rounded-full" />
          </span>
          .
        </motion.h1>

        {/* sub */}
        <motion.p
          initial={reduce ? false : { opacity: 0, y: 18 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.7, ease: EASE, delay: 0.16 }}
          className="mt-6 max-w-2xl text-pretty text-ink-muted [font-size:var(--text-lead)] leading-relaxed"
        >
          DevRites turns Claude Code into a disciplined senior engineer: it asks the right
          questions before writing a line, ships one verified slice at a time, and proves every
          claim with receipts on disk.
        </motion.p>

        {/* install line */}
        <motion.div
          initial={reduce ? false : { opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.7, ease: EASE, delay: 0.24 }}
          className="mt-8 flex items-center gap-2 rounded-xl border border-line bg-surface/70 py-2 pl-4 pr-2 backdrop-blur"
        >
          <code className="mono text-sm text-ink sm:text-[0.95rem]">
            <span className="text-accent">curl</span> -fsSL https://devrites.com | bash
          </code>
          <CopyButton text={INSTALL_CMD} />
        </motion.div>

        {/* CTAs */}
        <motion.div
          initial={reduce ? false : { opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.7, ease: EASE, delay: 0.32 }}
          className="mt-7 flex flex-col items-center gap-3 sm:flex-row"
        >
          <a
            href="#install"
            className="blade group inline-flex items-center gap-2 rounded-xl px-6 py-3 font-semibold text-bg-deep shadow-lg shadow-accent/20 transition-transform duration-200 hover:scale-[1.03]"
          >
            Install in one command
            <ArrowRight className="size-4 transition-transform duration-200 group-hover:translate-x-1" />
          </a>
          <a
            href="#workflow"
            className="inline-flex items-center gap-2 rounded-xl border border-line-bright px-6 py-3 font-medium text-ink transition-colors duration-200 hover:bg-surface-2/60"
          >
            <BookOpen className="size-4" />
            See how it works
          </a>
        </motion.div>

        {/* trust line */}
        <motion.div
          initial={reduce ? false : { opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.8, ease: EASE, delay: 0.5 }}
          className="mt-6 flex items-center gap-2 text-sm text-ink-faint"
        >
          <a href={REPO} rel="noopener" className="inline-flex items-center gap-1.5 transition-colors hover:text-ink">
            <Star className="size-4 text-warn" fill="currentColor" />
            {stars !== null ? <b className="text-ink">{stars.toLocaleString()}</b> : "—"}{" "}
            {stars === 1 ? "star" : "stars"}
          </a>
          <span aria-hidden>·</span>
          <span>Free &amp; open source (MIT)</span>
          <span aria-hidden>·</span>
          <span>
            never touches <span className="k">~/.claude</span>
          </span>
        </motion.div>

        {/* the console product visual */}
        <motion.div
          initial={reduce ? false : { opacity: 0, y: 40, scale: 0.97 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          transition={{ duration: 0.9, ease: EASE, delay: 0.4 }}
          className="glass glow-accent relative mt-14 w-full max-w-4xl overflow-hidden rounded-2xl text-left"
        >
          {/* title bar */}
          <div className="flex items-center gap-2 border-b border-line bg-surface-2/40 px-4 py-3">
            <span className="flex gap-1.5">
              <i className="size-3 rounded-full bg-danger/70" />
              <i className="size-3 rounded-full bg-warn/70" />
              <i className="size-3 rounded-full bg-go/70" />
            </span>
            <span className="mono ml-2 text-xs text-ink-muted">
              <b className="text-ink">auth-tokens</b> · .devrites/work
            </span>
            <span className="chip chip--run ml-auto">
              <span className="dot pulse-dot" />
              build 3/5
            </span>
          </div>

          <div className="p-5 sm:p-7">
            {/* pipeline rail */}
            <div className="flex items-center gap-1.5 overflow-x-auto pb-2" aria-hidden>
              {PIPE.map((p, i) => {
                const done = i < HERE;
                const here = i === HERE;
                return (
                  <div key={p} className="flex items-center gap-1.5">
                    <motion.div
                      initial={reduce ? false : { opacity: 0, scale: 0.6 }}
                      animate={{ opacity: 1, scale: 1 }}
                      transition={{ duration: 0.4, ease: EASE, delay: 0.6 + i * 0.07 }}
                      className={`flex shrink-0 flex-col items-center gap-1 ${
                        here ? "text-accent" : done ? "text-go" : "text-ink-faint"
                      }`}
                    >
                      <span
                        className={`flex size-6 items-center justify-center rounded-full border text-[0.6rem] font-bold ${
                          here
                            ? "border-accent bg-accent/15"
                            : done
                              ? "border-go/50 bg-go/10"
                              : "border-line"
                        }`}
                      >
                        {done ? "✓" : i + 1}
                      </span>
                      <span className="mono text-[0.6rem]">{p}</span>
                    </motion.div>
                    {i < PIPE.length - 1 && (
                      <span className={`h-px w-4 shrink-0 ${i < HERE ? "bg-go/50" : "bg-line"}`} />
                    )}
                  </div>
                );
              })}
            </div>

            {/* terminal */}
            <div className="mt-5 rounded-xl border border-line bg-bg-deep/60 p-4 font-mono text-sm">
              <div className="flex items-center">
                <span className="text-accent">›</span>
                <span className="ml-2 text-ink">{typed}</span>
                {!showOut && <span className="caret ml-0.5 h-[1.1em]">&nbsp;</span>}
              </div>
              {showOut && (
                <motion.div
                  initial={reduce ? false : { opacity: 0, y: 6 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.5, ease: EASE }}
                  className="mt-3 space-y-2 text-ink-muted"
                >
                  <div className="text-ink-faint">slice 3/5 · refresh-token rotation</div>
                  <div className="flex flex-wrap gap-x-5 gap-y-1">
                    <span className="text-go">✓ 14 tests green</span>
                    <span className="text-go">✓ types</span>
                    <span className="text-go">✓ build</span>
                  </div>
                  <div className="text-ink-faint">
                    → evidence.md, touched-files.md written. <span className="text-warn">stopped.</span>
                  </div>
                </motion.div>
              )}
            </div>
          </div>
        </motion.div>
      </div>
    </section>
  );
}
