"use client";

import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";

type Heading = { id: string; label: string };

export default function DocsToc() {
  const pathname = usePathname();
  const [headings, setHeadings] = useState<Heading[]>([]);
  const [active, setActive] = useState<string | null>(null);

  useEffect(() => {
    const elements = Array.from(document.querySelectorAll<HTMLElement>("#docs-content h2[id]"));
    setHeadings(elements.map((heading) => ({ id: heading.id, label: heading.textContent ?? heading.id })));
    setActive(elements[0]?.id ?? null);

    const observer = new IntersectionObserver(
      (entries) => {
        const current = entries.find((entry) => entry.isIntersecting);
        if (current) setActive(current.target.id);
      },
      { rootMargin: "-18% 0px -72% 0px" },
    );
    elements.forEach((heading) => observer.observe(heading));
    return () => observer.disconnect();
  }, [pathname]);

  if (!headings.length) return null;

  return (
    <aside className="sticky top-28 h-max" aria-label="On this page">
      <p className="mono text-xs text-ink-faint">On this page</p>
      <nav className="mt-4 grid gap-1">
        {headings.map((heading) => (
          <a
            key={heading.id}
            href={`#${heading.id}`}
            aria-current={active === heading.id ? "location" : undefined}
            className={`border-l px-3 py-1.5 text-sm leading-snug transition-colors ${
              active === heading.id
                ? "border-accent text-ink"
                : "border-line text-ink-faint hover:border-line-bright hover:text-ink"
            }`}
          >
            {heading.label}
          </a>
        ))}
      </nav>
    </aside>
  );
}
