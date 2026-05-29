// DevRites edge worker: serve the install script to CLI clients, the website to browsers.
//
//   curl -fsSL https://devrites.com | bash      -> install.sh (curl/wget UA at "/")
//   curl -fsSL https://devrites.com/install.sh  -> install.sh (always)
//   browser visit to https://devrites.com/      -> the static site
//
// Everything else is served straight from static assets (this worker only runs
// for "/" and "/install.sh" per wrangler.jsonc `run_worker_first`).

const INSTALL_URL = "https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/install.sh";
const CLI_UA = /\bcurl\b|\bwget\b|\bhttpie\b|libcurl|powershell|node-fetch|\bgot\b|\baxios\b/i;

async function serveInstall() {
  const upstream = await fetch(INSTALL_URL, { cf: { cacheTtl: 300, cacheEverything: true } });
  if (!upstream.ok) {
    return new Response("# DevRites install script is unavailable right now.\n", {
      status: 502,
      headers: { "content-type": "text/plain; charset=utf-8" },
    });
  }
  return new Response(upstream.body, {
    status: 200,
    headers: {
      "content-type": "text/x-shellscript; charset=utf-8",
      "cache-control": "public, max-age=300",
      "x-content-type-options": "nosniff",
    },
  });
}

export default {
  async fetch(request, env) {
    const { pathname } = new URL(request.url);

    if (pathname === "/install.sh") return serveInstall();

    if (pathname === "/") {
      const ua = request.headers.get("user-agent") || "";
      if (CLI_UA.test(ua)) return serveInstall();
    }

    // Browser / anything else -> static assets.
    return env.ASSETS.fetch(request);
  },
};
