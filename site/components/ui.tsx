"use client";

import { useEffect, useRef, useState, type ReactNode, type AnchorHTMLAttributes } from "react";
import { Check, Copy } from "lucide-react";
import { gsap, useGSAP } from "@/lib/gsap";

export function useStableReducedMotion() {
  const [reduce, setReduce] = useState(false);

  useEffect(() => {
    const query = window.matchMedia("(prefers-reduced-motion: reduce)");
    const update = () => setReduce(query.matches);
    update();
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);

  return reduce;
}

/* Content stays visible by default. Section-specific motion owns storytelling. */
export function Reveal({
  children,
  className,
  as = "div",
}: {
  children: ReactNode;
  delay?: number;
  y?: number;
  className?: string;
  as?: "div" | "section" | "li" | "span";
}) {
  const Tag = as;
  return (
    <Tag className={className}>
      {children}
    </Tag>
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
  const [status, setStatus] = useState<"idle" | "copied" | "error">("idle");
  const statusLabel = status === "copied" ? "copied" : status === "error" ? "copy failed" : label;
  const accessibleLabel = status === "copied"
    ? "Command copied to clipboard"
    : status === "error"
      ? "Could not copy command"
      : `${label} command to clipboard`;
  return (
    <button
      type="button"
      aria-label={accessibleLabel}
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
          setStatus("copied");
          setTimeout(() => setStatus("idle"), 1600);
        } catch {
          setStatus("error");
          setTimeout(() => setStatus("idle"), 2200);
        }
      }}
      className={`group inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-line bg-surface-2/70 px-2.5 py-1.5 font-mono text-xs text-ink-muted transition-colors duration-200 hover:border-accent/50 hover:text-ink ${className}`}
    >
      {status === "copied" ? (
        <Check className="size-3.5 text-go" strokeWidth={2.4} />
      ) : (
        <Copy className="size-3.5" strokeWidth={2} />
      )}
      <span aria-live="polite">{statusLabel}</span>
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
    <div className={`max-w-4xl ${center ? "mx-auto text-center" : ""}`}>
      {eyebrow ? (
        <Reveal>
          <span className="mono inline-block text-xs font-medium uppercase tracking-[0.16em] text-accent">
            {eyebrow}
          </span>
        </Reveal>
      ) : null}
      <Reveal delay={0.06}>
        <h2 className={`${eyebrow ? "mt-3" : ""} max-w-4xl font-bold text-ink [font-size:var(--text-h2)]`}>{title}</h2>
      </Reveal>
      {lead ? (
        <Reveal delay={0.12}>
          <p className={`mt-5 max-w-2xl text-pretty text-ink-muted [font-size:var(--text-lead)] leading-relaxed ${center ? "mx-auto" : ""}`}>
            {lead}
          </p>
        </Reveal>
      ) : null}
    </div>
  );
}

/* Pointer-magnetic anchor: the button slides toward the cursor, eases home on
   leave. GSAP owns this element's transform; reduced-motion renders a plain <a>. */
export function MagneticLink({
  children,
  strength = 0.16,
  ...rest
}: { children: ReactNode; strength?: number } & AnchorHTMLAttributes<HTMLAnchorElement>) {
  const ref = useRef<HTMLAnchorElement>(null);
  const reduce = useStableReducedMotion();

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
