#!/usr/bin/env node
// Per-file and aggregate byte ratchet for canonical prompt/instruction docs.
import { existsSync, mkdirSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';

const defaultRoot = new URL('..', import.meta.url).pathname.replace(/\/$/, '');
const argv = process.argv.slice(2);
function option(name, fallback) {
  const i = argv.indexOf(name);
  return i >= 0 ? argv[i + 1] : fallback;
}
const root = resolve(option('--root', defaultRoot));
const baselinePath = resolve(option('--baseline', join(root, 'tests', 'instruction-size-baseline.json')));
const write = argv.includes('--write');

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

function collect() {
  const roots = [
    join(root, 'pack', '.claude', 'skills'),
    join(root, 'pack', '.claude', 'agents'),
    join(root, 'pack', '.claude', 'rules'),
  ];
  const out = {};
  for (const file of roots.flatMap(markdownFiles)) {
    out[relative(root, file)] = lfBytes(readFileSync(file, 'utf8'));
  }
  return Object.fromEntries(Object.entries(out).sort(([a], [b]) => a.localeCompare(b)));
}

const files = collect();
const totalBytes = Object.values(files).reduce((sum, bytes) => sum + bytes, 0);
const current = { version: 2, total_bytes: totalBytes, files };
if (write) {
  mkdirSync(dirname(baselinePath), { recursive: true });
  writeFileSync(baselinePath, JSON.stringify(current, null, 2) + '\n');
  console.log(`instruction-size: wrote ${relative(root, baselinePath)} (${Object.keys(files).length} files, ${totalBytes} bytes)`);
  process.exit(0);
}

if (!existsSync(baselinePath)) {
  console.error(`FAIL: missing ${relative(root, baselinePath)}; run node scripts/check-instruction-size-baseline.mjs --write`);
  process.exit(1);
}
const baseline = JSON.parse(readFileSync(baselinePath, 'utf8'));
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
console.log(`instruction-size: ${Object.keys(files).length} files checked, ${totalBytes} bytes`);
process.exit(failures ? 1 : 0);
