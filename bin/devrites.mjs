#!/usr/bin/env node
// devrites - npx entry point for the engine-owned DevRites installer.
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { chmodSync, existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const root = dirname(here);

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
  npx devrites [install] [flags]   Install skills + agents + standards + hooks (default)
  npx devrites uninstall [flags]   Remove a DevRites install (preserves .devrites/ state)
  npx devrites update [flags]      Upgrade an existing install in place

Common flags:
  --target DIR          Install into DIR (default: current directory)
  --dry-run             Show the plan, change nothing
  --force               Overwrite existing non-DevRites files
  --no-codex            Skip Codex support files (.agents, .codex, AGENTS.md)
  --short-aliases=all   Also install /define /build /prove /seal
  --no-agents           Skip the review subagents
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
  const envEngine = process.env.DEVRITES_ENGINE_CLI || process.env.DEVRITES_CLI;
  if (envEngine && existsSync(envEngine)) return envEngine;

  const localBuilt = join(root, 'engine', 'devrites-engine');
  if (existsSync(localBuilt)) return localBuilt;

  const downloaded = await downloadEngine();
  if (downloaded) return downloaded;

  const engineDir = join(root, 'engine');
  if (existsSync(join(engineDir, 'go.mod'))) {
    const goBin = findGo();
    const out = join(mkdtempSync(join(tmpdir(), 'devrites-engine-')), process.platform === 'win32' ? 'devrites-engine.exe' : 'devrites-engine');
    const tag = `v${pkgVersion().replace(/^v/, '')}`;
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
  const tag = process.env.DEVRITES_REF || `v${pkgVersion().replace(/^v/, '')}`;
  const name = `devrites-${platform}-${arch}${platform === 'windows' ? '.exe' : ''}`;
  const dir = mkdtempSync(join(tmpdir(), 'devrites-engine-'));
  const out = join(dir, platform === 'windows' ? 'devrites-engine.exe' : 'devrites-engine');
  const url = `https://github.com/${repo}/releases/download/${tag}/${name}`;
  if (!await downloadURL(url, out, 0o755)) return null;
  if (!await downloadURL(`${url}.sha256`, `${out}.sha256`)) return null;
  const want = readFileSync(`${out}.sha256`, 'utf8').trim().split(/\s+/)[0];
  const got = createHash('sha256').update(readFileSync(out)).digest('hex');
  if (!want || got !== want) {
    rmSync(out, { force: true });
    return null;
  }
  chmodSync(out, 0o755);
  return out;
}

async function downloadURL(url, out, mode = 0o644) {
  try {
    const res = await fetch(url);
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    writeFileSync(out, Buffer.from(await res.arrayBuffer()), { mode });
    return true;
  } catch {
    rmSync(out, { force: true });
    return false;
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

const engine = await acquireEngine();
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
  console.error('devrites: devrites-engine was not found and could not be acquired.');
}
process.exit(127);
