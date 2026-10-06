#!/usr/bin/env python3
"""Observe composite-array typing/owned HIR; executable support is a later part."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
from verify_tagged_array_cases import POSITIVES, NEGATIVES, TRAPS

ROOT=Path(__file__).resolve().parent
def sha(path): return hashlib.sha256(Path(path).read_bytes()).hexdigest()
def main():
    ap=argparse.ArgumentParser();ap.add_argument('--images',nargs='+',required=True);ap.add_argument('--receipt',required=True)
    args=ap.parse_args();images=[Path(p).resolve() for p in args.images];dest=Path(args.receipt).resolve();dest.parent.mkdir(parents=True,exist_ok=True)
    rows=[];fixtures=[];passed=False
    def run(argv,code):
        p=subprocess.run(list(map(str,argv)),cwd=ROOT,capture_output=True,encoding='utf-8',timeout=120)
        row=dict(argv=list(map(str,argv)),cwd=str(ROOT),stdout=p.stdout,stderr=p.stderr,exit=p.returncode);rows.append(row)
        assert p.returncode==code,row;return row
    try:
        for rejected,cases in [(False,POSITIVES+TRAPS),(True,NEGATIVES)]:
            for name,source,expected in cases:
                path=dest.parent/(name+'.fy');path.write_text('module main;'+source,encoding='utf-8',newline='\n');digest=sha(path)
                fixtures.append(dict(name=name,source=path.read_text(encoding='utf-8'),sha256=digest,rejected=rejected))
                outputs={mode:[] for mode in ['--check-types-file','--hir-file']}
                for image in images:
                    for mode in outputs:
                        row=run([image,mode,path],2 if rejected else 0)
                        if rejected: assert not row['stdout'] and expected in row['stderr'],row
                        else: assert row['stdout'] and not row['stderr'],row
                        outputs[mode].append(row['stdout'])
                    if not rejected:
                        for mode in ['--mir-file','--c-file']:
                            row=run([image,mode,path],2);assert not row['stdout'] and row['stderr'],row
                assert sha(path)==digest
                assert all(len(set(v))==1 for v in outputs.values())
        passed=True
    finally:
        dest.write_text(json.dumps(dict(schema='fairyc.tagged-array-hir/v1',unit=14,part='array-types-hir',passed=passed,
            images=[dict(path=str(p),sha256=sha(p)) for p in images],fixtures=fixtures,cli=rows,
            source=[dict(path=str(p.relative_to(ROOT)),sha256=sha(p)) for p in sorted((ROOT/'src').glob('*.fy'))],
            skipped=['Composite-array MIR and C execution; kept fail-closed until next part','Linux consolidated batch','performance','stage1/stage2','full repository suite']),ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n')
    print(f'PASS: composite array HIR; {len(fixtures)} fixtures; {len(rows)} CLI observations; execution pending')
if __name__=='__main__': main()
