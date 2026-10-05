#!/usr/bin/env python3
"""Compare Fairy-owned nominal HIR with independent Go semantic HIR."""
import argparse, hashlib, json, subprocess
from pathlib import Path
from verify_hir import parse_tables, candidate_tree, oracle_tree, source_tokens

ROOT=Path(__file__).resolve().parent
def sha(path): return hashlib.sha256(Path(path).read_bytes()).hexdigest()
def compare(fixture, row):
    assert (row['exit'],row['stderr'])==(0,''),row
    source=bytes.fromhex(fixture['inputHex']); t=parse_tables(row['stdout'],source)
    assert candidate_tree(t,source)==oracle_tree(fixture),fixture['name']
    structs=fixture['hir']['Modules'][0]['Structs']
    assert len(structs)==len(t['N'])
    for actual,expected in zip(t['N'].values(),structs,strict=True):
        b=t['B'][actual['binding']]; assert b['type']==expected['Type'] and bool(actual['public'])==expected['Public'] and actual['public'] in (0,1)
        assert source[b['start']:b['end']].decode('ascii')==expected['Name']
        assert len(expected['Fields'] or [])==actual['field_count']
        for i,field in enumerate(expected['Fields'] or []):
            f=t['D'][actual['field_start']+i+1]
            span=field['Span']; start,end=span['StartByte'],span['EndByte']
            tokens=source_tokens(source[start:end]); name,tstart,tend=tokens[-2]
            assert (name.decode('ascii'),start+tstart,start+tend)==(field['Name'],f['start'],f['end'])
            assert f['type']==field['Type'] and f['owner']==actual['id']

def main():
    parser=argparse.ArgumentParser(); parser.add_argument('--images',nargs=2,required=True); parser.add_argument('--reference',required=True); parser.add_argument('--receipt',required=True)
    args=parser.parse_args(); images=[Path(p).resolve() for p in args.images]; reference=Path(args.reference).resolve(); receipt=Path(args.receipt).resolve(); receipt.parent.mkdir(parents=True,exist_ok=True)
    data=json.loads(reference.read_text(encoding='utf-8')); assert data['schema']=='fairyc.nominal-hir-oracle/v1'
    rows=[]
    for fixture in data['fixtures']:
        path=receipt.parent/(fixture['name']+'.fy'); path.write_bytes(bytes.fromhex(fixture['inputHex'])); digest=sha(path)
        for image in images:
            argv=[str(image),'--hir-file',str(path)]
            p=subprocess.run(argv,cwd=ROOT,capture_output=True,timeout=30)
            row=dict(case=fixture['name'],argv=argv,cwd=str(ROOT),exit=p.returncode,stdout=p.stdout.decode('utf-8'),stderr=p.stderr.decode('utf-8'),inputSHA256=digest,unchanged=sha(path)==digest)
            assert row['unchanged']; compare(fixture,row); rows.append(row)
    result=dict(schema='fairyc.nominal-hir-cli-verification/v1',passed=True,semanticComparisons=len(rows),images=[dict(path=str(p),sha256=sha(p)) for p in images],reference=dict(path=str(reference),sha256=sha(reference)),sources=[dict(path=str(ROOT/p),sha256=sha(ROOT/p)) for p in ('verify_nominal_hir.py','verify_hir.py')],runs=rows)
    receipt.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n'); print(f'PASS: {len(rows)} nominal HIR semantic comparisons')
if __name__=='__main__': main()
