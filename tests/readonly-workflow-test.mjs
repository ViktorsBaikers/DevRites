import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';

const source = (await readFile(new URL('../pack/.claude/workflows/devrites-readonly-review.js', import.meta.url), 'utf8'))
  .replace('export const meta =', 'const meta =');

const cleanReview = { outcome: 'no-findings', inspected: ['candidate'], findings: [], gaps: [] };
const finding = {
  severity: 'Important',
  file: 'src/example.js',
  line: 1,
  summary: 'reachable defect',
  evidence: 'line evidence',
  impact: 'wrong result',
};
const discovery = { outcome: 'ready', candidate: 'digest', files: ['src/example.js'], inputs: ['spec'], gaps: [] };
const complete = { complete: true, missing_modalities: [], unread_inputs: [], unverified_claims: [] };

async function run(overrides = {}) {
  const responses = {
    'discover:candidate': discovery,
    'review:code': cleanReview,
    'review:spec': cleanReview,
    'review:tests': cleanReview,
    'review:security': cleanReview,
    'verify:findings': { verdicts: [] },
    'complete:coverage': complete,
    ...overrides,
  };
  const context = {
    args: { candidate: 'digest', objective: 'review' },
    phase() {},
    log() {},
    parallel: thunks => Promise.all(thunks.map(thunk => thunk())),
    agent: async (_prompt, options) => responses[options.label],
  };
  return vm.runInNewContext(`(async () => {\n${source}\n})()`, context);
}

assert.equal((await run()).outcome, 'reviewed');
assert.equal((await run({ 'verify:findings': null })).outcome, 'gap');
assert.equal((await run({ 'review:security': null })).outcome, 'gap');
assert.equal((await run({
  'review:security': { outcome: 'gap', inspected: [], findings: [], gaps: ['unavailable'] },
})).outcome, 'gap');
assert.equal((await run({
  'review:code': { outcome: 'findings', inspected: ['candidate'], findings: [finding], gaps: [] },
  'verify:findings': { verdicts: [] },
})).outcome, 'gap');
assert.equal((await run({
  'review:code': { outcome: 'findings', inspected: ['candidate'], findings: [finding], gaps: [] },
  'verify:findings': { verdicts: [{ key: 'src/example.js:1:reachable defect', verdict: 'gap', reason: 'uncertain' }] },
})).outcome, 'gap');
assert.equal((await run({
  'complete:coverage': { complete: true, missing_modalities: [], unread_inputs: [], unverified_claims: ['claim'] },
})).outcome, 'gap');

console.log('readonly-workflow-test: PASS');
