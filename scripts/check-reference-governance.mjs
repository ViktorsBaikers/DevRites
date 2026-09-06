#!/usr/bin/env node
// Ensure supporting skill references are reachable or explicitly time-bounded.
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { basename, dirname, join, normalize, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const defaultRoot = fileURLToPath(new URL('..', import.meta.url));
const argv = process.argv.slice(2);
function option(name, fallback) {
  const i = argv.indexOf(name);
  return i >= 0 ? argv[i + 1] : fallback;
}
const skillsDir = resolve(option('--skills-dir', join(defaultRoot, 'pack', '.claude', 'skills')));
const allowlistPath = resolve(option('--allowlist', join(defaultRoot, 'tests', 'reference-orphan-allowlist.json')));

function markdownFiles(dir) {
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const file = join(dir, entry.name);
    if (entry.isDirectory()) files.push(...markdownFiles(file));
    else if (entry.isFile() && entry.name.endsWith('.md')) files.push(normalize(file));
  }
  return files;
}

const files = markdownFiles(skillsDir);
const fileSet = new Set(files);
const byBase = new Map();
for (const file of files) byBase.set(basename(file), [...(byBase.get(basename(file)) || []), file]);
const entries = files.filter((file) => basename(file) === 'SKILL.md');
const reachable = new Set(entries);
const queue = [...entries];
const markdownLink = /\]\(([^)#]+\.md)(?:#[^)]*)?\)/g;
const backtickPath = /`([A-Za-z0-9._/\-]+\.md)`/g;

function candidates(from, token) {
  if (/^(?:https?:|mailto:)/.test(token)) return [];
  if (token.startsWith('.claude/skills/')) return [join(skillsDir, token.slice('.claude/skills/'.length))];
  if (token.includes('/')) return [resolve(dirname(from), token)];
  return byBase.get(token) || [];
}

function refsFrom(from) {
  const text = readFileSync(from, 'utf8');
  const tokens = [
    ...[...text.matchAll(markdownLink)].map((match) => match[1]),
    ...[...text.matchAll(backtickPath)].map((match) => match[1]),
  ];
  const out = new Set();
  for (const token of tokens) {
    for (const candidate of candidates(from, token)) {
      const file = normalize(candidate);
      if (fileSet.has(file) && file !== from) out.add(file);
    }
  }
  return out;
}

function isIndex(file) {
  const name = basename(file);
  return name === 'core.md' || name === 'agents.md' || name === 'index.md' || name === 'README.md';
}

function isCatalog(file) {
  const rel = relative(skillsDir, file).split(sep).join('/');
  return rel.includes('/visual-playbooks/') || rel.includes('/reference/standards/');
}

function skillOwner(file) {
  return relative(skillsDir, file).split(sep)[0];
}

while (queue.length) {
  const from = queue.shift();
  for (const file of refsFrom(from)) {
    if (!reachable.has(file)) {
      reachable.add(file);
      queue.push(file);
    }
  }
}

const allowlist = existsSync(allowlistPath) ? JSON.parse(readFileSync(allowlistPath, 'utf8')) : {};
const today = new Date().toISOString().slice(0, 10);
let failures = 0;
function fail(message) { console.error(`FAIL: ${message}`); failures++; }
for (const [file, exception] of Object.entries(allowlist)) {
  if (!exception || !exception.owner || !exception.reason || !/^\d{4}-\d{2}-\d{2}$/.test(exception.expires || '')) {
    fail(`${file}: orphan exception requires owner, reason, and YYYY-MM-DD expires`);
  } else if (exception.expires < today) {
    fail(`${file}: orphan exception expired ${exception.expires}`);
  }
}
const orphans = files.filter((file) => basename(file) !== 'SKILL.md' && !reachable.has(file));
for (const file of orphans) {
  const rel = relative(skillsDir, file).split(sep).join('/');
  if (!(rel in allowlist)) fail(`${rel}: unreachable reference`);
}
for (const file of Object.keys(allowlist)) {
  if (!orphans.some((orphan) => relative(skillsDir, orphan).split(sep).join('/') === file)) {
    fail(`${file}: orphan exception is stale; reference is reachable or gone`);
  }
}
console.log(`reference-governance: ${files.length - entries.length} references, ${orphans.length} orphan(s)`);

const tocLineThreshold = 300;
const tocHeading = /^## Contents\s*$/m;
for (const file of files) {
  if (basename(file) === 'SKILL.md') continue;
  const text = readFileSync(file, 'utf8');
  const lineCount = text.split(/\r?\n/).length;
  if (lineCount > tocLineThreshold && !tocHeading.test(text)) {
    const rel = relative(skillsDir, file).split(sep).join('/');
    fail(`${rel}: reference over ${tocLineThreshold} lines needs a ## Contents table of contents`);
  }
}

// One hop from SKILL.md, except named indexes (core.md, agents.md, README.md,
// index.md) which may point onward. A skill-local reference must not be the
// only path to another file in that same skill.
for (const skill of entries) {
  const owner = skillOwner(skill);
  const hop1 = refsFrom(skill);
  for (const via of hop1) {
    if (isIndex(via) || basename(via) === 'SKILL.md') continue;
    if (skillOwner(via) !== owner) continue;
    for (const hidden of refsFrom(via)) {
      if (hidden === skill || hop1.has(hidden) || isIndex(hidden) || basename(hidden) === 'SKILL.md') continue;
      if (skillOwner(hidden) !== owner) continue;
      if (isCatalog(hidden) || isCatalog(via)) continue;
      const skillRel = relative(skillsDir, skill).split(sep).join('/');
      const viaRel = relative(skillsDir, via).split(sep).join('/');
      const hiddenRel = relative(skillsDir, hidden).split(sep).join('/');
      fail(`${skillRel}: two-hop via ${viaRel} -> ${hiddenRel}`);
    }
  }
}

process.exit(failures ? 1 : 0);
