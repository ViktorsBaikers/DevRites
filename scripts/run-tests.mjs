#!/usr/bin/env node
import { spawn } from 'node:child_process';
import { existsSync, mkdtempSync, readdirSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { basename, join } from 'node:path';

const root = new URL('..', import.meta.url).pathname.replace(/\/$/, '');
const testsDir = join(root, 'tests');
const args = process.argv.slice(2);
let jobs = Math.max(1, Math.min(5, Math.floor(Number(process.env.DEVRITES_TEST_JOBS || 4)) || 4));
let serial = false;
let fast = false;
const filters = [];

for (let i = 0; i < args.length; i++) {
  const arg = args[i];
  if (arg === '--serial') serial = true;
  else if (arg === '--fast') fast = true;
  else if (arg === '--jobs' || arg === '-j') jobs = Math.max(1, Number(args[++i] || 1) || 1);
  else if (arg.startsWith('--jobs=')) jobs = Math.max(1, Number(arg.slice('--jobs='.length)) || 1);
  else filters.push(arg);
}
if (serial) jobs = 1;

const allTests = readdirSync(testsDir)
  .filter((name) => name.endsWith('.sh'))
  .sort()
  .map((name) => join('tests', name));
const tests = filters.length
  ? allTests.filter((test) => filters.some((filter) => test.includes(filter)))
  : allTests;

const integrationTests = new Set([
  'binary-lifecycle-test.sh',
  'cli-smoke.sh',
  'codex-agent-generation-test.sh',
  'codex-runtime-smoke.sh',
  'fixture-install.sh',
  'install-flag-parser-legacy-smoke.sh',
  'host-artifacts-test.sh',
  'install-flag-parser-invalid-smoke.sh',
  'install-flag-parser-smoke.sh',
  'install-option-matrix-smoke.sh',
  'install-pin-no-global-smoke.sh',
  'install-shared-file-merge-smoke.sh',
  'install-smoke.sh',
  'npx-pack-smoke.sh',
  'uninstall-smoke.sh',
  'update-smoke.sh',
  'validate-pack.sh',
]);

const testWeights = new Map([
  ['binary-lifecycle-test.sh', 75],
  ['install-shared-file-merge-smoke.sh', 55],
  ['cli-smoke.sh', 55],
  ['uninstall-smoke.sh', 50],
  ['install-smoke.sh', 46],
  ['npx-pack-smoke.sh', 45],
  ['update-smoke.sh', 35],
  ['install-flag-parser-smoke.sh', 35],
  ['install-flag-parser-legacy-smoke.sh', 35],
  ['install-option-matrix-smoke.sh', 30],
  ['fixture-install.sh', 30],
  ['validate-pack.sh', 25],
  ['install-flag-parser-invalid-smoke.sh', 20],
  ['codex-agent-generation-test.sh', 20],
  ['codex-runtime-smoke.sh', 15],
  ['hooks-parity-test.sh', 15],
  ['host-artifacts-test.sh', 10],
  ['install-pin-no-global-smoke.sh', 10],
]);

const engineIsolatedTests = new Set([
  'binary-lifecycle-test.sh',
  'npx-pack-smoke.sh',
]);

tests.sort((a, b) => {
  const aw = testWeights.get(basename(a)) || 0;
  const bw = testWeights.get(basename(b)) || 0;
  return aw === bw ? a.localeCompare(b) : bw - aw;
});

if (fast) {
  for (let i = tests.length - 1; i >= 0; i--) {
    if (integrationTests.has(basename(tests[i]))) tests.splice(i, 1);
  }
}

if (tests.length === 0) {
  console.error(`no tests matched: ${filters.join(' ')}`);
  process.exit(1);
}

let failed = false;
const started = Date.now();
let sharedHostArtifacts = process.env.DEVRITES_HOST_ARTIFACT_DIR || '';
let sharedEngineDir = '';
let sharedEngine = process.env.DEVRITES_ENGINE_CLI || '';

if (!sharedHostArtifacts) {
  sharedHostArtifacts = mkdtempSync(join(tmpdir(), 'devrites-test-artifacts-'));
  const build = spawn('bash', ['scripts/build-host-artifacts.sh'], {
    cwd: root,
    env: { ...process.env, DEVRITES_HOST_ARTIFACT_DIR: sharedHostArtifacts },
    stdio: ['ignore', 'ignore', 'pipe'],
  });
  const chunks = [];
  build.stderr.on('data', (chunk) => chunks.push(chunk));
  const ok = await new Promise((resolve) => build.on('close', (code) => resolve(code === 0)));
  if (!ok) {
    for (const chunk of chunks) process.stderr.write(chunk);
    rmSync(sharedHostArtifacts, { recursive: true, force: true });
    process.exit(1);
  }
}

process.on('exit', () => {
  if (!process.env.DEVRITES_HOST_ARTIFACT_DIR) rmSync(sharedHostArtifacts, { recursive: true, force: true });
  if (sharedEngineDir) rmSync(sharedEngineDir, { recursive: true, force: true });
});

if (!sharedEngine && existsSync(join(root, 'engine', 'go.mod'))) {
  sharedEngineDir = mkdtempSync(join(tmpdir(), 'devrites-test-engine-'));
  sharedEngine = join(sharedEngineDir, 'devrites-engine');
  const build = spawn('go', ['build', '-trimpath', '-o', sharedEngine, '.'], {
    cwd: join(root, 'engine'),
    env: { ...process.env, CGO_ENABLED: '0' },
    stdio: ['ignore', 'ignore', 'pipe'],
  });
  const chunks = [];
  build.stderr.on('data', (chunk) => chunks.push(chunk));
  const ok = await new Promise((resolve) => {
    build.on('error', () => resolve(false));
    build.on('close', (code) => resolve(code === 0));
  });
  if (!ok) {
    for (const chunk of chunks) process.stderr.write(chunk);
    rmSync(sharedEngineDir, { recursive: true, force: true });
    process.exit(1);
  }
}

function runOne(test) {
  return new Promise((resolve) => {
    const label = basename(test);
    const chunks = [];
    const start = Date.now();
    const env = { ...process.env, DEVRITES_HOST_ARTIFACT_DIR: sharedHostArtifacts, DEVRITES_TEST_WORKER: label };
    if (engineIsolatedTests.has(label)) delete env.DEVRITES_ENGINE_CLI;
    else if (sharedEngine) env.DEVRITES_ENGINE_CLI = sharedEngine;
    const child = spawn('bash', [test], {
      cwd: root,
      env,
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    child.stdout.on('data', (chunk) => chunks.push(chunk));
    child.stderr.on('data', (chunk) => chunks.push(chunk));
    child.on('close', (code, signal) => {
      const elapsed = ((Date.now() - start) / 1000).toFixed(2);
      const status = code === 0 ? 'PASS' : 'FAIL';
      process.stdout.write(`== ${test} ==\n`);
      for (const chunk of chunks) process.stdout.write(chunk);
      if (chunks.length && !String(chunks[chunks.length - 1]).endsWith('\n')) process.stdout.write('\n');
      process.stdout.write(`${status}: ${test} (${elapsed}s)\n`);
      if (signal) process.stdout.write(`signal: ${signal}\n`);
      resolve(code === 0);
    });
  });
}

async function runBatch(batch, batchJobs) {
  let cursor = 0;
  async function worker() {
    while (cursor < batch.length) {
      const test = batch[cursor++];
      const ok = await runOne(test);
      if (!ok) failed = true;
    }
  }
  await Promise.all(Array.from({ length: Math.min(batchJobs, batch.length) }, worker));
}

async function runSerial(batch) {
  for (const test of batch) {
    const ok = await runOne(test);
    if (!ok) {
      failed = true;
      if (serial) break;
    }
  }
}

process.stdout.write(`Running ${tests.length} shell test(s) with ${jobs} job(s)`);
if (fast) process.stdout.write('; fast isolated-test subset');
if (!serial) process.stdout.write('; longest tests first');
process.stdout.write('\n');
if (serial) await runSerial(tests);
else await runBatch(tests, jobs);
const elapsed = ((Date.now() - started) / 1000).toFixed(2);
process.stdout.write(`\n${failed ? 'TESTS FAILED' : 'TESTS PASSED'} (${elapsed}s)\n`);
process.exit(failed ? 1 : 0);
