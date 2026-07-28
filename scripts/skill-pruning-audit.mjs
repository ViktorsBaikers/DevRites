#!/usr/bin/env node
// Advisory pruning audit plus blocking ordered-step completion contracts.
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('..', import.meta.url));
const skillsArg = process.argv.indexOf('--skills-dir');
const skillsDir = skillsArg >= 0 ? process.argv[skillsArg + 1] : join(root, 'pack', '.claude', 'skills');
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

function workflowSteps(src) {
  const steps = [];
  const sections = src.matchAll(/^## (?:Workflow|Protocol|Process|Steps|Run|Flow|The .*cycle)[^\n]*\n([\s\S]*?)(?=^##\s|(?![\s\S]))/gm);
  for (const section of sections) {
    const body = section[1];
    const starts = [...body.matchAll(/^(\d+[a-z]?)\.\s+/gm)];
    steps.push(...starts.map((match, i) => ({
      id: match[1],
      body: body.slice(match.index, starts[i + 1]?.index ?? body.length),
    })));
  }
  return steps;
}

function hasStepCompletion(body) {
  const explicit = /\b(done when|completion|stop(?:s|ped)?|pass(?:es|ed)?|fail(?:s|ed)?|written|updated|recorded|reported|returned|emitted|rendered|confirmed|verified|captured|mapped|resolved|green)\b|\brc=|\bexit\s+\d|\buntil\b/i;
  const observableAction = /\b(write|update|record|report|return|emit|render|confirm|verify|read|run|dispatch|apply|select|capture|map|append|persist|fetch|compare|classify|rank|fix|build)\b/i;
  const observableTarget = /`[^`]+`|\b(file|artifact|command|result|verdict|matrix|diff|workspace|slice|question|route|scenario|report|evidence|spec|plan|test|check|gate|surface|journey|finding|decision)\b/i;
  return explicit.test(body) || (observableAction.test(body) && observableTarget.test(body));
}

function markdownFiles(dir, skipResearch = false) {
  const out = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.name.startsWith('.') || entry.name === 'generated' || (skipResearch && entry.name === 'research')) continue;
    const path = join(dir, entry.name);
    if (entry.isDirectory()) out.push(...markdownFiles(path, skipResearch));
    else if (entry.isFile() && entry.name.endsWith('.md')) out.push(path);
  }
  return out;
}

const warnings = [];
const contractFailures = [];
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
  for (const step of workflowSteps(src)) {
    if (!hasStepCompletion(step.body)) {
      contractFailures.push(`${relative(root, file)}: ordered step ${step.id} needs a checkable completion criterion`);
    }
  }
}

const proseFiles = [
  ...markdownFiles(skillsDir),
  ...markdownFiles(join(root, 'pack', '.claude', 'agents')),
  ...markdownFiles(join(root, 'docs'), true),
  join(root, 'README.md'),
  join(root, 'CONTRIBUTING.md'),
];
for (const file of proseFiles) {
  const src = readFileSync(file, 'utf8');
  const lines = src.split(/\r?\n/).map((line) => line.trim());
  const seen = new Set();
  let inCode = false;
  for (const line of lines) {
    if (line.startsWith('```')) { inCode = !inCode; continue; }
    if (inCode || line.length < 100 || line.startsWith('|') || line.startsWith('> **Untrusted-input safety.**')) continue;
    if (seen.has(line)) warnings.push(`${relative(root, file)}: repeats a long sentence/line; keep one owner or use a leading word`);
    seen.add(line);
  }
  if (/^(?:[-*]\s+)?(?:be careful|be thorough|do your best|produce high quality)[.!]?$/im.test(src)) {
    warnings.push(`${relative(root, file)}: contains a likely no-op instruction; replace it with a checkable target or delete it`);
  }
  if (/[/$](?:humanizer|writing-great-skills)\b/i.test(src)) {
    warnings.push(`${relative(root, file)}: contains a foreign workflow command token in a promoted surface`);
  }
}

if (!quiet) {
  console.log(`skill-pruning-audit: ${warnings.length} advisory finding(s)`);
  for (const w of warnings.slice(0, 40)) console.log(`warn: ${w}`);
  if (warnings.length > 40) console.log(`warn: ${warnings.length - 40} more finding(s) suppressed`);
}
for (const failure of contractFailures) console.log(`FAIL: ${failure}`);
process.exit(contractFailures.length ? 1 : 0);
