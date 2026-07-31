#!/usr/bin/env node
// devrites - npx entry point for the engine-owned DevRites installer.
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { chmodSync, createReadStream, existsSync, mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { open } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const root = dirname(here);
const MAX_ENGINE_BYTES = 64 * 1024 * 1024;
const MAX_SIDECAR_BYTES = 4 * 1024;
const MAX_REDIRECTS = 5;
const REDIRECT_STATUSES = new Set([301, 302, 303, 307, 308]);
const temporaryDirs = new Set();
let acquisitionFailure = null;

class DownloadError extends Error {}

function privateTempDir() {
  const dir = mkdtempSync(join(tmpdir(), 'devrites-engine-'));
  temporaryDirs.add(dir);
  return dir;
}

function cleanupTemporaryDirs() {
  for (const dir of temporaryDirs) rmSync(dir, { force: true, recursive: true });
  temporaryDirs.clear();
}

process.on('exit', cleanupTemporaryDirs);

const SUBCOMMAND_ALIASES = new Map([
  ['install', 'install'],
  ['add', 'install'],
  ['update', 'update'],
  ['upgrade', 'update'],
  ['uninstall', 'uninstall'],
  ['remove', 'uninstall'],
]);

function pkgVersion() {
  try {
    return JSON.parse(readFileSync(join(root, 'package.json'), 'utf8')).version || 'unknown';
  } catch {
    return 'unknown';
  }
}

function printHelp() {
  process.stdout.write(`devrites - install the DevRites pack into a project

Usage:
  npx devrites [install] [flags]   Install skills, hook-free native agents, standards, and host settings (default)
  npx devrites uninstall [flags]   Remove a DevRites install (preserves .devrites/ state)
  npx devrites update [flags]      Upgrade an existing install in place

Common flags:
  --target DIR          Install into DIR (default: current directory)
  --dry-run             Show the plan, change nothing
  --force               Overwrite existing non-DevRites files
  --no-codex            Skip Codex support files (.agents, .codex, AGENTS.md)
  --short-aliases=all   Also install /define /build /prove /seal
  --no-agents           Skip hook-free native specialist profiles
  --no-skills           Skip skills and bundled engineering standards
  --no-binary           Skip installing the global devrites-engine binary
  --no-rules            Deprecated no-op; standards ship inside devrites-lib
  --rules-only          Deprecated no-op; installs normally for compatibility

  --version             Print the devrites version
  --help                Show this help
`);
}

function run(command, args, options = {}) {
  return spawnSync(command, args, { stdio: options.stdio || 'inherit', cwd: options.cwd || process.cwd(), env: options.env || process.env });
}

function firstWorking(candidates, args, env = process.env) {
  let lastError = null;
  for (const candidate of candidates.filter(Boolean)) {
    const res = run(candidate, args, { env });
    if (!res.error) process.exit(res.status === null ? 1 : res.status);
    lastError = res.error;
    if (res.error.code !== 'ENOENT') break;
  }
  return lastError;
}

async function acquireEngine() {
  const repo = process.env.DEVRITES_REPO || 'ViktorsBaikers/DevRites';
  const tag = normalizeReleaseTag(process.env.DEVRITES_REF || pkgVersion());
  if (!validRepository(repo) || !tag) throw new Error('DevRites repository and release tag must identify an exact release');
  const envEngine = process.env.DEVRITES_ENGINE_CLI || process.env.DEVRITES_CLI;
  if (envEngine && existsSync(envEngine)) return envEngine;

  const downloaded = await downloadEngine();
  if (downloaded) return downloaded;

  const engineDir = join(root, 'engine');
  if (existsSync(join(engineDir, 'go.mod'))) {
    const goBin = findGo();
    const out = join(privateTempDir(), process.platform === 'win32' ? 'devrites-engine.exe' : 'devrites-engine');
    if (goBin) {
      const env = { ...process.env, GOCACHE: join(dirname(out), 'go-cache'), CGO_ENABLED: '0' };
      const build = run(goBin, [
        'build',
        '-trimpath',
        '-ldflags',
        `-s -w -X github.com/devrites/devrites/internal/version.Version=${tag}`,
        '-o',
        out,
        '.',
      ], { cwd: engineDir, stdio: 'ignore', env });
      if (!build.error && build.status === 0) return out;
    }
  }

  return 'devrites-engine';
}

function findGo() {
  const direct = run('go', ['version'], { stdio: 'ignore' });
  if (!direct.error && direct.status === 0) return 'go';
  return null;
}

async function downloadEngine() {
  const platform = process.platform === 'darwin' ? 'darwin' : process.platform === 'linux' ? 'linux' : process.platform === 'win32' ? 'windows' : null;
  const arch = process.arch === 'x64' ? 'amd64' : process.arch === 'arm64' ? 'arm64' : null;
  if (!platform || !arch) return null;
  const repo = process.env.DEVRITES_REPO || 'ViktorsBaikers/DevRites';
  if (!validRepository(repo)) return null;
  const tag = normalizeReleaseTag(process.env.DEVRITES_REF || pkgVersion());
  if (!tag) return null;
  const name = `devrites-${platform}-${arch}${platform === 'windows' ? '.exe' : ''}`;
  const dir = privateTempDir();
  const out = join(dir, platform === 'windows' ? 'devrites-engine.exe' : 'devrites-engine');
  const url = `https://github.com/${repo}/releases/download/${tag}/${name}`;
  const assetFailure = await downloadURL(url, out, MAX_ENGINE_BYTES, 120_000, 0o755);
  if (assetFailure) {
    acquisitionFailure = `release ${tag} asset ${name}: ${assetFailure}`;
    return null;
  }
  const sidecarFailure = await downloadURL(`${url}.sha256`, `${out}.sha256`, MAX_SIDECAR_BYTES, 30_000);
  if (sidecarFailure) {
    acquisitionFailure = `release ${tag} asset ${name}.sha256: ${sidecarFailure}`;
    rmSync(out, { force: true });
    return null;
  }
  let want;
  let got;
  try {
    const match = readFileSync(`${out}.sha256`, 'utf8').match(/^([0-9A-Fa-f]{64})[ \t]+([^\r\n]+)\r?\n?$/);
    want = match && match[2] === name ? match[1].toLowerCase() : null;
    got = await hashFile(out);
  } catch {
    acquisitionFailure = `release ${tag} asset ${name}: checksum read failed`;
    rmSync(out, { force: true });
    rmSync(`${out}.sha256`, { force: true });
    return null;
  }
  if (!want || got !== want) {
    acquisitionFailure = `release ${tag} asset ${name}: checksum failed`;
    rmSync(out, { force: true });
    rmSync(`${out}.sha256`, { force: true });
    return null;
  }
  try {
    chmodSync(out, 0o755);
  } catch {
    acquisitionFailure = `release ${tag} asset ${name}: executable preparation failed`;
    rmSync(out, { force: true });
    rmSync(`${out}.sha256`, { force: true });
    return null;
  }
  return out;
}

function normalizeReleaseTag(value) {
  const version = String(value).replace(/^v/, '');
  const semver = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/;
  const match = version.match(semver);
  if (!match || (match[4] && match[4].split('.').some((part) => /^\d+$/.test(part) && part.length > 1 && part.startsWith('0')))) return null;
  return `v${version}`;
}

function validRepository(value) {
  return /^[A-Za-z0-9][A-Za-z0-9_.-]*\/[A-Za-z0-9][A-Za-z0-9_.-]*$/.test(value);
}

async function hashFile(path) {
  const hash = createHash('sha256');
  for await (const chunk of createReadStream(path)) hash.update(chunk);
  return hash.digest('hex');
}

async function downloadURL(url, out, maxBytes, timeoutMs, mode = 0o644) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  let file;
  try {
    let currentURL = new URL(url);
    let res;
    for (let redirects = 0; ; redirects += 1) {
      if (currentURL.protocol !== 'https:') throw new DownloadError('redirect failed');
      res = await fetch(currentURL.href, { redirect: 'manual', signal: controller.signal });
      if (!REDIRECT_STATUSES.has(res.status)) break;
      const location = res.headers.get('location');
      await res.body?.cancel();
      if (!location) throw new DownloadError('redirect failed');
      if (redirects >= MAX_REDIRECTS) throw new DownloadError('redirect limit failed');
      currentURL = new URL(location, currentURL);
    }
    if (!res.ok) throw new DownloadError(`HTTP status ${res.status} failed`);
    const contentLength = Number(res.headers.get('content-length'));
    if (Number.isFinite(contentLength) && contentLength > maxBytes) throw new DownloadError('size limit failed');
    if (!res.body) throw new DownloadError('response body failed');
    try {
      file = await open(out, 'wx', mode);
    } catch {
      throw new DownloadError('local write failed');
    }
    let bytes = 0;
    for await (const chunk of res.body) {
      bytes += chunk.byteLength;
      if (bytes > maxBytes) throw new DownloadError('size limit failed');
      try {
        await file.writeFile(Buffer.from(chunk));
      } catch {
        throw new DownloadError('local write failed');
      }
    }
    try {
      await file.close();
      file = undefined;
    } catch {
      throw new DownloadError('local write failed');
    }
    return null;
  } catch (error) {
    await file?.close().catch(() => {});
    rmSync(out, { force: true });
    if (error instanceof DownloadError) return error.message;
    return controller.signal.aborted ? 'timeout failed' : 'network failed';
  } finally {
    clearTimeout(timeout);
  }
}

const argv = process.argv.slice(2);
const first = argv[0];

if (first === '--version' || first === '-v') {
  process.stdout.write(`${pkgVersion()}\n`);
  process.exit(0);
}
if (first === '--help' || first === '-h') {
  printHelp();
  process.exit(0);
}

let command = 'install';
let rest = argv;
if (first && SUBCOMMAND_ALIASES.has(first)) {
  command = SUBCOMMAND_ALIASES.get(first);
  rest = argv.slice(1);
} else if (first && !first.startsWith('-')) {
  command = first;
  rest = argv.slice(1);
}

let engine;
try {
  engine = await acquireEngine();
} catch (error) {
  console.error(`devrites: ${error.message}`);
  process.exit(1);
}
const payloadDir = process.env.DEVRITES_HOST_ARTIFACT_DIR || join(root, 'pack', 'generated');
const installerCommand = command === 'install' || command === 'update' || command === 'uninstall';
const engineEnv = { ...process.env, DEVRITES_ENGINE_CLI: engine };
const args = [command];
if (installerCommand) {
  args.push('--source-dir', root);
}
if (command === 'install' || command === 'update') {
  args.push('--payload-dir', payloadDir);
}
args.push(...rest);

const candidates = [engine, 'devrites-engine'];
const lastError = firstWorking(candidates, args, engineEnv);
if (lastError && lastError.code !== 'ENOENT') {
  console.error('devrites: failed to launch devrites-engine:', lastError.message);
} else {
  console.error(`devrites: devrites-engine was not found and could not be acquired${acquisitionFailure ? ` (${acquisitionFailure})` : ''}.`);
}
process.exit(127);
