#!/usr/bin/env node
import { spawn, spawnSync } from 'node:child_process';
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
  // Per-index weights first: the WAI internal case split is uneven (measured
  // core shards ranged 7-49s), so a uniform per-mode weight lets one shard
  // stack two heavy WAI items that then serialize behind protected fixtures.
  if (item.waiMode === 'core') {
    return testWeights.get(`workflow-artifact-identity-test.sh#core-${item.waiCoreShard}`)
      || testWeights.get('workflow-artifact-identity-test.sh#core')
      || 80;
  }
  return testWeights.get(`workflow-artifact-identity-test.sh#boundary-${item.waiBoundaryShard}`)
    || testWeights.get('workflow-artifact-identity-test.sh#boundary')
    || 90;
}

function itemLabel(item) {
  if (typeof item === 'string') return item;
  if (item.waiMode === 'boundary') {
    return `${item.path}#boundary-${item.waiBoundaryShard}`;
  }
  if (item.waiCoreShard) {
    return `${item.path}#core-${item.waiCoreShard}`;
  }
  return `${item.path}#core`;
}

function assignWeightedShards(items, total) {
  const shards = Array.from({ length: total }, () => ({ items: [], weight: 0, wai: 0 }));
  for (const item of items) {
    const isWai = typeof item !== 'string';
    let target = shards[0];
    let targetCost = target.weight + (isWai ? target.wai * 2 : 0);
    for (const shard of shards) {
      const cost = shard.weight + (isWai ? shard.wai * 2 : 0);
      if (cost < targetCost) {
        target = shard;
        targetCost = cost;
      }
    }
    target.items.push(item);
    target.weight += itemWeight(item);
    if (isWai) target.wai += itemWeight(item);
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

const usage = `usage: node scripts/run-tests.mjs [options] [name-filter ...]

Runs tests/*.sh whose path contains any name filter (all tests if none).

options:
  --fast          skip the integration tests
  --serial        run one test at a time (same as --jobs 1)
  --jobs N, -j N  run N tests at once (also --jobs=N)
  --shard i/n     run only shard i of n, 1 <= i <= n (also --shard=i/n)
  --help, -h      print this message and exit`;

if (args.includes('--help') || args.includes('-h')) {
  console.log(usage);
  process.exit(0);
}

try {
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
} catch (error) {
  console.error(error.message);
  process.exit(2);
}
if (serial) jobs = 1;
const testTimeoutSec = Number(process.env.DEVRITES_TEST_TIMEOUT_SEC) || 900;

const allTests = readdirSync(testsDir)
  // *-lib.sh files are sourced helpers, not tests.
  .filter((name) => name.endsWith('.sh') && !name.endsWith('-lib.sh'))
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
// Refreshed 2026-09-01 from per-test PASS durations on the 8-shard CI run.
const testWeights = new Map([
  // Core checks split round-robin into 8 pieces (~117s of checks in total).
  ['workflow-artifact-identity-test.sh#core', 15],
  ['workflow-artifact-identity-test.sh#boundary', 27],
  ['workflow-artifact-identity-test.sh#boundary-1/6', 28],
  ['workflow-artifact-identity-test.sh#boundary-2/6', 20],
  ['workflow-artifact-identity-test.sh#boundary-3/6', 36],
  ['workflow-artifact-identity-test.sh#boundary-4/6', 33],
  ['workflow-artifact-identity-test.sh#boundary-5/6', 23],
  ['workflow-artifact-identity-test.sh#boundary-6/6', 31],
  ['workflow-artifact-identity-test.sh', 333],
  ['binary-lifecycle-test.sh', 32],
  ['validate-pack.sh', 34],
  ['uninstall-smoke.sh', 21],
  ['outcome-evals-test.sh', 31],
  ['release-tarball-test.sh', 51],
  ['validate-path-spaces-test.sh', 42],
  ['npx-pack-smoke.sh', 19],
  ['install-smoke.sh', 167],
  ['bootstrap-security-test.sh', 12],
  ['acceptance-preserving-reslice-policy-test.sh', 219],
  ['host-artifacts-test.sh', 137],
  ['workspace-schema-test.sh', 4],
  ['install-shared-file-merge-smoke.sh', 116],
  ['cli-smoke.sh', 2],
  ['engine-observation-contract-test.sh', 3],
  ['update-smoke.sh', 3],
  ['install-flag-parser-smoke.sh', 1],
  ['install-flag-parser-legacy-smoke.sh', 1],
  ['install-option-matrix-smoke.sh', 1],
  ['fixture-install.sh', 1],
  ['install-flag-parser-invalid-smoke.sh', 1],
  ['codex-agent-generation-test.sh', 2],
  ['claude-runtime-smoke.sh', 1],
  ['codex-runtime-smoke.sh', 1],
  ['hooks-parity-test.sh', 3],
  ['install-pin-no-global-smoke.sh', 1],
  ['native-host-loop-evals-test.sh', 132],
  ['validate-skip-summary-test.sh', 103],
  ['release-routes-test.sh', 45],
  ['scan-pack-security-test.sh', 4],
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

// WAI core and reslice policy both read live checkout entry identities; this
// chain keeps them from running at the same time.
const repoMutatingExclusiveTests = new Set([
  'acceptance-preserving-reslice-policy-test.sh',
  'workflow-artifact-identity-test.sh',
]);

function exclusiveChain(test) {
  const label = basename(typeof test === 'string' ? test : test.path);
  if (installExclusiveTests.has(label)) return 'install';
  if (repoMutatingExclusiveTests.has(label)) return 'repo';
  return '';
}

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
if (tests.includes(waiTest) || shardTotal > 0) {
  // Expand WAI into core + boundary pieces BEFORE weighting so each piece can
  // land on a different matrix runner (avoids packing ~5 heavy WAI jobs onto one VM)
  // and every piece gets its own timeout budget when the run is not sharded.
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
  tests.length = 0;
  if (shardTotal > 0) tests.push(...assignWeightedShards(expandable, shardTotal)[shardIndex - 1].items);
  else tests.push(...expandable);
}

if (tests.length === 0) {
  console.error(`no tests matched: ${filters.join(' ')}`);
  process.exit(1);
}

let failed = false;
const failedTests = new Set();
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

// Tests that build the engine reuse the user's Go build cache instead of
// falling back to a cold per-test temp cache.
const goCache = process.env.GOCACHE
  || spawnSync('go', ['env', 'GOCACHE'], { encoding: 'utf8' }).stdout?.trim()
  || '';

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
      env.DEVRITES_WAI_ALLOW_PARTIAL = '1';
      env.DEVRITES_WAI_SKIP_DELIVERY_MODES = '1';
      if (test.waiCoreShard) env.DEVRITES_WAI_CORE_SHARD = test.waiCoreShard;
    } else if (typeof test === 'object' && test.waiMode === 'boundary') {
      env.DEVRITES_WAI_ALLOW_PARTIAL = '1';
      env.DEVRITES_WAI_BOUNDARY_ONLY = '1';
      env.DEVRITES_WAI_BOUNDARY_SHARD = test.waiBoundaryShard;
    }
    if (goCache) env.GOCACHE = goCache;
    if (engineIsolatedTests.has(basename(path))) delete env.DEVRITES_ENGINE_CLI;
    else if (sharedEngine) env.DEVRITES_ENGINE_CLI = sharedEngine;
    const child = spawn('bash', [path], {
      cwd: root,
      env,
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    child.stdout.on('data', (chunk) => chunks.push(chunk));
    child.stderr.on('data', (chunk) => chunks.push(chunk));
    // SIGTERM first so the test's cleanup handlers can run; SIGKILL after the
    // grace period.
    // ponytail: signals only the bash child; orphaned grandchildren may linger,
    // but destroying the pipes lets the runner report the hung test and move on.
    let killTimer;
    const timer = setTimeout(() => {
      chunks.push(Buffer.from(`timeout: killed after ${testTimeoutSec}s (DEVRITES_TEST_TIMEOUT_SEC)\n`));
      child.kill('SIGTERM');
      killTimer = setTimeout(() => {
        child.kill('SIGKILL');
        child.stdout.destroy();
        child.stderr.destroy();
      }, 30000);
    }, testTimeoutSec * 1000);
    child.on('close', (code, signal) => {
      clearTimeout(timer);
      clearTimeout(killTimer);
      const elapsed = ((Date.now() - start) / 1000).toFixed(2);
      const status = code === 0 ? 'PASS' : 'FAIL';
      const displayName = typeof test === 'string' ? test : label;
      process.stdout.write(`== ${displayName} ==\n`);
      for (const chunk of chunks) process.stdout.write(chunk);
      if (chunks.length && !String(chunks.at(-1)).endsWith('\n')) process.stdout.write('\n');
      process.stdout.write(`${status}: ${displayName} (${elapsed}s)\n`);
      if (signal) process.stdout.write(`signal: ${signal}\n`);
      if (code !== 0) failedTests.add(path);
      resolve(code === 0);
    });
  });
}

// A test whose exclusive chain is busy stays queued; the worker takes the next
// runnable test instead of idling on a held slot.
async function runBatch(batch, batchJobs) {
  const pending = [...batch];
  const busy = new Set();
  let waiters = [];
  async function worker() {
    while (pending.length) {
      const index = pending.findIndex((test) => !busy.has(exclusiveChain(test)));
      if (index < 0) {
        await new Promise((resolve) => waiters.push(resolve));
        continue;
      }
      const [test] = pending.splice(index, 1);
      const chain = exclusiveChain(test);
      if (chain) busy.add(chain);
      try {
        if (!(await runOne(test))) failed = true;
      } finally {
        busy.delete(chain);
        const woken = waiters;
        waiters = [];
        for (const resolve of woken) resolve();
      }
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
for (const name of failedTests) {
  process.stdout.write(`\nfailed: ${name}\nrerun: node scripts/run-tests.mjs ${name}\n`);
}
process.stdout.write(`\n${failed ? 'TESTS FAILED' : 'TESTS PASSED'} (${elapsed}s)\n`);
process.exit(failed ? 1 : 0);
