#!/usr/bin/env python3
"""Validate Claude/Codex command parity for public DevRites skills."""
from __future__ import annotations
import argparse, re, sys
from pathlib import Path
ROOT = Path(__file__).resolve().parent.parent


def fm(text):
    if not text.startswith('---\n'): return {}
    end = text.find('\n---', 4)
    out = {}
    if end == -1: return out
    for line in text[4:end].splitlines():
        if not line.strip() or ':' not in line or line.startswith((' ', '\t')): continue
        k,v = line.split(':',1); out[k.strip()] = v.strip().strip('"\'')
    return out


def public_rites(skills_dir: Path):
    names=[]
    for f in sorted(skills_dir.glob('*/SKILL.md')):
        meta=fm(f.read_text(encoding='utf-8'))
        name=meta.get('name', f.parent.name)
        if meta.get('user-invocable')=='true' and name.startswith('rite-') and name!='rite':
            names.append(name)
    return names


def main():
    p=argparse.ArgumentParser()
    p.add_argument('--skills-dir', type=Path, default=ROOT/'pack/.claude/skills')
    p.add_argument('--docs-skills', type=Path, default=ROOT/'docs/skills.md')
    p.add_argument('--docs-command-map', type=Path, default=ROOT/'docs/command-map.md')
    p.add_argument('--readme', type=Path, default=ROOT/'README.md')
    p.add_argument('--generated-root', type=Path, default=ROOT/'pack/generated')
    p.add_argument('--quiet', action='store_true')
    args=p.parse_args()
    errors=[]
    skills=public_rites(args.skills_dir)
    docs_skills=args.docs_skills.read_text(encoding='utf-8') if args.docs_skills.exists() else ''
    docs_map=args.docs_command_map.read_text(encoding='utf-8') if args.docs_command_map.exists() else ''
    readme=args.readme.read_text(encoding='utf-8') if args.readme.exists() else ''
    all_docs='\n'.join([docs_skills, docs_map, readme])
    if 'npx devrites' not in all_docs:
        errors.append('docs missing npx devrites distribution contract')
    for line in all_docs.splitlines():
        if re.search(r'\b(install|installed|installing)\s+via\s+.*\b(plugin|marketplace)\b', line, re.I) and 'not' not in line.lower():
            errors.append('docs imply plugin distribution instead of npx install')
            break
    for name in skills:
        verb=name.removeprefix('rite-')
        needle=f'/rite-{verb}'
        if needle not in docs_map:
            errors.append(f'docs/command-map Claude direct: missing Claude command {needle} for {name}')
        if args.generated_root.exists():
            for rel in [f'claude/skills/{name}/SKILL.md', f'codex/skills/{name}/SKILL.md']:
                if not (args.generated_root/rel).exists():
                    errors.append(f'generated artifact missing {rel}')
    if errors:
        for e in errors: print(f'FAIL: {e}')
        print(f'validate-command-parity: {len(errors)} failure(s)')
        return 1
    if not args.quiet: print('validate-command-parity: PASS')
    return 0
if __name__=='__main__': sys.exit(main())
