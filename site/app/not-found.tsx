import type { Metadata } from "next";
import Image from "next/image";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import ThemeToggle from "@/components/ThemeToggle";

export const metadata: Metadata = {
  title: "Page not found · DevRites",
  description: "The requested DevRites route does not exist.",
};

export default function NotFound() {
  return (
    <div className="relative min-h-[100dvh] overflow-hidden">
      <a
        href="#not-found-main"
        className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50 focus:rounded-lg focus:bg-surface focus:px-4 focus:py-2 focus:text-ink"
      >
        Skip to content
      </a>

      <header className="wrap flex items-center justify-between py-6">
        <Link href="/" className="inline-flex items-center gap-2.5" aria-label="DevRites home">
          <Image src="/assets/img/mark-64.png" width={28} height={28} alt="" priority />
          <b className="text-lg">DevRites</b>
        </Link>
        <div className="flex items-center gap-3">
          <span className="mono text-xs text-ink-faint">HTTP 404</span>
          <ThemeToggle />
        </div>
      </header>

      <main id="not-found-main" className="wrap grid min-h-[calc(100dvh-5rem)] items-center gap-14 pb-24 pt-14 lg:grid-cols-[1.08fr_0.92fr]">
        <div>
          <h1 className="max-w-4xl font-bold [font-size:clamp(3.4rem,7vw,5.8rem)] leading-[0.9] tracking-[-0.04em]">
            We could not find that page.
          </h1>
          <p className="mt-8 max-w-xl text-lg leading-relaxed text-ink-muted">
            Check the address, return to the homepage, or open the documentation.
          </p>
          <div className="mt-10 flex flex-col items-start gap-4 sm:flex-row sm:items-center">
            <Link href="/" className="btn btn-primary px-7 py-3.5">
              Return home
            </Link>
            <Link href="/docs/" className="inline-flex items-center gap-2 px-2 py-3.5 font-medium text-ink-muted transition-colors duration-300 hover:text-ink">
              Read the docs <ArrowRight className="size-4" aria-hidden />
            </Link>
          </div>
        </div>

        <div className="relative lg:translate-y-16">
          <div className="glass rounded-card p-6 md:p-8">
            <div className="mono border-b border-line pb-4 text-xs text-ink-faint">route lookup</div>
            <div className="mono space-y-4 py-12 text-sm">
              <p className="text-ink"><span className="text-accent">›</span> checking requested path</p>
              <p className="text-danger">page not found</p>
              <p className="text-ink-muted">next: choose a link above</p>
            </div>
          </div>
        </div>
      </main>

      <div aria-hidden className="pointer-events-none absolute -left-64 top-1/4 -z-10 size-[42rem] rounded-full bg-accent/[0.09] blur-[160px]" />
    </div>
  );
}
