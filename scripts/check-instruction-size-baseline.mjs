#!/usr/bin/env node
// Track canonical instruction files individually and skills as a group.
import { existsSync, mkdirSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const defaultRoot = fileURLToPath(new URL('..', import.meta.url));
const argv = process.argv.slice(2);
function option(name, fallback) {
  const i = argv.indexOf(name);
  return i >= 0 ? argv[i + 1] : fallback;
}
const root = resolve(option('--root', defaultRoot));
const baselinePath = resolve(option('--baseline', join(root, 'tests', 'instruction-size-baseline.json')));
const write = argv.includes('--write');
const ratchetLimit = 860_000;

function lfBytes(text) {
  return Buffer.byteLength(text.replace(/\r\n/g, '\n'));
}

function markdownFiles(dir) {
  if (!existsSync(dir)) return [];
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const file = join(dir, entry.name);
    if (entry.isDirectory()) files.push(...markdownFiles(file));
    else if (entry.isFile() && entry.name.endsWith('.md')) files.push(file);
  }
  return files;
}

function collect(roots) {
  const out = {};
  for (const file of roots.flatMap(markdownFiles)) {
    out[relative(root, file)] = lfBytes(readFileSync(file, 'utf8'));
  }
  return Object.fromEntries(Object.entries(out).sort(([a], [b]) => a.localeCompare(b)));
}

const skillRoot = join(root, 'pack', '.claude', 'skills');
const files = collect([
  skillRoot,
  join(root, 'pack', '.claude', 'agents'),
  join(root, 'pack', '.claude', 'rules'),
]);
const skillFiles = collect([skillRoot]);
const totalBytes = Object.values(skillFiles).reduce((sum, bytes) => sum + bytes, 0);
const current = { version: 2, total_bytes: totalBytes, files };
if (totalBytes > ratchetLimit) {
  console.error(`FAIL: canonical skill Markdown is ${totalBytes} bytes (ratchet ${ratchetLimit})`);
  process.exit(1);
}
if (write) {
  mkdirSync(dirname(baselinePath), { recursive: true });
  writeFileSync(baselinePath, JSON.stringify(current, null, 2) + '\n');
  console.log(
    `instruction-size: wrote ${relative(root, baselinePath)} (${Object.keys(files).length} instruction files, ${totalBytes} skill bytes)`,
  );
  process.exit(0);
}

if (!existsSync(baselinePath)) {
  console.error(`FAIL: missing ${relative(root, baselinePath)}; run node scripts/check-instruction-size-baseline.mjs --write`);
  process.exit(1);
}
let baseline;
try {
  baseline = JSON.parse(readFileSync(baselinePath, 'utf8'));
} catch (error) {
  console.error(`FAIL: invalid ${relative(root, baselinePath)}: ${error.message}`);
  process.exit(1);
}
if (baseline.version !== 2 || !baseline.files || !Number.isInteger(baseline.total_bytes)) {
  console.error(`FAIL: ${relative(root, baselinePath)} uses an obsolete schema; regenerate it with --write`);
  process.exit(1);
}
let failures = 0;
function fail(msg) {
  console.error(`FAIL: ${msg}`);
  failures++;
}
for (const file of Object.keys(files)) {
  if (!(file in baseline.files)) fail(`${file} missing from baseline (${files[file]} bytes)`);
  else if (files[file] > baseline.files[file]) fail(`${file} grew by ${files[file] - baseline.files[file]} bytes (${baseline.files[file]} -> ${files[file]})`);
}
for (const file of Object.keys(baseline.files)) {
  if (!(file in files)) fail(`${file} in baseline but file is gone`);
}
if (totalBytes > baseline.total_bytes) {
  fail(`aggregate instruction payload grew by ${totalBytes - baseline.total_bytes} bytes (${baseline.total_bytes} -> ${totalBytes})`);
}
console.log(`instruction-size: ${Object.keys(files).length} instruction files checked, ${totalBytes} skill bytes`);
process.exit(failures ? 1 : 0);
