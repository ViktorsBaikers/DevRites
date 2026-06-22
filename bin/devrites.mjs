#!/usr/bin/env node
// devrites — npx entry point for the DevRites installer.
//
// A thin shim over the bundled bash installers. The npm package ships
// install.sh / uninstall.sh / update.sh alongside pack/ and scripts/, so the
// installer's network-bootstrap branch is skipped and the install runs offline,
// locked to whatever @version was invoked. The bash scripts remain the single
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

function pkgVersion() {
  try {
    return JSON.parse(readFileSync(join(root, 'package.json'), 'utf8')).version || 'unknown';
  } catch {
    return 'unknown';
  }
}

function printHelp() {
  process.stdout.write(`devrites — install the full DevRites pack into a project

Usage:
  npx devrites [install] [flags]   Install skills + agents + rules + hooks (default)
  npx devrites uninstall [flags]   Remove a DevRites install (preserves .devrites/ state)
  npx devrites update [flags]      Upgrade an existing install in place

Common flags (passed straight through to the installer):
  --target DIR          Install into DIR (default: current directory)
  --dry-run             Show the plan, change nothing
  --force               Overwrite existing non-DevRites files
  --rules-only          Install only the engineering rules
  --short-aliases=all   Also install /define /build /prove /seal
  --no-agents           Skip the review subagents

  --version             Print the devrites version
  --help                Show this help (use "<subcommand> --help" for installer-level detail)

DevRites is project-local — it never writes to ~/.claude.
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
