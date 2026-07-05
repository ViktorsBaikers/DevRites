#!/usr/bin/env node
import fs from 'node:fs';

const [, , mode, ...args] = process.argv;
let existingPath;
let devritesPath;
let outPath;

if (mode === 'merge') {
  [existingPath, devritesPath, outPath] = args;
} else if (mode === 'strip') {
  [existingPath, outPath] = args;
}

if (!['merge', 'strip'].includes(mode) || !existingPath || !outPath || (mode === 'merge' && !devritesPath)) {
  console.error('usage: merge-codex-hooks.mjs merge EXISTING DEVRITES OUT');
  console.error('   or: merge-codex-hooks.mjs strip EXISTING OUT');
  process.exit(2);
}

function readJson(path) {
  return JSON.parse(fs.readFileSync(path, 'utf8'));
}

function isDevRitesHook(value) {
  const s = JSON.stringify(value);
  return s.includes('.codex/hooks/devrites-') || s.includes('DevRites:') || s.includes('DEVRITES_');
}

function stripDevRitesHooks(config) {
  if (!config || typeof config !== 'object' || Array.isArray(config)) return {};
  const next = { ...config };
  if (typeof next.$comment === 'string' && next.$comment.includes('DevRites hooks for Codex')) {
    delete next.$comment;
  }
  const hooks = next.hooks && typeof next.hooks === 'object' && !Array.isArray(next.hooks)
    ? { ...next.hooks }
    : {};

  for (const [event, entries] of Object.entries(hooks)) {
    const kept = Array.isArray(entries) ? entries.filter((entry) => !isDevRitesHook(entry)) : entries;
    if (Array.isArray(kept) && kept.length === 0) {
      delete hooks[event];
    } else {
      hooks[event] = kept;
    }
  }

  if (Object.keys(hooks).length) {
    next.hooks = hooks;
  } else {
    delete next.hooks;
  }
  return next;
}

const existing = stripDevRitesHooks(readJson(existingPath));

if (mode === 'merge') {
  const devrites = readJson(devritesPath);
  const hooks = existing.hooks && typeof existing.hooks === 'object' && !Array.isArray(existing.hooks)
    ? existing.hooks
    : {};
  const devritesHooks = devrites.hooks && typeof devrites.hooks === 'object' && !Array.isArray(devrites.hooks)
    ? devrites.hooks
    : {};

  for (const [event, entries] of Object.entries(devritesHooks)) {
    hooks[event] = [
      ...(Array.isArray(hooks[event]) ? hooks[event] : []),
      ...(Array.isArray(entries) ? entries : []),
    ];
  }

  existing.hooks = hooks;
}

fs.writeFileSync(outPath, `${JSON.stringify(existing, null, 2)}\n`);
