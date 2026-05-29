// DevRites edge worker: serve install / update / uninstall scripts to the shell,
// the website to browsers.
//
//   curl -fsSL https://devrites.com | bash            -> install   (curl/wget UA at "/")
//   curl -fsSL https://devrites.com/install.sh | bash -> install
//   curl -fsSL https://devrites.com/update | bash     -> update    (update.sh)
//   curl -fsSL https://devrites.com/remove | bash     -> uninstall (uninstall.sh)
//   browser visit to https://devrites.com/            -> the static site
//
// Flags pass straight through, e.g.
//   curl -fsSL https://devrites.com/update | bash -s -- --check
//   curl -fsSL https://devrites.com | bash -s -- --target ./my-project
//
// The worker only runs for the script paths (see wrangler.jsonc run_worker_first);
// every other request is served directly from static assets.

const RAW = "https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main";
const SCRIPT = {
  install: RAW + "/install.sh",
  update: RAW + "/update.sh",
  uninstall: RAW + "/uninstall.sh",
};
const CLI_UA = /\bcurl\b|\bwget\b|\bhttpie\b|libcurl|powershell|node-fetch|\bgot\b|\baxios\b/i;

// path -> which script (script-only endpoints, served to any client)
const ROUTES = {
  "/install.sh": "install",
  "/update": "update",
  "/update.sh": "update",
  "/remove": "uninstall",
  "/uninstall": "uninstall",
  "/uninstall.sh": "uninstall",
};

async function serveScript(which) {
  const upstream = await fetch(SCRIPT[which], { cf: { cacheTtl: 300, cacheEverything: true } });
  if (!upstream.ok) {
    return new Response("# DevRites " + which + " script is unavailable right now.\n", {
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

    if (ROUTES[pathname]) return serveScript(ROUTES[pathname]);

    // Root: install script for CLI clients, website for browsers.
    if (pathname === "/" && CLI_UA.test(request.headers.get("user-agent") || "")) {
      return serveScript("install");
    }

    return env.ASSETS.fetch(request);
  },
};
