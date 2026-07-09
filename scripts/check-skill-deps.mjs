#!/usr/bin/env node
// Validate dependency-closed skill profiles. Tiny now; prevents future profile drift.
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { basename, join, relative } from 'node:path';

const root = new URL('..', import.meta.url).pathname.replace(/\/$/, '');
let skillsDir = join(root, 'pack', '.claude', 'skills');
let manifestPath = join(root, 'pack', 'skill-surface.json');
for (let i = 2; i < process.argv.length; i++) {
  if (process.argv[i] === '--skills-dir') skillsDir = process.argv[++i];
  else if (process.argv[i] === '--manifest') manifestPath = process.argv[++i];
}

function fail(msg) {
  console.error(`FAIL: ${msg}`);
  failures++;
}

let failures = 0;
if (!existsSync(skillsDir)) fail(`missing skills dir ${skillsDir}`);
if (!existsSync(manifestPath)) fail(`missing skill surface manifest ${manifestPath}`);
if (failures) process.exit(1);

const skillNames = readdirSync(skillsDir, { withFileTypes: true })
  .filter((d) => d.isDirectory() && existsSync(join(skillsDir, d.name, 'SKILL.md')))
  .map((d) => d.name)
  .sort();
const skillSet = new Set(skillNames);
const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
const profiles = manifest.profiles || {};

function bodyRefs(name) {
  const text = readFileSync(join(skillsDir, name, 'SKILL.md'), 'utf8');
  const refs = new Set();
  for (const other of skillNames) {
    if (other === name) continue;
    const esc = other.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const re = new RegExp(`(^|[^A-Za-z0-9_-])(?:/|\\$)?${esc}(?=$|[^A-Za-z0-9_-])`);
    if (re.test(text)) refs.add(other);
  }
  return [...refs].sort();
}

const deps = Object.fromEntries(skillNames.map((name) => [name, bodyRefs(name)]));

function expand(list) {
  if (!Array.isArray(list)) return null;
  if (list.includes('*')) return new Set(skillNames);
  const out = new Set();
  const visit = (name) => {
    if (out.has(name)) return;
    out.add(name);
    for (const dep of deps[name] || []) visit(dep);
  };
  for (const name of list) visit(name);
  return out;
}

for (const name of Object.keys(profiles).sort()) {
  const declared = profiles[name];
  const expanded = expand(declared);
  if (!expanded) {
    fail(`profile ${name} must be an array`);
    continue;
  }
  for (const skill of declared || []) {
    if (skill !== '*' && !skillSet.has(skill)) fail(`profile ${name} names unknown skill ${skill}`);
  }
  for (const skill of expanded) {
    if (!skillSet.has(skill)) fail(`profile ${name} dependency closure contains unknown skill ${skill}`);
  }
  if (!declared.includes('*')) {
    for (const skill of declared) {
      for (const dep of deps[skill] || []) {
        if (!expanded.has(dep)) fail(`profile ${name}: ${skill} references ${dep} but it is outside closure`);
      }
    }
  }
}

console.log(`skill-deps: ${Object.keys(profiles).length} profile(s), ${skillNames.length} skills checked via ${relative(root, manifestPath) || basename(manifestPath)}`);
process.exit(failures ? 1 : 0);
