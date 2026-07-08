#!/usr/bin/env node
// devrites — npx entry point for the DevRites installer.
//
// A thin shim over the bundled bash installers. The npm package ships
// install.sh / uninstall.sh / update.sh alongside pack/ and scripts/, so the
// installer's pack-bootstrap branch is skipped and the install is locked to
// whatever @version was invoked. The engine binary may still be fetched or built
// unless --no-binary is passed. The bash scripts remain the single
// source of truth for all install logic, flags, manifest, and guards.
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { existsSync, readFileSync } from 'node:fs';

const here = dirname(fileURLToPath(import.meta.url));
const root = dirname(here); // bin/ -> package root

const SUBCOMMANDS = {
  install: 'install.sh',
  add: 'install.sh',
  uninstall: 'uninstall.sh',
  remove: 'uninstall.sh',
  update: 'update.sh',
  upgrade: 'update.sh',
};

const ENGINE_COMMANDS = new Set([
  'status',
  'reindex',
  'readiness',
  'seal',
  'spec-validate',
  'spec-skeleton',
  'check-acceptance',
  'footprint',
  'evidence-fresh',
  'coverage',
  'doubt-coverage',
  'budget',
  'preamble',
  'progress',
  'stuck',
  'tick-afk',
  'build-readiness',
  'analyze',
  'mutation-gate',
  'test-integrity',
  'review-integrity',
  'package-existence',
  'reconcile',
  'resolve',
  'close-out',
  'archive-search',
  'ledger',
  'validate-pack',
  'harness-matrix',
  'learnings',
  'conventions',
  'extensions',
  'overrides',
  'doctor',
  'migrate',
  'version',
  'hook',
  'help',
]);

function pkgVersion() {
  try {
    return JSON.parse(readFileSync(join(root, 'package.json'), 'utf8')).version || 'unknown';
  } catch {
    return 'unknown';
  }
}

function printHelp() {
  process.stdout.write(`devrites — install the DevRites pack into a project

Usage:
  npx devrites [install] [flags]   Install skills + agents + standards + hooks (default)
  npx devrites uninstall [flags]   Remove a DevRites install (preserves .devrites/ state)
  npx devrites update [flags]      Upgrade an existing install in place

Common flags (passed straight through to the installer):
  --target DIR          Install into DIR (default: current directory)
  --dry-run             Show the plan, change nothing
  --force               Overwrite existing non-DevRites files
  --no-codex            Skip Codex support files (.agents, .codex, AGENTS.md)
  --short-aliases=all   Also install /define /build /prove /seal
  --no-agents           Skip the review subagents
  --no-skills           Skip skills and bundled engineering standards
  --no-binary           Skip the devrites-engine control-plane binary
  --no-rules            Deprecated no-op; standards ship inside devrites-lib
  --rules-only          Deprecated no-op; installs normally for compatibility

  --version             Print the devrites version
  --help                Show this help (use "<subcommand> --help" for installer-level detail)

DevRites is project-local for agent files — it never writes to ~/.claude or ~/.codex.
The installer also manages a global devrites-engine binary unless --no-binary is set.
Requires bash (Git Bash or WSL on Windows). No-Node fallback:
  curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/install.sh | bash
`);
}

const argv = process.argv.slice(2);
const first = argv[0];

if (first === '--version' || first === '-v') {
  process.stdout.write(pkgVersion() + '\n');
  process.exit(0);
}
if (first === '--help' || first === '-h') {
  printHelp();
  process.exit(0);
}

if (first && ENGINE_COMMANDS.has(first)) {
  const candidates = [
    process.env.DEVRITES_ENGINE_CLI,
    process.env.DEVRITES_CLI,
    join(root, 'engine', 'devrites'),
    'devrites-engine',
  ].filter(Boolean);
  let lastError = null;
  for (const engine of candidates) {
    const engineRes = spawnSync(engine, argv, { stdio: 'inherit', cwd: process.cwd() });
    if (!engineRes.error) {
      process.exit(engineRes.status === null ? 1 : engineRes.status);
    }
    if (engineRes.error.code !== 'ENOENT') {
      lastError = engineRes.error;
      break;
    }
    lastError = engineRes.error;
  }
  if (lastError && lastError.code !== 'ENOENT') {
    console.error('devrites: failed to launch devrites-engine:', lastError.message);
  } else {
    console.error('devrites: devrites-engine was not found on PATH.');
    console.error('devrites: run `npx devrites install`, or reinstall without --no-binary.');
  }
  process.exit(127);
}

// Route the subcommand; default to install when the first arg is a flag or absent.
let sub = 'install';
let rest = argv;
if (first && Object.prototype.hasOwnProperty.call(SUBCOMMANDS, first)) {
  sub = first;
  rest = argv.slice(1);
}

const script = join(root, SUBCOMMANDS[sub]);
if (!existsSync(script)) {
  console.error(`devrites: bundled ${SUBCOMMANDS[sub]} not found at ${script}`);
  console.error('devrites: the package looks incomplete — reinstall, or use the curl | bash installer.');
  process.exit(1);
}

const res = spawnSync('bash', [script, ...rest], { stdio: 'inherit', cwd: process.cwd() });

if (res.error) {
  if (res.error.code === 'ENOENT') {
    console.error('devrites: `bash` was not found on your PATH.');
    console.error('DevRites is bash-based. On Windows, run inside Git Bash or WSL,');
    console.error('or use the no-Node installer:');
    console.error('  curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/install.sh | bash');
    process.exit(127);
  }
  console.error('devrites: failed to launch installer:', res.error.message);
  process.exit(1);
}

process.exit(res.status === null ? 1 : res.status);
