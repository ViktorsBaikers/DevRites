#!/usr/bin/env node
// Advisory skill-pack pruning audit. It reports likely sediment; it never blocks release.
import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

const root = new URL('..', import.meta.url).pathname.replace(/\/$/, '');
const skillsDir = join(root, 'pack', '.claude', 'skills');
const standardsDir = join(root, 'pack', '.claude', 'skills', 'devrites-lib', 'reference', 'standards');
const docsResearch = join(root, 'docs', 'research');
const staleTokens = [
  'grill-with-docs',
  'setup-matt-pocock-skills',
  '/domain-modeling',
  '/improve-codebase-architecture',
  'skills.sh',
  '.claude-plugin',
];
const warnings = [];

function read(path) {
  return readFileSync(path, 'utf8');
}

function warn(path, message) {
  warnings.push(`${relative(root, path)}: ${message}`);
}

function walk(dir, out = []) {
  if (!existsSync(dir)) return out;
  for (const entry of readdirSync(dir)) {
    if (entry.startsWith('.')) continue;
    const path = join(dir, entry);
    if (statSync(path).isDirectory()) walk(path, out);
    else if (path.endsWith('.md')) out.push(path);
  }
  return out;
}

for (const skillDir of readdirSync(skillsDir).sort()) {
  if (skillDir.startsWith('.')) continue;
  const path = join(skillsDir, skillDir, 'SKILL.md');
  if (!existsSync(path)) continue;
  const text = read(path);
  const lineCount = text.split(/\r?\n/).length;
  if (lineCount > 220) warn(path, `${lineCount} lines; consider disclosed references before adding more`);

  const negations = (text.match(/\b(DO NOT|NEVER|NO |Not for|Don't)\b/g) || []).length;
  if (negations > 30) warn(path, `${negations} negation markers; check whether positive target rules would be shorter`);

  for (const token of staleTokens) {
    if (text.includes(token)) warn(path, `stale external workflow token: ${token}`);
  }
}

const seenSentences = new Map();
for (const path of walk(standardsDir)) {
  const text = read(path)
    .replace(/```[\s\S]*?```/g, '')
    .replace(/\[[^\]]+\]\([^)]+\)/g, '');
  const prose = text
    .split(/\r?\n/)
    .filter((line) => !line.trim().startsWith('|'))
    .join('\n');
  for (const sentence of prose.split(/(?<=[.!?])\s+/)) {
    const normalized = sentence.replace(/\s+/g, ' ').trim();
    if (normalized.length < 90) continue;
    const key = normalized.toLowerCase();
    const paths = seenSentences.get(key) || new Set();
    paths.add(relative(root, path));
    seenSentences.set(key, paths);
  }
}

for (const [sentence, paths] of seenSentences) {
  if (paths.size > 1) {
    warnings.push(`standards duplicate sentence across ${[...paths].join(', ')}: ${sentence.slice(0, 110)}...`);
  }
}

for (const path of walk(docsResearch)) {
  const text = read(path);
  const isAdoption = /adoption/i.test(path);
  if (isAdoption && !/(Source reviewed:|\*\*Source:\*\*)/i.test(text)) {
    warn(path, 'source-intake note should record source, commit/date, and files read');
  }
  if (isAdoption && !/^## Rejected\b/m.test(text)) {
    warn(path, 'source-intake note should include rejected ideas');
  }
}

if (!warnings.length) {
  console.log('skill-pruning-audit: no advisory findings');
  process.exit(0);
}

console.log(`skill-pruning-audit: ${warnings.length} advisory finding(s)`);
for (const warning of warnings.slice(0, 40)) {
  console.log(`warn: ${warning}`);
}
if (warnings.length > 40) console.log(`warn: ${warnings.length - 40} more omitted`);
