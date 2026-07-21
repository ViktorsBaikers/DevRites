#!/usr/bin/env node
// Deterministic fixture scorer for DevRites evals: no LLM judge, no deps.
import { readFileSync } from 'node:fs';

function norm(s) {
  return String(s || '')
    .normalize('NFKC')
    .toLowerCase()
    .replace(/[\p{P}\p{S}]+/gu, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function hit(text, kw) {
  const t = norm(text);
  const k = norm(kw);
  if (!k) return false;
  if (t.includes(k)) return true;
  const parts = k.split(' ').filter(Boolean);
  return parts.length > 1 && parts.every((p) => t.includes(p));
}

export function matchFinding(output, keywords) {
  const kws = Array.isArray(keywords) ? keywords : [];
  const matches = kws.filter((k) => hit(output, k)).length;
  const needed = Math.max(2, Math.ceil(kws.length * 0.4));
  return { matches, needed, matched: kws.length > 0 && matches >= Math.min(needed, kws.length) };
}

export function score(output, groundTruth) {
  const findings = Array.isArray(groundTruth?.findings) ? groundTruth.findings : [];
  const rows = findings.map((f) => ({ id: f.id || f.name || '', ...matchFinding(output, f.keywords || []) }));
  const found = rows.filter((r) => r.matched).length;
  return { found, total: findings.length, score: findings.length ? found / findings.length : 1, rows };
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const [, , outputPath, truthPath] = process.argv;
  if (!outputPath || !truthPath) {
    console.error('usage: node scripts/eval-scorer.mjs <model-output.txt> <ground-truth.json>');
    process.exit(2);
  }
  const result = score(readFileSync(outputPath, 'utf8'), JSON.parse(readFileSync(truthPath, 'utf8')));
  console.log(JSON.stringify(result, null, 2));
  process.exit(result.found === result.total ? 0 : 1);
}
