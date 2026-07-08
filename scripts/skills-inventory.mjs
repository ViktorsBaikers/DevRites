#!/usr/bin/env node
// skills-inventory.mjs — verify authored DevRites skill inventory and docs counts.
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { basename, join, relative } from 'node:path';

const root = new URL('..', import.meta.url).pathname.replace(/\/$/, '');
const skillsDir = join(root, 'pack', '.claude', 'skills');
const docsSkills = join(root, 'docs', 'skills.md');
const readme = join(root, 'README.md');
const arch = join(root, 'docs', 'architecture.md');
const DESCRIPTION_WORD_LIMITS = {
  public: 90,
  internal: 75,
  library: 60,
};

function fail(message) {
  console.error(`FAIL: ${message}`);
  failures++;
}

function frontmatter(text, file) {
  const lines = text.split(/\r?\n/);
  if (lines[0] !== '---') {
    fail(`${file}: missing opening frontmatter fence`);
    return new Map();
  }
  const fields = new Map();
  for (let i = 1; i < lines.length; i++) {
    const line = lines[i];
    if (line === '---') return fields;
    if (!line.trim() || line.trimStart().startsWith('#') || /^[ \t]/.test(line)) continue;
    const idx = line.indexOf(':');
    if (idx === -1) continue;
    const key = line.slice(0, idx).trim();
    const value = line.slice(idx + 1).trim().replace(/^['"]|['"]$/g, '');
    fields.set(key, value);
  }
  fail(`${file}: missing closing frontmatter fence`);
  return fields;
}

let failures = 0;
const skills = [];
for (const entry of readdirSync(skillsDir).sort()) {
  if (entry.startsWith('.')) continue;
  const dir = join(skillsDir, entry);
  if (!statSync(dir).isDirectory()) continue;
  const file = join(dir, 'SKILL.md');
  const text = readFileSync(file, 'utf8');
  const fm = frontmatter(text, relative(root, file));
  const name = fm.get('name') || '';
  const description = fm.get('description') || '';
  const invocable = fm.get('user-invocable') || '';
  const lines = text.split(/\r?\n/).length;
  const descriptionWords = description.trim().split(/\s+/).filter(Boolean).length;
  const descriptionSentences = (description.match(/[.!?](?:\s|$)/g) || []).length;
  const descriptionBudget = entry === 'devrites-lib'
    ? DESCRIPTION_WORD_LIMITS.library
    : invocable === 'true'
      ? DESCRIPTION_WORD_LIMITS.public
      : DESCRIPTION_WORD_LIMITS.internal;

  if (name !== entry) fail(`${relative(root, file)}: name ${JSON.stringify(name)} does not match directory ${JSON.stringify(entry)}`);
  if (!description) fail(`${relative(root, file)}: missing description`);
  if (description.length > 1024) fail(`${relative(root, file)}: description is ${description.length} chars (max 1024)`);
  if (descriptionWords > descriptionBudget) {
    fail(`${relative(root, file)}: description is ${descriptionWords} words (max ${descriptionBudget}; keep triggers tight and move reference into the body)`);
  }
  if (descriptionSentences > 4) {
    fail(`${relative(root, file)}: description has ${descriptionSentences} sentences (max 4; one purpose, one trigger branch, one boundary is enough)`);
  }
  for (const phrase of ['Use when', 'Not for']) {
    const count = (description.match(new RegExp(`\\b${phrase}\\b`, 'g')) || []).length;
    if (count > 1) {
      fail(`${relative(root, file)}: description repeats "${phrase}" ${count} times (collapse duplicate trigger branches)`);
    }
  }
  if (!['true', 'false'].includes(invocable)) fail(`${relative(root, file)}: user-invocable must be explicit true/false`);
  if (entry === 'devrites-lib' && fm.get('disable-model-invocation') !== 'true') {
    fail(`${relative(root, file)}: devrites-lib must set disable-model-invocation: true`);
  }
  if (lines > 500) fail(`${relative(root, file)}: SKILL.md has ${lines} lines (max 500)`);
  skills.push({ name: entry, invocable, lines, description });
}

const total = skills.length;
const publicCount = skills.filter((s) => s.invocable === 'true').length;
const internalCount = skills.filter((s) => s.invocable === 'false').length;
const publicRiteCount = skills.filter((s) => s.invocable === 'true' && s.name.startsWith('rite-')).length;
const modelInvokedInternalCount = skills.filter((s) => s.invocable === 'false' && s.name !== 'devrites-lib').length;

function assertDocContains(path, expected, label) {
  const text = readFileSync(path, 'utf8');
  if (!text.includes(expected)) {
    fail(`${relative(root, path)}: missing ${label}: ${JSON.stringify(expected)}`);
  }
}

assertDocContains(docsSkills, `# All ${total} skills`, 'total skill heading');
assertDocContains(docsSkills, `**${total} skills total**`, 'total skill prose');
assertDocContains(docsSkills, `${publicRiteCount} user-invocable \`rite-*\``, 'public rite-* count');
assertDocContains(docsSkills, `${modelInvokedInternalCount} model-invoked \`devrites-*\` specialists`, 'model-invoked internal count');
assertDocContains(readme, `**${total} skills total**`, 'README total skill prose');
assertDocContains(readme, `# skills/  ${total} skills`, 'README layout total count');
assertDocContains(arch, `${publicRiteCount} public \`rite-*\` skills (${total} total)`, 'architecture surface count');

console.log(`skills total: ${total}`);
console.log(`public skills: ${publicCount}`);
console.log(`public rite-* skills: ${publicRiteCount}`);
console.log(`internal skills: ${internalCount}`);
console.log(`model-invoked devrites-* specialists: ${modelInvokedInternalCount}`);

if (failures) {
  console.error(`skills-inventory: ${failures} failure(s)`);
  process.exit(1);
}
console.log('skills-inventory: PASS');
