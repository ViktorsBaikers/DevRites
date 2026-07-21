"use client";

import { Moon, Sun } from "lucide-react";
import { useEffect, useState } from "react";

type Theme = "light" | "dark";

function currentTheme(): Theme {
  return document.documentElement.dataset.theme === "light" ? "light" : "dark";
}

export default function ThemeToggle({ className = "" }: { className?: string }) {
  const [theme, setTheme] = useState<Theme | null>(null);

  useEffect(() => setTheme(currentTheme()), []);

  const next = (theme ?? "dark") === "dark" ? "light" : "dark";

  return (
    <button
      type="button"
      className={`inline-flex size-9 cursor-pointer items-center justify-center rounded-full border border-line text-ink-muted transition-colors hover:border-line-bright hover:bg-surface-2 hover:text-ink ${className}`}
      aria-label={theme ? `Use ${next} theme` : "Change theme"}
      title={theme ? `Use ${next} theme` : "Change theme"}
      onClick={() => {
        const selected = theme ? next : currentTheme() === "dark" ? "light" : "dark";
        document.documentElement.dataset.theme = selected;
        localStorage.setItem("devrites-theme", selected);
        setTheme(selected);
      }}
    >
      {theme === "light" ? <Moon className="size-4" aria-hidden /> : <Sun className="size-4" aria-hidden />}
    </button>
  );
}
