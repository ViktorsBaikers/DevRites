#!/usr/bin/env node
import { spawnSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const args = process.argv.slice(2);
const option = (name, fallback) => {
  const index = args.indexOf(name);
  return index >= 0 ? args[index + 1] : fallback;
};
const input = option('--input', '');
const exceptionsPath = resolve(option('--exceptions', resolve(root, 'scripts', 'npm-audit-exceptions.json')));
const blocking = new Set(['moderate', 'high', 'critical']);
const failures = [];
const fail = (message) => failures.push(message);
const load = (path) => JSON.parse(readFileSync(path, 'utf8'));

let report;
if (input) {
  report = load(resolve(input));
} else {
  const npm = process.platform === 'win32' ? 'npm.cmd' : 'npm';
  const result = spawnSync(npm, ['audit', '--json'], { cwd: root, encoding: 'utf8' });
  if (result.error || ![0, 1].includes(result.status)) {
    console.error(result.stderr || result.error?.message || `npm audit exited ${result.status}`);
    process.exit(1);
  }
  try {
    report = JSON.parse(result.stdout);
  } catch {
    console.error('FAIL: npm audit did not return valid JSON');
    process.exit(1);
  }
}

const concrete = new Map();
let affected = 0;
for (const [packageName, vulnerability] of Object.entries(report.vulnerabilities || {})) {
  if (!blocking.has(vulnerability.severity)) continue;
  affected++;
  for (const via of vulnerability.via || []) {
    if (!via || typeof via !== 'object' || !blocking.has(via.severity)) continue;
    const id = via.url?.match(/GHSA-[0-9a-z]{4}-[0-9a-z]{4}-[0-9a-z]{4}/i)?.[0];
    if (!id) {
      fail(`${packageName}: blocking advisory has no GHSA identifier`);
      continue;
    }
    const finding = concrete.get(id) || {
      id,
      package: via.name || packageName,
      range: via.range || vulnerability.range,
      source: via.url,
      nodes: [],
    };
    finding.nodes = [...new Set([...finding.nodes, ...(vulnerability.nodes || [])])].sort();
    concrete.set(id, finding);
  }
}
if (affected > 0 && concrete.size === 0) fail('blocking audit findings have no concrete advisory records');

const exceptions = load(exceptionsPath);
if (!Array.isArray(exceptions)) fail('npm audit exceptions must be a JSON array');
const today = new Date().toISOString().slice(0, 10);
const seen = new Set();
for (const exception of Array.isArray(exceptions) ? exceptions : []) {
  const required = ['id', 'package', 'range', 'source', 'owner', 'reason', 'expires'];
  const missing = required.filter((field) => !exception?.[field]);
  if (!Array.isArray(exception?.nodes) || exception.nodes.length === 0) missing.push('nodes');
  if (missing.length) {
    fail(`${exception?.id || 'unknown exception'}: missing ${missing.join(', ')}`);
    continue;
  }
  if (!/^GHSA-[0-9a-z]{4}-[0-9a-z]{4}-[0-9a-z]{4}$/i.test(exception.id) || !/^\d{4}-\d{2}-\d{2}$/.test(exception.expires)) {
    fail(`${exception.id}: invalid id or expiry`);
    continue;
  }
  if (seen.has(exception.id)) fail(`${exception.id}: duplicate exception`);
  seen.add(exception.id);
  if (exception.expires < today) fail(`${exception.id}: exception expired ${exception.expires}`);

  const finding = concrete.get(exception.id);
  if (!finding) {
    fail(`${exception.id}: exception is stale; advisory is absent`);
    continue;
  }
  for (const field of ['package', 'range', 'source']) {
    if (exception[field] !== finding[field]) {
      fail(`${exception.id}: ${field} mismatch (expected ${finding[field]})`);
    }
  }
  if (JSON.stringify([...exception.nodes].sort()) !== JSON.stringify(finding.nodes)) {
    fail(`${exception.id}: nodes mismatch (expected ${finding.nodes.join(', ')})`);
  }
}
for (const finding of concrete.values()) {
  if (!seen.has(finding.id)) fail(`${finding.id}: blocking advisory is not excepted`);
}

for (const message of failures) console.error(`FAIL: ${message}`);
if (failures.length) process.exit(1);
if (affected === 0) {
  console.log('npm-audit: no moderate-or-higher advisories');
} else {
  for (const exception of exceptions) {
    console.log(`npm-audit: temporary ${exception.id} exception owned by ${exception.owner}, expires ${exception.expires}`);
  }
  console.log(`npm-audit: ${concrete.size} exact temporary exception(s), ${affected} affected package record(s)`);
}
