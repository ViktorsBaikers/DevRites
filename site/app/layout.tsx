import type { Metadata, Viewport } from "next";
import localFont from "next/font/local";
import "./globals.css";
import { FAQ, SITE_URL, REPO, VERSION } from "@/lib/site";

const schibsted = localFont({
  src: [
    { path: "./fonts/schibsted-grotesk-latin.woff2", weight: "400 900", style: "normal" },
    { path: "./fonts/schibsted-grotesk-latin-ext.woff2", weight: "400 900", style: "normal" },
  ],
  variable: "--font-schibsted",
  display: "swap",
  fallback: ["ui-sans-serif", "system-ui", "sans-serif"],
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

const title = "DevRites: a disciplined senior engineer for Claude Code";
const description =
  "DevRites turns Claude Code into a disciplined senior engineer: spec, build one verified slice, prove with evidence, review, seal, ship. State lives on disk. Free and open source.";

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title,
  description,
  applicationName: "DevRites",
  authors: [{ name: "Viktors Baikers", url: "https://github.com/ViktorsBaikers" }],
  keywords: [
    "Claude Code", "AI coding agent", "spec-driven development", "agentic workflow",
    "AI pair programming", "Claude Code skills", "slash commands", "MCP server",
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
      "Spec, build one verified slice, prove with evidence, review, seal, ship. The agent that refuses to claim done without proof.",
    images: [{ url: "/assets/img/og.png", width: 1200, height: 630, alt: "DevRites: a senior engineer on every feature." }],
  },
  twitter: {
    card: "summary_large_image",
    title,
    description:
      "Spec, build one verified slice, prove with evidence, review, seal, ship. The agent that refuses to claim done without proof.",
    images: ["/assets/img/og.png"],
  },
};

export const viewport: Viewport = {
  themeColor: "#0c1330",
  colorScheme: "dark",
};

const jsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebSite",
      "@id": `${SITE_URL}/#website`,
      url: `${SITE_URL}/`,
      name: "DevRites",
      description: "A disciplined senior-engineer workflow for Claude Code.",
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
        "A project-local pack of Claude Code skills that runs a disciplined senior-engineer workflow: spec, build one verified slice, prove with evidence, review, seal, ship. State lives on disk so a fresh agent resumes where the last one stopped.",
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
    <html lang="en" className={`${schibsted.variable} ${jbmono.variable}`}>
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
