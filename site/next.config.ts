import type { NextConfig } from "next";
import path from "node:path";

const nextConfig: NextConfig = {
  // Pin the workspace root — the repo has sibling lockfiles that confuse inference.
  turbopack: { root: path.resolve(__dirname) },
  // Static export -> deployed as plain HTML/CSS/JS via the Cloudflare worker.
  output: "export",
  // Cloudflare assets use html_handling: auto-trailing-slash; emit /me/index.html.
  trailingSlash: true,
  // No image optimizer in a static export.
  images: { unoptimized: true },
};

export default nextConfig;
