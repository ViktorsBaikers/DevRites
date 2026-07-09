#!/usr/bin/env node
// Advisory pruning audit for skill bloat/no-op risk. Never blocks validate.
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

const root = new URL('..', import.meta.url).pathname.replace(/\/$/, '');
const skillsDir = join(root, 'pack', '.claude', 'skills');
const quiet = process.argv.includes('--quiet');

function frontmatter(src) {
  const m = src.match(/^---\n([\s\S]*?)\n---/);
  const out = {};
  if (!m) return out;
  for (const line of m[1].split(/\r?\n/)) {
    const i = line.indexOf(':');
    if (i > 0) out[line.slice(0, i).trim()] = line.slice(i + 1).trim().replace(/^['"]|['"]$/g, '');
  }
  return out;
}

function refs(src) {
  return [...src.matchAll(/\]\(([^)]+\.md)\)/g)].map((m) => m[1]);
}

const warnings = [];
for (const dir of readdirSync(skillsDir).sort()) {
  const file = join(skillsDir, dir, 'SKILL.md');
  try { if (!statSync(file).isFile()) continue; } catch { continue; }
  const src = readFileSync(file, 'utf8');
  const fm = frontmatter(src);
  const lines = src.split(/\r?\n/).length;
  const isPublic = fm['user-invocable']?.toLowerCase() === 'true';
  const budget = isPublic ? 220 : 180;
  if (lines > budget) warnings.push(`${relative(root, file)}: ${lines} lines > advisory budget ${budget}; consider progressive disclosure`);

  const desc = fm.description || '';
  const useWhen = (desc.match(/\bUse (when|for)\b/g) || []).length;
  if (useWhen > 1) warnings.push(`${relative(root, file)}: description has ${useWhen} trigger clauses; collapse duplicate branches if they are synonyms`);

  if (refs(src).some((r) => r.includes('/') && !r.startsWith('reference/') && !r.startsWith('../'))) {
    warnings.push(`${relative(root, file)}: has a multi-level reference pointer; prefer one-hop disclosure from SKILL.md`);
  }

  const hasCompletion = /Completion:|\bstop\b|output contract|reply contract|Verification|checklist/i.test(src);
  const hasPointer = /reference\/|\.md\)/.test(src);
  if (!hasCompletion && !hasPointer) warnings.push(`${relative(root, file)}: no obvious completion criterion or reference pointer`);
}

if (!quiet) {
  console.log(`skill-pruning-audit: ${warnings.length} advisory finding(s)`);
  for (const w of warnings.slice(0, 40)) console.log(`warn: ${w}`);
  if (warnings.length > 40) console.log(`warn: ${warnings.length - 40} more finding(s) suppressed`);
}
process.exit(0);
