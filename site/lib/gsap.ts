"use client";

// Central GSAP setup. ScrollTrigger + SplitText are free as of gsap 3.13.
// GSAP owns scroll-driven set-pieces and magnetic/parallax micro-motion;
// framer-motion keeps the mount entrances, tilt, and FAQ. The two never
// animate the same property on the same element.
import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { SplitText } from "gsap/SplitText";
import { useGSAP } from "@gsap/react";

if (typeof window !== "undefined") {
  gsap.registerPlugin(ScrollTrigger, SplitText, useGSAP);
}

// Matches the site's --ease-out-soft cubic-bezier(0.16, 1, 0.3, 1).
export const BLADE_EASE = "expo.out";

export { gsap, ScrollTrigger, SplitText, useGSAP };
