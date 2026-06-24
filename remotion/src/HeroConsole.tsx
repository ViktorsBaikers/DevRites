import {
  AbsoluteFill,
  Easing,
  interpolate,
  useCurrentFrame,
} from "remotion";
import { loadFont as loadMono } from "@remotion/google-fonts/JetBrainsMono";
import { loadFont as loadSans } from "@remotion/google-fonts/SchibstedGrotesk";

const mono = loadMono("normal", { weights: ["400", "500", "700"] }).fontFamily;
const sans = loadSans("normal", { weights: ["600", "700", "800"] }).fontFamily;

/* DevRites blade palette — OKLCH, matched to site/app/globals.css */
const C = {
  bg: "oklch(0.15 0.052 264)",
  bgDeep: "oklch(0.11 0.045 264)",
  surface: "oklch(0.214 0.05 264)",
  surface2: "oklch(0.262 0.045 264)",
  line: "oklch(0.34 0.04 264)",
  lineBright: "oklch(0.47 0.05 264)",
  ink: "oklch(0.975 0.008 250)",
  inkMuted: "oklch(0.808 0.018 250)",
  inkFaint: "oklch(0.66 0.02 250)",
  accent: "oklch(0.82 0.14 213)",
  accentBlue: "oklch(0.6 0.19 252)",
  go: "oklch(0.8 0.16 158)",
  warn: "oklch(0.82 0.13 78)",
  danger: "oklch(0.66 0.2 25)",
};

const PIPE = [
  "spec",
  "temper",
  "define",
  "vet",
  "build",
  "prove",
  "polish",
  "review",
  "seal",
  "ship",
];
const HERE = 4; // build is the live phase

const EASE = Easing.bezier(0.16, 1, 0.3, 1);

const clamp = { extrapolateLeft: "clamp", extrapolateRight: "clamp" } as const;

export const HeroConsole: React.FC = () => {
  const frame = useCurrentFrame();

  // soft loop seam: fade content in at the head, out at the tail
  const fadeIn = interpolate(frame, [0, 12], [0, 1], clamp);
  const fadeOut = interpolate(frame, [256, 270], [1, 0], clamp);
  const op = Math.min(fadeIn, fadeOut);

  // live "build" node pulse
  const pulse = 0.5 + 0.5 * Math.sin(frame / 9);

  // typed command
  const nChars = Math.round(interpolate(frame, [54, 104], [0, 11], clamp));
  const typed = "/rite-build".slice(0, nChars);
  const caretOn = frame % 30 < 16;
  const showOut = frame > 116;

  return (
    <AbsoluteFill style={{ backgroundColor: C.bgDeep }}>
      {/* depth: cyan bloom + dotted field */}
      <AbsoluteFill
        style={{
          background: `radial-gradient(120% 90% at 82% -15%, oklch(0.82 0.14 213 / 0.14), transparent 60%), radial-gradient(110% 90% at 10% 0%, oklch(0.6 0.19 252 / 0.10), transparent 55%), ${C.bg}`,
        }}
      />
      <AbsoluteFill
        style={{
          backgroundImage: `radial-gradient(${C.lineBright} 1px, transparent 1px)`,
          backgroundSize: "34px 34px",
          opacity: 0.1,
        }}
      />

      <AbsoluteFill style={{ opacity: op, fontFamily: sans }}>
        {/* title bar */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 14,
            padding: "0 32px",
            height: 76,
            borderBottom: `1px solid ${C.line}`,
            background: "oklch(0.262 0.045 264 / 0.35)",
          }}
        >
          <Dot color={C.danger} />
          <Dot color={C.warn} />
          <Dot color={C.go} />
          <span
            style={{
              fontFamily: mono,
              fontSize: 22,
              color: C.inkMuted,
              marginLeft: 10,
            }}
          >
            <b style={{ color: C.ink, fontWeight: 700 }}>auth-tokens</b>
            {"  ·  .devrites/work"}
          </span>
          <Chip pulse={pulse} />
        </div>

        {/* body */}
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 44,
            padding: "52px 52px 48px",
            height: 824,
          }}
        >
          {/* pipeline rail */}
          <div style={{ display: "flex", alignItems: "flex-start" }}>
            {PIPE.map((p, i) => {
              const done = i < HERE;
              const here = i === HERE;
              const delay = 14 + i * 4;
              const s = interpolate(frame, [delay, delay + 12], [0.55, 1], {
                ...clamp,
                easing: EASE,
              });
              const o = interpolate(frame, [delay, delay + 12], [0, 1], clamp);
              const tone = here ? C.accent : done ? C.go : C.inkFaint;
              return (
                <div
                  key={p}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    flex: i < PIPE.length - 1 ? 1 : "0 0 auto",
                  }}
                >
                  <div
                    style={{
                      display: "flex",
                      flexDirection: "column",
                      alignItems: "center",
                      gap: 12,
                      opacity: o,
                      scale: String(here ? s * (0.97 + pulse * 0.06) : s),
                    }}
                  >
                    <div
                      style={{
                        width: 56,
                        height: 56,
                        borderRadius: 999,
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                        fontFamily: mono,
                        fontSize: 22,
                        fontWeight: 700,
                        color: tone,
                        border: `2px solid ${here ? C.accent : done ? "oklch(0.8 0.16 158 / 0.5)" : C.line}`,
                        background: here
                          ? "oklch(0.82 0.14 213 / 0.16)"
                          : done
                            ? "oklch(0.8 0.16 158 / 0.12)"
                            : "transparent",
                        boxShadow: here
                          ? `0 0 ${18 + pulse * 22}px oklch(0.82 0.14 213 / ${0.35 + pulse * 0.3})`
                          : "none",
                      }}
                    >
                      {done ? "✓" : i + 1}
                    </div>
                    <span style={{ fontFamily: mono, fontSize: 19, color: tone }}>
                      {p}
                    </span>
                  </div>
                  {i < PIPE.length - 1 && (
                    <div
                      style={{
                        flex: 1,
                        height: 2,
                        marginTop: 27,
                        background: i < HERE ? "oklch(0.8 0.16 158 / 0.55)" : C.line,
                      }}
                    />
                  )}
                </div>
              );
            })}
          </div>

          {/* terminal */}
          <div
            style={{
              flex: 1,
              borderRadius: 18,
              border: `1px solid ${C.line}`,
              background: "oklch(0.11 0.045 264 / 0.6)",
              padding: 36,
              fontFamily: mono,
              fontSize: 25,
              lineHeight: 1.55,
            }}
          >
            <div style={{ display: "flex", alignItems: "center" }}>
              <span style={{ color: C.accent }}>›</span>
              <span style={{ marginLeft: 12, color: C.ink }}>{typed}</span>
              {!showOut && (
                <span
                  style={{
                    display: "inline-block",
                    width: 13,
                    height: 28,
                    marginLeft: 4,
                    background: C.accent,
                    opacity: caretOn ? 1 : 0,
                  }}
                />
              )}
            </div>

            {showOut && (
              <div style={{ marginTop: 26 }}>
                <Out from={120} delay={0}>
                  <span style={{ color: C.inkFaint }}>
                    slice 3/5 · refresh-token rotation
                  </span>
                </Out>
                <div
                  style={{
                    display: "flex",
                    gap: 36,
                    marginTop: 14,
                    flexWrap: "wrap",
                  }}
                >
                  <Out from={140}>
                    <span style={{ color: C.go }}>✓ 14 tests green</span>
                  </Out>
                  <Out from={150}>
                    <span style={{ color: C.go }}>✓ types</span>
                  </Out>
                  <Out from={160}>
                    <span style={{ color: C.go }}>✓ build</span>
                  </Out>
                </div>
                <Out from={176}>
                  <div style={{ color: C.inkFaint, marginTop: 16 }}>
                    → evidence.md, touched-files.md written.{" "}
                    <span style={{ color: C.warn }}>stopped.</span>
                  </div>
                </Out>
              </div>
            )}
          </div>
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};

const Dot: React.FC<{ color: string }> = ({ color }) => (
  <span
    style={{
      width: 15,
      height: 15,
      borderRadius: 999,
      background: color,
      opacity: 0.78,
    }}
  />
);

const Chip: React.FC<{ pulse: number }> = ({ pulse }) => (
  <span
    style={{
      marginLeft: "auto",
      display: "inline-flex",
      alignItems: "center",
      gap: 10,
      fontFamily: mono,
      fontSize: 18,
      letterSpacing: "0.06em",
      textTransform: "uppercase",
      color: C.accent,
      padding: "8px 16px",
      borderRadius: 999,
      border: "1px solid oklch(0.82 0.14 213 / 0.45)",
      background: "oklch(0.262 0.045 264 / 0.6)",
    }}
  >
    <span
      style={{
        width: 9,
        height: 9,
        borderRadius: 999,
        background: C.accent,
        scale: String(0.85 + pulse * 0.5),
        opacity: 0.55 + pulse * 0.45,
      }}
    />
    build 3/5
  </span>
);

/* rising fade-in for a terminal output line */
const Out: React.FC<{ from: number; delay?: number; children: React.ReactNode }> = ({
  from,
  children,
}) => {
  const frame = useCurrentFrame();
  const o = interpolate(frame, [from, from + 12], [0, 1], clamp);
  const y = interpolate(frame, [from, from + 12], [8, 0], { ...clamp, easing: EASE });
  return (
    <div style={{ opacity: o, translate: `0px ${y}px` }}>{children}</div>
  );
};
