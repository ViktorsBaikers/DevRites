#!/usr/bin/env node
// Per-file byte ratchet for prompt/instruction docs. Use --write after intentional growth.
import { existsSync, mkdirSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';

const root = new URL('..', import.meta.url).pathname.replace(/\/$/, '');
const baselinePath = join(root, 'tests', 'instruction-size-baseline.json');
const write = process.argv.includes('--write');

function lfBytes(text) {
  return Buffer.byteLength(text.replace(/\r\n/g, '\n'));
}

function collect() {
  const out = {};
  const skills = join(root, 'pack', '.claude', 'skills');
  if (existsSync(skills)) {
    for (const d of readdirSync(skills, { withFileTypes: true })) {
      const file = join(skills, d.name, 'SKILL.md');
      if (d.isDirectory() && existsSync(file)) out[relative(root, file)] = lfBytes(readFileSync(file, 'utf8'));
    }
  }
  const agents = join(root, 'pack', '.claude', 'agents');
  if (existsSync(agents)) {
    for (const d of readdirSync(agents, { withFileTypes: true })) {
      const file = join(agents, d.name);
      if (d.isFile() && d.name.endsWith('.md')) out[relative(root, file)] = lfBytes(readFileSync(file, 'utf8'));
    }
  }
  return Object.fromEntries(Object.entries(out).sort(([a], [b]) => a.localeCompare(b)));
}

const current = collect();
if (write) {
  mkdirSync(dirname(baselinePath), { recursive: true });
  writeFileSync(baselinePath, JSON.stringify(current, null, 2) + '\n');
  console.log(`instruction-size: wrote ${relative(root, baselinePath)} (${Object.keys(current).length} files)`);
  process.exit(0);
}

if (!existsSync(baselinePath)) {
  console.error(`FAIL: missing ${relative(root, baselinePath)}; run node scripts/check-instruction-size-baseline.mjs --write`);
  process.exit(1);
}
const baseline = JSON.parse(readFileSync(baselinePath, 'utf8'));
let failures = 0;
function fail(msg) {
  console.error(`FAIL: ${msg}`);
  failures++;
}
for (const file of Object.keys(current)) {
  if (!(file in baseline)) fail(`${file} missing from baseline (${current[file]} bytes)`);
  else if (current[file] > baseline[file]) fail(`${file} grew by ${current[file] - baseline[file]} bytes (${baseline[file]} -> ${current[file]})`);
}
for (const file of Object.keys(baseline)) {
  if (!(file in current)) fail(`${file} in baseline but file is gone`);
}
console.log(`instruction-size: ${Object.keys(current).length} files checked`);
process.exit(failures ? 1 : 0);
