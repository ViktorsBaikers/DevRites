"use client";

import {
  motion,
  useMotionValue,
  useSpring,
  useTransform,
  useReducedMotion,
} from "framer-motion";
import type { PointerEvent } from "react";
import {
  Files,
  ShieldAlert,
  KeyRound,
  Network,
  ShieldCheck,
  BookMarked,
  Plane,
  type LucideIcon,
} from "lucide-react";
import { Reveal, SectionHead } from "./ui";
import { MECHANISMS, type Mechanism } from "@/lib/site";

const ICONS: Record<string, LucideIcon> = {
  "paper-trail": Files,
  drift: ShieldAlert,
  "type-go": KeyRound,
  fanout: Network,
  security: ShieldCheck,
  learn: BookMarked,
  afk: Plane,
};

const TONE: Record<Mechanism["tone"], string> = {
  accent: "text-accent",
  go: "text-go",
  warn: "text-warn",
  danger: "text-danger",
};

const SPAN: Record<Mechanism["span"], string> = {
  wide: "md:col-span-2 lg:col-span-4",
  tall: "md:col-span-1 lg:col-span-2 lg:row-span-2",
  std: "md:col-span-1 lg:col-span-2",
};

export default function Bento() {
  return (
    <section id="mechanisms" className="wrap py-20 sm:py-28">
      <SectionHead
        center
        eyebrow="Proof, not promises"
        title="Every guarantee is a real mechanism."
        lead="No vibes. Each one is a named gate you can read in the repo. It's the reason an unattended agent won't do the expensive, irreversible thing on its own."
      />

      <div className="mt-12 grid grid-cols-1 gap-4 md:grid-cols-2 lg:auto-rows-[minmax(0,1fr)] lg:grid-cols-6">
        {MECHANISMS.map((m, i) => (
          <Tile key={m.key} m={m} index={i} />
        ))}
      </div>
    </section>
  );
}

function Tile({ m, index }: { m: Mechanism; index: number }) {
  const reduce = useReducedMotion() ?? false;
  const Icon = ICONS[m.key] ?? Files;

  const mx = useMotionValue(0);
  const my = useMotionValue(0);
  const rx = useSpring(useTransform(my, [-0.5, 0.5], [6, -6]), { stiffness: 200, damping: 18 });
  const ry = useSpring(useTransform(mx, [-0.5, 0.5], [-6, 6]), { stiffness: 200, damping: 18 });

  function onMove(e: PointerEvent<HTMLDivElement>) {
    if (reduce) return;
    const r = e.currentTarget.getBoundingClientRect();
    mx.set((e.clientX - r.left) / r.width - 0.5);
    my.set((e.clientY - r.top) / r.height - 0.5);
  }
  function onLeave() {
    mx.set(0);
    my.set(0);
  }

  return (
    <Reveal
      delay={Math.min(index * 0.06, 0.3)}
      className={`${SPAN[m.span]} group`}
    >
      <motion.div
        onPointerMove={onMove}
        onPointerLeave={onLeave}
        style={reduce ? undefined : { rotateX: rx, rotateY: ry, transformPerspective: 900 }}
        className="tile bento-tile relative flex h-full flex-col gap-3 overflow-hidden p-6"
      >
        {/* hover wash */}
        <div className="blade-soft pointer-events-none absolute -right-16 -top-16 size-40 rounded-full opacity-0 blur-2xl transition-opacity duration-500 group-hover:opacity-100" />

        <div className={`flex size-11 items-center justify-center rounded-xl border border-line bg-surface-2/60 ${TONE[m.tone]}`}>
          <Icon className="size-5" strokeWidth={1.8} />
        </div>

        <h3 className="text-lg font-bold text-ink">{m.title}</h3>
        <p className="text-[0.94rem] leading-relaxed text-ink-muted">{m.body}</p>

        {m.tags && (
          <div className="mt-auto flex flex-wrap gap-1.5 pt-2">
            {m.tags.map((t) => (
              <span
                key={t}
                className="mono rounded-md border border-line bg-surface-2/50 px-2 py-1 text-[0.7rem] text-ink-muted"
              >
                {t}
              </span>
            ))}
          </div>
        )}

        {m.demo && (
          <div className="mono mt-auto rounded-lg border border-line bg-bg-deep/60 px-3 py-2 text-[0.72rem] text-ink-muted">
            {m.demo}
          </div>
        )}
      </motion.div>
    </Reveal>
  );
}
