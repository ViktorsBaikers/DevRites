import type { Metadata, Viewport } from "next";
import localFont from "next/font/local";
import { Outfit } from "next/font/google";
import "./globals.css";
import { FAQ, SITE_URL, REPO, VERSION } from "@/lib/site";

const outfit = Outfit({
  variable: "--font-outfit",
  subsets: ["latin"],
  display: "optional",
});

const jbmono = localFont({
  src: [
    { path: "./fonts/jetbrains-mono-latin.woff2", weight: "400 700", style: "normal" },
    { path: "./fonts/jetbrains-mono-latin-ext.woff2", weight: "400 700", style: "normal" },
  ],
  variable: "--font-jbmono",
  display: "swap",
  fallback: ["ui-monospace", "monospace"],
});

const title = "DevRites: verify AI-written code before release";
const description =
  "Give Claude Code and Codex a shared workflow for specs, bounded builds, recorded checks, independent review, and human approval before release.";

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title,
  description,
  applicationName: "DevRites",
  authors: [{ name: "Viktors Baikers", url: "https://github.com/ViktorsBaikers" }],
  keywords: [
    "Claude Code", "Codex", "AI coding agent", "spec-driven development", "agentic workflow",
    "AI pair programming", "Claude Code skills", "Codex skills", "Go control plane",
    "spec kit alternative", "AI code review", "test-driven AI", "OWASP LLM Top 10",
    "brownfield AI onboarding",
  ],
  alternates: { canonical: "/" },
  robots: { index: true, follow: true, "max-image-preview": "large" } as Metadata["robots"],
  icons: {
    icon: [{ url: "/assets/img/mark-64.png", sizes: "64x64", type: "image/png" }],
    apple: [{ url: "/assets/img/mark-256.png" }],
  },
  openGraph: {
    type: "website",
    siteName: "DevRites",
    locale: "en_US",
    url: SITE_URL,
    title,
    description:
      "Move Claude Code and Codex work through bounded builds, recorded checks, independent review, and human approval.",
  },
  twitter: {
    card: "summary",
    title,
    description:
      "Move Claude Code and Codex work through bounded builds, recorded checks, independent review, and human approval.",
  },
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#f1f3f7" },
    { media: "(prefers-color-scheme: dark)", color: "#111722" },
  ],
  colorScheme: "light dark",
};

const jsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebSite",
      "@id": `${SITE_URL}/#website`,
      url: `${SITE_URL}/`,
      name: "DevRites",
      description: "A shared engineering workflow for Claude Code and Codex.",
      inLanguage: "en",
      publisher: { "@id": `${SITE_URL}/#org` },
    },
    {
      "@type": "SoftwareApplication",
      "@id": `${SITE_URL}/#app`,
      name: "DevRites",
      url: `${SITE_URL}/`,
      applicationCategory: "DeveloperApplication",
      operatingSystem: "macOS, Linux, Windows",
      softwareVersion: VERSION,
      description:
        "A spec-driven engineering system for Claude Code and Codex. Generated project-local host artifacts use the same Go control plane and git-diffable workspace, so a new agent can resume from the recorded project state.",
      license: `${REPO}/blob/main/LICENSE`,
      codeRepository: REPO,
      isAccessibleForFree: true,
      author: { "@id": `${SITE_URL}/#author` },
      offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
    },
    {
      "@type": "Organization",
      "@id": `${SITE_URL}/#org`,
      name: "DevRites",
      url: `${SITE_URL}/`,
      logo: `${SITE_URL}/assets/img/mark-256.png`,
      sameAs: [REPO],
    },
    {
      "@type": "Person",
      "@id": `${SITE_URL}/#author`,
      name: "Viktors Baikers",
      url: "https://github.com/ViktorsBaikers",
    },
    {
      "@type": "FAQPage",
      "@id": `${SITE_URL}/#faq`,
      mainEntity: FAQ.map((f) => ({
        "@type": "Question",
        name: f.q,
        acceptedAnswer: { "@type": "Answer", text: f.a },
      })),
    },
  ],
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" className={`${outfit.variable} ${jbmono.variable}`} suppressHydrationWarning>
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html: `(function(){try{var saved=localStorage.getItem('devrites-theme');var theme=saved==='light'||saved==='dark'?saved:(matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light');document.documentElement.dataset.theme=theme;}catch(_){}})();`,
          }}
        />
      </head>
      <body>
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
        />
        {children}
      </body>
    </html>
  );
}
