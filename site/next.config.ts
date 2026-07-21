import type { NextConfig } from "next";
import { readFileSync } from "node:fs";
import path from "node:path";

const { version } = JSON.parse(
  readFileSync(path.resolve(__dirname, "../package.json"), "utf8"),
) as { version: string };

const nextConfig: NextConfig = {
  // Pin the workspace root because sibling lockfiles confuse inference.
  turbopack: { root: path.resolve(__dirname) },
  // Static export -> deployed as plain HTML/CSS/JS via the Cloudflare worker.
  output: "export",
  // Cloudflare assets use html_handling: auto-trailing-slash; emit /me/index.html.
  trailingSlash: true,
  // No image optimizer in a static export.
  images: { unoptimized: true },
  // Keep the site on the repository's canonical release version.
  env: { NEXT_PUBLIC_DEVRITES_VERSION: version },
};

export default nextConfig;
