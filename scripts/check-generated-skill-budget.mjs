#!/usr/bin/env node
// Guard generated host skill payloads against accidental context bloat.
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('..', import.meta.url));
const base = process.argv[2] || join(root, 'pack', '.claude', 'skills');
const totalLimit = Number(process.env.DEVRITES_SKILL_TOTAL_BUDGET || 900_000);
const fileLimit = Number(process.env.DEVRITES_SKILL_FILE_BUDGET || 64_000);
const referenceFileLimit = Number(process.env.DEVRITES_REFERENCE_FILE_BUDGET || 32_000);
const routingLimit = Number(process.env.DEVRITES_SKILL_ROUTING_BUDGET || 5_200);
let total = 0;
let routingCharacters = 0;
let routingSkillCount = 0;
let failures = 0;

function fail(msg) {
  console.error(`FAIL: ${msg}`);
  failures++;
}

function frontmatter(text) {
  const match = text.match(/^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)/);
  const fields = new Map();
  if (!match) return fields;
  for (const line of match[1].split(/\r?\n/)) {
    const colon = line.indexOf(':');
    if (colon < 0 || /^[ \t]/.test(line)) continue;
    fields.set(line.slice(0, colon).trim(), line.slice(colon + 1).trim().replace(/^['"]|['"]$/g, ''));
  }
  return fields;
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
  const contents = readFileSync(file);
  const bytes = Buffer.byteLength(contents);
  total += bytes;
  const limit = file.endsWith('/SKILL.md') ? fileLimit : referenceFileLimit;
  if (bytes > limit) fail(`${relative(root, file)} is ${bytes} bytes (max ${limit})`);
  if (file.endsWith('/SKILL.md')) {
    const fields = frontmatter(contents.toString());
    if ((fields.get('disable-model-invocation') || '').toLowerCase() !== 'true') {
      routingCharacters += (fields.get('name') || '').length + (fields.get('description') || '').length;
      routingSkillCount++;
    }
  }
}
if (total > totalLimit) fail(`${relative(root, base)} markdown payload is ${total} bytes (max ${totalLimit})`);
if (routingCharacters > routingLimit) {
  fail(`${relative(root, base)} model-visible skill routing metadata is ${routingCharacters} characters (max ${routingLimit}); shorten name/description frontmatter, not on-demand skill bodies`);
}
console.log(`skill-budget: ${total} markdown bytes under ${relative(root, base) || base}`);
console.log(`skill-budget: ${routingCharacters} routing characters across ${routingSkillCount} model-visible skills (max ${routingLimit})`);
process.exit(failures ? 1 : 0);
