"use client";

import {
  motion,
  useInView,
  useReducedMotion,
  animate,
  type Variants,
} from "framer-motion";
import { useEffect, useRef, useState, type ReactNode, type AnchorHTMLAttributes } from "react";
import { Check, Copy } from "lucide-react";
import { gsap, useGSAP } from "@/lib/gsap";

export const EASE = [0.16, 1, 0.3, 1] as const;

/* Fade + rise into view. Collapses to instant when reduced-motion is set. */
export function Reveal({
  children,
  delay = 0,
  y = 22,
  className,
  as = "div",
}: {
  children: ReactNode;
  delay?: number;
  y?: number;
  className?: string;
  as?: "div" | "section" | "li" | "span";
}) {
  const reduce = useReducedMotion();
  const MotionTag = motion[as];
  return (
    <MotionTag
      className={className}
      initial={reduce ? false : { opacity: 0, y }}
      whileInView={reduce ? undefined : { opacity: 1, y: 0 }}
      viewport={{ once: true, margin: "-12% 0px" }}
      transition={{ duration: 0.7, ease: EASE, delay }}
    >
      {children}
    </MotionTag>
  );
}

export const staggerParent: Variants = {
  hidden: {},
  show: { transition: { staggerChildren: 0.08, delayChildren: 0.05 } },
};
export const staggerChild: Variants = {
  hidden: { opacity: 0, y: 20 },
  show: { opacity: 1, y: 0, transition: { duration: 0.6, ease: EASE } },
};

export function Stagger({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  const reduce = useReducedMotion();
  return (
    <motion.div
      className={className}
      variants={reduce ? undefined : staggerParent}
      initial={reduce ? false : "hidden"}
      whileInView={reduce ? undefined : "show"}
      viewport={{ once: true, margin: "-10% 0px" }}
    >
      {children}
    </motion.div>
  );
}

export function CopyButton({
  text,
  label = "copy",
  className = "",
}: {
  text: string;
  label?: string;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      aria-label="Copy command to clipboard"
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
          setCopied(true);
          setTimeout(() => setCopied(false), 1600);
        } catch {
          /* clipboard blocked — no-op */
        }
      }}
      className={`group inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-line bg-surface-2/70 px-2.5 py-1.5 font-mono text-xs text-ink-muted transition-colors duration-200 hover:border-accent/50 hover:text-ink ${className}`}
    >
      {copied ? (
        <Check className="size-3.5 text-go" strokeWidth={2.4} />
      ) : (
        <Copy className="size-3.5" strokeWidth={2} />
      )}
      <span>{copied ? "copied" : label}</span>
    </button>
  );
}

export function SectionHead({
  eyebrow,
  title,
  lead,
  center = false,
}: {
  eyebrow?: string;
  title: ReactNode;
  lead?: ReactNode;
  center?: boolean;
}) {
  return (
    <div className={`max-w-2xl ${center ? "mx-auto text-center" : ""}`}>
      {eyebrow ? (
        <Reveal>
          <span className="mono inline-block text-xs font-medium uppercase tracking-[0.16em] text-accent">
            {eyebrow}
          </span>
        </Reveal>
      ) : null}
      <Reveal delay={0.06}>
        <h2 className="mt-3 font-bold text-ink [font-size:var(--text-h2)]">{title}</h2>
      </Reveal>
      {lead ? (
        <Reveal delay={0.12}>
          <p className="mt-4 text-pretty text-ink-muted [font-size:var(--text-lead)] leading-relaxed">
            {lead}
          </p>
        </Reveal>
      ) : null}
    </div>
  );
}

/* Count from 0 -> value when scrolled into view. */
export function CountUp({
  value,
  suffix = "",
  prefix = "",
}: {
  value: number;
  suffix?: string;
  prefix?: string;
}) {
  const ref = useRef<HTMLSpanElement>(null);
  const inView = useInView(ref, { once: true, margin: "-20% 0px" });
  const reduce = useReducedMotion();
  const [display, setDisplay] = useState(reduce ? value : 0);

  useEffect(() => {
    if (!inView || reduce) return;
    const controls = animate(0, value, {
      duration: 1.4,
      ease: EASE,
      onUpdate: (v) => setDisplay(Math.round(v)),
    });
    return () => controls.stop();
  }, [inView, value, reduce]);

  return (
    <span ref={ref}>
      {prefix}
      {display}
      {suffix}
    </span>
  );
}

/* GitHub mark — lucide deprecated its brand icons, so we ship our own. */
export function GithubMark({ className = "size-4" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true" className={className}>
      <path d="M12 .5a12 12 0 0 0-3.79 23.4c.6.1.82-.26.82-.58v-2c-3.34.73-4.04-1.6-4.04-1.6-.55-1.4-1.34-1.77-1.34-1.77-1.1-.75.08-.73.08-.73 1.2.08 1.84 1.24 1.84 1.24 1.07 1.84 2.81 1.3 3.5 1 .1-.78.42-1.3.76-1.6-2.66-.3-5.46-1.33-5.46-5.93 0-1.3.47-2.38 1.24-3.22-.13-.3-.54-1.52.1-3.18 0 0 1-.32 3.3 1.23a11.5 11.5 0 0 1 6 0c2.28-1.55 3.29-1.23 3.29-1.23.65 1.66.24 2.88.12 3.18.77.84 1.23 1.91 1.23 3.22 0 4.61-2.8 5.62-5.48 5.92.43.37.81 1.1.81 2.22v3.29c0 .32.22.69.82.57A12 12 0 0 0 12 .5Z" />
    </svg>
  );
}

/* Pointer-magnetic anchor — the button slides toward the cursor, eases home on
   leave. GSAP owns this element's transform; reduced-motion renders a plain <a>. */
export function MagneticLink({
  children,
  strength = 0.16,
  ...rest
}: { children: ReactNode; strength?: number } & AnchorHTMLAttributes<HTMLAnchorElement>) {
  const ref = useRef<HTMLAnchorElement>(null);
  const reduce = useReducedMotion();

  useGSAP(
    () => {
      const el = ref.current;
      if (reduce || !el) return;
      const xTo = gsap.quickTo(el, "x", { duration: 0.5, ease: "power3.out" });
      const yTo = gsap.quickTo(el, "y", { duration: 0.5, ease: "power3.out" });
      const onMove = (e: PointerEvent) => {
        const r = el.getBoundingClientRect();
        xTo((e.clientX - (r.left + r.width / 2)) * strength);
        yTo((e.clientY - (r.top + r.height / 2)) * strength);
      };
      const onLeave = () => {
        xTo(0);
        yTo(0);
      };
      el.addEventListener("pointermove", onMove);
      el.addEventListener("pointerleave", onLeave);
      return () => {
        el.removeEventListener("pointermove", onMove);
        el.removeEventListener("pointerleave", onLeave);
      };
    },
    { dependencies: [reduce, strength], scope: ref },
  );

  return (
    <a ref={ref} {...rest}>
      {children}
    </a>
  );
}

/* Live GitHub star count; hidden until it resolves. */
export function useGitHubStars(repo = "ViktorsBaikers/DevRites") {
  const [stars, setStars] = useState<number | null>(null);
  useEffect(() => {
    let live = true;
    fetch(`https://api.github.com/repos/${repo}`)
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => {
        if (live && d && typeof d.stargazers_count === "number")
          setStars(d.stargazers_count);
      })
      .catch(() => {});
    return () => {
      live = false;
    };
  }, [repo]);
  return stars;
}
