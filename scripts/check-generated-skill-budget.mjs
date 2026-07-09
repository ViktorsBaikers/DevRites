#!/usr/bin/env node
// Guard generated host skill payloads against accidental context bloat.
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { join, relative } from 'node:path';

const root = new URL('..', import.meta.url).pathname.replace(/\/$/, '');
const base = process.argv[2] || join(root, 'pack', '.claude', 'skills');
const totalLimit = Number(process.env.DEVRITES_SKILL_TOTAL_BUDGET || 900_000);
const fileLimit = Number(process.env.DEVRITES_SKILL_FILE_BUDGET || 64_000);
const referenceFileLimit = Number(process.env.DEVRITES_REFERENCE_FILE_BUDGET || 32_000);
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
function markdownFiles(dir) {
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const file = join(dir, entry.name);
    if (entry.isDirectory()) files.push(...markdownFiles(file));
    else if (entry.isFile() && entry.name.endsWith('.md')) files.push(file);
  }
  return files;
}

for (const file of markdownFiles(base)) {
  const bytes = Buffer.byteLength(readFileSync(file));
  total += bytes;
  const limit = file.endsWith('/SKILL.md') ? fileLimit : referenceFileLimit;
  if (bytes > limit) fail(`${relative(root, file)} is ${bytes} bytes (max ${limit})`);
}
if (total > totalLimit) fail(`${relative(root, base)} markdown payload is ${total} bytes (max ${totalLimit})`);
console.log(`skill-budget: ${total} markdown bytes under ${relative(root, base) || base}`);
process.exit(failures ? 1 : 0);
