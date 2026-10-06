#!/usr/bin/env python3
"""Observe unit-15 resource source/owned HIR; native ownership execution follows."""
import argparse,hashlib,json,subprocess
from pathlib import Path
from verify_resource_type_cases import POSITIVES,NEGATIVES
ROOT=Path(__file__).resolve().parent
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def main():
    ap=argparse.ArgumentParser();ap.add_argument('--images',nargs='+',required=True);ap.add_argument('--receipt',required=True);args=ap.parse_args()
    images=[Path(p).resolve() for p in args.images];dest=Path(args.receipt).resolve();dest.parent.mkdir(parents=True,exist_ok=True)
    rows=[];fixtures=[];passed=False
    def run(image,mode,path,expected):
        argv=list(map(str,[image,mode,path]));p=subprocess.run(argv,cwd=ROOT,capture_output=True,encoding='utf-8',timeout=120)
        row=dict(argv=argv,cwd=str(ROOT),exit=p.returncode,stdout=p.stdout,stderr=p.stderr);rows.append(row);assert p.returncode==expected,row;return row
    try:
        for rejected,cases in [(False,POSITIVES),(True,NEGATIVES)]:
            for name,source,fact in cases:
                path=dest.parent/(name+'.fy');data=('module main;'+source).encode('utf-8');path.write_bytes(data);digest=sha(path)
                fixtures.append(dict(name=name,inputHex=data.hex(),sha256=digest,rejected=rejected,expected=fact))
                outputs=[]
                for image in images:
                    for mode in ['--check-types-file','--hir-file','--mir-file','--c-file']:
                        expected=2 if rejected or mode in ['--mir-file','--c-file'] else 0
                        row=run(image,mode,path,expected)
                        if expected==2:
                            assert not row['stdout'] and row['stderr'],row
                            if rejected:assert fact in row['stderr'],row
                            else:assert 'unsupported' in row['stderr'] or 'invalid' in row['stderr'],row
                        else:
                            assert row['stdout'] and not row['stderr'],row
                            if mode=='--hir-file':
                                text=row['stdout'];assert text.startswith('fairyc.resource-hir/v1\n'),row;outputs.append(text);found=set();resource=False
                                for line in text.splitlines():
                                    fields=line.split('\t')
                                    if fields[0]=='O' and 98304<=int(fields[1])<=98815:assert fields[2:]==['0','1'];resource=True
                                    if fields[0]=='D':
                                        field=data[int(fields[4]):int(fields[5])].decode('utf-8')
                                        if field in fact:assert tuple(map(int,fields[-2:]))==tuple(fact[field]),(name,field,line);found.add(field)
                                assert resource and found==set(fact),(name,found,fact)
                assert len(set(outputs))<=1 and sha(path)==digest
        passed=True
    finally:
        dest.write_text(json.dumps(dict(schema='fairyc.resource-type-hir/v1',unit=15,passed=passed,images=[dict(path=str(p),sha256=sha(p)) for p in images],fixtures=fixtures,cli=rows,sources=[dict(path=str(p.relative_to(ROOT)),sha256=sha(p)) for p in sorted((ROOT/'src').glob('*.fy'))],executionBoundary='Resource MIR/C stays closed until the subsequent move/cleanup units; API native tests execute compiler logic, not source resource programs.'),ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n')
    print(f'PASS resource types/HIR: {len(fixtures)} fixtures; {len(rows)} CLI observations')
if __name__=='__main__':main()
