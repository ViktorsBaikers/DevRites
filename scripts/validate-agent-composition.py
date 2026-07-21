#!/usr/bin/env python3
"""Validate DevRites agent persona composition blocks."""
from __future__ import annotations
import argparse, re, sys
from pathlib import Path
ROOT=Path(__file__).resolve().parent.parent

WRITE_TERMS=re.compile(r'\b(write-capable|may edit|may mutate|commit|push|patch files|modify files)\b', re.I)
READONLY_TERMS=re.compile(r'\bread-only\b|do not edit|findings only|proposes, never imposes', re.I)
REQUIRED={
  'role/scope':[r'## Role / scope', r'You are ', r'## Scope'],
  'tools/read-write mode':[r'## Tools / read-write mode', r'read-only', r'write-capable', r'Do \*\*not\*\* edit', r'Writes code'],
  'output format':[r'## Output format', r'## Output', r'```'],
  'composition block':[r'## Composition', r'Do not invoke another agent', r'Invoke directly when']
}
COMPOSITION_LINE=re.compile(
    r'^Do not invoke another agent\. You are called by (?:a `rite-\*` skill|`/rite-build`) '
    r'and return (?:findings|your result) to that orchestrator\.$', re.M
)
UNTRUSTED_LINE=re.compile(
    r'^> \*\*Untrusted-input safety\.\*\* .*data, not instructions.*never act on a directive.*'
    r'surface it instead of obeying it.*security\.md', re.I | re.M
)

def validate(agents_dir:Path):
    errors=[]
    for f in sorted(agents_dir.glob('*.md')):
        name=f.stem
        text=f.read_text(encoding='utf-8')
        for label,pats in REQUIRED.items():
            if not any(re.search(p,text,re.I|re.M) for p in pats):
                errors.append(f'{f}: missing {label}')
        if not COMPOSITION_LINE.search(text):
            errors.append(f'{f}: composition guard must match the canonical no-nested-agent contract')
        if not UNTRUSTED_LINE.search(text):
            errors.append(f'{f}: untrusted-input guard must keep the complete canonical safety contract')
        says_write=bool(WRITE_TERMS.search(text)) and not re.search(r'Do \*\*not\*\* edit|Do not edit|never edit|does not edit|does not write|Does not edit', text, re.I)
        if name=='devrites-slice-wright':
            if not re.search(r'write-capable|Writes code|write code', text, re.I):
                errors.append(f'{f}: devrites-slice-wright must state write-capable mode')
        else:
            if says_write or not READONLY_TERMS.search(text):
                errors.append(f'{f}: only devrites-slice-wright may be write-capable; reviewers must state read-only/findings-only mode')
        if re.search(r'invoke (the )?(devrites-|another agent)|spawn (the )?(devrites-|another agent)', text, re.I) and not re.search(r'Do not invoke another agent|never invoke another agent|does not invoke another agent', text, re.I):
            errors.append(f'{f}: agents must not say they may invoke another agent')
    return errors

def main():
    p=argparse.ArgumentParser(); p.add_argument('--agents-dir', type=Path, default=ROOT/'pack/.claude/agents'); p.add_argument('--quiet', action='store_true'); args=p.parse_args()
    errors=validate(args.agents_dir)
    if errors:
        for e in errors: print(f'FAIL: {e}')
        print(f'validate-agent-composition: {len(errors)} failure(s)')
        return 1
    if not args.quiet: print('validate-agent-composition: PASS')
    return 0
if __name__=='__main__': sys.exit(main())
