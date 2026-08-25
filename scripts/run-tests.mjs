#!/usr/bin/env node
import { spawn } from 'node:child_process';
import { existsSync, mkdtempSync, readFileSync, readdirSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { basename, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('..', import.meta.url));
const testsDir = join(root, 'tests');
const args = process.argv.slice(2);
const maxJobs = Math.max(1, Math.floor(Number(process.env.DEVRITES_TEST_JOBS_MAX || 16)) || 16);
let jobs = Math.max(1, Math.min(maxJobs, Math.floor(Number(process.env.DEVRITES_TEST_JOBS || 4)) || 4));
let serial = false;
let fast = false;
let shardIndex = 0;
let shardTotal = 0;
const filters = [];

function parseShard(value) {
  const match = /^(\d+)\/(\d+)$/.exec(value);
  if (!match) throw new Error(`invalid --shard value (want i/n): ${value}`);
  const index = Number(match[1]);
  const total = Number(match[2]);
  if (!Number.isInteger(index) || !Number.isInteger(total) || index < 1 || index > total || total < 1) {
    throw new Error(`invalid --shard value (want 1<=i<=n): ${value}`);
  }
  return { index, total };
}

function testWeight(name) {
  return testWeights.get(name) || 1;
}

function itemWeight(item) {
  if (typeof item === 'string') return testWeight(basename(item));
  if (item.waiMode === 'core') return testWeights.get('workflow-artifact-identity-test.sh#core') || 80;
  if (item.waiMode === 'matrix') return testWeights.get('workflow-artifact-identity-test.sh#matrix') || 60;
  return testWeights.get('workflow-artifact-identity-test.sh#boundary') || 90;
}

function itemLabel(item) {
  if (typeof item === 'string') return item;
  if (item.waiMode === 'boundary') {
    return `${item.path}#boundary-${item.waiBoundaryShard}`;
  }
  if (item.waiMode === 'matrix') {
    return `${item.path}#delivery-model-matrix`;
  }
  if (item.waiCoreShard) {
    return `${item.path}#core-${item.waiCoreShard}`;
  }
  return `${item.path}#core`;
}

function assignWeightedShards(items, total) {
  const shards = Array.from({ length: total }, () => ({ items: [], weight: 0 }));
  for (const item of items) {
    const target = shards.reduce((min, shard) => (shard.weight < min.weight ? shard : min));
    target.items.push(item);
    target.weight += itemWeight(item);
  }
  return shards;
}

function repositoryPackageVersion() {
  const version = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8')).version;
  const safeSemver = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/;
  if (typeof version !== 'string' || version.length > 128 || !safeSemver.test(version)) {
    throw new Error('package.json version must be a safe semantic version of at most 128 characters');
  }
  return version;
}

for (let i = 0; i < args.length; i++) {
  const arg = args[i];
  if (arg === '--serial') serial = true;
  else if (arg === '--fast') fast = true;
  else if (arg === '--jobs' || arg === '-j') jobs = Math.max(1, Number(args[++i] || 1) || 1);
  else if (arg.startsWith('--jobs=')) jobs = Math.max(1, Number(arg.slice('--jobs='.length)) || 1);
  else if (arg === '--shard') {
    const parsed = parseShard(String(args[++i] || ''));
    shardIndex = parsed.index;
    shardTotal = parsed.total;
  } else if (arg.startsWith('--shard=')) {
    const parsed = parseShard(arg.slice('--shard='.length));
    shardIndex = parsed.index;
    shardTotal = parsed.total;
  } else filters.push(arg);
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
  'claude-runtime-smoke.sh',
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

// Weights from CI wall times (seconds, rounded) for balanced shard assignment.
const testWeights = new Map([
  ['workflow-artifact-identity-test.sh#core', 80],
  ['workflow-artifact-identity-test.sh#matrix', 60],
  ['workflow-artifact-identity-test.sh#boundary', 90],
  ['workflow-artifact-identity-test.sh', 333],
  ['binary-lifecycle-test.sh', 200],
  ['validate-pack.sh', 80],
  ['uninstall-smoke.sh', 70],
  ['outcome-evals-test.sh', 70],
  ['release-tarball-test.sh', 50],
  ['validate-path-spaces-test.sh', 40],
  ['npx-pack-smoke.sh', 40],
  ['install-smoke.sh', 18],
  ['bootstrap-security-test.sh', 14],
  ['acceptance-preserving-reslice-policy-test.sh', 40],
  ['host-artifacts-test.sh', 9],
  ['workspace-schema-test.sh', 6],
  ['install-shared-file-merge-smoke.sh', 40],
  ['cli-smoke.sh', 20],
  ['update-smoke.sh', 20],
  ['install-flag-parser-smoke.sh', 10],
  ['install-flag-parser-legacy-smoke.sh', 10],
  ['install-option-matrix-smoke.sh', 15],
  ['fixture-install.sh', 20],
  ['install-flag-parser-invalid-smoke.sh', 10],
  ['codex-agent-generation-test.sh', 10],
  ['claude-runtime-smoke.sh', 5],
  ['codex-runtime-smoke.sh', 5],
  ['hooks-parity-test.sh', 10],
  ['install-pin-no-global-smoke.sh', 10],
]);

const engineIsolatedTests = new Set([
  'binary-lifecycle-test.sh',
]);

// Install smokes mutate shared host config (~/.codex, AGENTS bridges, etc.) and
// must not overlap on the same runner even when other tests parallelize freely.
const installExclusiveTests = new Set([
  'cli-smoke.sh',
  'fixture-install.sh',
  'install-flag-parser-invalid-smoke.sh',
  'install-flag-parser-legacy-smoke.sh',
  'install-flag-parser-smoke.sh',
  'install-option-matrix-smoke.sh',
  'install-pin-no-global-smoke.sh',
  'install-shared-file-merge-smoke.sh',
  'install-smoke.sh',
  'npx-pack-smoke.sh',
  'uninstall-smoke.sh',
  'update-smoke.sh',
]);

// WAI core installs live protected fixtures into the checkout; reslice policy
// snapshots repository entry identities. Running them together races on disk.
const repoMutatingExclusiveTests = new Set([
  'acceptance-preserving-reslice-policy-test.sh',
  'workflow-artifact-identity-test.sh',
]);

let installExclusiveChain = Promise.resolve();
let repoMutatingExclusiveChain = Promise.resolve();

tests.sort((a, b) => {
  const aw = itemWeight(a);
  const bw = itemWeight(b);
  return aw === bw ? itemLabel(a).localeCompare(itemLabel(b)) : bw - aw;
});

if (fast) {
  for (let i = tests.length - 1; i >= 0; i--) {
    if (integrationTests.has(basename(tests[i]))) tests.splice(i, 1);
  }
}

const waiTest = 'tests/workflow-artifact-identity-test.sh';
const waiBoundaryShards = Math.max(1, Math.floor(Number(process.env.DEVRITES_WAI_BOUNDARY_SHARDS || 4)) || 4);
const waiCoreShards = Math.max(1, Math.floor(Number(process.env.DEVRITES_WAI_CORE_SHARDS || 2)) || 2);
if (shardTotal > 0) {
  // Expand WAI into core + boundary pieces BEFORE weighting so each piece can
  // land on a different matrix runner (avoids packing ~5 heavy WAI jobs onto one VM).
  const expandable = [];
  for (const test of tests) {
    if (test === waiTest) {
      for (let coreShard = 1; coreShard <= waiCoreShards; coreShard++) {
        expandable.push({
          path: waiTest,
          waiMode: 'core',
          waiCoreShard: `${coreShard}/${waiCoreShards}`,
        });
      }
      expandable.push({ path: waiTest, waiMode: 'matrix' });
      for (let boundaryShard = 1; boundaryShard <= waiBoundaryShards; boundaryShard++) {
        expandable.push({
          path: waiTest,
          waiMode: 'boundary',
          waiBoundaryShard: `${boundaryShard}/${waiBoundaryShards}`,
        });
      }
    } else {
      expandable.push(test);
    }
  }
  expandable.sort((a, b) => {
    const aw = itemWeight(a);
    const bw = itemWeight(b);
    return aw === bw ? itemLabel(a).localeCompare(itemLabel(b)) : bw - aw;
  });
  const shards = assignWeightedShards(expandable, shardTotal);
  tests.length = 0;
  tests.push(...shards[shardIndex - 1].items);
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
  let engineVersion;
  try {
    engineVersion = `v${repositoryPackageVersion()}`;
  } catch (error) {
    console.error(`cannot build shared test engine: ${error.message}`);
    process.exit(1);
  }
  sharedEngineDir = mkdtempSync(join(tmpdir(), 'devrites-test-engine-'));
  sharedEngine = join(sharedEngineDir, 'devrites-engine');
  const build = spawn('go', [
    'build',
    '-trimpath',
    '-ldflags',
    `-s -w -X github.com/devrites/devrites/internal/version.Version=${engineVersion}`,
    '-o',
    sharedEngine,
    '.',
  ], {
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
    const path = typeof test === 'string' ? test : test.path;
    const label = typeof test === 'string'
      ? basename(path)
      : itemLabel(test);
    const chunks = [];
    const start = Date.now();
    const env = { ...process.env, DEVRITES_HOST_ARTIFACT_DIR: sharedHostArtifacts, DEVRITES_TEST_WORKER: label };
    if (typeof test === 'object' && test.waiMode === 'core') {
      env.DEVRITES_WAI_SKIP_DELIVERY_MODES = '1';
      env.DEVRITES_WAI_SKIP_DELIVERY_MODEL_MATRIX = '1';
      if (test.waiCoreShard) env.DEVRITES_WAI_CORE_SHARD = test.waiCoreShard;
    } else if (typeof test === 'object' && test.waiMode === 'matrix') {
      env.DEVRITES_WAI_DELIVERY_MODEL_ONLY = '1';
    } else if (typeof test === 'object' && test.waiMode === 'boundary') {
      env.DEVRITES_WAI_BOUNDARY_ONLY = '1';
      env.DEVRITES_WAI_BOUNDARY_SHARD = test.waiBoundaryShard;
    }
    if (engineIsolatedTests.has(basename(path))) delete env.DEVRITES_ENGINE_CLI;
    else if (sharedEngine) env.DEVRITES_ENGINE_CLI = sharedEngine;
    const child = spawn('bash', [path], {
      cwd: root,
      env,
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    child.stdout.on('data', (chunk) => chunks.push(chunk));
    child.stderr.on('data', (chunk) => chunks.push(chunk));
    child.on('close', (code, signal) => {
      const elapsed = ((Date.now() - start) / 1000).toFixed(2);
      const status = code === 0 ? 'PASS' : 'FAIL';
      const displayName = typeof test === 'string' ? test : label;
      process.stdout.write(`== ${displayName} ==\n`);
      for (const chunk of chunks) process.stdout.write(chunk);
      if (chunks.length && !String(chunks[chunks.length - 1]).endsWith('\n')) process.stdout.write('\n');
      process.stdout.write(`${status}: ${displayName} (${elapsed}s)\n`);
      if (signal) process.stdout.write(`signal: ${signal}\n`);
      resolve(code === 0);
    });
  });
}

async function runExclusive(getChain, setChain, test) {
  const previous = getChain();
  let release;
  setChain(new Promise((resolve) => {
    release = resolve;
  }));
  await previous;
  try {
    return await runOne(test);
  } finally {
    release();
  }
}

async function runBatch(batch, batchJobs) {
  let cursor = 0;
  async function worker() {
    while (cursor < batch.length) {
      const test = batch[cursor++];
      const path = typeof test === 'string' ? test : test.path;
      const label = basename(path);
      let ok;
      if (installExclusiveTests.has(label)) {
        ok = await runExclusive(
          () => installExclusiveChain,
          (next) => { installExclusiveChain = next; },
          test,
        );
      } else if (repoMutatingExclusiveTests.has(label)) {
        ok = await runExclusive(
          () => repoMutatingExclusiveChain,
          (next) => { repoMutatingExclusiveChain = next; },
          test,
        );
      } else {
        ok = await runOne(test);
      }
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
if (shardTotal > 0) process.stdout.write(`; shard ${shardIndex}/${shardTotal}`);
if (fast) process.stdout.write('; fast isolated-test subset');
if (!serial) process.stdout.write('; longest tests first');
process.stdout.write('\n');
if (serial) await runSerial(tests);
else await runBatch(tests, jobs);
const elapsed = ((Date.now() - started) / 1000).toFixed(2);
process.stdout.write(`\n${failed ? 'TESTS FAILED' : 'TESTS PASSED'} (${elapsed}s)\n`);
process.exit(failed ? 1 : 0);
