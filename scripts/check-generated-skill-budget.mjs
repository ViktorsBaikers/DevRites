#!/usr/bin/env node
// Guard generated host skill payloads against accidental context bloat.
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { join, relative } from 'node:path';

const root = new URL('..', import.meta.url).pathname.replace(/\/$/, '');
const base = process.argv[2] || join(root, 'pack', '.claude', 'skills');
const totalLimit = Number(process.env.DEVRITES_SKILL_TOTAL_BUDGET || 900_000);
const fileLimit = Number(process.env.DEVRITES_SKILL_FILE_BUDGET || 64_000);
let total = 0;
let failures = 0;

function fail(msg) {
  console.error(`FAIL: ${msg}`);
  failures++;
}

if (!existsSync(base)) {
  console.log(`skill-budget: skip missing ${base}`);
  process.exit(0);
}
for (const dirent of readdirSync(base, { withFileTypes: true })) {
  if (!dirent.isDirectory()) continue;
  const file = join(base, dirent.name, 'SKILL.md');
  if (!existsSync(file)) continue;
  const bytes = Buffer.byteLength(readFileSync(file));
  total += bytes;
  if (bytes > fileLimit) fail(`${relative(root, file)} is ${bytes} bytes (max ${fileLimit})`);
}
if (total > totalLimit) fail(`${relative(root, base)} SKILL.md payload is ${total} bytes (max ${totalLimit})`);
console.log(`skill-budget: ${total} bytes under ${relative(root, base) || base}`);
process.exit(failures ? 1 : 0);
