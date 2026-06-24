"use client";

import { useRef } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { ArrowRight, BookOpen, Star, Check } from "lucide-react";
import { CopyButton, MagneticLink, useGitHubStars, EASE } from "./ui";
import { INSTALL_CMD, REPO } from "@/lib/site";
import { gsap, SplitText, useGSAP } from "@/lib/gsap";

export default function Hero() {
  const reduce = useReducedMotion() ?? false;
  const stars = useGitHubStars();
  const sectionRef = useRef<HTMLElement>(null);
  const headlineRef = useRef<HTMLHeadingElement>(null);
  const ulRef = useRef<HTMLSpanElement>(null);
  // GSAP owns only the one-time headline reveal. No scroll-pin, no scrub: the
  // brand reads calmer without a hijack. Pulse, pipeline scrollytelling, and the
  // console video carry the motion the page actually needs.
  useGSAP(
    () => {
      if (reduce) return;
      const split = SplitText.create(headlineRef.current, { type: "words" });
      gsap.from(split.words, {
        yPercent: 115,
        opacity: 0,
        duration: 0.7,
        ease: "expo.out",
        stagger: 0.045,
        delay: 0.1,
      });
      gsap.from(ulRef.current, {
        scaleX: 0,
        transformOrigin: "left center",
        duration: 0.6,
        ease: "power2.inOut",
        delay: 0.55,
      });
      return () => split.revert();
    },
    { dependencies: [reduce], scope: sectionRef },
  );

  return (
    <section
      ref={sectionRef}
      id="top"
      className="relative overflow-hidden pt-28 pb-16 sm:pt-32 sm:pb-24"
    >
      {/* backdrop */}
      <div aria-hidden className="pointer-events-none absolute inset-0 -z-10">
        <div className="grid-field absolute inset-0 opacity-60" />
        <div className="float-slow absolute -top-24 left-[6%] size-[34rem] rounded-full bg-accent/10 blur-[130px]" />
        <div className="float-slow absolute top-[18%] right-[2%] size-[28rem] rounded-full bg-accent-blue/12 blur-[140px] [animation-delay:-4s]" />
      </div>

      <div className="wrap grid items-center gap-x-12 gap-y-14 lg:grid-cols-[1.12fr_0.88fr]">
        {/* left rail — the pitch */}
        <div className="flex flex-col items-start text-left">
          {/* status chips */}
          <motion.div
            initial={reduce ? false : { opacity: 0, y: 14 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, ease: EASE }}
            className="flex flex-wrap items-center gap-2"
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
          <h1
            ref={headlineRef}
            className="mt-6 font-bold [font-size:var(--text-display)] leading-[1.02]"
          >
            Ship AI code you don&rsquo;t have to{" "}
            <span className="relative whitespace-nowrap">
              babysit
              <span
                ref={ulRef}
                aria-hidden
                className="blade absolute -bottom-1 left-0 h-[0.14em] w-full rounded-full"
              />
            </span>
            .
          </h1>

          {/* sub */}
          <motion.p
            initial={reduce ? false : { opacity: 0, y: 18 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7, ease: EASE, delay: 0.16 }}
            className="mt-6 max-w-xl text-pretty text-ink-muted [font-size:var(--text-lead)] leading-relaxed"
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
              <span className="text-accent">npx</span> devrites@latest
            </code>
            <CopyButton text={INSTALL_CMD} />
          </motion.div>

          {/* CTAs — one primary, one quiet text link (DESIGN.md §4) */}
          <motion.div
            initial={reduce ? false : { opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7, ease: EASE, delay: 0.32 }}
            className="mt-7 flex flex-col items-start gap-x-6 gap-y-3 sm:flex-row sm:items-center"
          >
            <MagneticLink href="#install" className="btn btn-primary group px-6 py-3">
              Install in one command
              <ArrowRight className="size-4 transition-transform duration-200 group-hover:translate-x-1" />
            </MagneticLink>
            <a
              href="#workflow"
              className="group inline-flex items-center gap-1.5 text-sm font-medium text-ink-muted transition-colors duration-200 hover:text-ink"
            >
              <BookOpen className="size-4" />
              See how it works
              <ArrowRight className="size-3.5 transition-transform duration-200 group-hover:translate-x-0.5" />
            </a>
          </motion.div>

          {/* trust line */}
          <motion.div
            initial={reduce ? false : { opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.8, ease: EASE, delay: 0.5 }}
            className="mt-6 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-ink-faint"
          >
            <a href={REPO} rel="noopener" className="inline-flex items-center gap-1.5 transition-colors hover:text-ink">
              <Star className="size-4 text-warn" fill="currentColor" />
              {stars !== null ? <b className="text-ink">{stars.toLocaleString()}</b> : "-"}{" "}
              {stars === 1 ? "star" : "stars"}
            </a>
            <span aria-hidden>·</span>
            <span>Free &amp; open source (MIT)</span>
            <span aria-hidden>·</span>
            <span>
              never touches <span className="k">~/.claude</span>
            </span>
          </motion.div>
        </div>

        {/* right rail — the console product visual (rendered in Remotion) */}
        <motion.div
          initial={reduce ? false : { opacity: 0, y: 40, scale: 0.97 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          transition={{ duration: 0.9, ease: EASE, delay: 0.4 }}
          className="glass glow-accent relative w-full overflow-hidden rounded-2xl"
        >
          {reduce ? (
            <img
              src="/assets/video/hero-poster.png"
              width={1600}
              height={900}
              alt="DevRites console: the build phase paused at a gate after 14 tests, types, and build pass, with evidence written to .devrites/work."
              className="block w-full"
            />
          ) : (
            <video
              className="block w-full"
              width={1600}
              height={900}
              autoPlay
              loop
              muted
              playsInline
              poster="/assets/video/hero-poster.png"
              aria-label="DevRites console running the build phase: 14 tests, types, and build pass, then it stops at a gate with evidence written to disk."
            >
              <source src="/assets/video/hero.webm" type="video/webm" />
              <source src="/assets/video/hero.mp4" type="video/mp4" />
            </video>
          )}
        </motion.div>
      </div>
    </section>
  );
}
